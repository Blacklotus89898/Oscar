// Package kb reads the Oscar knowledge base (the markdown in this repo) so the
// TUI can put your own methodology next to the target you're working on:
// the per-target checklist, the docs and a search across them, the box note
// template, and the box tracker.
package kb

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// KB is an Oscar checkout.
type KB struct{ Root string }

const marker = "01-methodology/checklist.md"

// Find locates the knowledge base: $OSCAR_KB, then the directory holding the
// running binary or the current directory (or a parent), then ~/dev/Oscar
// and ~/Oscar.
func Find() (*KB, error) {
	var cands []string
	if d := os.Getenv("OSCAR_KB"); d != "" {
		cands = append(cands, d)
	}
	if exe, err := os.Executable(); err == nil {
		cands = append(cands, parents(filepath.Dir(exe))...)
	}
	if wd, err := os.Getwd(); err == nil {
		cands = append(cands, parents(wd)...)
	}
	if home, err := os.UserHomeDir(); err == nil {
		cands = append(cands, filepath.Join(home, "dev", "Oscar"), filepath.Join(home, "Oscar"))
	}
	for _, d := range cands {
		if _, err := os.Stat(filepath.Join(d, filepath.FromSlash(marker))); err == nil {
			return &KB{Root: d}, nil
		}
	}
	return nil, errors.New("Oscar knowledge base not found: set OSCAR_KB to your Oscar checkout")
}

func parents(d string) []string {
	var out []string
	for {
		out = append(out, d)
		p := filepath.Dir(d)
		if p == d {
			return out
		}
		d = p
	}
}

func (k *KB) path(rel string) string { return filepath.Join(k.Root, filepath.FromSlash(rel)) }

// ---- checklist

// Item is one "- [ ]" line of the checklist.
type Item struct {
	Section string // "Recon", "Web (per web port)"
	Text    string
}

// Key identifies an item in target.yaml; it survives reordering the file.
func (i Item) Key() string { return i.Section + " / " + i.Text }

var (
	headingRe = regexp.MustCompile(`^#{2,3}\s+(.*)$`)
	itemRe    = regexp.MustCompile(`^\s*[-*]\s+\[[ xX]\]\s+(.*)$`)
	emojiRe   = regexp.MustCompile(`^[^\p{L}\p{N}]+`)
)

// Checklist parses 01-methodology/checklist.md.
func (k *KB) Checklist() ([]Item, error) {
	f, err := os.Open(k.path(marker))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ParseChecklist(bufio.NewScanner(f))
}

// ParseChecklist reads "## Section" headings and "- [ ] item" lines.
func ParseChecklist(sc *bufio.Scanner) ([]Item, error) {
	var out []Item
	section := ""
	for sc.Scan() {
		line := sc.Text()
		if m := headingRe.FindStringSubmatch(line); m != nil {
			section = strings.TrimSpace(emojiRe.ReplaceAllString(m[1], ""))
			continue
		}
		if m := itemRe.FindStringSubmatch(line); m != nil {
			out = append(out, Item{Section: section, Text: strings.TrimSpace(m[1])})
		}
	}
	return out, sc.Err()
}

// ---- docs

// Doc is one markdown file of the knowledge base.
type Doc struct {
	Rel   string // "05-privesc/linux-privesc.md"
	Title string // first "# " heading
}

// Docs lists the knowledge base's markdown, in repo order.
func (k *KB) Docs() ([]Doc, error) {
	var out []Doc
	err := filepath.WalkDir(k.Root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if n := d.Name(); p != k.Root && (strings.HasPrefix(n, ".") || n == "cli" || n == "node_modules") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.EqualFold(filepath.Ext(p), ".md") {
			return nil
		}
		rel, _ := filepath.Rel(k.Root, p)
		rel = filepath.ToSlash(rel)
		out = append(out, Doc{Rel: rel, Title: title(p, rel)})
		return nil
	})
	sort.Slice(out, func(i, j int) bool {
		// README first, then numbered sections, then the rest.
		if (out[i].Rel == "README.md") != (out[j].Rel == "README.md") {
			return out[i].Rel == "README.md"
		}
		return out[i].Rel < out[j].Rel
	})
	return out, err
}

func title(p, fallback string) string {
	f, err := os.Open(p)
	if err != nil {
		return fallback
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for i := 0; sc.Scan() && i < 20; i++ {
		if t, ok := strings.CutPrefix(sc.Text(), "# "); ok {
			return strings.TrimSpace(t)
		}
	}
	return fallback
}

// Read returns a doc's text.
func (k *KB) Read(rel string) (string, error) {
	if strings.Contains(rel, "..") {
		return "", fmt.Errorf("bad path %q", rel)
	}
	b, err := os.ReadFile(k.path(rel))
	return string(b), err
}

// Hit is a search match.
type Hit struct {
	Doc  Doc
	Line int // 1-based
	Text string
	// Heading is the nearest heading above the match, for context.
	Heading string
}

// Search finds lines containing every word of query (case-insensitive),
// up to limit hits.
func (k *KB) Search(query string, limit int) ([]Hit, error) {
	words := strings.Fields(strings.ToLower(query))
	if len(words) == 0 {
		return nil, nil
	}
	docs, err := k.Docs()
	if err != nil {
		return nil, err
	}
	var hits []Hit
	for _, d := range docs {
		text, err := k.Read(d.Rel)
		if err != nil {
			continue
		}
		heading := ""
		for i, line := range strings.Split(text, "\n") {
			if strings.HasPrefix(line, "#") {
				heading = strings.TrimSpace(strings.TrimLeft(line, "# "))
			}
			low := strings.ToLower(line)
			all := true
			for _, w := range words {
				if !strings.Contains(low, w) {
					all = false
					break
				}
			}
			if all {
				hits = append(hits, Hit{Doc: d, Line: i + 1, Text: strings.TrimSpace(line), Heading: heading})
				if len(hits) >= limit {
					return hits, nil
				}
			}
		}
	}
	return hits, nil
}

// ---- box notes and tracker

// BoxNotes fills practice/box-note-template.md for a new box; ok is false
// when the template is missing.
func (k *KB) BoxNotes(name, ip, platform, date string) (string, bool) {
	b, err := os.ReadFile(k.path("practice/box-note-template.md"))
	if err != nil {
		return "", false
	}
	if name == "" {
		name = ip
	}
	s := strings.NewReplacer("<BOX NAME>", name, "<IP>", ip).Replace(string(b))
	s = fillField(s, "Platform", platform)
	s = fillField(s, "Date", date)
	s = fillField(s, "IPs / hostnames", ip)
	return s, true
}

// fillField sets "- **Field:** value" when the template leaves it blank or
// lists choices ("HTB / PG / TryHackMe").
func fillField(s, field, value string) string {
	if value == "" {
		return s
	}
	re := regexp.MustCompile(`(?m)^(- \*\*` + regexp.QuoteMeta(field) + `:\*\*).*$`)
	return re.ReplaceAllString(s, "${1} "+strings.ReplaceAll(value, "$", "$$"))
}

// TrackerRow is one box for practice/box-tracker.md.
type TrackerRow struct {
	Platform, Box, OS, Diff, Foothold, Privesc, Time, Unaided, Takeaway string
}

// AddToTracker writes row into the first empty line of the "Standalone boxes"
// table (adding a line if it's full) and returns the row number used.
func (k *KB) AddToTracker(row TrackerRow) (int, error) {
	p := k.path("practice/box-tracker.md")
	b, err := os.ReadFile(p)
	if err != nil {
		return 0, err
	}
	out, n, err := AddRow(string(b), row)
	if err != nil {
		return 0, err
	}
	return n, os.WriteFile(p, []byte(out), 0o644)
}

var rowRe = regexp.MustCompile(`^\|\s*(\d+)\s*\|(.*)\|\s*$`)

// AddRow is AddToTracker on text (pure, for tests).
func AddRow(text string, r TrackerRow) (string, int, error) {
	lines := strings.Split(text, "\n")
	in, lastRow, lastNum := false, -1, 0
	for i, line := range lines {
		if strings.HasPrefix(line, "## ") {
			if in {
				break
			}
			in = strings.Contains(strings.ToLower(line), "standalone")
			continue
		}
		if !in {
			continue
		}
		m := rowRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		fmt.Sscan(m[1], &lastNum)
		lastRow = i
		cells := strings.Split(m[2], "|")
		if len(cells) > 1 && strings.TrimSpace(cells[1]) == "" { // Box column empty
			lines[i] = formatRow(lastNum, r)
			return strings.Join(lines, "\n"), lastNum, nil
		}
	}
	if lastRow < 0 {
		return text, 0, errors.New("box-tracker.md has no \"Standalone boxes\" table")
	}
	lastNum++
	lines = append(lines[:lastRow+1], append([]string{formatRow(lastNum, r)}, lines[lastRow+1:]...)...)
	return strings.Join(lines, "\n"), lastNum, nil
}

func formatRow(n int, r TrackerRow) string {
	cells := []string{fmt.Sprint(n), r.Platform, r.Box, r.OS, r.Diff, r.Foothold, r.Privesc, r.Time, r.Unaided, r.Takeaway}
	for i, c := range cells {
		cells[i] = strings.ReplaceAll(strings.TrimSpace(c), "|", `\|`)
	}
	return "| " + strings.Join(cells, " | ") + " |"
}
