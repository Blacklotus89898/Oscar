package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newWS(t *testing.T, mode Mode) *Workspace {
	t.Helper()
	w, err := Create(t.TempDir(), "oscp", mode, []string{"10.10.10.0/24", "!10.10.10.1"})
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func TestCreateLayout(t *testing.T) {
	w := newWS(t, ModeLab)
	for _, p := range []string{"oscar.yaml", "scope.txt", "creds.md", ".gitignore", "targets", "www", "logs/sessions", "report"} {
		if _, err := os.Stat(w.Path(p)); err != nil {
			t.Errorf("missing %s: %v", p, err)
		}
	}
	if _, err := Create(w.Root, "again", ModeLab, nil); err == nil {
		t.Error("Create over an existing engagement succeeded")
	}
	if _, err := Create(t.TempDir(), "x", "yolo", nil); err == nil {
		t.Error("bad mode accepted")
	}
	if _, err := Create(t.TempDir(), "x", ModeLab, []string{"10.0.0.0/99"}); err == nil {
		t.Error("bad scope accepted")
	}
}

func TestAddTargetEnforcesScope(t *testing.T) {
	w := newWS(t, ModeLab)
	if _, err := w.AddTarget("10.10.11.5", ""); err == nil || !strings.Contains(err.Error(), "OUT OF SCOPE") {
		t.Fatalf("out-of-scope target added: %v", err)
	}
	if _, err := w.AddTarget("10.10.10.1", ""); err == nil {
		t.Fatal("excluded target added")
	}
	tg, err := w.AddTarget(" 10.10.10.5 ", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range []string{"scans", "loot", "exploits", "screenshots", "notes.md"} {
		if _, err := os.Stat(filepath.Join(tg.Dir, d)); err != nil {
			t.Errorf("missing %s", d)
		}
	}
	// Adding again returns the existing target, not an error.
	if _, err := w.AddTarget("10.10.10.5", ""); err != nil {
		t.Fatal(err)
	}
	w.AddTarget("10.10.10.40", "")
	w.AddTarget("10.10.10.9", "")
	ts, _ := w.Targets()
	var hosts []string
	for _, x := range ts {
		hosts = append(hosts, x.Host)
	}
	if strings.Join(hosts, " ") != "10.10.10.5 10.10.10.9 10.10.10.40" {
		t.Fatalf("not sorted numerically: %v", hosts)
	}
}

const (
	h1 = "0123456789abcdef0123456789abcdef"
	h2 = "fedcba9876543210fedcba9876543210"
	h3 = "00000000000000000000000000000003"
)

func TestFlagsAdvanceStatus(t *testing.T) {
	w := newWS(t, ModeExam)
	w.AddTarget("10.10.10.5", "")
	tg, err := w.SetFlag("10.10.10.5", "local.txt", " 0123456789abcdef0123456789abcdef \n")
	if err != nil || tg.Status != "foothold" || tg.Flag("local.txt") != h1 {
		t.Fatalf("after local: %+v %v", tg, err)
	}
	tg, _ = w.SetFlag("10.10.10.5", "proof.txt", h2)
	if tg.Status != "pwned" {
		t.Fatalf("after proof: %s", tg.Status)
	}
	tg, _ = w.SetFlag("10.10.10.5", "proof.txt", h3)
	if len(tg.Flags) != 2 || tg.Flag("proof.txt") != h3 {
		t.Fatalf("flag not replaced: %+v", tg.Flags)
	}
	if _, err := w.SetFlag("10.10.10.5", "local.txt", "  "); err == nil {
		t.Error("empty flag accepted")
	}
	// Round-trips from disk.
	again, _ := w.Target("10.10.10.5")
	if again.Flag("proof.txt") != h3 {
		t.Fatal("flag not persisted")
	}
}

func TestAssistantRules(t *testing.T) {
	yes := true
	cases := []struct {
		mode    Mode
		enabled *bool
		want    bool
	}{
		{ModeLab, nil, true},
		{ModeExam, nil, false},
		{ModeExam, &yes, false}, // can't be switched on in the exam
		{ModePentest, nil, false},
		{ModePentest, &yes, true},
	}
	for _, c := range cases {
		var cfg Config
		cfg.Mode, cfg.Assistant.Enabled = c.mode, c.enabled
		if got := cfg.AssistantAllowed(); got != c.want {
			t.Errorf("%s enabled=%v: got %v", c.mode, c.enabled, got)
		}
	}
}

func TestAuditLog(t *testing.T) {
	w := newWS(t, ModePentest)
	if err := w.Log(Entry{Target: "10.10.10.5", Action: "scan quick", Command: "nmap -sC -sV 10.10.10.5"}); err != nil {
		t.Fatal(err)
	}
	f, _ := os.OpenFile(w.Path("logs", "commands.jsonl"), os.O_APPEND|os.O_WRONLY, 0)
	f.WriteString("garbage\n")
	f.Close()
	w.Log(Entry{Action: "listener", Command: "nc -lvnp 4444"})
	h, err := w.History()
	if err != nil || len(h) != 2 || h[0].Mode != ModePentest || h[1].Action != "listener" {
		t.Fatalf("history = %+v %v", h, err)
	}
}

func TestFindViaEnvAndParents(t *testing.T) {
	w := newWS(t, ModeLab)
	t.Setenv("OSCAR_DIR", w.Root)
	got, err := Find()
	if err != nil || got.Cfg.Name != "oscp" {
		t.Fatalf("Find via env: %v", err)
	}
	t.Setenv("OSCAR_DIR", "")
	sub := w.Path("targets")
	old, _ := os.Getwd()
	defer os.Chdir(old)
	os.Chdir(sub)
	got, err = Find()
	if err != nil || got.Cfg.Name != "oscp" {
		t.Fatalf("Find via parent: %v", err)
	}
}

func TestStatusCycle(t *testing.T) {
	s := Status("pwned")
	if s.Next() != "skip" || Status("skip").Next() != "new" || Status("bogus").Next() != "new" {
		t.Fatal("cycle broken")
	}
}

func TestFlagFormat(t *testing.T) {
	w := newWS(t, ModeExam)
	w.AddTarget("10.10.10.5", "")
	if _, err := w.SetFlag("10.10.10.5", "local.txt", "HTB{nope}"); err == nil {
		t.Error("non-hex OffSec flag accepted")
	}
	w.Cfg.FlagFormat = "-"
	if _, err := w.SetFlag("10.10.10.5", "local.txt", "anything"); err != nil {
		t.Errorf("check not disabled: %v", err)
	}
	w.Cfg.FlagFormat = "(" // bad regexp is reported, not ignored
	if _, err := w.SetFlag("10.10.10.5", "local.txt", h1); err == nil {
		t.Error("bad flag_format ignored")
	}
}

func TestHTBMachine(t *testing.T) {
	w := newWS(t, ModeHTB)
	tg, err := w.AddTarget("10.10.10.3", "Lame")
	if err != nil || tg.Label() != "Lame (10.10.10.3)" {
		t.Fatalf("%+v %v", tg, err)
	}
	tg, _ = w.SetFlag(tg.ID, "user.txt", h1)
	if tg.Status != "foothold" {
		t.Fatalf("after user.txt: %s", tg.Status)
	}
	tg, _ = w.SetFlag(tg.ID, "root.txt", h2)
	if tg.Status != "pwned" {
		t.Fatalf("after root.txt: %s", tg.Status)
	}
	tg.Hostnames = []string{"lame.hackthebox.gr"}
	w.SaveTarget(tg)
	ts, _ := w.Targets()
	got := HostsLines(ts, ".htb")
	if len(got) != 1 || got[0] != "10.10.10.3\tlame.hackthebox.gr lame.htb" {
		t.Fatalf("hosts = %q", got)
	}
	if got := HostsLines(ts, ""); got[0] != "10.10.10.3\tlame.hackthebox.gr" {
		t.Fatalf("hosts without suffix = %q", got)
	}
}

func TestCTFChallenges(t *testing.T) {
	w, err := Create(t.TempDir(), "ctf", ModeCTF, []string{"chall.ctf.example.com", "10.0.5.0/24"})
	if err != nil {
		t.Fatal(err)
	}
	web, err := w.AddChallenge("Baby SQL!", "Web", 100, "chall.ctf.example.com:31337")
	if err != nil {
		t.Fatal(err)
	}
	if web.ID != "baby-sql" || web.Category != "web" || web.Remote() != "chall.ctf.example.com:31337" {
		t.Fatalf("%+v", web)
	}
	for _, d := range []string{"files", "solve", "notes.md"} {
		if _, err := os.Stat(filepath.Join(web.Dir, d)); err != nil {
			t.Errorf("missing %s", d)
		}
	}
	if _, err := w.AddChallenge("rsa-1", "crypto", 200, ""); err != nil {
		t.Fatal(err) // offline challenge: no remote, no scope check
	}
	if _, err := w.AddChallenge("evil", "pwn", 50, "8.8.8.8:1337"); err == nil {
		t.Error("out-of-scope remote accepted")
	}
	if _, err := w.AddChallenge("Baby SQL", "web", 1, ""); err == nil {
		t.Error("duplicate challenge accepted")
	}
	if _, err := w.AddChallenge("x", "web", 1, "chall.ctf.example.com:99999"); err == nil {
		t.Error("bad port accepted")
	}
	if _, err := w.SetFlag("baby-sql", "flag", "not a flag"); err == nil {
		t.Error("malformed CTF flag accepted")
	}
	tg, err := w.SetFlag("baby-sql", "flag", "flag{union_select_ftw}")
	if err != nil || tg.Status != "pwned" {
		t.Fatalf("%+v %v", tg, err)
	}
	w.AddTarget("10.0.5.9", "")
	ts, _ := w.Targets()
	if len(ts) != 3 || ts[0].Host != "10.0.5.9" || ts[1].ID != "rsa-1" || ts[2].ID != "baby-sql" {
		t.Fatalf("order: hosts first, then challenges by category: %v", ids(ts))
	}
	if got, total := Score(ts); got != 100 || total != 300 {
		t.Fatalf("score %d/%d", got, total)
	}
}

func ids(ts []Target) []string {
	var out []string
	for _, t := range ts {
		out = append(out, t.ID)
	}
	return out
}

func TestSlug(t *testing.T) {
	for in, want := range map[string]string{"Baby SQL!": "baby-sql", "  rsa--1 ": "rsa-1", "!!!": "", "Ünïcode pwn": "n-code-pwn"} {
		if got := Slug(in); got != want {
			t.Errorf("Slug(%q) = %q, want %q", in, got, want)
		}
	}
}
func TestToggle(t *testing.T) {
	w := newWS(t, ModeLab)
	w.AddTarget("10.10.10.5", "")
	if on, err := w.Toggle("10.10.10.5", "Recon / Full TCP port scan"); !on || err != nil {
		t.Fatal(on, err)
	}
	tg, _ := w.Target("10.10.10.5")
	if !tg.IsChecked("Recon / Full TCP port scan") {
		t.Fatal("not persisted")
	}
	if on, _ := w.Toggle("10.10.10.5", "Recon / Full TCP port scan"); on {
		t.Fatal("did not toggle off")
	}
}
