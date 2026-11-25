// Package workspace is the on-disk engagement: one directory per engagement,
// plain files a tester can grep, diff, back up and hand to a report writer.
//
//	<engagement>/
//	  oscar.yaml        name, mode (lab|exam|pentest), VPN, LHOST interface
//	  scope.txt            rules of engagement (see package scope)
//	  creds.md             credentials found (shared: reuse is half of AD)
//	  targets/<host>/
//	    target.yaml        status, flags, os, hostnames
//	    notes.md           your methodology notes for this box
//	    scans/             nmap -oA output (parsed for the dashboard)
//	    loot/ exploits/ screenshots/
//	  www/                 files you serve to targets (linpeas, winPEAS, nc.exe)
//	  logs/commands.jsonl  every command oscar ran: time, target, argv
//	  logs/sessions/       full transcripts of every session (script(1))
//	  report/              generated report.md
package workspace

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/Blacklotus89898/Oscar/cli/internal/assistant"
	"github.com/Blacklotus89898/Oscar/cli/internal/scope"
)

// Mode changes what oscar allows.
type Mode string

const (
	ModeLab     Mode = "lab"     // practice (OffSec labs, HTB, PG): everything on
	ModeExam    Mode = "exam"    // OSCP exam: no AI, restricted tools flagged, one Metasploit target
	ModePentest Mode = "pentest" // client work: strict scope, full logging, AI off unless enabled
	ModeHTB     Mode = "htb"     // Hack The Box / TryHackMe machines: user.txt + root.txt
	ModeCTF     Mode = "ctf"     // jeopardy CTF: challenges with categories, points, flag{...}
)

// Modes lists the valid modes.
var Modes = []Mode{ModeLab, ModeExam, ModePentest, ModeHTB, ModeCTF}

// FlagNames are the flags a mode expects per target, in capture order.
func (m Mode) FlagNames() []string {
	switch m {
	case ModeLab, ModeExam:
		return []string{"local.txt", "proof.txt"}
	case ModeHTB:
		return []string{"user.txt", "root.txt"}
	case ModeCTF:
		return []string{"flag"}
	}
	return nil
}

// defaultFlagFormat: OffSec and HTB flags are 32 lowercase hex characters.
func (m Mode) defaultFlagFormat() string {
	switch m {
	case ModeLab, ModeExam, ModeHTB:
		return `^[0-9a-f]{32}$`
	case ModeCTF:
		return `^[A-Za-z0-9_]+\{.+\}$`
	}
	return ""
}

// Config is oscar.yaml.
type Config struct {
	Name       string    `yaml:"name"`
	Mode       Mode      `yaml:"mode"`
	Created    time.Time `yaml:"created"`
	LHOSTIface string    `yaml:"lhost_iface"`          // tun0 for OffSec/HTB VPNs
	VPNConfig  string    `yaml:"vpn_config,omitempty"` // path to the .ovpn
	Shell      string    `yaml:"shell,omitempty"`      // default $SHELL, else bash
	Record     *bool     `yaml:"record,omitempty"`     // transcripts via script(1); default on
	Assistant  struct {
		// Enabled: lab default true; exam always false (OffSec bans AI in the
		// exam); pentest default false (client data must not leave the box).
		Enabled *bool `yaml:"enabled,omitempty"`
		// Which model: provider claude | opencode | openai, model, base_url,
		// api_key_env, command (see package assistant). Empty = auto.
		assistant.Config `yaml:",inline"`
	} `yaml:"assistant"`
	// MetasploitTarget is the one exam machine Metasploit was used on.
	MetasploitTarget string `yaml:"metasploit_target,omitempty"`
	// FlagFormat is a regexp flags must match (catches copy-paste mistakes);
	// empty uses the mode default, "-" turns checking off.
	FlagFormat string `yaml:"flag_format,omitempty"`
}

// CheckFlag reports a flag that doesn't look like this engagement's flags.
func (c Config) CheckFlag(v string) error {
	f := c.FlagFormat
	if f == "" {
		f = c.Mode.defaultFlagFormat()
	}
	if f == "" || f == "-" {
		return nil
	}
	re, err := regexp.Compile(f)
	if err != nil {
		return fmt.Errorf("flag_format in oscar.yaml: %w", err)
	}
	if !re.MatchString(v) {
		return fmt.Errorf("%q doesn't match the flag format %s (set flag_format: \"-\" to skip the check)", v, f)
	}
	return nil
}

// AssistantAllowed applies the mode rules.
func (c Config) AssistantAllowed() bool {
	switch c.Mode {
	case ModeExam:
		return false
	case ModePentest:
		return c.Assistant.Enabled != nil && *c.Assistant.Enabled
	}
	return c.Assistant.Enabled == nil || *c.Assistant.Enabled
}

// Recording is on unless record: false.
func (c Config) Recording() bool { return c.Record == nil || *c.Record }

// Status is how far you are on a target.
type Status string

var Statuses = []Status{"new", "enum", "foothold", "privesc", "pwned", "skip"}

// Next cycles through Statuses.
func (s Status) Next() Status {
	for i, x := range Statuses {
		if x == s {
			return Statuses[(i+1)%len(Statuses)]
		}
	}
	return Statuses[0]
}

// Flag is a captured proof file (local.txt, proof.txt, user.txt, root.txt…).
type Flag struct {
	Name  string    `yaml:"name"`
	Value string    `yaml:"value"`
	Time  time.Time `yaml:"time"`
}

// Categories are the usual jeopardy CTF categories.
var Categories = []string{"web", "pwn", "crypto", "rev", "forensics", "misc", "osint", "stego", "hardware"}

// Target is targets/<id>/target.yaml plus where it lives. A target is a host
// (OSCP, HTB, pentest) or a CTF challenge (Category set; Host is its optional
// remote host, Port its port).
type Target struct {
	ID        string   `yaml:"id"`             // directory name: the IP, or a challenge slug
	Name      string   `yaml:"name,omitempty"` // HTB machine or challenge name
	Category  string   `yaml:"category,omitempty"`
	Host      string   `yaml:"host,omitempty"`
	Port      int      `yaml:"port,omitempty"`
	Status    Status   `yaml:"status"`
	OS        string   `yaml:"os,omitempty"`
	Hostnames []string `yaml:"hostnames,omitempty"`
	Flags     []Flag   `yaml:"flags,omitempty"`
	Points    int      `yaml:"points,omitempty"`
	// Checked holds the methodology checklist items done on this target (kb.Item.Key).
	Checked []string `yaml:"checked,omitempty"`

	Dir string `yaml:"-"`
}

// IsChallenge reports whether this is a CTF challenge rather than a host.
func (t Target) IsChallenge() bool { return t.Category != "" }

// Label is what the dashboard calls the target.
func (t Target) Label() string {
	switch {
	case t.Name != "" && t.Host != "" && !t.IsChallenge():
		return t.Name + " (" + t.Host + ")"
	case t.Name != "":
		return t.Name
	}
	return t.Host
}

// Remote is "host:port" for a challenge with a service, else the host.
func (t Target) Remote() string {
	if t.Host != "" && t.Port > 0 {
		return net.JoinHostPort(t.Host, strconv.Itoa(t.Port))
	}
	return t.Host
}

// Flag returns the named flag's value, or "".
func (t Target) Flag(name string) string {
	for _, f := range t.Flags {
		if f.Name == name {
			return f.Value
		}
	}
	return ""
}

// Workspace is an open engagement.
type Workspace struct {
	Root  string
	Cfg   Config
	Scope *scope.Scope
}

const cfgFile = "oscar.yaml"

// Create lays out a new engagement in dir.
func Create(dir, name string, mode Mode, scopeLines []string) (*Workspace, error) {
	if !validMode(mode) {
		return nil, fmt.Errorf("mode must be one of %v, not %q", Modes, mode)
	}
	if _, err := os.Stat(filepath.Join(dir, cfgFile)); err == nil {
		return nil, fmt.Errorf("%s already holds an engagement", dir)
	}
	sc := "# Rules of engagement: one address, CIDR, range (10.11.1.5-20) or hostname per line.\n" +
		"# Prefix with ! to exclude. oscar refuses to touch anything not listed.\n"
	for _, l := range scopeLines {
		sc += strings.TrimSpace(l) + "\n"
	}
	if _, err := scope.Parse(strings.NewReader(sc)); err != nil {
		return nil, err
	}
	for _, d := range []string{"targets", "www", "logs/sessions", "report"} {
		if err := os.MkdirAll(filepath.Join(dir, d), 0o700); err != nil {
			return nil, err
		}
	}
	cfg := Config{Name: name, Mode: mode, Created: time.Now().UTC().Truncate(time.Second), LHOSTIface: "tun0"}
	files := map[string]string{
		"scope.txt": sc,
		"creds.md": "# Credentials\n\n| user | secret | type | where found | works on |\n" +
			"|------|--------|------|-------------|----------|\n",
		".gitignore": "# Engagement data is sensitive: keep it out of public repos.\nlogs/\nloot/\n*.ovpn\n",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			return nil, err
		}
	}
	w := &Workspace{Root: dir, Cfg: cfg}
	if err := w.SaveConfig(); err != nil {
		return nil, err
	}
	return Open(dir)
}

// Open loads the engagement at dir.
func Open(dir string) (*Workspace, error) {
	b, err := os.ReadFile(filepath.Join(dir, cfgFile))
	if err != nil {
		return nil, err
	}
	w := &Workspace{Root: dir}
	if err := yaml.Unmarshal(b, &w.Cfg); err != nil {
		return nil, fmt.Errorf("%s: %w", cfgFile, err)
	}
	if w.Cfg.LHOSTIface == "" {
		w.Cfg.LHOSTIface = "tun0"
	}
	if w.Cfg.Mode == "" {
		w.Cfg.Mode = ModeLab
	}
	if err := w.ReloadScope(); err != nil {
		return nil, err
	}
	return w, nil
}

// Find locates the engagement: $OSCAR_DIR, then oscar.yaml in the
// current directory or a parent, then the one last used (`oscar use`).
func Find() (*Workspace, error) {
	if d := os.Getenv("OSCAR_DIR"); d != "" {
		return Open(d)
	}
	if wd, err := os.Getwd(); err == nil {
		for d := wd; ; d = filepath.Dir(d) {
			if _, err := os.Stat(filepath.Join(d, cfgFile)); err == nil {
				return Open(d)
			}
			if filepath.Dir(d) == d {
				break
			}
		}
	}
	if b, err := os.ReadFile(currentFile()); err == nil {
		if d := strings.TrimSpace(string(b)); d != "" {
			return Open(d)
		}
	}
	return nil, errors.New("no engagement here: run `oscar new <name>` or cd into one")
}

func currentFile() string {
	d, err := os.UserConfigDir()
	if err != nil {
		d = os.TempDir()
	}
	return filepath.Join(d, "oscar", "current")
}

// SetCurrent remembers dir as the engagement `oscar` opens by default.
func SetCurrent(dir string) error {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(currentFile()), 0o700); err != nil {
		return err
	}
	return os.WriteFile(currentFile(), []byte(abs+"\n"), 0o600)
}

// SaveConfig writes oscar.yaml.
func (w *Workspace) SaveConfig() error {
	b, err := yaml.Marshal(w.Cfg)
	if err != nil {
		return err
	}
	return writeAtomic(filepath.Join(w.Root, cfgFile), b)
}

// ReloadScope re-reads scope.txt (edit it any time).
func (w *Workspace) ReloadScope() error {
	f, err := os.Open(w.Path("scope.txt"))
	if err != nil {
		return err
	}
	defer f.Close()
	s, err := scope.Parse(f)
	if err != nil {
		return err
	}
	w.Scope = s
	return nil
}

// Path joins elements onto the engagement root.
func (w *Workspace) Path(elem ...string) string {
	return filepath.Join(append([]string{w.Root}, elem...)...)
}

// validMode reports whether m is one of Modes.
func validMode(m Mode) bool {
	for _, x := range Modes {
		if x == m {
			return true
		}
	}
	return false
}

// hostID keeps IPv6 colons out of directory names.
func hostID(host string) string { return strings.ReplaceAll(host, ":", "_") }

// Slug turns a challenge name into a directory name: "Baby SQL!" → "baby-sql".
func Slug(name string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
			dash = false
		} else if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.TrimSuffix(b.String(), "-")
}

func normHost(host string) string {
	host = strings.TrimSpace(host)
	if a, err := netip.ParseAddr(host); err == nil {
		return a.Unmap().String()
	}
	return strings.ToLower(host)
}

// AddTarget creates targets/<host> if host is in scope. name is optional
// (an HTB machine name such as "Lame").
func (w *Workspace) AddTarget(host, name string) (Target, error) {
	host = normHost(host)
	if err := w.Scope.Check(host); err != nil {
		return Target{}, err
	}
	id := hostID(host)
	if t, err := w.Target(id); err == nil {
		if name != "" && t.Name == "" {
			t.Name = name
			return t, w.SaveTarget(t)
		}
		return t, nil
	}
	t := Target{ID: id, Name: strings.TrimSpace(name), Host: host, Status: "new", Dir: w.Path("targets", id)}
	title := host
	if t.Name != "" {
		title = t.Name + " (" + host + ")"
	}
	return t, w.scaffold(t, []string{"scans", "loot", "exploits", "screenshots"}, fmt.Sprintf(hostNotes, title))
}

// AddChallenge creates targets/<slug> for a CTF challenge. remote is optional
// ("host:port" or "host"); if given, it must be in scope.
func (w *Workspace) AddChallenge(name, category string, points int, remote string) (Target, error) {
	id := Slug(name)
	if id == "" {
		return Target{}, errors.New("challenge needs a name")
	}
	if net.ParseIP(id) != nil || strings.Count(id, ".") == 3 {
		return Target{}, fmt.Errorf("challenge name %q looks like an address", name)
	}
	category = strings.ToLower(strings.TrimSpace(category))
	if category == "" {
		category = "misc"
	}
	t := Target{ID: id, Name: strings.TrimSpace(name), Category: category, Points: points, Status: "new", Dir: w.Path("targets", id)}
	if remote = strings.TrimSpace(remote); remote != "" {
		host, port := remote, ""
		if h, p, err := net.SplitHostPort(remote); err == nil {
			host, port = h, p
		}
		host = normHost(host)
		if err := w.Scope.Check(host); err != nil {
			return Target{}, err
		}
		t.Host = host
		if port != "" {
			n, err := strconv.Atoi(port)
			if err != nil || n < 1 || n > 65535 {
				return Target{}, fmt.Errorf("bad port %q", port)
			}
			t.Port = n
		}
	}
	if _, err := w.Target(id); err == nil {
		return Target{}, fmt.Errorf("challenge %q already exists", id)
	}
	return t, w.scaffold(t, []string{"files", "solve"}, fmt.Sprintf(challengeNotes, t.Name, category, points))
}

func (w *Workspace) scaffold(t Target, dirs []string, notes string) error {
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(t.Dir, d), 0o700); err != nil {
			return err
		}
	}
	if err := os.WriteFile(filepath.Join(t.Dir, "notes.md"), []byte(notes), 0o600); err != nil {
		return err
	}
	return w.SaveTarget(t)
}

const hostNotes = `# %s

## Enumeration
- [ ] all TCP ports, then service/version detection on what's open; top UDP
- [ ] each service: version, known issues, default or reused credentials, anonymous access

## Foothold
<!-- the weakness, the exact steps, so someone else can reproduce it -->

## Privilege escalation
<!-- what you found, why it works, the exact steps -->

## Proof
<!-- one screenshot: whoami, hostname, IP address, and the flag file -->

## Lessons
<!-- what slowed you down; what you'd check first next time -->
`

const challengeNotes = `# %s  (%s, %d pts)

## Description
<!-- paste the challenge text and any hints -->

## Observations

## Solve
<!-- the steps; scripts live in solve/ -->

## Flag
`

// Target loads targets/<id>.
func (w *Workspace) Target(id string) (Target, error) {
	dir := w.Path("targets", hostID(id))
	b, err := os.ReadFile(filepath.Join(dir, "target.yaml"))
	if err != nil {
		return Target{}, err
	}
	var t Target
	if err := yaml.Unmarshal(b, &t); err != nil {
		return Target{}, fmt.Errorf("%s/target.yaml: %w", id, err)
	}
	t.Dir, t.ID = dir, filepath.Base(dir)
	if t.Status == "" {
		t.Status = "new"
	}
	return t, nil
}

// Targets lists all targets: hosts by address, then challenges by category and name.
func (w *Workspace) Targets() ([]Target, error) {
	ents, err := os.ReadDir(w.Path("targets"))
	if err != nil {
		return nil, err
	}
	var out []Target
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		if t, err := w.Target(e.Name()); err == nil {
			out = append(out, t)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.IsChallenge() != b.IsChallenge() {
			return !a.IsChallenge()
		}
		if a.IsChallenge() {
			if a.Category != b.Category {
				return a.Category < b.Category
			}
			return a.ID < b.ID
		}
		return lessHost(a.Host, b.Host)
	})
	return out, nil
}

func lessHost(a, b string) bool {
	x, errA := netip.ParseAddr(a)
	y, errB := netip.ParseAddr(b)
	if errA == nil && errB == nil {
		return x.Less(y)
	}
	if (errA == nil) != (errB == nil) {
		return errA == nil
	}
	return a < b
}

// SaveTarget writes target.yaml.
func (w *Workspace) SaveTarget(t Target) error {
	if t.Dir == "" {
		t.Dir = w.Path("targets", t.ID)
	}
	b, err := yaml.Marshal(t)
	if err != nil {
		return err
	}
	return writeAtomic(filepath.Join(t.Dir, "target.yaml"), b)
}

// SetFlag checks value against the flag format, records it (replacing one of
// the same name) and advances the status: the last flag of the mode means
// pwned (or solved), an earlier one means foothold.
func (w *Workspace) SetFlag(id, name, value string) (Target, error) {
	t, err := w.Target(id)
	if err != nil {
		return t, err
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return t, errors.New("empty flag")
	}
	if err := w.Cfg.CheckFlag(value); err != nil {
		return t, err
	}
	f := Flag{Name: name, Value: value, Time: time.Now().UTC().Truncate(time.Second)}
	replaced := false
	for i := range t.Flags {
		if t.Flags[i].Name == name {
			t.Flags[i], replaced = f, true
		}
	}
	if !replaced {
		t.Flags = append(t.Flags, f)
	}
	names := w.Cfg.Mode.FlagNames()
	switch {
	case t.IsChallenge() || len(names) > 0 && name == names[len(names)-1]:
		t.Status = "pwned"
	case t.Status == "new" || t.Status == "enum":
		t.Status = "foothold"
	}
	return t, w.SaveTarget(t)
}

// Toggle flips a checklist item and saves; it returns the new state.
func (w *Workspace) Toggle(id, key string) (bool, error) {
	t, err := w.Target(id)
	if err != nil {
		return false, err
	}
	for i, k := range t.Checked {
		if k == key {
			t.Checked = append(t.Checked[:i], t.Checked[i+1:]...)
			return false, w.SaveTarget(t)
		}
	}
	t.Checked = append(t.Checked, key)
	return true, w.SaveTarget(t)
}

// IsChecked reports whether a checklist item is done.
func (t Target) IsChecked(key string) bool {
	for _, k := range t.Checked {
		if k == key {
			return true
		}
	}
	return false
}

// Score sums points of pwned targets, and of all targets.
func Score(ts []Target) (got, total int) {
	for _, t := range ts {
		total += t.Points
		if t.Status == "pwned" {
			got += t.Points
		}
	}
	return got, total
}

// HostsLines are /etc/hosts lines for targets with names, for the user to
// add themselves (oscar never edits system files). suffix (".htb") turns
// a machine name into a hostname; "" uses only hostnames nmap found.
func HostsLines(ts []Target, suffix string) []string {
	var out []string
	for _, t := range ts {
		if t.IsChallenge() || net.ParseIP(t.Host) == nil {
			continue
		}
		names := append([]string{}, t.Hostnames...)
		if t.Name != "" && suffix != "" {
			names = append(names, Slug(t.Name)+suffix)
		}
		seen := map[string]bool{}
		var uniq []string
		for _, n := range names {
			if n != "" && !seen[n] {
				seen[n] = true
				uniq = append(uniq, n)
			}
		}
		if len(uniq) > 0 {
			out = append(out, t.Host+"\t"+strings.Join(uniq, " "))
		}
	}
	return out
}

// ---- audit log

// Entry is one line of logs/commands.jsonl.
type Entry struct {
	Time       time.Time `json:"time"`
	Target     string    `json:"target,omitempty"`
	Action     string    `json:"action"`
	Command    string    `json:"command"`
	Transcript string    `json:"transcript,omitempty"`
	Mode       Mode      `json:"mode"`
}

// Log appends an entry. Failing to log is an error: an engagement without an
// audit trail is one you can't defend.
func (w *Workspace) Log(e Entry) error {
	if e.Time.IsZero() {
		e.Time = time.Now().UTC()
	}
	e.Mode = w.Cfg.Mode
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(w.Path("logs", "commands.jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(b, '\n'))
	return err
}

// History reads the audit log (oldest first); a damaged line is skipped.
func (w *Workspace) History() ([]Entry, error) {
	b, err := os.ReadFile(w.Path("logs", "commands.jsonl"))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []Entry
	for _, line := range strings.Split(string(b), "\n") {
		var e Entry
		if json.Unmarshal([]byte(line), &e) == nil && e.Action != "" {
			out = append(out, e)
		}
	}
	return out, nil
}

// TranscriptPath is a fresh logs/sessions file for a session named name.
func (w *Workspace) TranscriptPath(name string) string {
	clean := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '.' || r == '-' {
			return r
		}
		return '_'
	}, name)
	if len(clean) > 60 {
		clean = clean[:60]
	}
	return w.Path("logs", "sessions", time.Now().UTC().Format("20060102T150405Z")+"-"+clean+".log")
}

func writeAtomic(path string, b []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
