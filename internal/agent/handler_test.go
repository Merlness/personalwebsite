package agent

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeRunner struct {
	reply   string
	written []Written
	err     error
	gotMsg  string
	gotHist []Turn
}

func (f *fakeRunner) Run(_ context.Context, history []Turn, message string) (string, []Written, error) {
	f.gotMsg = message
	f.gotHist = history
	return f.reply, f.written, f.err
}

func TestHandlerHappyPath(t *testing.T) {
	runner := &fakeRunner{reply: "done", written: []Written{{Path: "tasks.md", Content: "x"}}}
	h := &Handler{Runner: runner}

	body := `{"message":"add a task","history":[{"role":"user","content":"earlier"}]}`
	req := httptest.NewRequest("POST", "/agent", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	var res response
	json.Unmarshal(rec.Body.Bytes(), &res)
	if res.Reply != "done" || len(res.Written) != 1 || res.Written[0].Path != "tasks.md" {
		t.Fatalf("res = %+v", res)
	}
	if runner.gotMsg != "add a task" || len(runner.gotHist) != 1 {
		t.Fatalf("runner got %q %v", runner.gotMsg, runner.gotHist)
	}
}

func TestHandlerRejectsBadRequests(t *testing.T) {
	h := &Handler{Runner: &fakeRunner{}}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/agent", nil))
	if rec.Code != 405 {
		t.Fatalf("GET status %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("POST", "/agent", strings.NewReader(`{"message":""}`)))
	if rec.Code != 400 {
		t.Fatalf("empty message status %d", rec.Code)
	}
}

// fakeStreamer implements both halves, so the handler picks the SSE path.
type fakeStreamer struct {
	fakeRunner
	events []Event
}

func (f *fakeStreamer) Stream(_ context.Context, history []Turn, message string, emit func(Event)) (string, []Written, error) {
	f.gotMsg = message
	f.gotHist = history
	for _, e := range f.events {
		emit(e)
	}
	return f.reply, f.written, f.err
}

func sseRequest(body string) *http.Request {
	req := httptest.NewRequest("POST", "/agent", strings.NewReader(body))
	req.Header.Set("Accept", "text/event-stream")
	return req
}

func TestHandlerStreamsEventsThenDone(t *testing.T) {
	runner := &fakeStreamer{
		fakeRunner: fakeRunner{reply: "Swapped it.", written: []Written{{Path: "pulse/today-workout.md", Content: "upper"}}},
		events: []Event{
			{Type: EventText, Text: "Let me look.\nOne moment."},
			{Type: EventStep, Text: "reading pulse/today-workout.md", Path: "pulse/today-workout.md"},
			{Type: EventWritten, Path: "pulse/today-workout.md", Content: "upper"},
		},
	}
	h := &Handler{Runner: runner}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, sseRequest(`{"message":"swap today to upper body"}`))

	if rec.Code != 200 {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("content type %q", ct)
	}
	body := rec.Body.String()

	// Every frame must be a single "data:" line: a raw newline inside one
	// would split it into two frames on the client.
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "data: ") && !json.Valid([]byte(strings.TrimPrefix(line, "data: "))) {
			t.Fatalf("frame is not single-line JSON: %q", line)
		}
	}

	frames := strings.Split(strings.TrimSpace(body), "\n\n")
	if len(frames) != 4 {
		t.Fatalf("got %d frames, want 4:\n%s", len(frames), body)
	}
	var first Event
	json.Unmarshal([]byte(strings.TrimPrefix(frames[0], "data: ")), &first)
	if first.Type != EventText || first.Text != "Let me look.\nOne moment." {
		t.Fatalf("first frame = %+v", first)
	}
	if !strings.HasPrefix(frames[3], "event: done\n") {
		t.Fatalf("last frame not done: %q", frames[3])
	}
	var res response
	json.Unmarshal([]byte(strings.TrimPrefix(frames[3], "event: done\ndata: ")), &res)
	if res.Reply != "Swapped it." || len(res.Written) != 1 {
		t.Fatalf("done payload = %+v", res)
	}
}

func TestHandlerStreamErrorSendsErrorFrame(t *testing.T) {
	h := &Handler{Runner: &fakeStreamer{fakeRunner: fakeRunner{err: errors.New("boom")}}}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, sseRequest(`{"message":"x"}`))

	body := rec.Body.String()
	if !strings.Contains(body, `"type":"error"`) {
		t.Fatalf("no error frame: %q", body)
	}
	if strings.Contains(body, "boom") {
		t.Fatal("internal error detail leaked to the client")
	}
	if strings.Contains(body, "event: done") {
		t.Fatal("a failed run must not send done")
	}
}

func TestHandlerFallsBackToJSONWithoutStreamSupport(t *testing.T) {
	// fakeRunner has no Stream method, so Accept must not change the shape.
	h := &Handler{Runner: &fakeRunner{reply: "done"}}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, sseRequest(`{"message":"x"}`))

	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content type %q", ct)
	}
	var res response
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil || res.Reply != "done" {
		t.Fatalf("body = %q", rec.Body)
	}
}

func TestHandlerMapsRunnerErrorTo502(t *testing.T) {
	h := &Handler{Runner: &fakeRunner{err: errors.New("boom")}}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("POST", "/agent", strings.NewReader(`{"message":"x"}`)))
	if rec.Code != 502 {
		t.Fatalf("status %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "boom") {
		t.Fatal("internal error detail leaked to the client")
	}
}
