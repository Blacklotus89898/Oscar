package assistant

import (
	"regexp"
	"slices"
	"strings"
	"testing"
)

const stream = `{"type":"system","subtype":"init"}
{"type":"stream_event","event":{"type":"message_start"}}
{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"text_delta","text":"Hel"}}}
{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"text_delta","text":"lo"}}}
{"type":"assistant"}
not json
{"type":"result","subtype":"success","is_error":false,"result":"Hello","session_id":"11ceff1e-41f5-48ac-a893-5490ed88ca4d"}
`

func TestParseStream(t *testing.T) {
	var got []string
	reply, sid, err := ParseStream(strings.NewReader(stream), func(s string) { got = append(got, s) })
	if err != nil || reply != "Hello" || sid != "11ceff1e-41f5-48ac-a893-5490ed88ca4d" {
		t.Fatalf("reply=%q sid=%q err=%v", reply, sid, err)
	}
	if !slices.Equal(got, []string{"Hel", "lo"}) {
		t.Fatalf("deltas %v", got)
	}
}

func TestParseStreamError(t *testing.T) {
	_, _, err := ParseStream(strings.NewReader(`{"type":"result","subtype":"error_during_execution","is_error":true,"result":"Not logged in\nrun claude"}`), nil)
	if err == nil || !strings.Contains(err.Error(), "Not logged in") || strings.Contains(err.Error(), "run claude") {
		t.Fatalf("got %v", err)
	}
}

func TestParseStreamEmpty(t *testing.T) {
	if _, _, err := ParseStream(strings.NewReader(""), nil); err == nil {
		t.Fatal("no output must be an error")
	}
}

func TestArgsAreToolFreeAndResume(t *testing.T) {
	a := &Assistant{System: "sys", Model: "sonnet", session: newUUID()}
	first := a.Args("hi")
	for _, want := range [][]string{{"--tools", ""}, {"--strict-mcp-config"}, {"--session-id", a.session}, {"--system-prompt", "sys"}, {"--model", "sonnet"}} {
		if !containsSeq(first, want) {
			t.Errorf("first turn missing %v in %q", want, first)
		}
	}
	a.started = true
	next := a.Args("again")
	if !containsSeq(next, []string{"--resume", a.session}) || slices.Contains(next, "--session-id") {
		t.Fatalf("later turns resume: %q", next)
	}
}

func TestUUIDv4(t *testing.T) {
	if !regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`).MatchString(newUUID()) {
		t.Fatal("not a v4 uuid")
	}
}

func containsSeq(s, seq []string) bool {
	for i := 0; i+len(seq) <= len(s); i++ {
		if slices.Equal(s[i:i+len(seq)], seq) {
			return true
		}
	}
	return false
}
