# OSCP+ Exam Guide

> ⚠️ Rules and format change. **The official [OSCP Exam Guide](https://help.offsec.com/hc/en-us/articles/360040165632-OSCP-Exam-Guide) and [Exam FAQ](https://help.offsec.com/hc/en-us/articles/4412170923924-OSCP-Exam-FAQ) are the only authority.** Re-read both the week before your exam. This doc reflects the format as of the OSCP+ update (Nov 2024).

## Format at a glance

| Item | Detail |
|---|---|
| Duration | **23 hours 45 minutes** hacking, then **24 hours** to write & submit the report |
| Passing score | **70 / 100** |
| Proctored | Yes — webcam, mic, screen share the whole time |
| Certification | Passing now grants **OSCP+** (a 3-year "continuing" cert); the base OSCP name was retired into OSCP+ |

## Scoring

- **3 standalone machines** — 20 points each (typically 10 for initial/`local.txt`, 10 for privesc/`proof.txt`).
- **1 Active Directory set** — **40 points**, awarded as a block: 10 for initial access + 10 for each of the additional machines/escalations in the domain chain. AD is scored **all-or-nothing per step** — partial credit on AD is limited, so the AD set is the highest-value, highest-priority target.
- **No more bonus points.** The old +10 for lab report/exercises was removed with OSCP+.

### Passing combinations (need 70)

- 40 (AD) + 3 × `local.txt` (10 each) = 70 ✅
- 40 (AD) + 2 × `local.txt` + 1 × `proof.txt` = 70 ✅
- 20 (partial AD) + 3 standalone fully rooted (60) = 80 ✅ (if AD partial credit applies)
- 3 standalone fully rooted (60) alone = **60 ❌ — not enough.** You essentially must engage the AD set.

**Takeaway:** The AD set is 40% of the exam and hard to pass without. Prioritize AD skills.

## Tool rules (know these cold — violating = fail)

- **Metasploit / Meterpreter: limited use.** You may use the full Metasploit framework (auto-exploits, meterpreter, post modules) against **only ONE single standalone target** of your choice — and **never** against the AD set. Use it wisely or not at all.
- **Allowed unlimited:** `msfvenom` (payload generation), `msfconsole`'s `multi/handler` (listener), and manual exploitation everywhere.
- **Banned:** commercial/automated exploitation tools and frameworks — Cobalt Strike, Core Impact, **SQLmap**, **Nessus/Nexpose/OpenVAS** (automated vuln scanners), automatic AD-exploitation tools that chain attacks for you, mass-scanners, and any "one-click root" tooling. Chisel, Ligolo-ng, BloodHound, Impacket, CrackMapExec/NetExec, evil-winrm, effectively all manual tooling = **fine**.
- **AI assistants / ChatGPT / Copilot: not allowed during the exam.** (You can use them to *study* — like right now — just not on exam day.)
- When in doubt, **check the official guide**, not a forum post.

## What's tested (PEN-200 domains)

Enumeration · Web attacks · Client-side (limited) · Common services · Buffer overflow is **de-emphasized** (may or may not appear; don't over-invest) · Linux privesc · Windows privesc · Password attacks · Port forwarding & tunneling · **Active Directory** (enumeration, Kerberos attacks, lateral movement).

## Exam-day strategy

**Before:**
- Sleep. A rested brain out-hacks a caffeinated wreck. Do **not** cram the night before.
- Prep your environment: VPN pack tested, note tool (Obsidian/CherryTree/Sublime + `Sysreptor`/template) open, screenshot tool bound to a hotkey, [`09-reporting/`](../09-reporting/report-template.md) template ready.
- Set up a screenshot + command-log habit **from minute one**. Every flag needs proof: screenshot of the flag + `ipconfig`/`ip a` showing the target's IP.

**During:**
1. **Read the control panel** — note IPs, which box allows Metasploit, revert rules.
2. **Kick off enumeration on ALL targets in parallel.** Nmap every box first thing; don't serialize.
3. **Start with the AD set** — it's 40 points and the initial foothold is often a "known" path. If AD stalls hard, bank standalone points, then return.
4. **Timebox.** ~1.5–2h per standalone before rotating. Stuck = enumerate more, don't force one exploit.
5. **Take the screenshot the moment you get each flag.** People fail by rooting boxes and losing points to missing proof.
6. **Eat, drink, walk.** Take a real break every few hours. Breakthroughs happen away from the keyboard.
7. **~70 points reached? Stop chasing more, verify all your proof/screenshots are captured, then rest before the report.**

**Common ways people fail (avoid these):**
- No/insufficient screenshots → points revoked.
- Rabbit-holing on one box/exploit for 6 hours.
- Ignoring the AD set until too late.
- Using a banned tool "just to check."
- Weak enumeration — 90% of "stuck" is missed enumeration.

## The report

- Due within **24 hours** of the hacking window ending.
- Must be **reproducible**: someone should be able to follow your steps and re-own each box.
- Include: every step, every command, screenshots of shells + `local.txt`/`proof.txt` with target IP visible.
- Use the [template](../09-reporting/report-template.md). Consider [Sysreptor](https://docs.sysreptor.com/) (OffSec-friendly) for authoring.
- **A rooted box with a bad report = 0 points.** Write as you go.

See also: [`../practice/study-plan.md`](../practice/study-plan.md) for how to build up to this.
