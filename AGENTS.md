# AGENTS.md — Oscar

Guide for any coding agent (Claude Code, Codex, Cursor, Copilot…) and human contributor.

## What this is

A **public** OSCP+/HTB/CTF knowledge base (numbered markdown sections, `practice/`, an
offline `dashboard.html`) plus `cli/`: `oscar`, a Go TUI engagement workspace that reads this
knowledge base (checklist, docs, box-note template, box tracker).

## Commands (cli/)

```bash
cd cli
gofmt -l .                 # must print nothing
go vet ./...
go test ./...              # CI: Linux with -race
GOOS=linux go build -o bin/ ./cmd/oscar
```

## Layout (cli/)

```
cmd/oscar/          main.go CLI; actions.go = every target action (scope check → audit log → script(1));
                    tui*.go Bubble Tea dashboard, sessions, Claude panel, checklist/docs views
internal/scope      rules of engagement parsing + Check
internal/workspace  engagement dir, targets/challenges, flags + format, audit log, modes
internal/nmap       nmap XML parse + merge
internal/kb         this repo: checklist, docs/search, note template, box tracker
internal/report     report.md draft
internal/term       from tether: pty sessions
internal/assistant  chat-only Backend: claude (Claude Code CLI), opencode (OpenCode CLI), openai (HTTP)
                    OPENCODE_BIN=... OPENCODE_MODEL=p/m go test -run Live ./internal/assistant/  (opt-in)
```

## Non-negotiables

- **Authorised use only.** Every action against a host goes through `actions.go` and
  `Scope.Check`; never add a path that skips it. Nothing is logged after the fact: log first,
  refuse to run if logging fails.
- **No offensive payloads in the tool.** No exploit code, reverse shells, payload generators
  or attack automation in `cli/`. oscar organises, records, scans with standard nmap presets and
  surfaces the user's own notes.
- **Exam mode has no AI.** `Config.AssistantAllowed` must stay false for `exam` regardless of
  config; pentest is opt-in. Every assistant backend stays chat-only: claude (`--tools ""`,
  `--strict-mcp-config`), opencode (tools denied top-level *and* per agent in a config written to
  a private dir, `--pure`, tool events abort), openai (never send `tools`). A new backend needs a
  test proving the model is offered no tools.
- **No engagement data in this repo.** Engagements live outside git (`oscar new` refuses a git
  work tree). Never commit flags, creds, scans or logs; test fixtures must be synthetic.
- **The knowledge base is the user's.** `internal/kb` reads it; the only write is a row in
  `practice/box-tracker.md`, after confirmation.
- Linux is the target platform; the code must still build on Windows (`go vet` both).

## Docs changes

Keep the numbered-section structure; update the README contents table and the structure
tree when adding a file. Checklist items are `- [ ]` lines under `##` headings (the CLI
parses them; `TestParseChecklistRealFile` guards it).
