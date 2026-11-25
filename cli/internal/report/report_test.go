package report

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Blacklotus89898/Oscar/cli/internal/workspace"
)

const scan = `<nmaprun><host><status state="up"/><address addr="10.10.10.3" addrtype="ipv4"/>
<ports><port protocol="tcp" portid="445"><state state="open"/><service name="netbios-ssn" product="Samba smbd" version="3.0.20"/></port></ports></host></nmaprun>`

func TestBuild(t *testing.T) {
	w, err := workspace.Create(t.TempDir(), "htb-week1", workspace.ModeHTB, []string{"10.10.10.0/24"})
	if err != nil {
		t.Fatal(err)
	}
	tg, _ := w.AddTarget("10.10.10.3", "Lame")
	os.WriteFile(filepath.Join(tg.Dir, "scans", "full.xml"), []byte(scan), 0o600)
	os.WriteFile(filepath.Join(tg.Dir, "screenshots", "root.png"), []byte("png"), 0o600)
	os.WriteFile(filepath.Join(tg.Dir, "notes.md"), []byte("# Lame\n## Foothold\nold samba\n```\n# not a heading\n```\n"), 0o600)
	w.SetFlag(tg.ID, "root.txt", "0123456789abcdef0123456789abcdef")
	w.Log(workspace.Entry{Target: tg.ID, Action: "scan full", Command: "nmap -p- 10.10.10.3"})
	w.AddTarget("10.10.10.4", "")

	got, err := Build(w, time.Date(2026, 11, 25, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`title: "htb-week1 — Hack The Box report"`,
		"1 of 2 targets fully compromised",
		"| Lame (10.10.10.3) | pwned | root.txt |",
		"| 445/tcp | netbios-ssn | Samba smbd 3.0.20 |",
		"## Lame\n### Foothold", // notes demoted under the target heading
		"# not a heading",       // but not inside code fences
		"![root.png](../targets/10.10.10.3/screenshots/root.png)",
		"| scan full | `nmap -p- 10.10.10.3` |",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("report missing %q", want)
		}
	}
	if strings.Contains(got, "\n## # not a heading") {
		t.Error("heading inside a code fence was demoted")
	}
}

func TestCTFScore(t *testing.T) {
	w, _ := workspace.Create(t.TempDir(), "ctf", workspace.ModeCTF, nil)
	w.AddChallenge("baby rsa", "crypto", 100, "")
	w.AddChallenge("heap 1", "pwn", 400, "")
	w.SetFlag("baby-rsa", "flag", "flag{small_e}")
	p, err := Write(w)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(p)
	if !strings.Contains(string(b), "100 / 500 points") || !strings.Contains(string(b), "CTF write-up") {
		t.Fatalf("report:\n%s", b)
	}
}
