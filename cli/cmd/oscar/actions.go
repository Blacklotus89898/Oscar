package main

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Blacklotus89898/Oscar/cli/internal/workspace"
)

// Every action that touches a target is built here, so the rules apply in one
// place: the target must be in scope, the command goes into the audit log, and
// the session is recorded with script(1) into logs/sessions/.

// action is a command ready to run in an embedded session.
type action struct {
	name string   // session tab: "scan full 10.10.10.3"
	argv []string // what actually runs (possibly wrapped by script)
	cmd  string   // the human-readable command, for the log
	dir  string   // working directory
	env  []string // extra environment
}

// scanPreset is an nmap run whose -oA output lands in the target's scans/.
type scanPreset struct {
	key, name, desc string
	args            func(ports string) []string
	root            bool // needs sudo (UDP)
}

var scanPresets = []scanPreset{
	{"q", "quick", "top 1000 TCP, default scripts + versions", func(string) []string { return []string{"-sC", "-sV"} }, false},
	{"f", "full", "all 65535 TCP ports, fast", func(string) []string { return []string{"-p-", "--min-rate", "1000", "-T4"} }, false},
	{"d", "deep", "scripts + versions on the open ports found so far", func(p string) []string { return []string{"-sC", "-sV", "-p", p} }, false},
	{"u", "udp", "top 50 UDP ports (sudo)", func(string) []string { return []string{"-sU", "--top-ports", "50"} }, true},
}

func presetByKey(k string) (scanPreset, bool) {
	for _, p := range scanPresets {
		if p.key == k {
			return p, true
		}
	}
	return scanPreset{}, false
}

// scanAction builds an nmap run for t; ports is the known open TCP list (for deep).
func scanAction(w *workspace.Workspace, t workspace.Target, p scanPreset, ports string) (action, error) {
	if t.Host == "" {
		return action{}, errors.New("this challenge has no host to scan")
	}
	if err := w.Scope.Check(t.Host); err != nil {
		return action{}, err
	}
	if p.name == "deep" && ports == "" {
		return action{}, errors.New("no open ports known yet: run a quick or full scan first")
	}
	argv := []string{"nmap"}
	argv = append(argv, p.args(ports)...)
	argv = append(argv, "-oA", filepath.Join(t.Dir, "scans", p.name), t.Host)
	if p.root {
		argv = append([]string{"sudo"}, argv...)
	}
	return action{name: "scan " + p.name + " " + t.ID, argv: argv, cmd: strings.Join(argv, " "), dir: t.Dir}, nil
}

// shellAction opens your shell in the target's directory with RHOST, LHOST
// and friends exported, so your own tools and notes are one keystroke away.
func shellAction(w *workspace.Workspace, t workspace.Target, lhost string) (action, error) {
	if t.Host != "" {
		if err := w.Scope.Check(t.Host); err != nil {
			return action{}, err
		}
	}
	sh := w.Cfg.Shell
	if sh == "" {
		sh = os.Getenv("SHELL")
	}
	if sh == "" {
		sh = "bash"
	}
	env := []string{"OSCAR_TARGET=" + t.ID, "OSCAR_DIR=" + w.Root, "TARGET_DIR=" + t.Dir}
	if t.Host != "" {
		env = append(env, "RHOST="+t.Host)
	}
	if t.Port > 0 {
		env = append(env, fmt.Sprintf("RPORT=%d", t.Port))
	}
	if lhost != "" {
		env = append(env, "LHOST="+lhost)
	}
	return action{name: "shell " + t.ID, argv: []string{sh}, cmd: sh, dir: t.Dir, env: env}, nil
}

// notesAction edits the target's notes.md in $EDITOR.
func notesAction(t workspace.Target) action {
	ed := os.Getenv("EDITOR")
	if ed == "" {
		ed = "vi"
	}
	argv := append(strings.Fields(ed), filepath.Join(t.Dir, "notes.md"))
	return action{name: "notes " + t.ID, argv: argv, cmd: strings.Join(argv, " "), dir: t.Dir}
}

// serveAction serves www/ over HTTP on LHOST, for moving your files to a target.
func serveAction(w *workspace.Workspace, lhost string) (action, error) {
	if lhost == "" {
		return action{}, fmt.Errorf("no LHOST: %s has no IPv4 address (VPN down?)", w.Cfg.LHOSTIface)
	}
	argv := []string{"python3", "-m", "http.server", "8000", "--bind", lhost, "--directory", w.Path("www")}
	return action{name: "serve www", argv: argv, cmd: strings.Join(argv, " "), dir: w.Root}, nil
}

// vpnAction runs OpenVPN in a session (sudo asks for your password there).
func vpnAction(w *workspace.Workspace) (action, error) {
	if w.Cfg.VPNConfig == "" {
		return action{}, fmt.Errorf("set vpn_config: /path/to/lab.ovpn in %s", w.Path("oscar.yaml"))
	}
	if _, err := os.Stat(w.Cfg.VPNConfig); err != nil {
		return action{}, err
	}
	argv := []string{"sudo", "openvpn", "--config", w.Cfg.VPNConfig}
	return action{name: "vpn", argv: argv, cmd: strings.Join(argv, " "), dir: w.Root}, nil
}

// prepare logs the action and wraps it in script(1) when recording is on.
func prepare(w *workspace.Workspace, target string, a action) (action, error) {
	e := workspace.Entry{Target: target, Action: a.name, Command: a.cmd}
	if w.Cfg.Recording() {
		if path, err := exec.LookPath("script"); err == nil {
			e.Transcript = w.TranscriptPath(a.name)
			a.argv = []string{path, "-q", "-f", "-e", "-c", shellJoin(withEnv(a)), e.Transcript}
			a.env = nil // folded into the command line
		}
	}
	if err := w.Log(e); err != nil {
		return a, fmt.Errorf("audit log: %w (refusing to run unlogged)", err)
	}
	return a, nil
}

// withEnv prefixes argv with `env K=V ... ` and `cd`, so script(1) runs the
// exact same thing.
func withEnv(a action) []string {
	out := []string{}
	if len(a.env) > 0 {
		out = append(out, "env")
		out = append(out, a.env...)
	}
	return append(out, a.argv...)
}

// shellJoin quotes argv for sh -c.
func shellJoin(argv []string) string {
	q := make([]string, len(argv))
	for i, a := range argv {
		if a != "" && strings.IndexFunc(a, func(r rune) bool {
			return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("-_./=:,@%+", r))
		}) < 0 {
			q[i] = a
			continue
		}
		q[i] = "'" + strings.ReplaceAll(a, "'", `'\''`) + "'"
	}
	return strings.Join(q, " ")
}

// lhost is the first IPv4 address on the VPN interface (tun0), or "".
func lhost(iface string) string {
	ifc, err := net.InterfaceByName(iface)
	if err != nil {
		return ""
	}
	addrs, err := ifc.Addrs()
	if err != nil {
		return ""
	}
	for _, a := range addrs {
		if n, ok := a.(*net.IPNet); ok && n.IP.To4() != nil {
			return n.IP.String()
		}
	}
	return ""
}
