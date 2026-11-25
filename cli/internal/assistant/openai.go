package assistant

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

// OpenAI talks to any OpenAI-compatible chat-completions API (Ollama,
// LM Studio, vLLM, llama.cpp server, OpenRouter, OpenAI…). It never sends a
// tools list, so the model has nothing to call: chat-only by construction.
// The conversation is kept here and sent with each turn.
type OpenAI struct {
	BaseURL string // e.g. http://localhost:11434/v1
	Model   string
	APIKey  string
	System  string
	Client  *http.Client

	mu      sync.Mutex
	history []chatMsg
}

type chatMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// maxHistory caps the turns re-sent (each carries a dashboard snapshot).
const maxHistory = 20

// Name implements Backend.
func (o *OpenAI) Name() string { return o.Model }

// Ask implements Backend.
func (o *OpenAI) Ask(ctx context.Context, prompt string, onDelta func(string)) (string, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	msgs := []chatMsg{}
	if o.System != "" {
		msgs = append(msgs, chatMsg{"system", o.System})
	}
	hist := o.history
	if len(hist) > maxHistory {
		hist = hist[len(hist)-maxHistory:]
	}
	msgs = append(msgs, hist...)
	msgs = append(msgs, chatMsg{"user", prompt})
	body, err := json.Marshal(map[string]any{"model": o.Model, "messages": msgs, "stream": true})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(o.BaseURL, "/")+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if o.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+o.APIKey)
	}
	c := o.Client
	if c == nil {
		c = http.DefaultClient
	}
	resp, err := c.Do(req)
	if err != nil {
		return "", fmt.Errorf("%s: %w", o.BaseURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("%s: %s: %s", o.Model, resp.Status, firstLine(apiError(b)))
	}
	reply, err := ParseSSE(resp.Body, onDelta)
	if err != nil {
		return reply, err
	}
	if strings.TrimSpace(reply) == "" {
		return "", fmt.Errorf("%s returned no answer", o.Model)
	}
	o.history = append(o.history, chatMsg{"user", prompt}, chatMsg{"assistant", reply})
	return reply, nil
}

// ParseSSE reads a streamed chat completion ("data: {...}" lines).
func ParseSSE(r io.Reader, onDelta func(string)) (string, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 8*1024*1024)
	var b strings.Builder
	for sc.Scan() {
		data, ok := strings.CutPrefix(sc.Text(), "data:")
		if !ok {
			continue
		}
		data = strings.TrimSpace(data)
		if data == "[DONE]" {
			break
		}
		var ch struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
			Error json.RawMessage `json:"error"`
		}
		if json.Unmarshal([]byte(data), &ch) != nil {
			continue
		}
		if len(ch.Error) > 0 && string(ch.Error) != "null" {
			return b.String(), fmt.Errorf("%s", apiError([]byte(data)))
		}
		for _, c := range ch.Choices {
			if c.Delta.Content != "" {
				b.WriteString(c.Delta.Content)
				if onDelta != nil {
					onDelta(c.Delta.Content)
				}
			}
		}
	}
	return b.String(), sc.Err()
}

// apiError pulls the message out of an OpenAI-style error body.
func apiError(b []byte) string {
	var e struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(b, &e) == nil && e.Error.Message != "" {
		return e.Error.Message
	}
	var s struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(b, &s) == nil && s.Error != "" {
		return s.Error
	}
	return strings.TrimSpace(string(b))
}
