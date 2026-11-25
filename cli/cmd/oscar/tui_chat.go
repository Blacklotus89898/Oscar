package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Blacklotus89898/Oscar/cli/internal/assistant"
	"github.com/Blacklotus89898/Oscar/cli/internal/workspace"
)

// The assistant panel (Claude, OpenCode or any OpenAI-compatible model): chat-only (no tools, no MCP), and it sees a snapshot of
// the dashboard with each question. It is a tutor, not an autopilot. It never
// exists in exam mode: OffSec prohibits AI assistance during the exam, and in
// pentest mode it is off unless enabled (client data stays on the box).

type chunkMsg string

type chatDoneMsg struct{ err error }

type turn struct{ who, text string }

type chat struct {
	a        assistant.Backend
	off      string // why a is nil
	input    textinput.Model
	vp       viewport.Model
	turns    []turn
	show     bool
	focus    bool
	insert   bool
	thinking bool
	ch       chan tea.Msg
}

var (
	youStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
	claudeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Bold(true)
	panelStyle  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240")).Padding(0, 1)
	panelFocus  = panelStyle.BorderForeground(lipgloss.Color("208"))
)

const welcome = `I'm your study partner. I see this dashboard (target, open ports, your checklist progress) and any session's screen, but I can't run anything.

I'll explain what you're looking at, ask the question that gets you unstuck, and point you to the right page of your Oscar notes — not hand you the answer, because the exam won't.

tab ask · l focus · x explain · \ hide`

const assistantSystem = `You are the study-partner panel inside oscar, a terminal workspace for OSCP preparation, Hack The Box and CTFs. The user owns an "Oscar" knowledge base: 00-exam-guide, 01-methodology (methodology + per-target checklist), 02-enumeration, 03-web, 04-exploitation, 05-privesc (linux/windows), 06-active-directory, 07-pivoting, 08-password-attacks, 09-reporting, practice/ (box list, tracker, note template). Point to those docs by path when relevant (the user opens them with d or searches with /).

How to help: teach methodology. Prefer Socratic hints in increasing strength — what to look at, why it matters, which checklist item they skipped — over complete solutions or spoilers for a named box. Explain tool output on screen. Encourage note-taking and screenshots for the report. Be concise: the panel is about 45 columns wide; short paragraphs, plain text, no tables.

Boundaries: only discuss the targets in the dashboard snapshot, which are in the user's authorized scope (lab, HTB, CTF). If asked about anything outside that scope, decline. You cannot run commands or read files.`

func newChat(w *workspace.Workspace) chat {
	ti := textinput.New()
	ti.Placeholder = "ask…"
	ti.Prompt = "› "
	ti.CharLimit = 2000
	c := chat{input: ti, vp: viewport.New(40, 10), turns: []turn{{"claude", welcome}}}
	switch {
	case w.Cfg.Mode == workspace.ModeExam:
		c.off = "exam mode: no AI assistance (OffSec rules)"
		return c
	case !w.Cfg.AssistantAllowed():
		c.off = "the assistant is off in pentest mode; set assistant.enabled: true in oscar.yaml if your client allows it"
		return c
	}
	// A private directory: some CLIs (OpenCode) read instruction files from
	// their working directory, and none of them should see the engagement.
	dir := filepath.Join(os.TempDir(), "oscar-assistant")
	if d, err := os.UserCacheDir(); err == nil {
		dir = filepath.Join(d, "oscar", "assistant")
	}
	b, err := assistant.New(w.Cfg.Assistant.Config, assistantSystem, dir)
	if err != nil {
		c.off = "assistant unavailable: " + err.Error()
		return c
	}
	c.a = b
	c.show = true
	return c
}

// snapshot is the dashboard state sent with each question. Flag values and
// credentials are never included.
func (m *tui) snapshot() string {
	var b strings.Builder
	fmt.Fprintf(&b, "[dashboard] engagement %q, mode %s, LHOST %s.\n", m.w.Cfg.Name, m.w.Cfg.Mode, orDash(m.lhost))
	if r, ok := m.selected(); ok {
		t := r.t
		fmt.Fprintf(&b, "Selected: %s, status %s", t.Label(), t.Status)
		if t.IsChallenge() {
			fmt.Fprintf(&b, ", %s challenge worth %d", t.Category, t.Points)
		}
		var have []string
		for _, f := range t.Flags {
			have = append(have, f.Name)
		}
		if len(have) > 0 {
			fmt.Fprintf(&b, ", flags captured: %s", strings.Join(have, ", "))
		}
		b.WriteString(".\n")
		if os := firstNonEmpty(t.OS, r.h.OS); os != "" {
			fmt.Fprintf(&b, "OS guess: %s.\n", os)
		}
		if len(r.h.Ports) > 0 {
			b.WriteString("Open ports:")
			for _, p := range r.h.Ports {
				b.WriteString(" " + p.Label() + ";")
			}
			b.WriteString("\n")
		}
		if len(m.items) > 0 && !t.IsChallenge() {
			var done, todo []string
			for _, it := range m.items {
				if t.IsChecked(it.Key()) {
					done = append(done, it.Text)
				} else if len(todo) < 8 {
					todo = append(todo, it.Text)
				}
			}
			fmt.Fprintf(&b, "Checklist: %d/%d done. Next unticked: %s.\n", len(done), len(m.items), strings.Join(todo, " | "))
		}
	}
	if name, screen := m.screenForClaude(); name != "" {
		fmt.Fprintf(&b, "The user is looking at session %q. Visible screen:\n```\n%s\n```\n", name, screen)
	} else if m.view == viewDocs && m.docOpen != "" {
		fmt.Fprintf(&b, "The user is reading %s.\n", m.docOpen)
	}
	if m.msg != "" {
		fmt.Fprintf(&b, "Last message: %s\n", m.msg)
	}
	return b.String()
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func (m *tui) askClaude(question string) tea.Cmd {
	c := &m.chat
	if c.a == nil {
		m.setMsg(c.off, true)
		return nil
	}
	if c.thinking {
		return nil
	}
	c.show = true
	m.layout()
	c.turns = append(c.turns, turn{"you", question}, turn{"claude", ""})
	c.thinking = true
	m.refreshChat()

	prompt := m.snapshot() + "\n" + question
	ch := make(chan tea.Msg, 256)
	c.ch = ch
	a := c.a
	go func() {
		ctx, cancel := context.WithTimeout(m.ctx, 3*time.Minute)
		defer cancel()
		streamed := false
		reply, err := a.Ask(ctx, prompt, func(s string) { streamed = true; ch <- chunkMsg(s) })
		if !streamed && reply != "" {
			ch <- chunkMsg(reply)
		}
		ch <- chatDoneMsg{err}
		close(ch)
	}()
	return waitChat(ch)
}

func waitChat(ch chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return nil
		}
		return msg
	}
}

func (m *tui) chatUpdate(msg tea.Msg) (tea.Cmd, bool) {
	c := &m.chat
	switch msg := msg.(type) {
	case chunkMsg:
		c.turns[len(c.turns)-1].text += string(msg)
		m.refreshChat()
		return waitChat(c.ch), true
	case chatDoneMsg:
		c.thinking = false
		last := &c.turns[len(c.turns)-1]
		if msg.err != nil {
			last.text = strings.TrimSpace(last.text + "\n(" + msg.err.Error() + ")")
		}
		if strings.TrimSpace(last.text) == "" {
			last.text = "(no answer)"
		}
		m.refreshChat()
		return nil, true
	}
	return nil, false
}

func (m *tui) chatKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	c := &m.chat
	if k.String() == "ctrl+c" && m.view == viewDashboard {
		return m, tea.Quit
	}
	if !c.insert {
		switch k.String() {
		case "i", "a", "enter":
			c.insert = true
			return m, c.input.Focus()
		case "h", "esc", "tab", "left":
			m.leaveChat()
		case "j", "down":
			c.vp.LineDown(1)
		case "k", "up":
			c.vp.LineUp(1)
		case "ctrl+d", "pgdown":
			c.vp.HalfViewDown()
		case "ctrl+u", "pgup":
			c.vp.HalfViewUp()
		case "g", "home":
			c.vp.GotoTop()
		case "G", "end":
			c.vp.GotoBottom()
		case "?":
			m.help = true
		}
		return m, nil
	}
	switch k.String() {
	case "esc":
		c.insert = false
		c.input.Blur()
		return m, nil
	case "tab":
		m.leaveChat()
		return m, nil
	case "enter":
		q := strings.TrimSpace(c.input.Value())
		if q == "" || c.thinking {
			return m, nil
		}
		c.input.SetValue("")
		return m, m.askClaude(q)
	}
	var cmd tea.Cmd
	c.input, cmd = c.input.Update(k)
	return m, cmd
}

func (m *tui) focusChat(insert bool) tea.Cmd {
	if m.chat.a == nil {
		m.askClaude("") // sets the reason it's unavailable
		return nil
	}
	m.chat.show, m.chat.focus, m.chat.insert = true, true, insert
	m.tbl.Blur()
	m.layout()
	if insert {
		return m.chat.input.Focus()
	}
	m.chat.input.Blur()
	return nil
}

func (m *tui) leaveChat() {
	m.chat.focus, m.chat.insert = false, false
	m.chat.input.Blur()
	if m.view == viewDashboard {
		m.tbl.Focus()
	}
}

func (m *tui) chatHint() string {
	if m.chat.insert {
		return "INSERT · enter send · esc normal mode · tab back"
	}
	return "NORMAL · i type · j/k scroll · g/G top/bottom · h back"
}

func (m *tui) refreshChat() {
	c := &m.chat
	w := max(20, c.vp.Width)
	var b strings.Builder
	for i, t := range c.turns {
		if i > 0 {
			b.WriteString("\n\n")
		}
		label := claudeStyle.Render("claude")
		if t.who == "you" {
			label = youStyle.Render("you")
		}
		text := t.text
		if t.who == "claude" && text == "" && c.thinking && i == len(c.turns)-1 {
			text = m.spin.View() + " thinking…"
		}
		b.WriteString(label + "\n" + lipgloss.NewStyle().Width(w).Render(text))
	}
	c.vp.SetContent(b.String())
	c.vp.GotoBottom()
}

// sideBySide is the width from which the panel sits beside the main view.
const sideBySide = 110

func (m *tui) panelWidth() int { return min(60, max(38, m.width/3)) }

func (m *tui) layout() {
	if m.view == viewSession {
		cols, rows := m.termSize()
		for _, ts := range m.sessions {
			ts.t.Resize(cols, rows)
		}
		if m.sessionPanel() {
			inner := m.panelWidth() - 4
			m.chat.vp.Width, m.chat.vp.Height = inner, max(3, rows-2)
			m.chat.input.Width = inner - 3
			m.refreshChat()
		}
		return
	}
	panel := m.chat.show && m.width >= sideBySide
	mainW := m.width
	if panel {
		mainW = m.width - m.panelWidth() - 1
	}
	compact := mainW < wideWidth
	if compact != m.compact || len(m.tbl.Columns()) == 0 {
		m.tbl.SetRows(nil) // rows must match the column count
		if compact {
			m.tbl.SetColumns(compactColumns)
		} else {
			m.tbl.SetColumns(wideColumns)
		}
		m.compact = compact
		m.tbl.SetRows(m.tableRows())
	}
	m.tbl.SetHeight(max(4, min(len(m.rows)+2, (m.height-6)/2)))
	m.docVP.Width, m.docVP.Height = max(20, mainW-2), max(5, m.height-5)
	if panel {
		inner := m.panelWidth() - 4
		m.chat.vp.Width, m.chat.vp.Height = inner, max(3, m.height-8)
		m.chat.input.Width = inner - 3
		m.refreshChat()
	}
}

func (m *tui) panelView() string {
	style := panelStyle
	if m.chat.focus {
		style = panelFocus
	}
	title := claudeStyle.Render(m.chat.a.Name())
	switch {
	case m.chat.focus && m.chat.insert:
		title += dimStyle.Render("  INSERT · esc normal")
	case m.chat.focus:
		title += dimStyle.Render("  NORMAL · i type · h back")
	default:
		title += dimStyle.Render("  tab ask · l focus")
	}
	return style.Render(title + "\n" + m.chat.vp.View() + "\n" + m.chat.input.View())
}
