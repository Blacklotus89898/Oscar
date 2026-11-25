// Package assistant runs the user's installed Claude Code CLI headless as a
// chat-only helper: no built-in tools, no MCP servers, so asking it something
// can never change the machine. One conversation per oscar run.
package assistant

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
)

// Assistant holds one conversation.
type Assistant struct {
	Bin    string // claude executable
	Dir    string // working directory (keeps sessions out of project dirs)
	Model  string // optional --model
	System string // system prompt, sent with every turn

	mu      sync.Mutex
	session string
	started bool
}

// Find locates the claude CLI; "" when it isn't installed.
func Find() string {
	p, err := exec.LookPath("claude")
	if err != nil {
		return ""
	}
	return p
}

// Args builds the CLI arguments for one turn. The first turn names the session;
// later turns resume it.
func (a *Assistant) Args(prompt string) []string {
	args := []string{"-p", prompt,
		"--output-format", "stream-json", "--include-partial-messages", "--verbose",
		"--tools", "", "--strict-mcp-config",
	}
	if a.System != "" {
		args = append(args, "--system-prompt", a.System)
	}
	if a.Model != "" {
		args = append(args, "--model", a.Model)
	}
	if a.started {
		args = append(args, "--resume", a.session)
	} else {
		args = append(args, "--session-id", a.session)
	}
	return args
}

// Ask sends one turn and streams text to onDelta; it returns the final reply.
func (a *Assistant) Ask(ctx context.Context, prompt string, onDelta func(string)) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.Bin == "" {
		return "", errors.New("Claude Code CLI not found on PATH")
	}
	if a.session == "" {
		a.session = newUUID()
	}
	cmd := exec.CommandContext(ctx, a.Bin, a.Args(prompt)...)
	cmd.Dir = a.Dir
	out, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return "", err
	}
	reply, sid, perr := ParseStream(out, onDelta)
	werr := cmd.Wait()
	if sid != "" {
		a.session, a.started = sid, true
	}
	switch {
	case perr != nil:
		return reply, perr
	case werr != nil:
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = werr.Error()
		}
		return reply, fmt.Errorf("claude: %s", firstLine(msg))
	}
	return reply, nil
}

// ParseStream reads Claude Code's stream-json output: text deltas go to
// onDelta as they arrive; the final result supplies the reply and session id.
func ParseStream(r io.Reader, onDelta func(string)) (reply, session string, err error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 8*1024*1024)
	var streamed strings.Builder
	for sc.Scan() {
		var ev struct {
			Type  string `json:"type"`
			Event struct {
				Type  string `json:"type"`
				Delta struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"delta"`
			} `json:"event"`
			Subtype   string `json:"subtype"`
			IsError   bool   `json:"is_error"`
			Result    string `json:"result"`
			SessionID string `json:"session_id"`
		}
		if json.Unmarshal(sc.Bytes(), &ev) != nil {
			continue
		}
		switch ev.Type {
		case "stream_event":
			if ev.Event.Type == "content_block_delta" && ev.Event.Delta.Type == "text_delta" {
				streamed.WriteString(ev.Event.Delta.Text)
				if onDelta != nil {
					onDelta(ev.Event.Delta.Text)
				}
			}
		case "result":
			session = ev.SessionID
			reply = ev.Result
			if reply == "" {
				reply = streamed.String()
			}
			if ev.IsError {
				return reply, session, fmt.Errorf("claude: %s", firstLine(orDefault(ev.Result, ev.Subtype)))
			}
		}
	}
	if err := sc.Err(); err != nil {
		return streamed.String(), session, err
	}
	if session == "" && reply == "" {
		reply = streamed.String()
		if reply == "" {
			return "", "", errors.New("claude returned no answer")
		}
	}
	return reply, session, nil
}

func orDefault(s, d string) string {
	if strings.TrimSpace(s) == "" {
		return d
	}
	return s
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

func newUUID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
