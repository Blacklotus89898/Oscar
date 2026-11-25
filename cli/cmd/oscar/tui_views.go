package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	cursorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("0")).Background(lipgloss.Color("42"))
	sectionStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	mdHeading    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42"))
	mdCode       = lipgloss.NewStyle().Foreground(lipgloss.Color("180"))
)

// ---- checklist: 01-methodology/checklist.md, ticked per target

func (m *tui) checklistKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	r, ok := m.selected()
	if !ok {
		m.view = viewDashboard
		return m, nil
	}
	switch k.String() {
	case "esc", "q", "h", "c":
		m.view = viewDashboard
		m.tbl.Focus()
	case "j", "down":
		m.clCursor = min(m.clCursor+1, len(m.items)-1)
	case "k", "up":
		m.clCursor = max(m.clCursor-1, 0)
	case "g":
		m.clCursor = 0
	case "G":
		m.clCursor = len(m.items) - 1
	case "ctrl+d":
		m.clCursor = min(m.clCursor+10, len(m.items)-1)
	case "ctrl+u":
		m.clCursor = max(m.clCursor-10, 0)
	case "n": // next unticked
		for i := m.clCursor + 1; i < len(m.items); i++ {
			if !r.t.IsChecked(m.items[i].Key()) {
				m.clCursor = i
				break
			}
		}
	case " ", "x", "enter":
		if _, err := m.w.Toggle(r.t.ID, m.items[m.clCursor].Key()); err != nil {
			m.setMsg(err.Error(), true)
			return m, nil
		}
		m.clCursor = min(m.clCursor+1, len(m.items)-1)
		return m, m.loadRows()
	case "l", "tab":
		return m, m.focusChat(k.String() == "tab")
	case "?":
		m.help = true
	}
	return m, nil
}

func (m *tui) checklistView() string {
	r, _ := m.selected()
	w := m.mainWidth()
	var lines []string
	cursorLine := 0
	section := ""
	done := 0
	for i, it := range m.items {
		if it.Section != section {
			section = it.Section
			lines = append(lines, sectionStyle.Render(section))
		}
		box := "☐"
		if r.t.IsChecked(it.Key()) {
			box = "☑"
			done++
		}
		line := truncate(fmt.Sprintf("  %s %s", box, plainMD(it.Text)), w-1)
		if i == m.clCursor {
			line = cursorStyle.Render(line)
			cursorLine = len(lines)
		}
		lines = append(lines, line)
	}
	h := max(3, m.height-4)
	start := max(0, min(cursorLine-h/2, len(lines)-h))
	end := min(len(lines), start+h-1)
	title := headStyle.Render("checklist · "+r.t.Label()) + dimStyle.Render(fmt.Sprintf("  %d/%d · space tick · n next unticked · esc back", done, len(m.items)))
	return lipgloss.NewStyle().Width(w).Render(title + "\n" + strings.Join(lines[start:end], "\n"))
}

// plainMD drops markdown emphasis for terminal display.
func plainMD(s string) string {
	return strings.NewReplacer("**", "", "`", "").Replace(s)
}

// ---- docs: browse and search the Oscar knowledge base

func (m *tui) openDocs() {
	if m.k == nil {
		m.setMsg("Oscar knowledge base not found: set OSCAR_KB to your Oscar checkout", true)
		return
	}
	docs, err := m.k.Docs()
	if err != nil {
		m.setMsg(err.Error(), true)
		return
	}
	m.docs, m.hitMode, m.docOpen = docs, false, ""
	m.docCur = min(m.docCur, max(0, len(docs)-1))
	m.view = viewDocs
	m.tbl.Blur()
	m.layout()
}

func (m *tui) searchPrompt() tea.Cmd {
	if m.k == nil {
		m.setMsg("Oscar knowledge base not found: set OSCAR_KB to your Oscar checkout", true)
		return nil
	}
	return m.ask1("search notes", "e.g. kerberoast, sudo -l, smb null session", func(q string) tea.Cmd {
		hits, err := m.k.Search(q, 200)
		if err != nil {
			m.setMsg(err.Error(), true)
			return nil
		}
		if len(hits) == 0 {
			m.setMsg("no match for "+q+" in your notes — when you learn it, write it down there", false)
			return nil
		}
		m.openDocs()
		m.hits, m.hitMode, m.docCur = hits, true, 0
		m.setMsg(fmt.Sprintf("%d matches for %q", len(hits), q), false)
		return nil
	})
}

func (m *tui) openDoc(rel string, line int) {
	text, err := m.k.Read(rel)
	if err != nil {
		m.setMsg(err.Error(), true)
		return
	}
	m.docOpen = rel
	m.docVP.SetContent(renderMD(text, m.docVP.Width))
	m.docVP.SetYOffset(max(0, line-3))
}

func (m *tui) docsKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.docOpen != "" {
		switch k.String() {
		case "esc", "q", "h", "backspace":
			m.docOpen = ""
		case "/":
			return m, m.searchPrompt()
		case "l", "tab":
			return m, m.focusChat(k.String() == "tab")
		default:
			var cmd tea.Cmd
			m.docVP, cmd = m.docVP.Update(k) // j/k, ctrl+d/u, pgup/pgdn
			switch k.String() {
			case "g":
				m.docVP.GotoTop()
			case "G":
				m.docVP.GotoBottom()
			}
			return m, cmd
		}
		return m, nil
	}
	n := len(m.docs)
	if m.hitMode {
		n = len(m.hits)
	}
	switch k.String() {
	case "esc", "q", "d":
		if m.hitMode {
			m.hitMode = false
			m.docCur = 0
			return m, nil
		}
		m.view = viewDashboard
		m.tbl.Focus()
		m.layout()
	case "j", "down":
		m.docCur = min(m.docCur+1, n-1)
	case "k", "up":
		m.docCur = max(m.docCur-1, 0)
	case "g":
		m.docCur = 0
	case "G":
		m.docCur = n - 1
	case "/":
		return m, m.searchPrompt()
	case "enter", "l":
		if m.hitMode && m.docCur < len(m.hits) {
			h := m.hits[m.docCur]
			m.openDoc(h.Doc.Rel, h.Line)
		} else if !m.hitMode && m.docCur < len(m.docs) {
			m.openDoc(m.docs[m.docCur].Rel, 0)
		}
	case "tab":
		return m, m.focusChat(true)
	case "?":
		m.help = true
	}
	return m, nil
}

func (m *tui) docsView() string {
	w := m.mainWidth()
	if m.docOpen != "" {
		title := headStyle.Render(m.docOpen) + dimStyle.Render(fmt.Sprintf("  %3.0f%% · j/k scroll · / search · esc back", m.docVP.ScrollPercent()*100))
		return lipgloss.NewStyle().Width(w).Render(title + "\n" + m.docVP.View())
	}
	var lines []string
	var title string
	if m.hitMode {
		title = headStyle.Render("search results") + dimStyle.Render("  enter open · esc all docs")
		for i, h := range m.hits {
			line := truncate(fmt.Sprintf("%-34s %s", truncate(h.Doc.Rel+" › "+h.Heading, 34), plainMD(h.Text)), w-1)
			if i == m.docCur {
				line = cursorStyle.Render(line)
			}
			lines = append(lines, line)
		}
	} else {
		title = headStyle.Render("Oscar knowledge base") + dimStyle.Render("  enter open · / search · esc back")
		for i, d := range m.docs {
			line := truncate(fmt.Sprintf("%-40s %s", d.Rel, dimStyle.Render(d.Title)), w-1)
			if i == m.docCur {
				line = cursorStyle.Render(truncate(fmt.Sprintf("%-40s %s", d.Rel, d.Title), w-1))
			}
			lines = append(lines, line)
		}
	}
	h := max(3, m.height-4)
	start := max(0, min(m.docCur-h/2, len(lines)-h+1))
	end := min(len(lines), start+h-1)
	return lipgloss.NewStyle().Width(w).Render(title + "\n" + strings.Join(lines[start:end], "\n"))
}

// renderMD is a light markdown pass for the terminal: headings and code
// blocks coloured, emphasis markers dropped, long lines wrapped.
func renderMD(s string, width int) string {
	var out []string
	fence := false
	wrap := lipgloss.NewStyle().Width(max(20, width))
	for _, l := range strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n") {
		switch {
		case strings.HasPrefix(l, "```"):
			fence = !fence
			out = append(out, dimStyle.Render(l))
		case fence:
			out = append(out, mdCode.Render(truncate(l, width)))
		case strings.HasPrefix(l, "#"):
			out = append(out, mdHeading.Render(strings.TrimSpace(strings.TrimLeft(l, "#"))))
		default:
			out = append(out, wrap.Render(plainMD(l)))
		}
	}
	return strings.Join(out, "\n")
}
