# Best Courses to Get Ready for Easy & Medium HTB Boxes

No single course is enough on its own. What reliably works is **structured course + guided practice + watching experts think**, then volume. This page names the best of each and how to combine them.

> Goal here: become fluent at **easy and medium** HTB boxes (the on-ramp to OSCP). Everything below doubles as OSCP prep — nothing is wasted if you continue to the cert.

---

## 🥇 Primary pick: HTB Academy — "Penetration Tester" job-role path

The best-matched course, because HTB makes it and its modules map directly onto the boxes you'll solve.

- **What:** a long structured path — enumeration → web → privilege escalation → Active Directory → pivoting → reporting — with hands-on labs after every concept.
- **Why it fits:** teaches the *methodology* HTB boxes test, not just tools. Finishing even ~60% makes easy boxes routine and medium boxes approachable.
- **Cost:** subscription (student tier is cheap — verify current pricing). Modules also sold à la carte for patching weak spots.
- **Bonus:** it's the standard OSCP-prep path too.

## 🥈 Beginner-friendly alternative: TCM Security — "Practical Ethical Hacking" (PEH)

Start here if HTB Academy feels too dense.

- Taught by **The Cyber Mentor** — video-based, beginner-friendly, one-time purchase (cheap, frequent sales).
- Covers the full workflow + an AD lab + buffer overflow.
- Pair with TCM's **Linux Privilege Escalation** and **Windows Privilege Escalation** courses — those two alone unlock most easy/medium privescs.

## 🆓 The free on-ramp (do in parallel, week one)

1. **HTB "Starting Point"** — free guided boxes designed to teach how HTB works. Do all tiers. Fastest way to stop feeling lost.
2. **IppSec — https://ippsec.rocks/** — free video walkthroughs of retired boxes. **Highest-leverage free resource.** Watch how he enumerates and switches ideas.
3. **0xdf — https://0xdf.gitlab.io/** — best written walkthroughs; read *after* you attempt a box.
4. **TryHackMe "Jr Penetration Tester" path** — gentler than HTB; great for fundamentals if you're a true beginner.

---

## The part that actually matters (method > course)

Courses get you ~40% of the way; **volume of boxes + reviewing writeups** gets the rest. The loop that works:

1. Learn a topic (a course module) →
2. Do 2–3 **easy** boxes on that topic from [`../practice/box-list.md`](../practice/box-list.md) (Phases 1–4) →
3. Stuck 60–90 min → watch IppSec / read 0xdf **for that box** →
4. Root it, then write down the technique you missed ([note-taking](../09-reporting/note-taking.md)) →
5. Repeat until easy boxes are boring, then move to medium.

Track every box in the [dashboard](../dashboard.html) or [`../practice/box-tracker.md`](../practice/box-tracker.md).

## Concrete 8-week plan → "easy + medium comfortable"

| Weeks | Do this |
|---|---|
| 1 | HTB **Starting Point** (all tiers) + begin HTB Academy PT path (or TCM PEH) |
| 2–3 | Academy: Enumeration + Web modules → Phase 1–2 boxes from the box-list |
| 4–5 | Academy: Linux + Windows privesc → Phase 3–4 boxes + TCM privesc courses |
| 6 | Grind easy boxes solo — aim: root an easy box in **under 90 min unaided** |
| 7–8 | First **medium** boxes; IppSec/0xdf only after a real attempt |

## If you only pick three things

1. **HTB Academy PT path** (or **TCM PEH** if budget/beginner)
2. **HTB Starting Point** (free)
3. **IppSec** (free)

That combination reliably gets people to easy/medium HTB competency.

---

## Quick comparison

| Resource | Format | Cost | Best for |
|---|---|---|---|
| HTB Academy — PT path | Text + labs | Subscription | Structured, box-aligned, OSCP-ready |
| TCM PEH (+ privesc courses) | Video | One-time (cheap) | Beginners, full workflow |
| HTB Starting Point | Guided boxes | Free | Learning the platform |
| IppSec | Video walkthroughs | Free | Seeing methodology in action |
| 0xdf writeups | Written | Free | Post-attempt review |
| TryHackMe — Jr Pentester | Text + labs | Freemium | True beginners |

> Pricing and course contents change — confirm on each provider's site before buying.
