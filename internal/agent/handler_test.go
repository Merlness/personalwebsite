package agent

import (
	"context"
	"encoding/json"
	"errors"
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
