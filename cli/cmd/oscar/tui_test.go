package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Blacklotus89898/Oscar/cli/internal/kb"
	"github.com/Blacklotus89898/Oscar/cli/internal/term"
	"github.com/Blacklotus89898/Oscar/cli/internal/workspace"
)

func newTestTUI(t *testing.T, mode workspace.Mode, scope ...string) *tui {
	t.Helper()
	t.Setenv("PATH", "") // no claude, no script: deterministic
	w, err := workspace.Create(t.TempDir(), "test", mode, scope)
	if err != nil {
		t.Fatal(err)
	}
	k := &kb.KB{Root: t.TempDir()}
	os.MkdirAll(filepath.Join(k.Root, "01-methodology"), 0o755)
	os.WriteFile(filepath.Join(k.Root, "01-methodology", "checklist.md"),
		[]byte("## Recon\n- [ ] Full TCP port scan\n- [ ] UDP scan\n"), 0o644)
	m := newTUI(context.Background(), w, k)
	m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	return m
}

// do runs cmd and feeds the resulting message back, like the runtime.
// Timers (cursor blink, spinner) are dropped: anything slower than 200ms.
func do(m *tui, cmd tea.Cmd) {
	for i := 0; cmd != nil && i < 10; i++ {
		ch := make(chan tea.Msg, 1)
		go func(c tea.Cmd) { ch <- c() }(cmd)
		var msg tea.Msg
		select {
		case msg = <-ch:
		case <-time.After(200 * time.Millisecond):
		}
		if msg == nil {
			return
		}
		if b, ok := msg.(tea.BatchMsg); ok {
			for _, c := range b {
				do(m, c)
			}
			return
		}
		_, cmd = m.Update(msg)
	}
}

func press(m *tui, keys ...string) {
	for _, k := range keys {
		var msg tea.KeyMsg
		switch k {
		case "enter":
			msg = tea.KeyMsg{Type: tea.KeyEnter}
		case "esc":
			msg = tea.KeyMsg{Type: tea.KeyEsc}
		case " ":
			msg = tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}
		default:
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
		}
		_, cmd := m.Update(msg)
		do(m, cmd)
	}
}

func typeText(m *tui, s string) {
	for _, r := range s {
		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		do(m, cmd)
	}
}

func TestAddTargetAndScopeRefusal(t *testing.T) {
	m := newTestTUI(t, workspace.ModeHTB, "10.10.10.0/24")
	do(m, m.Init())
	if !strings.Contains(m.View(), "No targets yet") {
		t.Fatal("empty state not shown")
	}
	press(m, "a")
	typeText(m, "8.8.8.8 google")
	press(m, "enter")
	if !m.msgErr || !strings.Contains(m.msg, "OUT OF SCOPE") {
		t.Fatalf("out-of-scope add not refused: %q", m.msg)
	}
	press(m, "a")
	typeText(m, "10.10.10.3 Lame")
	press(m, "enter")
	if m.msgErr || len(m.rows) != 1 || m.rows[0].t.Label() != "Lame (10.10.10.3)" {
		t.Fatalf("add failed: %q rows=%d", m.msg, len(m.rows))
	}
	v := m.View()
	for _, want := range []string{"Lame (10.10.10.3)", "u· r·", "0/2", "no scan results yet", "Full TCP port scan"} {
		if !strings.Contains(v, want) {
			t.Errorf("view missing %q", want)
		}
	}
}

func TestFlagMenuAndFormat(t *testing.T) {
	m := newTestTUI(t, workspace.ModeHTB, "10.10.10.0/24")
	m.w.AddTarget("10.10.10.3", "Lame")
	do(m, m.loadRows())
	press(m, "f", "2") // root.txt
	typeText(m, "nope")
	press(m, "enter")
	if !m.msgErr || !strings.Contains(m.msg, "flag format") {
		t.Fatalf("bad flag accepted: %q", m.msg)
	}
	press(m, "f", "2")
	typeText(m, "0123456789abcdef0123456789abcdef")
	press(m, "enter")
	if m.msgErr || m.rows[0].t.Status != "pwned" || !strings.Contains(m.msg, "box tracker") {
		t.Fatalf("flag not saved: %q %s", m.msg, m.rows[0].t.Status)
	}
	h, _ := m.w.History()
	if len(h) == 0 || h[len(h)-1].Action != "flag root.txt" || strings.Contains(h[len(h)-1].Command, "0123") {
		t.Fatalf("flag not logged (or value leaked into the log): %+v", h)
	}
}

func TestChecklistToggle(t *testing.T) {
	m := newTestTUI(t, workspace.ModeLab, "10.10.10.0/24")
	m.w.AddTarget("10.10.10.5", "")
	do(m, m.loadRows())
	press(m, "c")
	if m.view != viewChecklist || !strings.Contains(m.View(), "0/2") {
		t.Fatal("checklist not open")
	}
	press(m, " ")
	if !strings.Contains(m.View(), "1/2") {
		t.Fatalf("tick not shown:\n%s", m.View())
	}
	tg, _ := m.w.Target("10.10.10.5")
	if !tg.IsChecked("Recon / Full TCP port scan") {
		t.Fatal("tick not saved")
	}
	press(m, "esc")
	if m.view != viewDashboard {
		t.Fatal("esc did not return")
	}
}

func TestExamModeHasNoClaude(t *testing.T) {
	m := newTestTUI(t, workspace.ModeExam, "10.10.10.0/24")
	if m.chat.a != nil || m.chat.show {
		t.Fatal("assistant available in exam mode")
	}
	press(m, "tab")
	if !strings.Contains(m.msg, "exam mode") {
		t.Fatalf("no explanation: %q", m.msg)
	}
	if !strings.Contains(m.View(), "EXAM") {
		t.Fatal("exam badge missing")
	}
}

func TestCTFAddChallengeAndScore(t *testing.T) {
	m := newTestTUI(t, workspace.ModeCTF, "chall.example.com")
	press(m, "a")
	typeText(m, "Baby SQL | web | 100 | chall.example.com:31337")
	press(m, "enter")
	if m.msgErr || len(m.rows) != 1 {
		t.Fatalf("challenge not added: %q", m.msg)
	}
	if v := m.View(); !strings.Contains(v, "  web · 100 pts · chall.example.com:31337") || !strings.Contains(v, "0/100 pts") {
		t.Fatalf("view:\n%s", v)
	}
	press(m, "f") // single flag name: straight to the prompt
	typeText(m, "flag{union}")
	press(m, "enter")
	if !strings.Contains(m.View(), "100/100 pts") {
		t.Fatalf("score not updated: %q", m.msg)
	}
}

// fakeTerm starts a real local process if possible; otherwise reports failure.
func TestScanLaunchesLoggedSession(t *testing.T) {
	m := newTestTUI(t, workspace.ModeLab, "10.10.10.0/24")
	m.w.AddTarget("10.10.10.5", "")
	do(m, m.loadRows())
	var got []string
	var gotDir string
	startTerm = func(argv []string, dir string, env []string, cols, rows int) (*term.Session, error) {
		got, gotDir = argv, dir
		return nil, errors.New("no pty in tests")
	}
	defer func() { startTerm = term.StartIn }()
	press(m, "s", "f")
	joined := strings.Join(got, " ")
	if !strings.HasPrefix(joined, "nmap -p- ") || !strings.HasSuffix(joined, "10.10.10.5") || !strings.Contains(joined, filepath.Join("scans", "full")) {
		t.Fatalf("argv = %q", joined)
	}
	if !strings.HasSuffix(gotDir, "10.10.10.5") {
		t.Fatalf("dir = %q", gotDir)
	}
	h, _ := m.w.History()
	if len(h) != 1 || h[0].Action != "scan full 10.10.10.5" || h[0].Target != "10.10.10.5" {
		t.Fatalf("scan not logged before running: %+v", h)
	}
	// deep scan needs known ports
	press(m, "s", "d")
	if !strings.Contains(m.msg, "no open ports known") {
		t.Fatalf("deep without ports: %q", m.msg)
	}
}

func TestShellJoin(t *testing.T) {
	got := shellJoin([]string{"env", "RHOST=10.0.0.1", "nmap", "-oA", "/tmp/a b/full", "it's"})
	want := `env RHOST=10.0.0.1 nmap -oA '/tmp/a b/full' 'it'\''s'`
	if got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
}

func TestViewSurvivesTinyAndHugeTerminals(t *testing.T) {
	m := newTestTUI(t, workspace.ModeLab, "10.10.10.0/24")
	m.w.AddTarget("10.10.10.5", "a-very-long-machine-name-that-overflows")
	do(m, m.loadRows())
	for _, sz := range [][2]int{{20, 5}, {80, 24}, {300, 90}} {
		m.Update(tea.WindowSizeMsg{Width: sz[0], Height: sz[1]})
		_ = m.View()
		press(m, "c")
		_ = m.View()
		press(m, "esc", "?")
		_ = m.View()
		press(m, "x")
	}
}

func TestNewRefusesGitRepo(t *testing.T) {
	repo := t.TempDir()
	os.Mkdir(filepath.Join(repo, ".git"), 0o755)
	err := cmdNew([]string{"x", "-dir", filepath.Join(repo, "engagements", "x")})
	if err == nil || !strings.Contains(err.Error(), "inside the git repo") {
		t.Fatalf("err = %v", err)
	}
}
