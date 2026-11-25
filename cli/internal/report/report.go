// Package report turns an engagement into report/report.md: a summary,
// one section per target (flags, open ports, your notes, proof screenshots)
// and a timeline from the command log. It is a draft for you to edit — the
// narrative still has to be yours — and converts cleanly with pandoc.
package report

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Blacklotus89898/Oscar/cli/internal/nmap"
	"github.com/Blacklotus89898/Oscar/cli/internal/workspace"
)

// Build renders the report text.
func Build(w *workspace.Workspace, now time.Time) (string, error) {
	targets, err := w.Targets()
	if err != nil {
		return "", err
	}
	hist, err := w.History()
	if err != nil {
		return "", err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "---\ntitle: \"%s — %s report\"\ndate: %s\n---\n\n", w.Cfg.Name, modeTitle(w.Cfg.Mode), now.Format("2006-01-02"))

	b.WriteString("# Summary\n\n")
	fmt.Fprintf(&b, "- **Engagement:** %s (%s mode)\n", w.Cfg.Name, w.Cfg.Mode)
	fmt.Fprintf(&b, "- **Scope:** %s\n", strings.Join(w.Scope.Rules(), ", "))
	if len(hist) > 0 {
		fmt.Fprintf(&b, "- **Testing window:** %s – %s (UTC)\n", hist[0].Time.Format("2006-01-02 15:04"), hist[len(hist)-1].Time.Format("2006-01-02 15:04"))
	}
	pwned := 0
	for _, t := range targets {
		if t.Status == "pwned" {
			pwned++
		}
	}
	fmt.Fprintf(&b, "- **Result:** %d of %d targets fully compromised", pwned, len(targets))
	if got, total := workspace.Score(targets); total > 0 {
		fmt.Fprintf(&b, "; %d / %d points", got, total)
	}
	b.WriteString("\n\n")

	if len(targets) > 0 {
		b.WriteString("| Target | Status | Flags |\n|---|---|---|\n")
		for _, t := range targets {
			var fl []string
			for _, f := range t.Flags {
				fl = append(fl, f.Name)
			}
			fmt.Fprintf(&b, "| %s | %s | %s |\n", cell(t.Label()), t.Status, cell(strings.Join(fl, ", ")))
		}
		b.WriteString("\n")
	}

	for _, t := range targets {
		fmt.Fprintf(&b, "# %s\n\n", t.Label())
		if t.IsChallenge() {
			fmt.Fprintf(&b, "- **Category:** %s · **Points:** %d\n", t.Category, t.Points)
			if r := t.Remote(); r != "" {
				fmt.Fprintf(&b, "- **Remote:** `%s`\n", r)
			}
		} else {
			fmt.Fprintf(&b, "- **Address:** %s\n", t.Host)
			if t.OS != "" {
				fmt.Fprintf(&b, "- **OS:** %s\n", t.OS)
			}
		}
		fmt.Fprintf(&b, "- **Status:** %s\n", t.Status)
		for _, f := range t.Flags {
			fmt.Fprintf(&b, "- **%s:** `%s` (captured %s UTC)\n", f.Name, f.Value, f.Time.Format("2006-01-02 15:04"))
		}
		b.WriteString("\n")

		if !t.IsChallenge() {
			if h, _ := nmap.LoadDir(filepath.Join(t.Dir, "scans"), t.Host); len(h.Ports) > 0 {
				b.WriteString("## Open ports\n\n| Port | Service | Version |\n|---|---|---|\n")
				for _, p := range h.Ports {
					fmt.Fprintf(&b, "| %d/%s | %s | %s |\n", p.Number, p.Proto, cell(p.Service), cell(p.Product))
				}
				b.WriteString("\n")
			}
		}

		if notes, err := os.ReadFile(filepath.Join(t.Dir, "notes.md")); err == nil {
			b.WriteString("## Notes\n\n")
			b.WriteString(demote(string(notes)))
			b.WriteString("\n\n")
		}

		if shots := screenshots(t.Dir); len(shots) > 0 {
			b.WriteString("## Proof\n\n")
			for _, s := range shots {
				rel, _ := filepath.Rel(w.Path("report"), s)
				fmt.Fprintf(&b, "![%s](%s)\n\n", filepath.Base(s), filepath.ToSlash(rel))
			}
		}

		var mine []workspace.Entry
		for _, e := range hist {
			if e.Target == t.ID || e.Target == t.Host && t.Host != "" {
				mine = append(mine, e)
			}
		}
		if len(mine) > 0 {
			b.WriteString("## Timeline\n\n| Time (UTC) | Action | Command |\n|---|---|---|\n")
			for _, e := range mine {
				fmt.Fprintf(&b, "| %s | %s | `%s` |\n", e.Time.Format("01-02 15:04"), cell(e.Action), cell(e.Command))
			}
			b.WriteString("\n")
		}
	}
	return b.String(), nil
}

// Write builds the report into report/report.md and returns its path.
func Write(w *workspace.Workspace) (string, error) {
	s, err := Build(w, time.Now())
	if err != nil {
		return "", err
	}
	p := w.Path("report", "report.md")
	return p, os.WriteFile(p, []byte(s), 0o600)
}

func modeTitle(m workspace.Mode) string {
	switch m {
	case workspace.ModeExam:
		return "OSCP exam"
	case workspace.ModeHTB:
		return "Hack The Box"
	case workspace.ModeCTF:
		return "CTF write-up"
	case workspace.ModePentest:
		return "penetration test"
	}
	return "lab"
}

// demote pushes the notes' headings one level down, under the target heading.
func demote(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	fence := false
	for i, l := range lines {
		if strings.HasPrefix(l, "```") {
			fence = !fence
		}
		if !fence && strings.HasPrefix(l, "#") {
			lines[i] = "#" + l
		}
	}
	return strings.Join(lines, "\n")
}

func screenshots(dir string) []string {
	var out []string
	for _, pat := range []string{"*.png", "*.jpg", "*.jpeg"} {
		m, _ := filepath.Glob(filepath.Join(dir, "screenshots", pat))
		out = append(out, m...)
	}
	sort.Strings(out)
	return out
}

func cell(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "|", `\|`), "\n", " ")
}
