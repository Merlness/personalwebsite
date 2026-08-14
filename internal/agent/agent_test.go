package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

func TestWritableAllowlist(t *testing.T) {
	allowed := []string{"tasks.md", "inbox.md", "pulse/today-workout.md", "drafts/cards/inbox/x.md"}
	for _, p := range allowed {
		if !writable(p) {
			t.Errorf("writable(%q) = false, want true", p)
		}
	}
	denied := []string{
		"README.md", "contexts/health.md", "pulse/../secrets.md", "../other/tasks.md",
		"/etc/passwd", "pulse", "drafts", "", ".github/workflows/x.yml",
	}
	for _, p := range denied {
		if writable(p) {
			t.Errorf("writable(%q) = true, want false", p)
		}
	}
}

func TestSafeReadRejectsNonCanonicalAndTraversal(t *testing.T) {
	allowed := []string{"", "tasks.md", "pulse/today-workout.md", "drafts/cards/inbox/x.md"}
	for _, p := range allowed {
		if !safeRead(p) {
			t.Errorf("safeRead(%q) = false, want true", p)
		}
	}
	denied := []string{"/etc/passwd", "../x", "a/../b", "pulse/", "./tasks.md", "a//b"}
	for _, p := range denied {
		if safeRead(p) {
			t.Errorf("safeRead(%q) = true, want false", p)
		}
	}
}

type fakeStore struct {
	files map[string]string
	puts  []Written
}

func (f *fakeStore) Get(_ context.Context, p string) (string, error) {
	return f.files[p], nil
}
func (f *fakeStore) Put(_ context.Context, p, content, _ string) error {
	f.files[p] = content
	f.puts = append(f.puts, Written{Path: p, Content: content})
	return nil
}
func (f *fakeStore) List(_ context.Context, _ string) ([]string, error) {
	return []string{"tasks.md"}, nil
}

// fakeAnthropic scripts a sequence of Messages API responses.
func fakeAnthropic(t *testing.T, responses []string) *httptest.Server {
	i := 0
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if i >= len(responses) {
			t.Errorf("unexpected extra anthropic call %d", i)
			http.Error(w, "{}", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(responses[i]))
		i++
	}))
}

const toolUseResponse = `{
  "id": "msg_1", "type": "message", "role": "assistant", "model": "claude-haiku-4-5",
  "stop_reason": "tool_use",
  "content": [
    {"type": "tool_use", "id": "toolu_1", "name": "write_file",
     "input": {"path": "pulse/today-workout.md", "content": "upper body day", "message": "swap workout"}}
  ],
  "usage": {"input_tokens": 10, "output_tokens": 10}
}`

const doneResponse = `{
  "id": "msg_2", "type": "message", "role": "assistant", "model": "claude-haiku-4-5",
  "stop_reason": "end_turn",
  "content": [{"type": "text", "text": "Swapped today to upper body in pulse/today-workout.md."}],
  "usage": {"input_tokens": 10, "output_tokens": 10}
}`

func newTestAgent(srvURL string, store *fakeStore) *Agent {
	return &Agent{
		Client: anthropic.NewClient(option.WithBaseURL(srvURL), option.WithAPIKey("test")),
		Store:  store,
	}
}

func TestRunExecutesWriteToolAndReturnsReply(t *testing.T) {
	srv := fakeAnthropic(t, []string{toolUseResponse, doneResponse})
	defer srv.Close()
	store := &fakeStore{files: map[string]string{}}

	reply, written, err := newTestAgent(srv.URL, store).Run(context.Background(), nil, "swap today's workout to upper body")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(reply, "upper body") {
		t.Fatalf("reply = %q", reply)
	}
	if len(written) != 1 || written[0].Path != "pulse/today-workout.md" || written[0].Content != "upper body day" {
		t.Fatalf("written = %+v", written)
	}
	if store.files["pulse/today-workout.md"] != "upper body day" {
		t.Fatal("store was not written")
	}
}

const badWriteResponse = `{
  "id": "msg_1", "type": "message", "role": "assistant", "model": "claude-haiku-4-5",
  "stop_reason": "tool_use",
  "content": [
    {"type": "tool_use", "id": "toolu_1", "name": "write_file",
     "input": {"path": "contexts/health.md", "content": "x", "message": "m"}}
  ],
  "usage": {"input_tokens": 10, "output_tokens": 10}
}`

func TestRunRefusesWriteOutsideAllowlist(t *testing.T) {
	srv := fakeAnthropic(t, []string{badWriteResponse, doneResponse})
	defer srv.Close()
	store := &fakeStore{files: map[string]string{}}

	_, written, err := newTestAgent(srv.URL, store).Run(context.Background(), nil, "edit health context")
	if err != nil {
		t.Fatal(err)
	}
	if len(written) != 0 || len(store.puts) != 0 {
		t.Fatalf("disallowed write went through: %+v", store.puts)
	}
}

func TestRunPassesHistoryThrough(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(doneResponse))
	}))
	defer srv.Close()

	history := []Turn{{Role: "user", Content: "hi"}, {Role: "assistant", Content: "hello"}}
	if _, _, err := newTestAgent(srv.URL, &fakeStore{files: map[string]string{}}).Run(context.Background(), history, "next"); err != nil {
		t.Fatal(err)
	}
	msgs := gotBody["messages"].([]any)
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(msgs))
	}
}

// ---------- streaming ----------

// sseAnthropic scripts a sequence of streaming Messages API responses. Each
// entry is the list of SSE events for one call, already JSON encoded.
func sseAnthropic(t *testing.T, calls [][]string) *httptest.Server {
	i := 0
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if i >= len(calls) {
			t.Errorf("unexpected extra anthropic call %d", i)
			http.Error(w, "{}", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		for _, ev := range calls[i] {
			var probe struct {
				Type string `json:"type"`
			}
			json.Unmarshal([]byte(ev), &probe)
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", probe.Type, ev)
		}
		w.(http.Flusher).Flush()
		i++
	}))
}

// One streamed turn that calls write_file, then one that just talks.
func sseToolUseTurn() []string {
	return []string{
		`{"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","model":"claude-haiku-4-5","content":[],"stop_reason":null,"usage":{"input_tokens":10,"output_tokens":0}}}`,
		`{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
		`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Swapping it now."}}`,
		`{"type":"content_block_stop","index":0}`,
		`{"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"toolu_1","name":"write_file","input":{}}}`,
		`{"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{\"path\":\"pulse/today-workout.md\",\"content\":\"upper body day\",\"message\":\"swap workout\"}"}}`,
		`{"type":"content_block_stop","index":1}`,
		`{"type":"message_delta","delta":{"stop_reason":"tool_use"},"usage":{"output_tokens":20}}`,
		`{"type":"message_stop"}`,
	}
}

func sseFinalTurn() []string {
	return []string{
		`{"type":"message_start","message":{"id":"msg_2","type":"message","role":"assistant","model":"claude-haiku-4-5","content":[],"stop_reason":null,"usage":{"input_tokens":10,"output_tokens":0}}}`,
		`{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
		`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Done, "}}`,
		`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"upper body today."}}`,
		`{"type":"content_block_stop","index":0}`,
		`{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":20}}`,
		`{"type":"message_stop"}`,
	}
}

func TestStreamEmitsTextStepsAndWrites(t *testing.T) {
	srv := sseAnthropic(t, [][]string{sseToolUseTurn(), sseFinalTurn()})
	defer srv.Close()
	store := &fakeStore{files: map[string]string{}}

	var events []Event
	reply, written, err := newTestAgent(srv.URL, store).
		Stream(context.Background(), nil, "swap today's workout to upper body", func(e Event) {
			events = append(events, e)
		})
	if err != nil {
		t.Fatal(err)
	}

	var text, steps string
	var writes int
	for _, e := range events {
		switch e.Type {
		case EventText:
			text += e.Text
		case EventStep:
			steps += e.Text
		case EventWritten:
			writes++
			if e.Path != "pulse/today-workout.md" || e.Content != "upper body day" {
				t.Fatalf("written event = %+v", e)
			}
		}
	}
	if text != "Swapping it now.Done, upper body today." {
		t.Fatalf("streamed text = %q", text)
	}
	if steps != "writing pulse/today-workout.md" {
		t.Fatalf("steps = %q", steps)
	}
	if writes != 1 {
		t.Fatalf("written events = %d", writes)
	}

	// The return values must still match what the non-streaming path gives.
	if reply != "Done, upper body today." {
		t.Fatalf("reply = %q", reply)
	}
	if len(written) != 1 || written[0].Path != "pulse/today-workout.md" {
		t.Fatalf("written = %+v", written)
	}
	if store.files["pulse/today-workout.md"] != "upper body day" {
		t.Fatal("store was not written")
	}
}

// The step must be announced before the tool runs, so the phone shows what is
// happening rather than what already happened.
func TestStreamAnnouncesStepBeforeWrite(t *testing.T) {
	srv := sseAnthropic(t, [][]string{sseToolUseTurn(), sseFinalTurn()})
	defer srv.Close()

	var order []string
	_, _, err := newTestAgent(srv.URL, &fakeStore{files: map[string]string{}}).
		Stream(context.Background(), nil, "x", func(e Event) {
			if e.Type == EventStep || e.Type == EventWritten {
				order = append(order, e.Type)
			}
		})
	if err != nil {
		t.Fatal(err)
	}
	if len(order) != 2 || order[0] != EventStep || order[1] != EventWritten {
		t.Fatalf("event order = %v", order)
	}
}
