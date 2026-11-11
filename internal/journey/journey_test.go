package journey

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sample = `---
title: "Test Trip"
gallery: "Trip"
palette:
  paper: "#fff"
stops:
  - name: "Phoenix"
    region: "Maricopa County, AZ"
    date: "2020-06-07"
    lat: 33.45
    lon: -112.07
    elev_ft: 1086
    photos: ["A1"]
  - name: "Flagstaff"
    region: "Coconino County, AZ"
    date: "2020-06-07"
    lat: 35.20
    lon: -111.65
  - name: "Moab"
    region: "Grand County, UT"
    date: "2020-06-09"
    lat: 38.57
    lon: -109.55
---

Hello *there*.
`

func writeSample(t *testing.T) Service {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "test-trip.md"), []byte(sample), 0644); err != nil {
		t.Fatal(err)
	}
	return NewFilesystemService(dir, "/assets/portfolio")
}

func TestGetBuildsTheRoute(t *testing.T) {
	trip, err := writeSample(t).Get("test-trip")
	if err != nil {
		t.Fatal(err)
	}
	if trip.Title != "Test Trip" || trip.Slug != "test-trip" {
		t.Errorf("title or slug wrong: %q %q", trip.Title, trip.Slug)
	}
	if len(trip.Stops) != 3 {
		t.Fatalf("want 3 stops, got %d", len(trip.Stops))
	}
	if trip.Days != 3 {
		t.Errorf("Jun 7 to Jun 9 is 3 days, got %d", trip.Days)
	}
	if trip.States != 2 {
		t.Errorf("want 2 states (AZ, UT), got %d", trip.States)
	}
	if !strings.Contains(trip.Intro, "<em>there</em>") {
		t.Errorf("intro Markdown not rendered: %q", trip.Intro)
	}
	if !strings.HasPrefix(trip.RoutePath, "M") || strings.Count(trip.RoutePath, "L") != 2 {
		t.Errorf("route path wrong: %q", trip.RoutePath)
	}

	first, second, third := trip.Stops[0], trip.Stops[1], trip.Stops[2]
	if first.PathLen != 0 || first.Miles != 0 || first.Day != 1 {
		t.Errorf("first stop must start at zero on day 1: %+v", first)
	}
	if !(second.PathLen > 0 && third.PathLen > second.PathLen) {
		t.Errorf("path length must grow: %v %v", second.PathLen, third.PathLen)
	}
	if third.PathLen != trip.RouteLen || third.Miles != trip.Miles {
		t.Errorf("totals must equal the last stop")
	}
	// Phoenix to Flagstaff is about 123 miles in a straight line, 145 by road.
	if second.Miles < 130 || second.Miles > 170 {
		t.Errorf("Phoenix to Flagstaff estimate out of range: %d", second.Miles)
	}
	if third.Day != 3 || third.DateLabel != "Jun 9" {
		t.Errorf("third stop day or label wrong: %d %q", third.Day, third.DateLabel)
	}
	if len(first.PhotoURLs) != 1 || first.PhotoURLs[0].Thumb != "/assets/portfolio/Trip/A1_w600.jpg" {
		t.Errorf("photo URL wrong: %+v", first.PhotoURLs)
	}
	// North is up and east is right.
	if !(second.Y < first.Y && third.X > second.X) {
		t.Errorf("projection is flipped: %+v %+v %+v", first, second, third)
	}
}

func TestGetRejectsBadSlugs(t *testing.T) {
	svc := writeSample(t)
	for _, slug := range []string{"", "nope", "../test-trip", "a/b"} {
		if _, err := svc.Get(slug); err != ErrNotFound {
			t.Errorf("slug %q: want ErrNotFound, got %v", slug, err)
		}
	}
}

func TestAllWithNoFolderIsEmpty(t *testing.T) {
	for _, dir := range []string{"", filepath.Join(t.TempDir(), "missing")} {
		trips, err := NewFilesystemService(dir, "/x").All()
		if err != nil || len(trips) != 0 {
			t.Errorf("dir %q: want empty and no error, got %d, %v", dir, len(trips), err)
		}
	}
}

func TestBadDateIsAnError(t *testing.T) {
	dir := t.TempDir()
	bad := strings.Replace(sample, `"2020-06-09"`, `"June 9"`, 1)
	os.WriteFile(filepath.Join(dir, "bad.md"), []byte(bad), 0644)
	if _, err := NewFilesystemService(dir, "/x").Get("bad"); err == nil {
		t.Error("want an error for a bad date")
	}
}

func TestRealRoadTripLoads(t *testing.T) {
	trip, err := NewFilesystemService("../../content/journeys", "/assets/portfolio").Get("road-trip-2020")
	if err != nil {
		t.Fatal(err)
	}
	for _, stop := range trip.Stops {
		if stop.X < 0 || stop.X > trip.Map.Width || stop.Y < 0 || stop.Y > trip.Map.Height {
			t.Errorf("%s is off the map: %.1f, %.1f", stop.Name, stop.X, stop.Y)
		}
		for _, photo := range stop.Photos {
			file := filepath.Join("../../content/portfolio_optimized", trip.Gallery, photo+"_w600.jpg")
			if _, err := os.Stat(file); err != nil {
				t.Errorf("%s: photo %s is missing", stop.Name, photo)
			}
		}
	}
}

func TestOdometerAndStopColors(t *testing.T) {
	dir := t.TempDir()
	withExtras := strings.Replace(sample, "gallery: \"Trip\"\n", "gallery: \"Trip\"\nodometer_miles: 900\n", 1)
	withExtras = strings.Replace(withExtras, "    lon: -111.65\n", "    lon: -111.65\n    elev_ft: 6900\n    colors:\n      paper: \"#eef\"\n      accent: \"#06c\"\n", 1)
	os.WriteFile(filepath.Join(dir, "t.md"), []byte(withExtras), 0644)
	trip, err := NewFilesystemService(dir, "/x").Get("t")
	if err != nil {
		t.Fatal(err)
	}
	if !trip.Measured || trip.Miles != 900 || trip.Stops[2].Miles != 900 {
		t.Errorf("odometer must set the total: %+v", trip.Miles)
	}
	if trip.Stops[0].Miles != 0 || !(trip.Stops[1].Miles > 0 && trip.Stops[1].Miles < 900) {
		t.Errorf("legs must keep their share: %d %d", trip.Stops[0].Miles, trip.Stops[1].Miles)
	}
	if trip.HighFt != 6900 || trip.HighName != "Flagstaff" {
		t.Errorf("high point wrong: %d %s", trip.HighFt, trip.HighName)
	}
	// Stop 1 has the trip colors, stop 2 sets its own, stop 3 keeps stop 2's.
	if trip.Stops[0].Theme.Paper != "#fff" {
		t.Errorf("first stop must use the trip colors: %+v", trip.Stops[0].Theme)
	}
	if trip.Stops[1].Theme.Paper != "#eef" || trip.Stops[2].Theme.Accent != "#06c" {
		t.Errorf("stop colors must apply and carry on: %+v %+v", trip.Stops[1].Theme, trip.Stops[2].Theme)
	}
}

func TestRouteFollowsSavedRoads(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "t.md"), []byte(sample), 0644)
	svc := NewFilesystemService(dir, "/x")
	straight, _ := svc.Get("t")

	// Two legs for three stops. The first leg bends east before Flagstaff.
	roads := `{"source":"test","legs":[
	  {"miles":150,"points":[[-112.07,33.45],[-111.0,34.3],[-111.65,35.20]]},
	  {"miles":320,"points":[[-111.65,35.20],[-109.55,38.57]]}]}`
	os.WriteFile(filepath.Join(dir, "t.route.json"), []byte(roads), 0644)
	trip, err := svc.Get("t")
	if err != nil {
		t.Fatal(err)
	}
	if trip.RouteCredit != "test" || strings.Count(trip.RoutePath, "L") != 3 {
		t.Errorf("route must use the saved line: %q %q", trip.RouteCredit, trip.RoutePath)
	}
	if trip.Stops[1].Miles != 150 || trip.Miles != 470 {
		t.Errorf("miles must come from the legs: %d %d", trip.Stops[1].Miles, trip.Miles)
	}
	if !(trip.Stops[1].PathLen > straight.Stops[1].PathLen) {
		t.Errorf("a bent leg must be longer than a straight one")
	}

	// A file that does not match the stops is ignored.
	os.WriteFile(filepath.Join(dir, "t.route.json"), []byte(`{"legs":[{"miles":1,"points":[[0,0],[1,1]]}]}`), 0644)
	trip, _ = svc.Get("t")
	if trip.RouteCredit != "" || trip.RoutePath != straight.RoutePath {
		t.Errorf("a wrong road file must fall back to straight lines")
	}
}
