# OSCP Exam Report Template

> A rooted box with a bad report = **0 points**. Write as you go, not at the end. The report must let someone **reproduce** every compromise. OffSec provides an official `.docx` template — this mirrors its structure so you can draft in Markdown and paste in, or use [Sysreptor](https://docs.sysreptor.com/).

**Proof rules:** every `local.txt`/`proof.txt` needs a screenshot showing the flag **and** the machine's IP (`ip a` / `ipconfig`) in the same shot. Include the commands you ran.

---

```markdown
# OffSec OSCP Exam Report

**Student:** <name>
**OSID:** OS-XXXXX
**Email:** <email>
**Date:** <date>

## High-Level Summary
Brief narrative: which machines were compromised, overall approach, total points claimed.

## Recommendations
Generic remediation guidance (patch, least privilege, strong passwords, disable unused services).

---

# Methodology / Attack Narrative

For EACH target repeat this block:

## Target: <IP / hostname>  (<Standalone | AD - role>)

### Service Enumeration
| Port | Service | Version |
|------|---------|---------|
| 80   | http    | Apache 2.4.x |
| ...  | ...     | ... |

**Nmap:**
```
<paste the nmap command + relevant output>
```

### Initial Access / Foothold
- **Vulnerability:** <what + why it worked>
- **Steps (reproducible):**
  1. <command>  — <what it did>
  2. ...
- **Proof of exploitation:** <screenshot: shell as `user`>
- **local.txt:**
  ```
  <hash>
  ```
  <screenshot: `type local.txt` / `cat local.txt` + `ipconfig`/`ip a`>

### Privilege Escalation
- **Vulnerability:** <SUID / sudo / token / service / kernel / etc.>
- **Steps (reproducible):**
  1. <command>
  2. ...
- **Proof:** <screenshot: shell as root/SYSTEM/Administrator>
- **proof.txt:**
  ```
  <hash>
  ```
  <screenshot: proof.txt + IP>

---

# Active Directory Set

Document the full chain: initial foothold → each user/credential obtained → lateral movement → Domain Admin / DC. Include a short path diagram and every credential/hash used, with the command that used it.

---

# Appendix (optional)
Extra commands, scripts written, notes.
```

---

## Report tips

- **One folder per box** for screenshots, named clearly (`web-foothold.png`, `root-proof.png`).
- Screenshot the **command + its output together**; crop nothing important.
- Show `whoami`/`id` in shells to prove the privilege level.
- If you used your one Metasploit allowance, say which box and how.
- Don't include tool spam — include the **steps that reproduce** the compromise.
- Submit as a single **PDF**, follow OffSec's exact naming/upload instructions in the exam email.
- Double-check the report actually opens and images render before you submit.
