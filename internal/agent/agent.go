// Package agent runs Merl's real-time organizer commands: a small tool-use
// loop over the Anthropic API whose tools read and write the life-organizer
// repo. The write surface is pinned by an allowlist so the agent can never
// touch anything outside the organizer files.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
)

const maxIterations = 8

// FileStore is the storage seam; lifeorg.Client is the real implementation.
type FileStore interface {
	Get(ctx context.Context, path string) (string, error)
	Put(ctx context.Context, path, content, message string) error
	List(ctx context.Context, dir string) ([]string, error)
}

type Turn struct {
	Role    string `json:"role"` // "user" or "assistant"
	Content string `json:"content"`
}

type Written struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type Agent struct {
	Client anthropic.Client
	Store  FileStore
	Model  string
	// Now overrides the clock in tests.
	Now func() time.Time
}

// Phoenix has no DST, so a fixed offset is exact year-round.
var phoenix = time.FixedZone("America/Phoenix", -7*3600)

func (a *Agent) now() time.Time {
	if a.Now != nil {
		return a.Now()
	}
	return time.Now()
}

func (a *Agent) systemPrompt() string {
	today := a.now().In(phoenix).Format("Monday, January 2, 2006")
	return `You are the agent behind Merl Martin's personal organizer app. Today is ` + today + ` (Phoenix, Arizona).

Your tools read and write markdown files in Merl's life-organizer repo. The files you will use most:
- tasks.md: the task list, with sections Business, Personal, Financial. Tasks are markdown checkboxes.
- inbox.md: unprocessed captures, newest under the Unprocessed heading.
- pulse/today-workout.md: today's workout card. pulse/workout-program.md: the program. pulse/workout-ledger.md: the log of completed workouts.
- drafts/: LinkedIn post drafts and business-card captures.

Rules:
- Read a file before you edit it, and preserve its existing structure and formatting exactly. Make the smallest edit that does the job.
- Never delete a file, and never drop content you were not asked to change. Additive and targeted edits only.
- If the request is ambiguous, do not guess: reply with one short clarifying question instead of writing.
- Use short, specific commit messages.
- No em dashes anywhere. Use commas or periods.
- When done, confirm what changed in one or two short sentences, naming the file.`
}

func toolDefs() []anthropic.ToolUnionParam {
	str := func(desc string) map[string]any {
		return map[string]any{"type": "string", "description": desc}
	}
	tools := []anthropic.ToolParam{
		{
			Name:        "list_dir",
			Description: anthropic.String("List the files in a directory of the organizer repo. Use an empty path for the repo root. Directories end with a slash."),
			InputSchema: anthropic.ToolInputSchemaParam{Properties: map[string]any{
				"path": str("directory path, e.g. \"pulse\" or empty for the root (required)"),
			}},
		},
		{
			Name:        "read_file",
			Description: anthropic.String("Read one file from the organizer repo and return its full contents."),
			InputSchema: anthropic.ToolInputSchemaParam{Properties: map[string]any{
				"path": str("file path, e.g. \"tasks.md\" or \"pulse/today-workout.md\" (required)"),
			}},
		},
		{
			Name:        "write_file",
			Description: anthropic.String("Write the full new contents of one file in the organizer repo. Only tasks.md, inbox.md, and files under pulse/ or drafts/ are writable. Always read the file first and send back the complete document with your edit applied."),
			InputSchema: anthropic.ToolInputSchemaParam{Properties: map[string]any{
				"path":    str("file path to write (required)"),
				"content": str("the complete new file contents (required)"),
				"message": str("short commit message describing the change (required)"),
			}},
		},
	}
	out := make([]anthropic.ToolUnionParam, len(tools))
	for i := range tools {
		out[i] = anthropic.ToolUnionParam{OfTool: &tools[i]}
	}
	return out
}

var writablePrefixes = []string{"pulse/", "drafts/"}
var writableFiles = []string{"tasks.md", "inbox.md"}

// writable pins the agent's write surface to the organizer files.
func writable(p string) bool {
	clean := path.Clean(p)
	if clean != p || strings.HasPrefix(clean, "/") || strings.HasPrefix(clean, "..") {
		return false
	}
	for _, f := range writableFiles {
		if clean == f {
			return true
		}
	}
	for _, pre := range writablePrefixes {
		if strings.HasPrefix(clean, pre) && clean != strings.TrimSuffix(pre, "/") {
			return true
		}
	}
	return false
}

// safeRead rejects traversal for reads and listings.
func safeRead(p string) bool {
	clean := path.Clean(p)
	return !strings.HasPrefix(clean, "/") && !strings.HasPrefix(clean, "..")
}

func (a *Agent) model() anthropic.Model {
	if a.Model != "" {
		return anthropic.Model(a.Model)
	}
	return anthropic.Model("claude-haiku-4-5")
}

func textMessage(role anthropic.MessageParamRole, text string) anthropic.MessageParam {
	return anthropic.MessageParam{Role: role, Content: []anthropic.ContentBlockParamUnion{anthropic.NewTextBlock(text)}}
}

// Run executes one command end to end and returns the reply plus every file
// the agent wrote, so the client can refresh its views without refetching.
func (a *Agent) Run(ctx context.Context, history []Turn, message string) (string, []Written, error) {
	msgs := make([]anthropic.MessageParam, 0, len(history)+1)
	for _, t := range history {
		switch t.Role {
		case "user":
			msgs = append(msgs, textMessage(anthropic.MessageParamRoleUser, t.Content))
		case "assistant":
			msgs = append(msgs, textMessage(anthropic.MessageParamRoleAssistant, t.Content))
		}
	}
	msgs = append(msgs, textMessage(anthropic.MessageParamRoleUser, message))

	params := anthropic.MessageNewParams{
		Model:     a.model(),
		MaxTokens: 4096,
		System:    []anthropic.TextBlockParam{{Text: a.systemPrompt()}},
		Tools:     toolDefs(),
		Messages:  msgs,
	}

	var written []Written
	for i := 0; i < maxIterations; i++ {
		resp, err := a.Client.Messages.New(ctx, params)
		if err != nil {
			return "", written, fmt.Errorf("anthropic: %w", err)
		}
		params.Messages = append(params.Messages, resp.ToParam())

		if resp.StopReason != anthropic.StopReasonToolUse {
			return textOf(resp), written, nil
		}

		var results []anthropic.ContentBlockParamUnion
		for _, block := range resp.Content {
			if variant, ok := block.AsAny().(anthropic.ToolUseBlock); ok {
				out, isErr := a.execTool(ctx, variant.Name, []byte(variant.JSON.Input.Raw()), &written)
				results = append(results, anthropic.NewToolResultBlock(variant.ID, out, isErr))
			}
		}
		if len(results) == 0 {
			return textOf(resp), written, nil
		}
		params.Messages = append(params.Messages, anthropic.NewUserMessage(results...))
	}
	return "I hit the tool budget before finishing. The changes listed below did land; please retry the rest.", written, nil
}

func (a *Agent) execTool(ctx context.Context, name string, input []byte, written *[]Written) (string, bool) {
	switch name {
	case "list_dir":
		var in struct{ Path string }
		if err := json.Unmarshal(input, &in); err != nil {
			return "bad input: " + err.Error(), true
		}
		if !safeRead(in.Path) {
			return "path not allowed", true
		}
		entries, err := a.Store.List(ctx, in.Path)
		if err != nil {
			return err.Error(), true
		}
		return strings.Join(entries, "\n"), false
	case "read_file":
		var in struct{ Path string }
		if err := json.Unmarshal(input, &in); err != nil {
			return "bad input: " + err.Error(), true
		}
		if in.Path == "" || !safeRead(in.Path) {
			return "path not allowed", true
		}
		content, err := a.Store.Get(ctx, in.Path)
		if err != nil {
			return err.Error(), true
		}
		return content, false
	case "write_file":
		var in struct{ Path, Content, Message string }
		if err := json.Unmarshal(input, &in); err != nil {
			return "bad input: " + err.Error(), true
		}
		if !writable(in.Path) {
			return "write not allowed: only tasks.md, inbox.md, pulse/, and drafts/ are writable", true
		}
		if in.Message == "" {
			in.Message = "organizer agent edit"
		}
		if err := a.Store.Put(ctx, in.Path, in.Content, in.Message); err != nil {
			return err.Error(), true
		}
		*written = append(*written, Written{Path: in.Path, Content: in.Content})
		return "wrote " + in.Path, false
	default:
		return "unknown tool " + name, true
	}
}

func textOf(m *anthropic.Message) string {
	var parts []string
	for _, block := range m.Content {
		if t, ok := block.AsAny().(anthropic.TextBlock); ok {
			parts = append(parts, t.Text)
		}
	}
	return strings.Join(parts, "\n")
}
