package assistant

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

// TestOpenCodeLive drives a real opencode binary. Opt in:
//
//	OPENCODE_BIN=/path/to/opencode OPENCODE_MODEL=provider/model go test -run Live ./internal/assistant/
func TestOpenCodeLive(t *testing.T) {
	bin := os.Getenv("OPENCODE_BIN")
	if bin == "" {
		t.Skip("set OPENCODE_BIN to run against a real opencode")
	}
	o := &OpenCode{Bin: bin, Dir: t.TempDir(), Model: os.Getenv("OPENCODE_MODEL"), System: "Reply briefly."}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	var streamed strings.Builder
	reply, err := o.Ask(ctx, "-say pong", func(s string) { streamed.WriteString(s) })
	if err != nil || reply == "" || streamed.String() != reply {
		t.Fatalf("turn 1: reply=%q streamed=%q err=%v", reply, streamed.String(), err)
	}
	first := o.session
	if first == "" {
		t.Fatal("no session id")
	}
	if _, err := o.Ask(ctx, "again", nil); err != nil {
		t.Fatalf("turn 2: %v", err)
	}
	if o.session != first {
		t.Fatalf("session changed: %s → %s", first, o.session)
	}
}
