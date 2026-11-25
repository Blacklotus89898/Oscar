package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Blacklotus89898/Oscar/cli/internal/kb"
	"github.com/Blacklotus89898/Oscar/cli/internal/nmap"
	"github.com/Blacklotus89898/Oscar/cli/internal/report"
	"github.com/Blacklotus89898/Oscar/cli/internal/workspace"
)

func runTUI(ctx context.Context) error {
	w, err := workspace.Find()
	if err != nil {
		return err
	}
	k, _ := kb.Find() // optional: without it, no checklist/docs
	m := newTUI(ctx, w, k)
	_, err = tea.NewProgram(m, tea.WithAltScreen(), tea.WithContext(ctx)).Run()
	for _, ts := range m.sessions {
		ts.t.Close()
	}
	if errors.Is(err, tea.ErrProgramKilled) {
		return nil
	}
	return err
}

// ---- messages and small modal states

type row struct {
	t workspace.Target
	h nmap.Host
}

type rowsMsg struct {
	rows []row
	err  error
}

type doneMsg struct {
	text   string
	err    error
	reload bool
}

type confirm struct {
	prompt string
	run    tea.Cmd
}

// menu waits for one key.
type menu struct {
	prompt string
	opts   map[string]func() tea.Cmd
}

// prompt reads one line.
type prompt struct {
	label  string
	in     textinput.Model
	submit func(string) tea.Cmd
}

// ---- model

type tui struct {
	ctx     context.Context
	w       *workspace.Workspace
	k       *kb.KB
	items   []kb.Item // methodology checklist
	rows    []row
	tbl     table.Model
	spin    spinner.Model
	lhost   string
	busy    string
	msg     string
	msgErr  bool
	ask     *confirm
	menu    *menu
	prompt  *prompt
	help    bool
	width   int
	height  int
	compact bool
	chat    chat

	// checklist view
	clCursor int
	// docs view
	docs    []kb.Doc
	docCur  int
	hits    []kb.Hit
	hitMode bool
	docOpen string
	docVP   viewport.Model

	sessions   []*termSession
	active     int
	view       viewMode
	prev       viewMode // where ctrl+] d returns to
	prefix     bool
	lastScreen string
	nextID     int
}

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42"))
	dimStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	okStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	warnStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	errStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	askStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("0")).Background(lipgloss.Color("214")).Padding(0, 1)
	examBadge  = lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Background(lipgloss.Color("160")).Bold(true).Padding(0, 1)
	modeBadge  = lipgloss.NewStyle().Foreground(lipgloss.Color("0")).Background(lipgloss.Color("42")).Bold(true).Padding(0, 1)
	headStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
)

var wideColumns = []table.Column{
	{Title: "TARGET", Width: 24}, {Title: "STATUS", Width: 9}, {Title: "PORTS / CATEGORY", Width: 30},
	{Title: "FLAGS", Width: 12}, {Title: "CHECK", Width: 7},
}

var compactColumns = []table.Column{
	{Title: "TARGET", Width: 22}, {Title: "STATUS", Width: 9}, {Title: "FLAGS", Width: 9},
}

const wideWidth = 24 + 9 + 30 + 12 + 7 + 2*5
const compactWidth = 22 + 9 + 9 + 2*3

func newTUI(ctx context.Context, w *workspace.Workspace, k *kb.KB) *tui {
	t := table.New(table.WithColumns(wideColumns), table.WithFocused(true), table.WithHeight(10))
	st := table.DefaultStyles()
	st.Header = st.Header.BorderStyle(lipgloss.NormalBorder()).BorderBottom(true).Bold(true)
	st.Selected = st.Selected.Foreground(lipgloss.Color("0")).Background(lipgloss.Color("42")).Bold(true)
	t.SetStyles(st)
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	m := &tui{ctx: ctx, w: w, k: k, tbl: t, spin: sp, width: 120, height: 32, docVP: viewport.New(80, 20)}
	if k != nil {
		m.items, _ = k.Checklist()
	}
	m.lhost = lhost(w.Cfg.LHOSTIface)
	m.chat = newChat(w)
	m.layout()
	return m
}

func (m *tui) Init() tea.Cmd { return tea.Batch(m.spin.Tick, m.loadRows()) }

func (m *tui) loadRows() tea.Cmd {
	w := m.w
	return func() tea.Msg {
		if err := w.ReloadScope(); err != nil {
			return rowsMsg{err: err}
		}
		ts, err := w.Targets()
		if err != nil {
			return rowsMsg{err: err}
		}
		rows := make([]row, len(ts))
		for i, t := range ts {
			rows[i] = row{t: t}
			if !t.IsChallenge() {
				rows[i].h, _ = nmap.LoadDir(filepath.Join(t.Dir, "scans"), t.Host)
			}
		}
		return rowsMsg{rows: rows}
	}
}

func (m *tui) setMsg(text string, isErr bool) { m.msg, m.msgErr, m.busy = text, isErr, "" }

func (m *tui) selected() (row, bool) {
	i := m.tbl.Cursor()
	if i < 0 || i >= len(m.rows) {
		return row{}, false
	}
	return m.rows[i], true
}

func (m *tui) background(label string, reload bool, fn func() (string, error)) tea.Cmd {
	m.busy, m.msg = label, ""
	return func() tea.Msg {
		text, err := fn()
		return doneMsg{text: text, err: err, reload: reload}
	}
}

// launch runs an action (built with its scope check) in an embedded session.
func (m *tui) launch(target string, a action, err error) tea.Cmd {
	if err != nil {
		m.setMsg(err.Error(), true)
		return nil
	}
	if i := m.findSession(a.name); i >= 0 {
		m.showSession(i)
		return nil
	}
	a, err = prepare(m.w, target, a)
	if err != nil {
		m.setMsg(err.Error(), true)
		return nil
	}
	return m.embed(a)
}

// ---- update

func (m *tui) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if cmd, handled := m.chatUpdate(msg); handled {
		return m, cmd
	}
	if cmd, handled := m.termUpdate(msg); handled {
		return m, cmd
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.layout()
		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		if m.chat.thinking {
			m.refreshChat()
		}
		return m, cmd
	case rowsMsg:
		m.busy = ""
		if msg.err != nil {
			m.setMsg("refresh failed: "+msg.err.Error(), true)
			return m, nil
		}
		m.rows = msg.rows
		m.lhost = lhost(m.w.Cfg.LHOSTIface)
		m.tbl.SetRows(m.tableRows())
		return m, nil
	case doneMsg:
		m.setMsg(msg.text, msg.err != nil)
		if msg.err != nil {
			m.msg = msg.err.Error()
		}
		if msg.reload {
			return m, m.loadRows()
		}
		return m, nil
	case tea.KeyMsg:
		return m.key(msg)
	}
	var cmd tea.Cmd
	m.tbl, cmd = m.tbl.Update(msg)
	return m, cmd
}

func (m *tui) key(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.prompt != nil {
		return m.promptKey(k)
	}
	if m.ask != nil {
		a := m.ask
		m.ask = nil
		if k.String() == "y" || k.String() == "Y" {
			return m, a.run
		}
		m.setMsg("cancelled", false)
		return m, nil
	}
	if m.menu != nil {
		mn := m.menu
		m.menu = nil
		if f, ok := mn.opts[k.String()]; ok {
			return m, f()
		}
		m.setMsg("cancelled", false)
		return m, nil
	}
	if m.help {
		m.help = false
		return m, nil
	}
	switch m.view {
	case viewSession:
		return m.sessionKey(k)
	case viewChecklist:
		if !m.chat.focus {
			return m.checklistKey(k)
		}
	case viewDocs:
		if !m.chat.focus {
			return m.docsKey(k)
		}
	}
	if m.chat.focus {
		return m.chatKey(k)
	}
	return m.dashboardKey(k)
}

func (m *tui) promptKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	p := m.prompt
	switch k.String() {
	case "esc", "ctrl+c":
		m.prompt = nil
		m.setMsg("cancelled", false)
		return m, nil
	case "enter":
		m.prompt = nil
		v := strings.TrimSpace(p.in.Value())
		if v == "" {
			return m, nil
		}
		return m, p.submit(v)
	}
	var cmd tea.Cmd
	p.in, cmd = p.in.Update(k)
	return m, cmd
}

func (m *tui) ask1(label, placeholder string, submit func(string) tea.Cmd) tea.Cmd {
	in := textinput.New()
	in.Prompt = "› "
	in.Placeholder = placeholder
	in.CharLimit = 300
	in.Width = max(20, m.width-len(label)-6)
	m.prompt = &prompt{label: label, in: in, submit: submit}
	return m.prompt.in.Focus() // focus the stored copy, not the local one
}

func (m *tui) dashboardKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	r, ok := m.selected()
	switch k.String() {
	case "q", "ctrl+c":
		if len(m.sessions) > 0 {
			m.ask = &confirm{fmt.Sprintf("Quit and close %d open session(s)? y/n", len(m.sessions)), tea.Quit}
			return m, nil
		}
		return m, tea.Quit
	case "tab":
		return m, m.focusChat(true)
	case "l":
		return m, m.focusChat(false)
	case `\`:
		m.chat.show = !m.chat.show && m.chat.a != nil
		m.layout()
		return m, nil
	case "?":
		m.help = true
		return m, nil
	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		if i := int(k.Runes[0] - '1'); i < len(m.sessions) {
			m.showSession(i)
		}
		return m, nil
	case "r":
		m.busy = "refreshing"
		return m, m.loadRows()
	case "a":
		return m, m.addPrompt()
	case "d":
		m.openDocs()
		return m, nil
	case "/":
		return m, m.searchPrompt()
	case "v":
		if i := m.findSession("vpn"); i >= 0 {
			m.showSession(i)
			m.setMsg("VPN session: ctrl+c stops OpenVPN", false)
			return m, nil
		}
		a, err := vpnAction(m.w)
		return m, m.launch("", a, err)
	case "H":
		a, err := serveAction(m.w, m.lhost)
		return m, m.launch("", a, err)
	case "R":
		w := m.w
		return m, m.background("writing report", false, func() (string, error) {
			p, err := report.Write(w)
			if err != nil {
				return "", err
			}
			return "report draft: " + p + "  (pandoc it to PDF)", nil
		})
	case "x":
		if m.msg == "" {
			m.setMsg("nothing to explain yet", false)
			return m, nil
		}
		q := "Explain this message and what I should do about it: " + m.msg
		if m.msgErr && m.lastScreen != "" {
			q += "\nThe session's last screen was:\n```\n" + lastLines(m.lastScreen, 25) + "\n```"
		}
		return m, m.askClaude(q)
	}
	if !ok {
		var cmd tea.Cmd
		m.tbl, cmd = m.tbl.Update(k)
		return m, cmd
	}
	t := r.t
	switch k.String() {
	case "enter":
		a, err := shellAction(m.w, t, m.lhost)
		return m, m.launch(t.ID, a, err)
	case "e":
		return m, m.launch(t.ID, notesAction(t), nil)
	case "s":
		if t.Host == "" {
			m.setMsg(t.Label()+" has no host to scan", true)
			return m, nil
		}
		opts := map[string]func() tea.Cmd{}
		var labels []string
		for _, p := range scanPresets {
			p := p
			labels = append(labels, p.key+" "+p.name)
			opts[p.key] = func() tea.Cmd {
				a, err := scanAction(m.w, t, p, r.h.PortNumbers("tcp"))
				return m.launch(t.ID, a, err)
			}
		}
		m.menu = &menu{"scan " + t.Host + ": " + strings.Join(labels, " · ") + " · esc cancel", opts}
		return m, nil
	case "m":
		t.Status = t.Status.Next()
		if err := m.w.SaveTarget(t); err != nil {
			m.setMsg(err.Error(), true)
			return m, nil
		}
		return m, m.loadRows()
	case "f":
		return m, m.flagMenu(t)
	case "c":
		if len(m.items) == 0 {
			m.setMsg("no checklist: Oscar knowledge base not found (set OSCAR_KB)", true)
			return m, nil
		}
		m.view, m.clCursor = viewChecklist, 0
		m.tbl.Blur()
		return m, nil
	case "T":
		return m, m.trackerConfirm(t)
	}
	var cmd tea.Cmd
	m.tbl, cmd = m.tbl.Update(k)
	return m, cmd
}

// addPrompt adds a host (all modes) or a challenge (CTF).
func (m *tui) addPrompt() tea.Cmd {
	w := m.w
	label, ph := "add target", "10.10.10.3 Lame   (IP or hostname, optional name)"
	if w.Cfg.Mode == workspace.ModeCTF {
		label, ph = "add challenge", "name | category | points | host:port   (or an IP to add a host)"
	}
	return m.ask1(label, ph, func(v string) tea.Cmd {
		return m.background("adding", true, func() (string, error) {
			if w.Cfg.Mode == workspace.ModeCTF && net.ParseIP(strings.Fields(v)[0]) == nil {
				parts := strings.Split(v, "|")
				for len(parts) < 4 {
					parts = append(parts, "")
				}
				pts, _ := strconv.Atoi(strings.TrimSpace(parts[2]))
				t, err := w.AddChallenge(parts[0], parts[1], pts, parts[3])
				if err != nil {
					return "", err
				}
				return "added " + t.Label() + " (" + t.Category + ")", nil
			}
			f := strings.Fields(v)
			name := ""
			if len(f) > 1 {
				name = strings.Join(f[1:], " ")
			}
			t, err := w.AddTarget(f[0], name)
			if err != nil {
				return "", err
			}
			return "added " + t.Label() + " — s to scan", nil
		})
	})
}

func (m *tui) flagMenu(t workspace.Target) tea.Cmd {
	names := m.w.Cfg.Mode.FlagNames()
	if len(names) == 0 {
		names = []string{"proof"}
	}
	record := func(name string) func() tea.Cmd {
		return func() tea.Cmd {
			return m.ask1(name+" for "+t.Label(), "paste the flag", func(v string) tea.Cmd {
				w := m.w
				return m.background("saving flag", true, func() (string, error) {
					nt, err := w.SetFlag(t.ID, name, v)
					if err != nil {
						return "", err
					}
					_ = w.Log(workspace.Entry{Target: t.ID, Action: "flag " + name})
					s := fmt.Sprintf("%s: %s saved, status %s", t.Label(), name, nt.Status)
					if nt.Status == "pwned" && !nt.IsChallenge() {
						s += " — screenshot the proof, then T logs it to the box tracker"
					}
					return s, nil
				})
			})
		}
	}
	if len(names) == 1 {
		return record(names[0])()
	}
	opts := map[string]func() tea.Cmd{}
	var labels []string
	for i, n := range names {
		key := strconv.Itoa(i + 1)
		opts[key] = record(n)
		labels = append(labels, key+" "+n)
	}
	m.menu = &menu{"which flag? " + strings.Join(labels, " · "), opts}
	return nil
}

func (m *tui) trackerConfirm(t workspace.Target) tea.Cmd {
	if m.k == nil {
		m.setMsg("Oscar knowledge base not found (set OSCAR_KB)", true)
		return nil
	}
	if t.IsChallenge() || t.Status != "pwned" {
		m.setMsg("log a box to the tracker once it's pwned", true)
		return nil
	}
	platform := map[workspace.Mode]string{workspace.ModeHTB: "HTB", workspace.ModeLab: "PG/OffSec", workspace.ModeExam: "OSCP exam"}[m.w.Cfg.Mode]
	name := t.Name
	if name == "" {
		name = t.Host
	}
	row := kb.TrackerRow{Platform: platform, Box: name, OS: t.OS, Time: m.timeOn(t)}
	k := m.k
	m.ask = &confirm{fmt.Sprintf("Add %s to practice/box-tracker.md (you fill in technique & takeaway)? y/n", name),
		m.background("updating tracker", false, func() (string, error) {
			n, err := k.AddToTracker(row)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("box tracker row %d: %s — add the technique/takeaway; that column is the point", n, name), nil
		})}
	return nil
}

// timeOn is first logged action → last flag, rounded, e.g. "2h10m".
func (m *tui) timeOn(t workspace.Target) string {
	hist, _ := m.w.History()
	for _, e := range hist {
		if e.Target == t.ID {
			end := e.Time
			for _, f := range t.Flags {
				if f.Time.After(end) {
					end = f.Time
				}
			}
			d := end.Sub(e.Time).Round(10 * 60e9)
			return strings.TrimSuffix(d.String(), "0s")
		}
	}
	return ""
}

// ---- table

func (m *tui) flagCell(t workspace.Target) string {
	names := m.w.Cfg.Mode.FlagNames()
	if len(names) == 0 {
		return dash(fmt.Sprint(len(t.Flags)))
	}
	var parts []string
	for _, n := range names {
		mark := "·"
		if t.Flag(n) != "" {
			mark = "✓"
		}
		short := strings.TrimSuffix(n, ".txt")
		if len(names) > 1 {
			short = short[:1]
		}
		parts = append(parts, short+mark)
	}
	return strings.Join(parts, " ")
}

func (m *tui) checkCell(t workspace.Target) string {
	if len(m.items) == 0 || t.IsChallenge() {
		return "-"
	}
	n := 0
	for _, it := range m.items {
		if t.IsChecked(it.Key()) {
			n++
		}
	}
	return fmt.Sprintf("%d/%d", n, len(m.items))
}

func (m *tui) tableRows() []table.Row {
	out := make([]table.Row, len(m.rows))
	for i, r := range m.rows {
		t := r.t
		if m.compact {
			out[i] = table.Row{t.Label(), string(t.Status), m.flagCell(t)}
			continue
		}
		detail := ""
		if t.IsChallenge() {
			detail = fmt.Sprintf("%s · %d pts", t.Category, t.Points)
			if rem := t.Remote(); rem != "" {
				detail += " · " + rem
			}
		} else {
			var ps []string
			for _, p := range r.h.Ports {
				s := strconv.Itoa(p.Number)
				if p.Proto == "udp" {
					s += "u"
				}
				ps = append(ps, s)
			}
			detail = dash(strings.Join(ps, ","))
		}
		out[i] = table.Row{t.Label(), string(t.Status), detail, m.flagCell(t), m.checkCell(t)}
	}
	return out
}

func dash(s string) string {
	if s == "" || s == "0" {
		return "-"
	}
	return s
}

// ---- view

func (m *tui) header() string {
	mode := strings.ToUpper(string(m.w.Cfg.Mode))
	badge := modeBadge.Render(mode)
	if m.w.Cfg.Mode == workspace.ModeExam {
		badge = examBadge.Render("EXAM · no AI")
	}
	lh := warnStyle.Render(m.w.Cfg.LHOSTIface + " down")
	if m.lhost != "" {
		lh = okStyle.Render("LHOST " + m.lhost)
	}
	ts := make([]workspace.Target, len(m.rows))
	pwned := 0
	for i, r := range m.rows {
		ts[i] = r.t
		if r.t.Status == "pwned" {
			pwned++
		}
	}
	progress := fmt.Sprintf("%d/%d pwned", pwned, len(m.rows))
	if got, total := workspace.Score(ts); total > 0 {
		progress = fmt.Sprintf("%d/%d pts", got, total)
	}
	parts := []string{badge, dimStyle.Render(m.w.Cfg.Name), lh, progress}
	if m.k == nil {
		parts = append(parts, warnStyle.Render("no KB"))
	}
	return titleStyle.Render("oscar") + "  " + strings.Join(parts, " · ")
}

func (m *tui) View() string {
	if m.help {
		return m.header() + "\n\n" + helpText + dimStyle.Render("\n(any key to close)")
	}
	switch m.view {
	case viewSession:
		if ts := m.activeSession(); ts != nil {
			return m.sessionView(ts)
		}
	case viewChecklist:
		return m.withPanel(m.checklistView())
	case viewDocs:
		return m.withPanel(m.docsView())
	}
	var b strings.Builder
	b.WriteString(m.header() + "\n")
	if len(m.sessions) > 0 {
		b.WriteString(dimStyle.Render("sessions ") + m.tabsView() + dimStyle.Render(" 1-9 to open"))
	}
	b.WriteString("\n")
	main := m.tbl.View() + "\n" + m.detailView()
	if len(m.rows) == 0 {
		main = m.emptyView()
	}
	if m.chat.show && m.width >= sideBySide {
		main = lipgloss.JoinHorizontal(lipgloss.Top, main, " ", m.panelView())
	}
	b.WriteString(main + "\n")
	b.WriteString(m.statusLine() + "\n")
	b.WriteString(dimStyle.Render(m.keyHint()))
	return b.String()
}

// withPanel puts a full-screen view next to the Claude panel.
func (m *tui) withPanel(body string) string {
	if m.chat.show && m.width >= sideBySide {
		body = lipgloss.JoinHorizontal(lipgloss.Top, body, " ", m.panelView())
	}
	return m.header() + "\n" + body + "\n" + m.statusLine()
}

func (m *tui) statusLine() string {
	switch {
	case m.prompt != nil:
		return askStyle.Render(m.prompt.label) + " " + m.prompt.in.View()
	case m.ask != nil:
		return askStyle.Render(m.ask.prompt)
	case m.menu != nil:
		return askStyle.Render(m.menu.prompt)
	case m.busy != "":
		return m.spin.View() + " " + m.busy + "…"
	case m.msgErr:
		return errStyle.Render(m.msg)
	}
	return m.msg
}

func (m *tui) emptyView() string {
	var b strings.Builder
	b.WriteString("\n  No targets yet.\n\n")
	if len(m.w.Scope.Rules()) == 0 {
		b.WriteString("  1. List what you may test in " + m.w.Path("scope.txt") + "\n")
		b.WriteString("  2. Press r, then a to add a target\n")
	} else {
		b.WriteString("  Scope: " + strings.Join(m.w.Scope.Rules(), ", ") + "\n")
		b.WriteString("  Press a to add a target.\n")
	}
	if m.w.Cfg.Mode == workspace.ModeExam {
		b.WriteString("\n  " + examBadge.Render("EXAM") + " The assistant is off. Re-read 00-exam-guide/exam-guide.md (d) for tool rules.\n")
	}
	return b.String()
}

// detailView is the selected target's ports, flags and next checklist steps.
func (m *tui) detailView() string {
	r, ok := m.selected()
	if !ok {
		return ""
	}
	t := r.t
	var b strings.Builder
	b.WriteString(headStyle.Render(t.Label()))
	if t.OS != "" || r.h.OS != "" {
		b.WriteString(dimStyle.Render("  " + firstNonEmpty(t.OS, r.h.OS)))
	}
	b.WriteString("\n")
	if len(r.h.Ports) > 0 {
		for i, p := range r.h.Ports {
			if i == 8 {
				b.WriteString(dimStyle.Render(fmt.Sprintf("  … %d more", len(r.h.Ports)-8)) + "\n")
				break
			}
			b.WriteString("  " + p.Label() + "\n")
		}
	} else if !t.IsChallenge() {
		b.WriteString(dimStyle.Render("  no scan results yet — s to scan") + "\n")
	}
	if t.IsChallenge() {
		fmt.Fprintf(&b, "  %s · %d pts", t.Category, t.Points)
		if rem := t.Remote(); rem != "" {
			b.WriteString(" · " + rem)
		}
		b.WriteString(dimStyle.Render("  · files/ and solve/ in "+t.Dir) + "\n")
	}
	if len(m.items) > 0 && !t.IsChallenge() {
		var next []string
		for _, it := range m.items {
			if !t.IsChecked(it.Key()) {
				next = append(next, it.Section+": "+it.Text)
				if len(next) == 3 {
					break
				}
			}
		}
		if len(next) > 0 {
			b.WriteString(dimStyle.Render("next on your checklist (c):") + "\n")
			for _, n := range next {
				b.WriteString("  ☐ " + truncate(n, m.mainWidth()-4) + "\n")
			}
		}
	}
	return b.String()
}

func (m *tui) mainWidth() int {
	if m.chat.show && m.width >= sideBySide {
		return m.width - m.panelWidth() - 1
	}
	return m.width
}

func firstNonEmpty(xs ...string) string {
	for _, x := range xs {
		if x != "" {
			return x
		}
	}
	return ""
}

func truncate(s string, n int) string {
	if n < 4 || lipgloss.Width(s) <= n {
		return s
	}
	r := []rune(s)
	if len(r) > n-1 {
		r = r[:n-1]
	}
	return string(r) + "…"
}

func (m *tui) keyHint() string {
	full := "j/k move · enter shell · s scan · e notes · f flag · m status · c checklist · d docs · / search · a add · v vpn · H serve · R report · ? help · q quit"
	if lipgloss.Width(full) <= m.width {
		return full
	}
	return "enter shell · s scan · e notes · f flag · c checklist · d docs · ? help"
}

const helpText = `Dashboard (vim keys)
  j/k g/G     move                    a     add a target (CTF: name | category | points | host:port)
  enter       your shell in the target's folder, with RHOST, LHOST, TARGET_DIR set
  s           nmap: q quick · f all TCP · d scripts+versions on open ports · u top UDP (sudo)
  e           edit the target's notes.md in $EDITOR
  f           record a flag (format-checked; advances the status)
  m           cycle status: new → enum → foothold → privesc → pwned → skip
  c           the Oscar methodology checklist for this target (space ticks)
  d  /        browse / search the Oscar knowledge base
  T           log a pwned box to practice/box-tracker.md
  v           OpenVPN session (vpn_config in oscar.yaml)     H  serve www/ on LHOST:8000
  R           write report/report.md (draft: notes, flags, ports, timeline)
  1-9         jump to session N       r refresh     q quit
  l / tab     assistant panel (Claude/OpenCode/any model; never in exam mode)   x explain the last error

In a session every key goes to it. ctrl+] then: d back · n/p or 1-9 switch · l Claude · x explain screen

Every target must be in scope.txt; every command is logged to logs/commands.jsonl and
recorded to logs/sessions/ — your timeline for the report, and proof of what you did.
`
