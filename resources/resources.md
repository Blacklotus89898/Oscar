# Resources

Curated, high-signal links for OSCP prep. Verify anything time-sensitive against the official OffSec docs.

## Official (authoritative)
- **OSCP Exam Guide** — https://help.offsec.com/hc/en-us/articles/360040165632-OSCP-Exam-Guide
- **OSCP Exam FAQ** — https://help.offsec.com/hc/en-us/articles/4412170923924-OSCP-Exam-FAQ
- **PEN-200 course** — https://www.offsec.com/courses/pen-200/
- **OffSec Proving Grounds** — https://www.offsec.com/labs/ (Practice = OSCP-like boxes)

## Practice platforms
- **Hack The Box** — https://www.hackthebox.com/ (retired boxes + official writeups with VIP)
- **Proving Grounds Practice** — closest 3rd-party analog to exam boxes
- **TryHackMe** — https://tryhackme.com/ (gentler on-ramp; good for fundamentals)
- **VulnHub** — https://www.vulnhub.com/ (free downloadable VMs)

## The box lists (what to actually practice)
- **TJnull's OSCP-like list** (NetSecFocus Google Sheet) — the canonical list of HTB/PG boxes that mirror exam difficulty. Search "TJnull OSCP list" / NetSecFocus.
- **LainKusanagi's OSCP list** — another well-regarded curated list.

## Reference / cheatsheets
- **HackTricks** — https://book.hacktricks.wiki/ (enumeration + privesc encyclopedia)
- **GTFOBins** — https://gtfobins.github.io/ (Linux SUID/sudo escapes)
- **LOLBAS** — https://lolbas-project.github.io/ (Windows living-off-the-land binaries)
- **PayloadsAllTheThings** — https://github.com/swisskyrepo/PayloadsAllTheThings
- **revshells.com** — https://www.revshells.com/ (reverse shell generator)
- **Reverse Shell Cheat Sheet** — pentestmonkey / highon.coffee
- **Total OSCP Guide** — https://sushant747.gitbooks.io/total-oscp-guide/

## Privilege escalation
- **PEASS-ng (linpeas/winpeas)** — https://github.com/peass-ng/PEASS-ng
- **PowerUp / PowerSploit** — https://github.com/PowerShellMafia/PowerSploit
- **GTFOBins** (Linux) · **LOLBAS** (Windows) — above
- **Windows/Linux privesc courses (TCM)** — solid paid deep-dives

## Active Directory
- **BloodHound** — https://github.com/SpecterOps/BloodHound
- **The Hacker Recipes** — https://www.thehacker.recipes/ (excellent AD attack reference)
- **NetExec (CME successor)** — https://github.com/Pennyw0rth/NetExec
- **Impacket** — https://github.com/fortra/impacket
- **Ired.team** — https://www.ired.team/ (AD/offensive technique notes)

## Tunneling / pivoting
- **Ligolo-ng** — https://github.com/nicocha30/ligolo-ng
- **Chisel** — https://github.com/jpillora/chisel

## Wordlists
- **SecLists** — https://github.com/danielmiessler/SecLists
- `rockyou.txt` (ships with Kali under `/usr/share/wordlists/`)

## Note-taking / reporting
- **Obsidian** — https://obsidian.md/
- **CherryTree** — https://www.giuspen.net/cherrytree/
- **Sysreptor** — https://docs.sysreptor.com/ (pentest reporting; free community edition)
- OffSec official report template (in your exam email / PEN-200 materials)

## Communities
- **NetSecFocus Discord**, **OffSec Discord**, r/oscp — for hints, motivation, and up-to-date exam experience reports.

---

### How to use these without wasting time
1. **TJnull list + HTB/PG** = your practice pipeline. Don't distract yourself with 20 platforms.
2. **HackTricks + GTFOBins + LOLBAS + The Hacker Recipes** open in tabs while you work — reference, don't read cover-to-cover.
3. When stuck on a *retired* box, a writeup **after** an honest 90-min attempt teaches methodology faster than grinding blind.
4. On the **exam**: none of these AI/writeup shortcuts are allowed — so train without them now.
