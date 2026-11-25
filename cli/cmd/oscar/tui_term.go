package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Blacklotus89898/Oscar/cli/internal/term"
)

// Embedded sessions (shells, scans, editors, VPN) run in a pseudo-terminal
// drawn inside the dashboard. While one has focus every key goes to it except
// the prefix, ctrl+], which reaches oscar (tmux-style). Adapted from tether.

// startTerm launches sessions; tests swap it out.
var startTerm = term.StartIn

type termSession struct {
	id   int
	name string
	t    *term.Session
}

type termDirtyMsg struct{ id int }

type termEndMsg struct {
	id  int
	err error
}

type viewMode int

const (
	viewDashboard viewMode = iota
	viewSession
	viewChecklist
	viewDocs
)

var (
	termBorder      = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240"))
	termBorderFocus = termBorder.BorderForeground(lipgloss.Color("42"))
	tabStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Padding(0, 1)
	tabActive       = lipgloss.NewStyle().Foreground(lipgloss.Color("0")).Background(lipgloss.Color("42")).Bold(true).Padding(0, 1)
	prefixStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("0")).Background(lipgloss.Color("39")).Padding(0, 1)
)

const prefixHelp = "d back · n/p or 1-9 sessions · l Claude · x explain screen · \\ panel · ] send ctrl+]"

func waitTerm(ts *termSession) tea.Cmd {
	return func() tea.Msg {
		select {
		case <-ts.t.Dirty():
			return termDirtyMsg{ts.id}
		case <-ts.t.Done():
			return termEndMsg{ts.id, ts.t.Err()}
		}
	}
}

func (m *tui) findSession(name string) int {
	for i, ts := range m.sessions {
		if ts.name == name {
			return i
		}
	}
	return -1
}

func (m *tui) activeSession() *termSession {
	if m.view != viewSession || m.active < 0 || m.active >= len(m.sessions) {
		return nil
	}
	return m.sessions[m.active]
}

func (m *tui) embed(a action) tea.Cmd {
	if m.view != viewSession {
		m.prev = m.view
	}
	m.view = viewSession // so termSize measures the session layout
	cols, rows := m.termSize()
	t, err := startTerm(a.argv, a.dir, a.env, cols, rows)
	if err != nil {
		m.view = m.prev
		m.layout()
		m.setMsg(fmt.Sprintf("could not start %s: %v", a.name, err), true)
		return nil
	}
	m.nextID++
	ts := &termSession{id: m.nextID, name: a.name, t: t}
	m.sessions = append(m.sessions, ts)
	m.showSession(len(m.sessions) - 1)
	m.setMsg("", false)
	return waitTerm(ts)
}

func (m *tui) showSession(i int) {
	if m.view != viewSession {
		m.prev = m.view
	}
	m.active, m.view, m.prefix = i, viewSession, false
	m.leaveChat()
	m.tbl.Blur()
	m.layout()
}

func (m *tui) showDashboard() {
	m.view, m.prefix = viewDashboard, false
	if m.prev == viewChecklist || m.prev == viewDocs {
		m.view = m.prev
	}
	m.leaveChat()
	m.layout()
}

func (m *tui) termSize() (int, int) {
	w := m.width
	if m.sessionPanel() {
		w -= m.panelWidth() + 1
	}
	return max(10, w-2), max(3, m.height-4)
}

func (m *tui) sessionPanel() bool { return m.chat.show && m.width >= 90 }

func (m *tui) termUpdate(msg tea.Msg) (tea.Cmd, bool) {
	switch msg := msg.(type) {
	case termDirtyMsg:
		for _, ts := range m.sessions {
			if ts.id == msg.id {
				return waitTerm(ts), true
			}
		}
		return nil, true
	case termEndMsg:
		for i, ts := range m.sessions {
			if ts.id != msg.id {
				continue
			}
			m.lastScreen = ts.t.Text()
			m.sessions = append(m.sessions[:i], m.sessions[i+1:]...)
			wasActive := m.view == viewSession && m.active == i
			if m.active >= i && m.active > 0 {
				m.active--
			}
			last := lastLines(m.lastScreen, 1)
			switch {
			case msg.err != nil:
				m.setMsg(fmt.Sprintf("%s ended (%v): %s", ts.name, msg.err, last), true)
			case strings.HasPrefix(ts.name, "scan "):
				m.setMsg(ts.name+" finished — results parsed into the dashboard", false)
			default:
				m.setMsg(ts.name+" closed", false)
			}
			if wasActive {
				m.showDashboard()
			}
			return m.loadRows(), true // scans and notes change what we show
		}
		return nil, true
	}
	return nil, false
}

func (m *tui) sessionKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	ts := m.activeSession()
	if ts == nil {
		m.showDashboard()
		return m, nil
	}
	if m.chat.focus {
		return m.chatKey(k)
	}
	if m.prefix {
		m.prefix = false
		switch s := k.String(); s {
		case "l", "tab":
			return m, m.focusChat(s == "tab")
		case "d", "q", "esc":
			m.showDashboard()
			return m, nil
		case "n", "p":
			step := 1
			if s == "p" {
				step = len(m.sessions) - 1
			}
			m.showSession((m.active + step) % len(m.sessions))
			return m, nil
		case `\`:
			m.chat.show = !m.chat.show && m.chat.a != nil
			m.layout()
			return m, nil
		case "x":
			return m, m.askClaude("Look at my " + ts.name + " screen (in the snapshot). Explain what is going on and what I should learn from it.")
		case "?":
			m.help = true
			return m, nil
		case "]", "ctrl+]":
			_ = ts.t.Write([]byte{0x1d})
			return m, nil
		}
		if len(k.Runes) == 1 && k.Runes[0] >= '1' && k.Runes[0] <= '9' {
			if i := int(k.Runes[0] - '1'); i < len(m.sessions) {
				m.showSession(i)
			}
		}
		return m, nil
	}
	if k.Type == tea.KeyCtrlCloseBracket {
		m.prefix = true
		return m, nil
	}
	if b := term.KeyBytes(k, ts.t.AppCursor()); len(b) > 0 {
		if err := ts.t.Write(b); err != nil {
			m.setMsg(ts.name+": "+err.Error(), true)
		}
	}
	return m, nil
}

func (m *tui) tabsView() string {
	var tabs []string
	for i, ts := range m.sessions {
		label := fmt.Sprintf("%d %s", i+1, ts.name)
		if m.view == viewSession && i == m.active {
			tabs = append(tabs, tabActive.Render(label))
		} else {
			tabs = append(tabs, tabStyle.Render(label))
		}
	}
	return strings.Join(tabs, "")
}

func (m *tui) sessionView(ts *termSession) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("oscar") + "  " + m.tabsView() + "\n")
	focused := !m.chat.focus
	box := termBorder
	if focused {
		box = termBorderFocus
	}
	screen := box.Render(ts.t.Render(focused && !m.prefix))
	if m.sessionPanel() {
		screen = lipgloss.JoinHorizontal(lipgloss.Top, screen, " ", m.panelView())
	}
	b.WriteString(screen + "\n")
	switch {
	case m.prefix:
		b.WriteString(prefixStyle.Render("ctrl+]") + " " + dimStyle.Render(prefixHelp))
	case m.msg != "" && m.msgErr:
		b.WriteString(errStyle.Render(m.msg))
	case m.chat.focus:
		b.WriteString(dimStyle.Render(m.chatHint()))
	default:
		b.WriteString(dimStyle.Render("recorded to logs/sessions · ctrl+] then: d back · n next · l Claude · ? keys"))
	}
	return b.String()
}

func (m *tui) screenForClaude() (string, string) {
	if ts := m.activeSession(); ts != nil {
		return ts.name, lastLines(ts.t.Text(), 40)
	}
	return "", ""
}

func lastLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}
