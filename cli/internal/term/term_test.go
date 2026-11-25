package term

import (
	"io"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hinshun/vt10x"
)

// fakePTY feeds scripted output and records input.
type fakePTY struct {
	out    *io.PipeReader
	feed   *io.PipeWriter
	mu     sync.Mutex
	in     []byte
	exited chan struct{}
}

func newFake() *fakePTY {
	r, w := io.Pipe()
	return &fakePTY{out: r, feed: w, exited: make(chan struct{})}
}

func (f *fakePTY) Read(b []byte) (int, error) { return f.out.Read(b) }
func (f *fakePTY) Write(b []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.in = append(f.in, b...)
	return len(b), nil
}
func (f *fakePTY) Resize(int, int) error { return nil }
func (f *fakePTY) Wait() error           { <-f.exited; return nil }
func (f *fakePTY) Close() error          { f.feed.Close(); return nil }

func waitDirty(t *testing.T, s *Session) {
	t.Helper()
	select {
	case <-s.Dirty():
	case <-time.After(2 * time.Second):
		t.Fatal("no screen update")
	}
}

func TestScreenColoursAndSplitUTF8(t *testing.T) {
	f := newFake()
	s := newSession(f, 20, 3)
	// "é" split across two writes must not be lost.
	go func() {
		f.feed.Write([]byte("\x1b[31mred\x1b[0m caf\xc3"))
		f.feed.Write([]byte("\xa9"))
	}()
	deadline := time.Now().Add(2 * time.Second)
	for !strings.Contains(s.Text(), "café") {
		if time.Now().After(deadline) {
			t.Fatalf("screen %q", s.Text())
		}
		waitDirty(t, s)
	}
	r := s.Render(false)
	if !strings.Contains(r, "\x1b[0;31mr") {
		t.Fatalf("red not rendered: %q", r)
	}
	if lines := strings.Split(r, "\n"); len(lines) != 3 {
		t.Fatalf("want 3 rows, got %d", len(lines))
	}
	close(f.exited)
	select {
	case <-s.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("session did not finish after exit")
	}
}

func TestCursorIsReversed(t *testing.T) {
	v := vt10x.New(vt10x.WithSize(5, 1))
	v.Write([]byte("ab"))
	v.Lock()
	r := render(v, true)
	v.Unlock()
	if !strings.Contains(r, "\x1b[0;7m ") {
		t.Fatalf("cursor cell not reversed: %q", r)
	}
}

func TestKeyBytes(t *testing.T) {
	cases := []struct {
		k    tea.KeyMsg
		app  bool
		want string
	}{
		{tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("hé")}, false, "hé"},
		{tea.KeyMsg{Type: tea.KeyEnter}, false, "\r"},
		{tea.KeyMsg{Type: tea.KeyBackspace}, false, "\x7f"},
		{tea.KeyMsg{Type: tea.KeyCtrlC}, false, "\x03"},
		{tea.KeyMsg{Type: tea.KeyEsc}, false, "\x1b"},
		{tea.KeyMsg{Type: tea.KeyTab}, false, "\t"},
		{tea.KeyMsg{Type: tea.KeySpace}, false, " "},
		{tea.KeyMsg{Type: tea.KeyUp}, false, "\x1b[A"},
		{tea.KeyMsg{Type: tea.KeyUp}, true, "\x1bOA"},
		{tea.KeyMsg{Type: tea.KeyPgDown}, false, "\x1b[6~"},
		{tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x"), Alt: true}, false, "\x1bx"},
	}
	for _, c := range cases {
		if got := string(KeyBytes(c.k, c.app)); got != c.want {
			t.Errorf("%v (app=%v): got %q want %q", c.k, c.app, got, c.want)
		}
	}
}

// TestRealPTY runs a real program through ConPTY / a Unix pty.
func TestRealPTY(t *testing.T) {
	argv := []string{"sh", "-c", "echo hello-from-pty"}
	if runtime.GOOS == "windows" {
		argv = []string{"cmd", "/c", "echo hello-from-pty"}
	}
	s, err := Start(argv, 40, 5)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	select {
	case <-s.Done():
	case <-time.After(10 * time.Second):
		t.Fatalf("program did not finish; screen %q", s.Text())
	}
	if !strings.Contains(s.Text(), "hello-from-pty") || s.Err() != nil {
		t.Fatalf("screen %q err %v", s.Text(), s.Err())
	}
}
