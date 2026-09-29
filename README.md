# OSCP Study, Practice & Documentation

> A complete, self-contained knowledge base for the **OffSec Certified Professional (OSCP+)** exam and the PEN-200 course — built for legal, authorized practice on platforms like Hack The Box, OffSec Proving Grounds, and your own lab.

## ⚖️ Scope & Ethics

Everything here is for **authorized security testing and education only**: your own lab, CTF platforms (HTB, PG Play/Practice, TryHackMe), and machines you have **explicit written permission** to test. Never point any of these techniques at systems you don't own or aren't contracted to assess. Unauthorized access is a crime in nearly every jurisdiction.

## 🖥️ Interactive dashboard

Open [`dashboard.html`](dashboard.html) in any browser (double-click it — no server needed) for a live tracker: box log, per-target checklist, exam-readiness meter, and a copy-paste cheatsheet. Progress saves in your browser; use its Export/Import buttons to back it up or move between machines.

## 🗺️ How to use this repo

1. **Start with the exam guide** — understand what you're training for before you grind boxes.
2. **Internalize the methodology** — OSCP is a methodology exam, not a memorization exam. The checklist in `01-methodology/` is your anti-panic tool.
3. **Use the topic docs as reference** while doing boxes. Don't read them cover to cover — do a box, get stuck, come back, learn the technique, add your own notes.
4. **Track everything** in `practice/`. Volume of boxes + quality of notes = a pass.
5. **Practice the report** from box #1, not the week before the exam.

## 📚 Contents

| Section | What's inside |
|---|---|
| [`00-exam-guide/`](00-exam-guide/exam-guide.md) | Exam format, scoring, rules, tool restrictions, exam-day strategy |
| [`01-methodology/`](01-methodology/methodology.md) | The full attack workflow + a per-target checklist |
| [`02-enumeration/`](02-enumeration/enumeration.md) | Nmap + per-service enumeration (SMB, web, FTP, SNMP, LDAP, DNS, RPC, NFS, mail, databases) |
| [`03-web/`](03-web/web-attacks.md) | Web enumeration & attacks: SQLi, LFI/RFI, upload, command injection, auth bypass |
| [`04-exploitation/`](04-exploitation/shells-and-transfers.md) | Reverse shells, shell upgrading, msfvenom, file transfers, searchsploit, Metasploit |
| [`05-privesc/`](05-privesc/linux-privesc.md) | [Linux](05-privesc/linux-privesc.md) & [Windows](05-privesc/windows-privesc.md) privilege escalation |
| [`06-active-directory/`](06-active-directory/active-directory.md) | AD enumeration, Kerberoasting, AS-REP, PtH, lateral movement, DCSync |
| [`07-pivoting/`](07-pivoting/pivoting-tunneling.md) | Port forwarding, SSH tunnels, Chisel, Ligolo-ng, proxychains |
| [`08-password-attacks/`](08-password-attacks/password-attacks.md) | Hydra, hashcat, john, wordlist generation, hash cracking |
| [`09-reporting/`](09-reporting/report-template.md) | Report template + note-taking discipline |
| [`10-buffer-overflow/`](10-buffer-overflow/buffer-overflow.md) | Stack BOF workflow (de-emphasized on the exam, but free points if it appears) |
| [`practice/`](practice/study-plan.md) | 90-day study plan, box tracker, per-box note template |
| [`cheatsheets/`](cheatsheets/quick-reference.md) | One-page command quick reference |
| [`resources/`](resources/resources.md) | Curated links, tools, box lists (TJnull), reading |

## 🗂️ Repository structure

```
Oscar/
├── README.md                     ← you are here
├── dashboard.html                ← interactive offline tracker (open in browser)
├── 00-exam-guide/
│   └── exam-guide.md             format · scoring · rules · exam-day strategy
├── 01-methodology/
│   ├── methodology.md            the attack loop + mindset
│   └── checklist.md              per-target anti-panic checklist
├── 02-enumeration/
│   └── enumeration.md            nmap + every service
├── 03-web/
│   └── web-attacks.md            SQLi · LFI/RFI · upload · cmd injection
├── 04-exploitation/
│   └── shells-and-transfers.md   shells · TTY upgrade · msfvenom · transfers
├── 05-privesc/
│   ├── linux-privesc.md
│   └── windows-privesc.md
├── 06-active-directory/
│   └── active-directory.md       BloodHound · Kerberoast · PtH · DCSync (40 pts)
├── 07-pivoting/
│   └── pivoting-tunneling.md     Ligolo-ng · Chisel · SSH · proxychains
├── 08-password-attacks/
│   └── password-attacks.md       hydra · hashcat · john · Responder
├── 09-reporting/
│   ├── report-template.md
│   └── note-taking.md
├── 10-buffer-overflow/
│   └── buffer-overflow.md        32-bit stack BOF workflow
├── cheatsheets/
│   └── quick-reference.md        the one-pager
├── practice/
│   ├── study-plan.md             phased ~90-day plan
│   ├── box-tracker.md            log every box
│   └── box-note-template.md      copy per box
└── resources/
    └── resources.md              TJnull list · HackTricks · GTFOBins · tools
```

## ✅ The one-sentence method

> **Enumerate → find a foothold → get a shell → stabilize it → enumerate again as the new user → escalate → loot → pivot → repeat, writing down every command and every finding as you go.**

## 🚦 Readiness signal

You're likely ready when you can consistently root a "medium" HTB/PG box in **under ~2 hours without walkthroughs**, and complete a small AD set end to end. See [`practice/study-plan.md`](practice/study-plan.md).
