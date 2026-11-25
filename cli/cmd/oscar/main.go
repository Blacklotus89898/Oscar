// Command oscar is the terminal side of the Oscar knowledge base: an
// engagement workspace for OSCP labs and exam, Hack The Box and CTFs.
// Run it with no arguments for the dashboard.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/Blacklotus89898/Oscar/cli/internal/kb"
	"github.com/Blacklotus89898/Oscar/cli/internal/report"
	"github.com/Blacklotus89898/Oscar/cli/internal/workspace"
)

const usage = `oscar — engagement workspace for OSCP, Hack The Box and CTFs

  oscar                                 dashboard (TUI)
  oscar new <name> [-mode M] [-scope S] [-dir D]
                                        start an engagement; M = lab|exam|htb|ctf|pentest
                                        S = comma-separated scope (CIDR, IP, range, host)
  oscar use <dir>                       make <dir> the default engagement
  oscar add <ip|host> [name]            add a host target (must be in scope)
  oscar chal <name> <category> <points> [host:port]
                                        add a CTF challenge
  oscar ls                              list targets
  oscar flag <target> <name> <value>    record a flag (format-checked)
  oscar check <host>                    is <host> in scope?
  oscar hosts                           /etc/hosts lines for your targets (add them yourself)
  oscar report                          write report/report.md
  oscar where                           engagement and knowledge-base paths

Engagements live outside this (public) repo, in ~/engagements by default.
`

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := run(ctx, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "oscar:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	if len(args) == 0 || args[0] == "ui" {
		if runtime.GOOS == "windows" {
			fmt.Fprintln(os.Stderr, "note: oscar is built for your Linux attack box (Kali, Parrot, WSL); sessions run Linux tools.")
		}
		return runTUI(ctx)
	}
	switch args[0] {
	case "help", "-h", "--help":
		fmt.Print(usage)
		return nil
	case "new":
		return cmdNew(args[1:])
	case "use":
		if len(args) != 2 {
			return errors.New("usage: oscar use <dir>")
		}
		if _, err := workspace.Open(args[1]); err != nil {
			return err
		}
		return workspace.SetCurrent(args[1])
	}

	w, err := workspace.Find()
	if err != nil {
		return err
	}
	switch args[0] {
	case "add":
		if len(args) < 2 || len(args) > 3 {
			return errors.New("usage: oscar add <ip|host> [name]")
		}
		name := ""
		if len(args) == 3 {
			name = args[2]
		}
		t, err := w.AddTarget(args[1], name)
		if err != nil {
			return err
		}
		fmt.Println("added", t.Label(), "→", t.Dir)
	case "chal":
		if len(args) < 4 || len(args) > 5 {
			return errors.New("usage: oscar chal <name> <category> <points> [host:port]")
		}
		pts, err := strconv.Atoi(args[3])
		if err != nil {
			return fmt.Errorf("points must be a number: %q", args[3])
		}
		remote := ""
		if len(args) == 5 {
			remote = args[4]
		}
		t, err := w.AddChallenge(args[1], args[2], pts, remote)
		if err != nil {
			return err
		}
		fmt.Println("added", t.Label(), "→", t.Dir)
	case "ls":
		ts, err := w.Targets()
		if err != nil {
			return err
		}
		for _, t := range ts {
			var fl []string
			for _, f := range t.Flags {
				fl = append(fl, f.Name)
			}
			fmt.Printf("%-24s %-9s %-10s %s\n", t.ID, t.Status, t.Category, strings.Join(fl, ","))
		}
		if got, total := workspace.Score(ts); total > 0 {
			fmt.Printf("\nscore %d / %d\n", got, total)
		}
	case "flag":
		if len(args) != 4 {
			return errors.New("usage: oscar flag <target> <name> <value>")
		}
		t, err := w.SetFlag(args[1], args[2], args[3])
		if err != nil {
			return err
		}
		_ = w.Log(workspace.Entry{Target: t.ID, Action: "flag " + args[2]})
		fmt.Printf("%s: %s recorded, status %s\n", t.Label(), args[2], t.Status)
	case "check":
		if len(args) != 2 {
			return errors.New("usage: oscar check <host>")
		}
		if err := w.Scope.Check(args[1]); err != nil {
			return err
		}
		fmt.Println(args[1], "is in scope")
	case "hosts":
		ts, err := w.Targets()
		if err != nil {
			return err
		}
		suffix := ""
		if w.Cfg.Mode == workspace.ModeHTB {
			suffix = ".htb"
		}
		lines := workspace.HostsLines(ts, suffix)
		if len(lines) == 0 {
			fmt.Println("# no named targets yet (oscar add <ip> <name>, or scan to learn hostnames)")
		}
		for _, l := range lines {
			fmt.Println(l)
		}
		fmt.Fprintln(os.Stderr, "# review, then append with: oscar hosts | sudo tee -a /etc/hosts")
	case "report":
		p, err := report.Write(w)
		if err != nil {
			return err
		}
		fmt.Println("wrote", p)
		fmt.Println("PDF: pandoc", p, "-o report.pdf   (OffSec wants PDF for the exam report)")
	case "where":
		fmt.Println("engagement:", w.Root)
		if k, err := kb.Find(); err == nil {
			fmt.Println("knowledge base:", k.Root)
		} else {
			fmt.Println("knowledge base:", err)
		}
	default:
		return fmt.Errorf("unknown command %q (oscar help)", args[0])
	}
	return nil
}

func cmdNew(args []string) error {
	fs := flag.NewFlagSet("new", flag.ContinueOnError)
	mode := fs.String("mode", "lab", "lab|exam|htb|ctf|pentest")
	scopeArg := fs.String("scope", "", "comma-separated scope")
	dir := fs.String("dir", "", "engagement directory (default ~/engagements/<name>)")
	// Allow the name before or after the flags.
	var name string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		name, args = args[0], args[1:]
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if name == "" && fs.NArg() > 0 {
		name = fs.Arg(0)
	}
	if name == "" {
		return errors.New("usage: oscar new <name> [-mode lab|exam|htb|ctf|pentest] [-scope 10.10.10.0/24,...]")
	}
	d := *dir
	if d == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		d = filepath.Join(home, "engagements", workspace.Slug(name))
	}
	if repo := gitRoot(d); repo != "" {
		return fmt.Errorf("%s is inside the git repo %s: engagement data (flags, creds, scans) must not be committed — pick a directory outside it", d, repo)
	}
	var scopeLines []string
	if *scopeArg != "" {
		scopeLines = strings.Split(*scopeArg, ",")
	}
	if err := os.MkdirAll(d, 0o700); err != nil {
		return err
	}
	w, err := workspace.Create(d, name, workspace.Mode(*mode), scopeLines)
	if err != nil {
		return err
	}
	if err := workspace.SetCurrent(d); err != nil {
		return err
	}
	fmt.Printf("engagement %q (%s) at %s\n", name, w.Cfg.Mode, d)
	if len(scopeLines) == 0 {
		fmt.Println("scope is empty: list targets in", w.Path("scope.txt"), "before adding them")
	}
	if w.Cfg.Mode == workspace.ModeExam {
		fmt.Println("exam mode: the Claude panel is disabled (OffSec forbids AI tools during the exam).")
		fmt.Println("Re-read 00-exam-guide/exam-guide.md for the tool restrictions before you start.")
	}
	fmt.Println("run `oscar` for the dashboard")
	return nil
}

// gitRoot returns the work tree containing dir (or its nearest existing
// parent), or "".
func gitRoot(dir string) string {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return ""
	}
	for d := abs; ; d = filepath.Dir(d) {
		if _, err := os.Stat(filepath.Join(d, ".git")); err == nil {
			return d
		}
		if filepath.Dir(d) == d {
			return ""
		}
	}
}
