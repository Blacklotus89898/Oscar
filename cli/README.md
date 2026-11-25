# oscar — the terminal side of Oscar

**One workspace per engagement, with your own methodology beside every target.** A single Go
binary for your Linux attack box (Kali, Parrot, WSL) that keeps OSCP labs, the exam, Hack The
Box and CTFs organised the way a professional engagement is: scoped, logged, recorded, and
ready to report.

```
oscar   HTB  · week-3 · LHOST 10.10.14.7 · 1/2 pwned
 TARGET                    STATUS     PORTS / CATEGORY                FLAGS         CHECK
 Lame (10.10.10.3)         pwned      21,22,139,445,3632              u✓ r✓         31/48
 Bashed (10.10.10.68)      enum       80                              u· r·         6/48
Bashed (10.10.10.68)
  80/tcp http Apache httpd 2.4.18 (Ubuntu)
next on your checklist (c):
  ☐ Web (per web port): Directory brute force with a good wordlist + extensions
  ☐ Web (per web port): Checked robots.txt, sitemap.xml, /.git, backup files
enter shell · s scan · e notes · f flag · c checklist · d docs · / search · ? help
```

## What it does

- **Scope is enforced.** `scope.txt` lists what you may test (IPs, CIDRs, ranges, hostnames,
  `!exclusions`). oscar refuses to add, scan or open a shell on anything else. A typo can't
  send a scan to someone else's network.
- **Everything is on the record.** Every command oscar runs goes to `logs/commands.jsonl`.
  Every session (shell, scan, editor) is recorded with `script(1)` into `logs/sessions/`, so
  you have a timeline for the report and proof of what you did.
- **Your methodology, per target.** The checklist in `01-methodology/checklist.md` is ticked
  separately for each box (`c`). The dashboard always shows the next three items you haven't
  done. `d` browses and `/` searches this whole knowledge base without leaving the terminal.
- **Scans that feed the dashboard.** nmap presets (`s`): quick, all TCP, scripts and versions
  on the ports found so far, and top UDP. Output goes to the target's `scans/` folder and is
  parsed back into the table.
- **Flags and progress.** `f` records a flag, checks its format (32 hex for OffSec/HTB,
  `flag{...}` for CTFs) and advances the status. `T` writes a pwned box into
  `practice/box-tracker.md`.
- **Report draft.** `R` writes `report/report.md`: summary, then per target the flags, ports,
  your notes, screenshots and timeline. Convert it with `pandoc` to the PDF OffSec wants.
- **Claude as a study partner** (lab/HTB/CTF only). A chat-only panel that sees the dashboard
  and your session screen but has no tools. It's told to teach with hints and point you to
  your own notes, not hand you answers. In **exam mode it doesn't exist**, because OffSec
  forbids AI tools in the exam. In pentest mode it's off unless you enable it.

## Modes

| Mode | For | Flags | Claude |
|---|---|---|---|
| `lab` | PEN-200 labs, Proving Grounds | `local.txt`, `proof.txt` | on |
| `exam` | the OSCP exam | `local.txt`, `proof.txt` | **never** |
| `htb` | Hack The Box, TryHackMe | `user.txt`, `root.txt` | on |
| `ctf` | jeopardy CTFs: challenges with category and points, scoreboard | `flag` (`flag{...}`) | on |
| `pentest` | authorised client work | none | off unless enabled |

## Install & start

```bash
cd cli && go install ./cmd/oscar        # Go 1.24+; or: go build -o ~/.local/bin/oscar ./cmd/oscar
oscar new week-3 -mode htb -scope 10.10.10.0/24
oscar add 10.10.10.3 Lame
oscar                                   # the dashboard
```

CTF: `oscar new defcon-quals -mode ctf -scope chall.example.com`, then in the dashboard
press `a` and enter `Baby SQL | web | 100 | chall.example.com:31337` (offline challenges
need no host).

Engagements live in `~/engagements/<name>`, **outside this public repo**, and oscar refuses
to create one inside any git repo. oscar finds this knowledge base through `$OSCAR_KB`, the
directory you run it from, or `~/dev/Oscar`.

### Engagement layout

```
~/engagements/week-3/
  oscar.yaml        mode, LHOST interface (tun0), vpn_config, flag_format, assistant
  scope.txt         rules of engagement
  creds.md          credentials found (shared across targets)
  targets/10.10.10.3/
    target.yaml     status, flags, checklist ticks
    notes.md        your notes (template below)
    scans/ loot/ exploits/ screenshots/
  targets/baby-sql/ (CTF) files/ solve/ notes.md
  www/              files you serve with H (python http.server on LHOST:8000)
  logs/commands.jsonl  logs/sessions/
  report/report.md
```

## Keys

| Key | Action |
|---|---|
| `enter` | your `$SHELL` in the target's folder, with `RHOST`, `LHOST`, `RPORT`, `TARGET_DIR` exported |
| `s` | nmap: `q` quick · `f` all TCP · `d` scripts+versions on open ports · `u` top UDP (sudo) |
| `e` | edit `notes.md` in `$EDITOR` |
| `f` / `m` | record a flag / cycle status (new → enum → foothold → privesc → pwned → skip) |
| `c` | methodology checklist for this target (`space` tick, `n` next unticked) |
| `d` / `/` | browse / search the Oscar knowledge base |
| `a` | add a target (CTF: `name \| category \| points \| host:port`) |
| `v` / `H` | OpenVPN session (`vpn_config` in oscar.yaml) / serve `www/` on LHOST:8000 |
| `R` / `T` | report draft / add a pwned box to `practice/box-tracker.md` |
| `1`–`9`, `ctrl+]` | jump to a session; in a session `ctrl+]` then `d` returns, `n` next, `l` Claude |
| `l` / `tab`, `x` | Claude panel; explain the last error |

The CLI covers the scriptable parts: `oscar ls`, `flag`, `check <host>`, `hosts` (prints
`/etc/hosts` lines; oscar never edits system files), `report`, `where`.

## Why it doesn't do more

oscar doesn't ship exploits, payloads or attack automation. It organises *your* work, runs
standard recon, and puts your own notes beside you. The OSCP rewards methodology you can
repeat under pressure, and the exam restricts automated exploitation. Write the techniques
into the knowledge base as you learn them, and `/` will find them when you need them.

## Development

```bash
gofmt -l . && go vet ./... && go test ./...
GOOS=linux go build -o bin/ ./cmd/oscar
```

`internal/term` and `internal/assistant` come from
[tether](https://github.com/Blacklotus89898/tether) (embedded pty sessions, headless Claude
Code). See [AGENTS.md](../AGENTS.md) for the rules.
