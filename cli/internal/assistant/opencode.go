package assistant

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// OpenCode runs the OpenCode CLI headless (`opencode run --format json`),
// so any provider OpenCode is set up for (Anthropic, OpenAI, Ollama, OpenRouter,
// its own models…) can be the assistant.
//
// Chat-only is enforced, not hoped for. OpenCode falls back to its default
// agent, which has bash/edit/write, when the named agent is missing, so tools
// are denied twice in a config that this package owns: at the top level
// (covers any agent, including that fallback) and on the agent itself. It runs
// in Dir, a private directory holding that opencode.json, so the user's own
// OpenCode config and projects are never touched, and --pure keeps plugins
// (which could add tools) out. Verified against opencode 1.19: every request
// to the model carried an empty tool list, even with a bogus --agent.
type OpenCode struct {
	Bin    string
	Dir    string // private; opencode.json is (re)written here
	Model  string // provider/model; empty = the user's OpenCode default
	System string

	mu      sync.Mutex
	session string
}

const openCodeAgent = "chat-only"

// Name implements Backend.
func (o *OpenCode) Name() string {
	if o.Model != "" {
		return "OpenCode (" + o.Model + ")"
	}
	return "OpenCode"
}

// Config is the opencode.json written into Dir.
func (o *OpenCode) Config() ([]byte, error) {
	deny := map[string]string{"edit": "deny", "bash": "deny", "webfetch": "deny"}
	noTools := map[string]bool{"*": false}
	cfg := map[string]any{
		"$schema":    "https://opencode.ai/config.json",
		"tools":      noTools,
		"permission": deny,
		"agent": map[string]any{
			openCodeAgent: map[string]any{
				"mode":       "primary",
				"prompt":     o.System,
				"tools":      noTools,
				"permission": deny,
			},
		},
	}
	return json.MarshalIndent(cfg, "", "  ")
}

// Args builds one turn's arguments.
func (o *OpenCode) Args(prompt string) []string {
	args := []string{"run", "--pure", "--format", "json", "--agent", openCodeAgent, "--dir", o.Dir}
	if o.Model != "" {
		args = append(args, "--model", o.Model)
	}
	if o.session != "" {
		args = append(args, "--session", o.session)
	}
	// "--" so a question starting with "-" isn't taken for a flag.
	return append(args, "--", prompt)
}

// Ask implements Backend.
func (o *OpenCode) Ask(ctx context.Context, prompt string, onDelta func(string)) (string, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.Dir == "" {
		return "", errors.New("opencode: no private directory")
	}
	if err := os.MkdirAll(o.Dir, 0o700); err != nil {
		return "", err
	}
	cfg, err := o.Config()
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(o.Dir, "opencode.json"), cfg, 0o600); err != nil {
		return "", err
	}
	cmd := exec.CommandContext(ctx, o.Bin, o.Args(prompt)...)
	cmd.Dir = o.Dir
	cmd.Stdin = nil // no prompts: a permission ask must not block
	out, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return "", err
	}
	reply, sid, perr := ParseOpenCode(out, onDelta)
	werr := cmd.Wait()
	if sid != "" {
		o.session = sid
	}
	switch {
	case perr != nil:
		return reply, perr
	case ctx.Err() != nil:
		return reply, fmt.Errorf("opencode: no answer in time (%s)", lastLine(stderr.String(), "timed out"))
	case werr != nil:
		return reply, fmt.Errorf("opencode: %s", lastLine(stderr.String(), werr.Error()))
	case reply == "":
		return "", fmt.Errorf("opencode returned no answer: %s", lastLine(stderr.String(), "empty reply"))
	}
	return reply, nil
}

// ParseOpenCode reads `opencode run --format json` events: one JSON object
// per line, text in {"type":"text","part":{"text":…}}, the session id on
// every event, failures as {"type":"error",…}.
func ParseOpenCode(r io.Reader, onDelta func(string)) (reply, session string, err error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 8*1024*1024)
	var b strings.Builder
	for sc.Scan() {
		var ev struct {
			Type      string          `json:"type"`
			SessionID string          `json:"sessionID"`
			Error     json.RawMessage `json:"error"`
			Part      struct {
				Type string `json:"type"`
				Text string `json:"text"`
				Tool string `json:"tool"`
			} `json:"part"`
		}
		if json.Unmarshal(sc.Bytes(), &ev) != nil {
			continue
		}
		if ev.SessionID != "" {
			session = ev.SessionID
		}
		if ev.Part.Type == "tool" {
			ev.Type = "tool"
		}
		switch ev.Type {
		case "text":
			if ev.Part.Text == "" {
				continue
			}
			if b.Len() > 0 {
				b.WriteString("\n\n")
				if onDelta != nil {
					onDelta("\n\n")
				}
			}
			b.WriteString(ev.Part.Text)
			if onDelta != nil {
				onDelta(ev.Part.Text)
			}
		case "tool_use", "tool":
			// Should be impossible with tools denied; refuse to continue if not.
			return b.String(), session, fmt.Errorf("opencode tried to use a tool (%s): chat-only guard tripped", orDefault(ev.Part.Tool, "unknown"))
		case "error":
			return b.String(), session, fmt.Errorf("opencode: %s", errText(ev.Error))
		}
	}
	return b.String(), session, sc.Err()
}

func errText(raw json.RawMessage) string {
	var e struct {
		Name string `json:"name"`
		Data struct {
			Message string `json:"message"`
		} `json:"data"`
		Message string `json:"message"`
	}
	if json.Unmarshal(raw, &e) == nil {
		if m := orDefault(e.Data.Message, e.Message); m != "" {
			return firstLine(m)
		}
		if e.Name != "" {
			return e.Name
		}
	}
	var s string
	if json.Unmarshal(raw, &s) == nil && s != "" {
		return firstLine(s)
	}
	return "unknown error"
}

func lastLine(s, def string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if l := strings.TrimSpace(lines[i]); l != "" {
			return l
		}
	}
	return def
}
