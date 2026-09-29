# OSCP Study Plan

A pass is built on **volume of boxes + solid methodology + clean notes**, not on watching more videos. Adjust the timeline to your hours/week; this assumes ~15–20 hrs/week over ~3 months.

## Phase 0 — Setup (a few days)
- Kali VM ready; snapshot it. Install: SecLists, feroxbuster, netexec, impacket, bloodhound, evil-winrm, chisel, ligolo-ng, hashcat/john.
- Pick and set up your **note tool** ([note-taking](../09-reporting/note-taking.md)) and screenshot hotkey.
- Read [`../00-exam-guide/exam-guide.md`](../00-exam-guide/exam-guide.md) and [`../01-methodology/methodology.md`](../01-methodology/methodology.md).
- Get accounts: **PEN-200** (if enrolled), **Hack The Box** (VIP for retired boxes + writeups), **Proving Grounds Practice**.

## Phase 1 — Fundamentals (weeks 1–4)
- Work the **PEN-200 course material + exercises** if enrolled. Otherwise use HTB Academy modules (Footprinting, Web, Privesc paths).
- Drill each topic doc here against 1–2 easy boxes:
  - Enumeration → do 5 easy boxes focusing only on thorough enum.
  - Web attacks → boxes with web footholds.
  - Linux & Windows privesc → grind privesc-focused boxes.
- **Goal:** internalize the loop. Follow the [checklist](../01-methodology/checklist.md) every time.

## Phase 2 — Volume on standalones (weeks 5–8)
- Grind the **TJnull OSCP-like list** (HTB + PG). See [`../resources/resources.md`](../resources/resources.md).
- Target **~40–60 boxes**. Easy → medium. Root every one, write notes for every one.
- Rule: try **90 min unaided** before peeking at a hint; after rooting, read a writeup to learn the intended path and what you missed.
- Practice **without Metasploit** (manual only) — that's the exam reality.

## Phase 3 — Active Directory (weeks 7–10, overlaps)
- **This is 40% of the exam — do not shortchange it.**
- Work AD: HTB Academy AD paths, PG Practice AD boxes, and the PEN-200 challenge labs.
- Build fluency with: BloodHound, NetExec, Kerberoast/AS-REP, PtH, evil-winrm/psexec, secretsdump.
- Do full **AD chains** (foothold → lateral → DA) end to end, including [pivoting](../07-pivoting/pivoting-tunneling.md).

## Phase 4 — Exam simulation (weeks 10–12)
- Do the **PEN-200 Challenge Labs** (OSCP-A/B/C) — these are the closest thing to the exam.
- Run **timed full simulations**: 3 standalones + an AD set in 24h, then write a real report.
- Fix your weak spots. Refine your personal cheatsheet.
- **Practice the report** at least twice before exam day.

## Daily/weekly rhythm
- Each session: 1 box (or progress on an AD chain) → complete notes → screenshots.
- Weekly: review what tripped you up; add missing commands to [`../cheatsheets/quick-reference.md`](../cheatsheets/quick-reference.md).

## Readiness checklist (you're likely ready when...)
- [ ] You root most **medium** boxes in under ~2h with no walkthrough.
- [ ] You can complete a full **AD chain** unaided.
- [ ] You never get stuck for lack of enumeration ideas.
- [ ] You comfortably **pivot** into a second subnet.
- [ ] Your notes+screenshots are automatic, and you've written 2+ practice reports.
- [ ] You've passed at least one **timed 24h simulation**.

## Anti-burnout
- Don't marathon daily; consistency beats cramming.
- Reverts and rabbit holes are normal — timebox and rotate.
- Read a writeup after rooting; it's the fastest way to fill methodology gaps.
