package lifeorg

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newClient(upstream string) *Client {
	return &Client{Token: "tok", Owner: "Merlness", Repo: "life-organizer", Branch: "main", Upstream: upstream}
}

func TestGetDecodesContent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			t.Errorf("missing auth header")
		}
		json.NewEncoder(w).Encode(map[string]string{
			"content": base64.StdEncoding.EncodeToString([]byte("hello café")),
			"sha":     "abc",
		})
	}))
	defer srv.Close()

	got, err := newClient(srv.URL).Get(context.Background(), "tasks.md")
	if err != nil {
		t.Fatal(err)
	}
	if got != "hello café" {
		t.Fatalf("got %q", got)
	}
}

func TestPutSendsSHAFromExistingFile(t *testing.T) {
	var put map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			json.NewEncoder(w).Encode(map[string]string{"content": "", "sha": "oldsha"})
		case http.MethodPut:
			json.NewDecoder(r.Body).Decode(&put)
			json.NewEncoder(w).Encode(map[string]any{})
		}
	}))
	defer srv.Close()

	if err := newClient(srv.URL).Put(context.Background(), "tasks.md", "new body", "update tasks"); err != nil {
		t.Fatal(err)
	}
	if put["sha"] != "oldsha" {
		t.Fatalf("sha = %q, want oldsha", put["sha"])
	}
	raw, _ := base64.StdEncoding.DecodeString(put["content"])
	if string(raw) != "new body" {
		t.Fatalf("content = %q", raw)
	}
	if put["message"] != "update tasks" {
		t.Fatalf("message = %q", put["message"])
	}
}

func TestPutCreatesWhenMissing(t *testing.T) {
	var put map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			http.Error(w, `{"message":"Not Found"}`, http.StatusNotFound)
		case http.MethodPut:
			json.NewDecoder(r.Body).Decode(&put)
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte("{}"))
		}
	}))
	defer srv.Close()

	if err := newClient(srv.URL).Put(context.Background(), "pulse/new.md", "x", "create"); err != nil {
		t.Fatal(err)
	}
	if _, ok := put["sha"]; ok {
		t.Fatal("sha should be absent when creating a new file")
	}
}

func TestPutSurfacesUpstreamErrorBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			json.NewEncoder(w).Encode(map[string]string{"content": "", "sha": "s"})
			return
		}
		http.Error(w, `{"message":"conflict"}`, http.StatusConflict)
	}))
	defer srv.Close()

	err := newClient(srv.URL).Put(context.Background(), "tasks.md", "x", "m")
	if err == nil || !strings.Contains(err.Error(), "409") || !strings.Contains(err.Error(), "conflict") {
		t.Fatalf("error should carry status and body, got %v", err)
	}
}

func TestListMarksDirectories(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]map[string]string{
			{"path": "pulse", "type": "dir"},
			{"path": "tasks.md", "type": "file"},
		})
	}))
	defer srv.Close()

	got, err := newClient(srv.URL).List(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "pulse/" || got[1] != "tasks.md" {
		t.Fatalf("got %v", got)
	}
}
