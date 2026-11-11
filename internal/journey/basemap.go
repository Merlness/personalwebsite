package journey

import (
	"fmt"
	"math"
)

// The base map shows the western United States with real state borders. The
// lines live in basemap_data.go, which a script generates from the us-atlas
// data set (US Census boundaries) with the projection below.

const (
	mapLonMin = -125.6
	mapLonMax = -100.4
	mapLatMin = 30.9
	mapLatMax = 49.6
	mapScale  = 40.0 // map units per degree of latitude
	mapMidLat = 40.0 // latitude where east-west distance is true to scale
)

var lonScale = math.Cos(mapMidLat*math.Pi/180) * mapScale

// Project turns a latitude and longitude into map units (x right, y down).
func Project(lat, lon float64) (float64, float64) {
	return (lon - mapLonMin) * lonScale, (mapLatMax - lat) * mapScale
}

type MapLabel struct {
	X, Y float64
	Text string
}

type Basemap struct {
	Width, Height float64
	Outline       []string // coast and country borders, the heaviest lines
	Borders       []string // lines between states
	Labels        []MapLabel
}

func (b Basemap) ViewBox() string {
	return fmt.Sprintf("0 0 %.0f %.0f", b.Width, b.Height)
}

// WesternUS returns the map used by road trips in the American West.
func WesternUS() Basemap {
	w, _ := Project(mapLatMax, mapLonMax)
	_, h := Project(mapLatMin, mapLonMin)

	label := func(lon, lat float64, text string) MapLabel {
		x, y := Project(lat, lon)
		return MapLabel{X: round1(x), Y: round1(y), Text: text}
	}
	labels := []MapLabel{
		label(-120.3, 47.3, "WA"), label(-120.6, 43.9, "OR"), label(-120.9, 37.9, "CA"),
		label(-116.8, 39.4, "NV"), label(-114.7, 43.5, "ID"), label(-109.5, 47.0, "MT"),
		label(-107.5, 43.0, "WY"), label(-111.7, 39.4, "UT"), label(-105.4, 38.4, "CO"),
		label(-111.9, 33.9, "AZ"), label(-106.0, 34.4, "NM"), label(-101.6, 44.5, "SD"),
		label(-101.6, 47.4, "ND"), label(-101.6, 41.8, "NE"),
	}

	return Basemap{Width: math.Round(w), Height: math.Round(h), Outline: westOutline, Borders: westBorders, Labels: labels}
}
