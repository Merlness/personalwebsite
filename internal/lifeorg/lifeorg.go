// Package lifeorg is a server-side client for the files in Merl's
// life-organizer repo, via the GitHub Contents API. It is the storage
// layer the agent's tools operate on.
package lifeorg

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type Client struct {
	Token  string
	Owner  string
	Repo   string
	Branch string
	// Upstream overrides the GitHub API base URL in tests.
	Upstream string
	// HTTP is the client for upstream calls; http.DefaultClient if nil.
	HTTP *http.Client
}

func (c *Client) upstream() string {
	if c.Upstream != "" {
		return strings.TrimSuffix(c.Upstream, "/")
	}
	return "https://api.github.com"
}

func (c *Client) http() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return http.DefaultClient
}

// escapePath escapes each segment but keeps the slashes, which the
// Contents API expects literally.
func escapePath(p string) string {
	parts := strings.Split(p, "/")
	for i, s := range parts {
		parts[i] = url.PathEscape(s)
	}
	return strings.Join(parts, "/")
}

func (c *Client) contentsURL(path string) string {
	return fmt.Sprintf("%s/repos/%s/%s/contents/%s", c.upstream(), c.Owner, c.Repo, escapePath(path))
}

func (c *Client) do(ctx context.Context, method, u string, body []byte) (int, []byte, error) {
	var rdr io.Reader
	if body != nil {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, rdr)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	res, err := c.http().Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer res.Body.Close()
	b, err := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	return res.StatusCode, b, err
}

// snippet keeps error messages diagnosable without dumping whole bodies.
func snippet(b []byte) string {
	s := string(b)
	if len(s) > 200 {
		s = s[:200] + "..."
	}
	return s
}

// Get returns the decoded content of one file.
func (c *Client) Get(ctx context.Context, path string) (string, error) {
	content, _, err := c.getWithSHA(ctx, path)
	return content, err
}

func (c *Client) getWithSHA(ctx context.Context, path string) (string, string, error) {
	status, body, err := c.do(ctx, http.MethodGet, c.contentsURL(path)+"?ref="+url.QueryEscape(c.Branch), nil)
	if err != nil {
		return "", "", fmt.Errorf("get %s: %w", path, err)
	}
	if status == http.StatusNotFound {
		return "", "", fmt.Errorf("get %s: not found", path)
	}
	if status != http.StatusOK {
		return "", "", fmt.Errorf("get %s: status %d: %s", path, status, snippet(body))
	}
	var f struct {
		Content string `json:"content"`
		SHA     string `json:"sha"`
	}
	if err := json.Unmarshal(body, &f); err != nil {
		return "", "", fmt.Errorf("get %s: decode: %w", path, err)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(f.Content, "\n", ""))
	if err != nil {
		return "", "", fmt.Errorf("get %s: base64: %w", path, err)
	}
	return string(raw), f.SHA, nil
}

// Put writes a file, creating it if absent and otherwise using the current
// sha so GitHub's optimistic concurrency check applies.
func (c *Client) Put(ctx context.Context, path, content, message string) error {
	_, sha, err := c.getWithSHA(ctx, path)
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return err
	}
	payload := map[string]string{
		"message": message,
		"content": base64.StdEncoding.EncodeToString([]byte(content)),
		"branch":  c.Branch,
	}
	if sha != "" {
		payload["sha"] = sha
	}
	body, _ := json.Marshal(payload)
	status, resBody, err := c.do(ctx, http.MethodPut, c.contentsURL(path), body)
	if err != nil {
		return fmt.Errorf("put %s: %w", path, err)
	}
	if status != http.StatusOK && status != http.StatusCreated {
		return fmt.Errorf("put %s: status %d: %s", path, status, snippet(resBody))
	}
	return nil
}

// List returns the entries of a directory; directories carry a trailing "/".
func (c *Client) List(ctx context.Context, dir string) ([]string, error) {
	u := c.contentsURL(strings.TrimSuffix(dir, "/")) + "?ref=" + url.QueryEscape(c.Branch)
	if dir == "" || dir == "." {
		u = fmt.Sprintf("%s/repos/%s/%s/contents?ref=%s", c.upstream(), c.Owner, c.Repo, url.QueryEscape(c.Branch))
	}
	status, body, err := c.do(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("list %s: %w", dir, err)
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("list %s: status %d: %s", dir, status, snippet(body))
	}
	var entries []struct {
		Path string `json:"path"`
		Type string `json:"type"`
	}
	if err := json.Unmarshal(body, &entries); err != nil {
		return nil, fmt.Errorf("list %s: decode: %w", dir, err)
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.Type == "dir" {
			out = append(out, e.Path+"/")
		} else {
			out = append(out, e.Path)
		}
	}
	return out, nil
}
