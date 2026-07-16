package agent

import (
	"context"
	"encoding/json"
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
