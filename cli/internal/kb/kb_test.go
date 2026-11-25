package kb

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeKB builds a minimal Oscar layout.
func fakeKB(t *testing.T) *KB {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"README.md":                     "# OSCP Study\n",
		"01-methodology/checklist.md":   "# Per-Target Checklist\n\n## 🔍 Recon\n- [ ] Full TCP port scan\n- [x] UDP scan\n\n## 🐚 Foothold\n- [ ] Got a shell\n",
		"05-privesc/linux-privesc.md":   "# Linux Privilege Escalation\n\n## Sudo\nCheck sudo -l first.\n\n## SUID\nFind SUID binaries.\n",
		"practice/box-note-template.md": "# Box Notes — <BOX NAME> (<IP>)\n\n- **Platform:** HTB / PG / TryHackMe\n- **OS:** \n- **Date:** \n- **IPs / hostnames:** \n",
		"practice/box-tracker.md":       tracker,
		"cli/README.md":                 "# not part of the KB\n",
	}
	for rel, body := range files {
		p := filepath.Join(root, filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return &KB{Root: root}
}

const tracker = `# Box Tracker

## Standalone boxes

| # | Platform | Box | OS | Diff | Foothold | Privesc | Time | Root'd unaided? | Technique / Takeaway |
|---|----------|-----|----|----|----------|---------|------|-----------------|----------------------|
| 1 | HTB | Lame | Linux | Easy | smb | none | 1h | yes | old samba |
| 2 |  |  |  |  |  |  |  |  |  |

## Active Directory chains

| # | Platform | Set | Foothold | Lateral path | Domain compromise | Time | Notes |
|---|----------|-----|----------|--------------|-------------------|------|-------|
| 1 |  |  |  |  |  |  |  |
`

func TestFindViaEnv(t *testing.T) {
	k := fakeKB(t)
	t.Setenv("OSCAR_KB", k.Root)
	got, err := Find()
	if err != nil || got.Root != k.Root {
		t.Fatalf("%v %v", got, err)
	}
}

func TestChecklist(t *testing.T) {
	items, err := fakeKB(t).Checklist()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 || items[0].Section != "Recon" || items[2].Key() != "Foothold / Got a shell" {
		t.Fatalf("%+v", items)
	}
}

func TestParseChecklistRealFile(t *testing.T) {
	// The repo's own checklist must parse into a useful number of items.
	f, err := os.Open(filepath.Join("..", "..", "..", "01-methodology", "checklist.md"))
	if err != nil {
		t.Skip("not inside the Oscar repo")
	}
	defer f.Close()
	items, err := ParseChecklist(bufio.NewScanner(f))
	if err != nil || len(items) < 30 {
		t.Fatalf("%d items, %v", len(items), err)
	}
	for _, it := range items {
		if it.Section == "" || strings.HasPrefix(it.Section, "🔍") {
			t.Fatalf("bad section for %q: %q", it.Text, it.Section)
		}
	}
}

func TestDocsAndSearch(t *testing.T) {
	k := fakeKB(t)
	docs, err := k.Docs()
	if err != nil {
		t.Fatal(err)
	}
	if docs[0].Rel != "README.md" {
		t.Fatalf("README not first: %v", docs)
	}
	for _, d := range docs {
		if strings.HasPrefix(d.Rel, "cli/") {
			t.Fatal("cli/ docs leaked into the KB")
		}
		if d.Rel == "05-privesc/linux-privesc.md" && d.Title != "Linux Privilege Escalation" {
			t.Fatalf("title = %q", d.Title)
		}
	}
	hits, _ := k.Search("SUID binaries", 10)
	if len(hits) != 1 || hits[0].Heading != "SUID" || hits[0].Line != 7 {
		t.Fatalf("hits = %+v", hits)
	}
	if hits, _ := k.Search("  ", 10); hits != nil {
		t.Fatal("empty query matched")
	}
	if _, err := k.Read("../secret"); err == nil {
		t.Fatal("path traversal allowed")
	}
}

func TestBoxNotes(t *testing.T) {
	s, ok := fakeKB(t).BoxNotes("Lame", "10.10.10.3", "HTB", "2026-11-25")
	if !ok {
		t.Fatal("template not found")
	}
	for _, want := range []string{"# Box Notes — Lame (10.10.10.3)", "- **Platform:** HTB\n", "- **Date:** 2026-11-25", "- **IPs / hostnames:** 10.10.10.3", "- **OS:** \n"} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in:\n%s", want, s)
		}
	}
}

func TestAddRow(t *testing.T) {
	out, n, err := AddRow(tracker, TrackerRow{Platform: "HTB", Box: "Bashed", OS: "Linux", Takeaway: "a | b"})
	if err != nil || n != 2 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	if !strings.Contains(out, `| 2 | HTB | Bashed | Linux |  |  |  |  |  | a \| b |`) {
		t.Fatalf("row 2 not filled:\n%s", out)
	}
	// Table full: a new row 3 is appended to the standalone table, not the AD one.
	out, n, _ = AddRow(out, TrackerRow{Box: "Nibbles"})
	if n != 3 || strings.Index(out, "| 3 |") > strings.Index(out, "## Active Directory") {
		t.Fatalf("row 3 misplaced (n=%d):\n%s", n, out)
	}
	if _, _, err := AddRow("# nothing\n", TrackerRow{}); err == nil {
		t.Fatal("missing table not reported")
	}
}
