# Note-Taking Discipline

Good notes are the difference between a pass and "I rooted it but lost the points." Build the habit on **box #1**, not the week before the exam.

## Tools (pick one, learn it well)

- **Obsidian** — markdown, linkable, great for a personal knowledge base. Most popular for OSCP.
- **CherryTree** — hierarchical, offline, screenshots inline. A classic OSCP choice.
- **Sublime/VSCode + a folder of `.md`** — simple and fast.
- **Sysreptor** — purpose-built pentest reporting; can double as your note tool → report.

## The per-box note habit

For every box, capture as you go:
1. **Target header** — IP, hostname(s), OS, date.
2. **Every command you run** — paste it, even the failures (so you don't repeat them).
3. **Every finding** — open ports, versions, usernames, creds, paths, interesting files.
4. **Screenshots at every milestone** — foothold shell, `local.txt`+IP, root shell, `proof.txt`+IP.
5. **The "why"** — one line on what worked and what the intended path was (for review).

Use the template in [`../practice/box-note-template.md`](../practice/box-note-template.md).

## Screenshot discipline (exam-critical)

- Bind a screenshot hotkey (Flameshot on Linux: `flameshot gui`).
- **Every flag screenshot must show the flag AND the IP** (`ip a` / `ipconfig`) in the same image.
- Screenshot the moment you get the flag — reverts wipe unsaved proof.
- Keep one folder per box for images.

## A credentials log (your most valuable file)

Keep a running table for the whole engagement/exam — creds are keys to other locks:

| Credential | Source | Works on |
|---|---|---|
| jdoe : Summer2025! | SMB share config | SSH box1, WinRM DC |
| svc_sql : (NTLM ...) | secretsdump box2 | psexec box3 |

## Why this matters

- The report is due **24h after** hacking ends — if your notes are complete, the report writes itself.
- "I definitely rooted it" without a screenshot = **0 points**.
- Notes let you rotate between stuck boxes without losing your place.
