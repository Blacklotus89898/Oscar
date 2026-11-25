package assistant

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Captured from opencode 1.19.6 `run --format json` against a test server.
const openCodeEvents = `{"type":"step_start","timestamp":1,"sessionID":"ses_abc","part":{"type":"step-start"}}
{"type":"text","timestamp":2,"sessionID":"ses_abc","part":{"type":"text","text":"pong"}}
{"type":"step_finish","timestamp":3,"sessionID":"ses_abc","part":{"reason":"stop","type":"step-finish"}}
`

func TestParseOpenCode(t *testing.T) {
	var deltas []string
	reply, sid, err := ParseOpenCode(strings.NewReader(openCodeEvents+"not json\n"), func(s string) { deltas = append(deltas, s) })
	if err != nil || reply != "pong" || sid != "ses_abc" || strings.Join(deltas, "") != "pong" {
		t.Fatalf("reply=%q sid=%q deltas=%q err=%v", reply, sid, deltas, err)
	}
	two := openCodeEvents + `{"type":"text","sessionID":"ses_abc","part":{"type":"text","text":"again"}}` + "\n"
	if reply, _, _ := ParseOpenCode(strings.NewReader(two), nil); reply != "pong\n\nagain" {
		t.Fatalf("parts not joined: %q", reply)
	}
}

func TestParseOpenCodeErrorsAndToolGuard(t *testing.T) {
	_, _, err := ParseOpenCode(strings.NewReader(`{"type":"error","sessionID":"s","error":{"name":"APIError","data":{"message":"Free usage limit reached.\nmore"}}}`), nil)
	if err == nil || err.Error() != "opencode: Free usage limit reached." {
		t.Fatalf("err = %v", err)
	}
	for _, ev := range []string{
		`{"type":"tool_use","sessionID":"s","part":{"type":"tool","tool":"bash"}}`,
		`{"type":"something","sessionID":"s","part":{"type":"tool","tool":"edit"}}`,
	} {
		if _, _, err := ParseOpenCode(strings.NewReader(ev), nil); err == nil || !strings.Contains(err.Error(), "chat-only guard") {
			t.Fatalf("tool event not refused: %v", err)
		}
	}
}

func TestOpenCodeConfigDeniesToolsEverywhere(t *testing.T) {
	o := &OpenCode{Dir: t.TempDir(), System: "be \"nice\""}
	b, err := o.Config()
	if err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		Tools      map[string]bool   `json:"tools"`
		Permission map[string]string `json:"permission"`
		Agent      map[string]struct {
			Prompt     string            `json:"prompt"`
			Tools      map[string]bool   `json:"tools"`
			Permission map[string]string `json:"permission"`
		} `json:"agent"`
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		t.Fatal(err)
	}
	a, ok := cfg.Agent[openCodeAgent]
	// Top level covers OpenCode's fallback agent if ours is ever not found.
	if cfg.Tools["*"] || !ok || a.Tools["*"] || a.Prompt != `be "nice"` {
		t.Fatalf("config = %s", b)
	}
	for _, p := range []string{"edit", "bash", "webfetch"} {
		if cfg.Permission[p] != "deny" || a.Permission[p] != "deny" {
			t.Fatalf("%s not denied: %s", p, b)
		}
	}
}

func TestOpenCodeArgs(t *testing.T) {
	o := &OpenCode{Dir: "/priv", Model: "ollama/qwen3"}
	got := strings.Join(o.Args("-rf hi"), " ")
	want := "run --pure --format json --agent chat-only --dir /priv --model ollama/qwen3 -- -rf hi"
	if got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
	o.session = "ses_1"
	if !strings.Contains(strings.Join(o.Args("x"), " "), "--session ses_1 -- x") {
		t.Fatal("session not resumed")
	}
}

// fakeAPI is an OpenAI-compatible server that records requests.
func fakeAPI(t *testing.T, chunks ...string) (*httptest.Server, *[]map[string]any) {
	t.Helper()
	var reqs []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") == "Bearer bad" {
			w.WriteHeader(http.StatusUnauthorized)
			io.WriteString(w, `{"error":{"message":"invalid key"}}`)
			return
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		reqs = append(reqs, body)
		w.Header().Set("Content-Type", "text/event-stream")
		for _, c := range chunks {
			fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":%q}}]}\n\n", c)
		}
		io.WriteString(w, "data: [DONE]\n\n")
	}))
	t.Cleanup(srv.Close)
	return srv, &reqs
}

func TestOpenAIStreamsAndKeepsHistory(t *testing.T) {
	srv, reqs := fakeAPI(t, "po", "ng")
	o := &OpenAI{BaseURL: srv.URL + "/v1/", Model: "qwen3", System: "sys"}
	var got strings.Builder
	reply, err := o.Ask(context.Background(), "ping", func(s string) { got.WriteString(s) })
	if err != nil || reply != "pong" || got.String() != "pong" {
		t.Fatalf("reply=%q streamed=%q err=%v", reply, got.String(), err)
	}
	if _, err := o.Ask(context.Background(), "again", nil); err != nil {
		t.Fatal(err)
	}
	second := (*reqs)[1]
	msgs := second["messages"].([]any)
	if len(msgs) != 4 || msgs[0].(map[string]any)["role"] != "system" || msgs[2].(map[string]any)["content"] != "pong" {
		t.Fatalf("history not sent: %v", msgs)
	}
	if _, has := second["tools"]; has || second["stream"] != true || second["model"] != "qwen3" {
		t.Fatalf("request = %v", second)
	}
}

func TestOpenAIErrors(t *testing.T) {
	srv, _ := fakeAPI(t)
	o := &OpenAI{BaseURL: srv.URL + "/v1", Model: "m", APIKey: "bad"}
	if _, err := o.Ask(context.Background(), "x", nil); err == nil || !strings.Contains(err.Error(), "invalid key") {
		t.Fatalf("err = %v", err)
	}
	o.APIKey = ""
	if _, err := o.Ask(context.Background(), "x", nil); err == nil || !strings.Contains(err.Error(), "no answer") {
		t.Fatalf("empty reply not reported: %v", err)
	}
	if len(o.history) != 0 {
		t.Fatal("failed turn kept in history")
	}
}

func TestNew(t *testing.T) {
	t.Setenv("PATH", "")
	if _, err := New(Config{}, "", t.TempDir()); err == nil || !strings.Contains(err.Error(), "no assistant found") {
		t.Fatalf("auto with nothing installed: %v", err)
	}
	if _, err := New(Config{Provider: "gpt"}, "", ""); err == nil {
		t.Fatal("bad provider accepted")
	}
	if _, err := New(Config{Provider: "openai", Model: "m"}, "", ""); err == nil {
		t.Fatal("openai without base_url accepted")
	}
	if _, err := New(Config{Provider: "openai", Model: "m", BaseURL: "http://x/v1", APIKeyEnv: "OSCAR_TEST_KEY_UNSET"}, "", ""); err == nil {
		t.Fatal("missing key env accepted")
	}
	t.Setenv("OSCAR_TEST_KEY", "k")
	b, err := New(Config{Provider: "OpenAI", Model: "llama3", BaseURL: "http://x/v1", APIKeyEnv: "OSCAR_TEST_KEY"}, "s", "")
	if err != nil || b.Name() != "llama3" || b.(*OpenAI).APIKey != "k" {
		t.Fatalf("%v %v", b, err)
	}
	oc, err := New(Config{Provider: "opencode", Command: "/opt/opencode", Model: "ollama/qwen3"}, "s", "/priv")
	if err != nil || oc.Name() != "OpenCode (ollama/qwen3)" {
		t.Fatalf("%v %v", oc, err)
	}
	cl, err := New(Config{Provider: "claude", Command: "/opt/claude"}, "s", "/priv")
	if err != nil || cl.Name() != "Claude" {
		t.Fatalf("%v %v", cl, err)
	}
}
