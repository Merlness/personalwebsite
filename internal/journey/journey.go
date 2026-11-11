// Package journey loads a trip (an ordered list of stops with dates, places,
// and photos) and turns it into the geometry the scroll map needs.
package journey

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/adrg/frontmatter"
	"github.com/yuin/goldmark"
)

var ErrNotFound = errors.New("journey not found")

// roadFactor turns straight-line distance into a road estimate. Mountain and
// coast roads wander, so the real number is higher than the great circle.
const roadFactor = 1.25

type Palette struct {
	Paper   string `yaml:"paper"`
	Ink     string `yaml:"ink"`
	Accent  string `yaml:"accent"`
	Accent2 string `yaml:"accent2"`
}

type Stop struct {
	Name    string   `yaml:"name"`
	Region  string   `yaml:"region"`
	Date    string   `yaml:"date"`
	Lat     float64  `yaml:"lat"`
	Lon     float64  `yaml:"lon"`
	ElevFt  int      `yaml:"elev_ft"`
	Note    string   `yaml:"note"`
	Photos  []string `yaml:"photos"`
	Scene   string   `yaml:"scene"`
	Chapter string   `yaml:"chapter"`
	Colors  Palette  `yaml:"colors"`
	Icon    string   `yaml:"icon"`
}

// Stat is one extra fact for the trip header, written by hand in the file.
type Stat struct {
	Label string `yaml:"label"`
	Value string `yaml:"value"`
}

type meta struct {
	Title    string  `yaml:"title"`
	Subtitle string  `yaml:"subtitle"`
	Summary  string  `yaml:"summary"`
	Gallery  string  `yaml:"gallery"`
	Palette  Palette `yaml:"palette"`
	Odometer int     `yaml:"odometer_miles"`
	Stats    []Stat  `yaml:"stats"`
	Stops    []Stop  `yaml:"stops"`
}

// StopView is a stop plus everything the page needs to place and count it.
type StopView struct {
	Stop
	Index     int
	X, Y      float64
	PathLen   float64 // route length in map units from the first stop to here
	Miles     int     // estimated road miles from the first stop to here
	Day       int     // 1 on the first date of the trip
	DateLabel string  // "Jun 7"
	PhotoURLs []Photo
	Theme     Palette // colors while this stop is on screen
}

type Photo struct {
	Thumb string
	Full  string
}

type Journey struct {
	Slug        string
	Title       string
	Subtitle    string
	Summary     string
	Intro       string // HTML from the Markdown body
	Gallery     string
	Palette     Palette
	Stops       []StopView
	Start       time.Time
	End         time.Time
	Days        int
	Miles       int
	Measured    bool // true when Miles is the odometer reading, not an estimate
	States      int
	HighFt      int
	HighName    string
	Stats       []Stat
	RouteCredit string // who made the road lines, when the route follows roads
	RoutePath   string
	RouteLen    float64
	Map         Basemap
}

func (j Journey) DateRange() string {
	if j.Start.IsZero() {
		return ""
	}
	return fmt.Sprintf("%s to %s", j.Start.Format("January 2"), j.End.Format("January 2, 2006"))
}

type Service interface {
	All() ([]Journey, error)
	Get(slug string) (Journey, error)
}

type filesystemService struct {
	dir            string
	photoURLPrefix string
}

// NewFilesystemService reads journeys from Markdown files in dir. Photo names
// in a journey resolve to photoURLPrefix/<gallery>/<name>.
func NewFilesystemService(dir, photoURLPrefix string) Service {
	return &filesystemService{dir: dir, photoURLPrefix: photoURLPrefix}
}

func (s *filesystemService) All() ([]Journey, error) {
	if s.dir == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var journeys []Journey
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		j, err := s.parse(filepath.Join(s.dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", entry.Name(), err)
		}
		journeys = append(journeys, j)
	}
	sort.Slice(journeys, func(a, b int) bool { return journeys[a].Start.After(journeys[b].Start) })
	return journeys, nil
}

func (s *filesystemService) Get(slug string) (Journey, error) {
	if s.dir == "" || slug == "" || strings.ContainsAny(slug, "/\\") || strings.Contains(slug, "..") {
		return Journey{}, ErrNotFound
	}
	path := filepath.Join(s.dir, slug+".md")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return Journey{}, ErrNotFound
		}
		return Journey{}, err
	}
	return s.parse(path)
}

func (s *filesystemService) parse(path string) (Journey, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Journey{}, err
	}
	var m meta
	body, err := frontmatter.Parse(bytes.NewReader(raw), &m)
	if err != nil {
		return Journey{}, err
	}
	if len(m.Stops) == 0 {
		return Journey{}, errors.New("journey has no stops")
	}
	var intro bytes.Buffer
	if err := goldmark.Convert(body, &intro); err != nil {
		return Journey{}, err
	}

	name := filepath.Base(path)
	j := Journey{
		Slug:     strings.TrimSuffix(name, filepath.Ext(name)),
		Title:    m.Title,
		Subtitle: m.Subtitle,
		Summary:  m.Summary,
		Intro:    intro.String(),
		Gallery:  m.Gallery,
		Palette:  m.Palette,
		Map:      WesternUS(),
	}

	roads, roadCredit := loadRoads(strings.TrimSuffix(path, filepath.Ext(path))+".route.json", len(m.Stops))
	j.RouteCredit = roadCredit
	theme := m.Palette
	states := map[string]bool{}
	var path2 strings.Builder
	var prev Stop
	var lenSoFar, milesSoFar float64
	for i, stop := range m.Stops {
		date, err := time.Parse("2006-01-02", stop.Date)
		if err != nil {
			return Journey{}, fmt.Errorf("stop %q: bad date %q", stop.Name, stop.Date)
		}
		if i == 0 {
			j.Start = date
		}
		j.End = date

		x, y := Project(stop.Lat, stop.Lon)
		if i == 0 {
			fmt.Fprintf(&path2, "M%.1f %.1f", x, y)
		} else {
			px, py := Project(prev.Lat, prev.Lon)
			if roads != nil {
				// Follow the saved driving line for this leg.
				leg := roads[i-1]
				for _, point := range leg.Points[1:] {
					rx, ry := Project(point[1], point[0])
					lenSoFar += math.Hypot(rx-px, ry-py)
					fmt.Fprintf(&path2, " L%.1f %.1f", rx, ry)
					px, py = rx, ry
				}
				milesSoFar += leg.Miles
			} else {
				lenSoFar += math.Hypot(x-px, y-py)
				milesSoFar += HaversineMiles(prev.Lat, prev.Lon, stop.Lat, stop.Lon) * roadFactor
				fmt.Fprintf(&path2, " L%.1f %.1f", x, y)
			}
		}
		prev = stop

		view := StopView{
			Stop:      stop,
			Index:     i,
			X:         round1(x),
			Y:         round1(y),
			PathLen:   round1(lenSoFar),
			Miles:     int(math.Round(milesSoFar/10) * 10),
			Day:       int(date.Sub(j.Start).Hours()/24) + 1,
			DateLabel: date.Format("Jan 2"),
		}
		// A stop with its own colors changes the theme. A stop with none
		// keeps the colors of the stop before it.
		theme = theme.merge(stop.Colors)
		view.Theme = theme
		if stop.ElevFt > j.HighFt {
			j.HighFt, j.HighName = stop.ElevFt, stop.Name
		}
		for _, photo := range stop.Photos {
			base := s.photoURLPrefix + "/" + m.Gallery + "/" + photo
			view.PhotoURLs = append(view.PhotoURLs, Photo{Thumb: base + "_w600.jpg", Full: base + "_w1600.jpg"})
		}
		j.Stops = append(j.Stops, view)

		if abbr := stateOf(stop.Region); abbr != "" {
			states[abbr] = true
		}
	}

	j.RoutePath = path2.String()
	j.RouteLen = round1(lenSoFar)
	j.Miles = j.Stops[len(j.Stops)-1].Miles
	j.Stats = m.Stats
	if m.Odometer > 0 && j.Miles > 0 {
		// The straight-line estimate gives each leg its share of the trip.
		// The odometer gives the true total.
		scale := float64(m.Odometer) / float64(j.Miles)
		for i := range j.Stops {
			j.Stops[i].Miles = int(math.Round(float64(j.Stops[i].Miles)*scale/10) * 10)
		}
		j.Stops[len(j.Stops)-1].Miles = m.Odometer
		j.Miles = m.Odometer
		j.Measured = true
	}
	j.Days = j.Stops[len(j.Stops)-1].Day
	j.States = len(states)
	return j, nil
}

type roadLeg struct {
	Miles  float64      `json:"miles"`
	Points [][2]float64 `json:"points"` // lon, lat
}

// loadRoads reads the optional driving lines saved next to a journey. It
// returns nil when the file is missing or does not match the stops, and the
// route then falls back to straight lines.
func loadRoads(file string, stops int) ([]roadLeg, string) {
	raw, err := os.ReadFile(file)
	if err != nil {
		return nil, ""
	}
	var data struct {
		Source string    `json:"source"`
		Legs   []roadLeg `json:"legs"`
	}
	if json.Unmarshal(raw, &data) != nil || len(data.Legs) != stops-1 {
		return nil, ""
	}
	for _, leg := range data.Legs {
		if len(leg.Points) < 2 || leg.Miles <= 0 {
			return nil, ""
		}
	}
	return data.Legs, data.Source
}

// merge returns p with every non-empty color of over applied.
func (p Palette) merge(over Palette) Palette {
	if over.Paper != "" {
		p.Paper = over.Paper
	}
	if over.Ink != "" {
		p.Ink = over.Ink
	}
	if over.Accent != "" {
		p.Accent = over.Accent
	}
	if over.Accent2 != "" {
		p.Accent2 = over.Accent2
	}
	return p
}

// stateOf takes the last comma-separated part of a region ("Aspen, CO").
func stateOf(region string) string {
	parts := strings.Split(region, ",")
	return strings.TrimSpace(parts[len(parts)-1])
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }

// HaversineMiles is the great-circle distance between two points.
func HaversineMiles(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusMiles = 3958.8
	rad := math.Pi / 180
	dLat := (lat2 - lat1) * rad
	dLon := (lon2 - lon1) * rad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return 2 * earthRadiusMiles * math.Asin(math.Sqrt(a))
}
