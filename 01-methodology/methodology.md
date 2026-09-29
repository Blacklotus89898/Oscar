# The OSCP Methodology

OSCP rewards **process over tricks**. When you're stuck, you have almost always **missed something in enumeration** — not failed to find the one magic exploit. This is the loop to run on every target.

```
┌─────────────────────────────────────────────────────────────┐
│  1. RECON        Full port scan → service/version scan       │
│  2. ENUMERATE    Dig into EVERY open service                 │
│  3. FOOTHOLD     Find a vuln / cred / misconfig → get shell  │
│  4. STABILIZE    Upgrade to a proper TTY, note the user      │
│  5. LOCAL ENUM   Re-enumerate AS this user (creds, files)    │
│  6. PRIVESC      Escalate to root / SYSTEM / Administrator    │
│  7. LOOT         Grab hashes, creds, keys, flags             │
│  8. PIVOT        Use this host to reach the next network     │
│         ─────────  repeat from step 1 on new targets  ─────  │
└─────────────────────────────────────────────────────────────┘
```

## 1. Recon (find the doors)

- Full TCP port scan first — **never assume top-1000 is enough.** Boxes hide services on high ports.
- Then a deep version/script scan on the open ports only.
- UDP scan the common ports (SNMP 161, DNS 53, TFTP 69) — easy wins get missed here.
- See [`../02-enumeration/enumeration.md`](../02-enumeration/enumeration.md).

## 2. Enumerate (understand every door)

For **each** open service, ask: *What is it? What version? Is it misconfigured? Any default/weak creds? Any public exploit? Any info it leaks?*

- Enumerate the **highest-value** services first: web (80/443/8080/etc.), SMB (445), then everything else.
- Web = the #1 foothold source. Always: tech fingerprint → directory/vhost bruteforce → look at source → find login/upload/param.
- Write down **every** username, hostname, email, version string, path, and interesting file you see. These become creds, exploits, and pivots later.

## 3. Foothold (get in)

Foothold sources, roughly in order of how often they appear:
1. **Web app vuln** (SQLi, file upload, LFI→log poison, command injection, auth bypass, known CVE in the CMS/software).
2. **Default / weak / reused credentials** on any service (SSH, SMB, RDP, WinRM, database, web login).
3. **Known exploit for an outdated service version** (`searchsploit`). Read the exploit before running it.
4. **Anonymous access** (FTP anon, SMB null session, NFS export) leaking creds or a writable path.

Prefer a **shell** but a **credential** is often the real foothold — a valid password sprayed across services opens more than one exploit.

## 4. Stabilize

Get off a dumb reverse shell onto a real TTY before doing anything delicate. See [`../04-exploitation/shells-and-transfers.md`](../04-exploitation/shells-and-transfers.md#upgrading-your-shell).

## 5. Local enumeration (the step people skip)

You are now a user on the box. **Re-run enumeration from the inside** — this is where privesc paths live.
- Run [`linpeas`/`winpeas`](../05-privesc/linux-privesc.md) but **read the output**, don't just grep for "root".
- Manually check: who am I, what can I run, what's running as root/SYSTEM, what's listening on localhost only, config files with creds, other users' home dirs, cron/scheduled tasks, sudo rights.
- **Localhost-only services** (found via `netstat`/`ss`) are pivot & privesc gold — tunnel to them.

## 6. Privesc

- [Linux privesc](../05-privesc/linux-privesc.md) · [Windows privesc](../05-privesc/windows-privesc.md).
- Match findings to a technique. Verify the exploit fits the exact OS/kernel/software version before firing.

## 7. Loot

- Grab `local.txt` / `proof.txt` (**screenshot immediately** with IP visible).
- Dump password hashes, SSH keys, browser/creds stores, config secrets, `/etc/shadow`, SAM/`NTDS.dit`.
- **Every credential is a key to another lock** — spray it everywhere (other users, other hosts, the domain).

## 8. Pivot

- Note this host's other interfaces / routes / ARP neighbors — they reveal the next subnet.
- Set up tunneling ([`../07-pivoting/pivoting-tunneling.md`](../07-pivoting/pivoting-tunneling.md)) and scan the newly reachable network. Restart the loop.

---

## Mindset rules

- **"I'm stuck" almost always means "I under-enumerated."** Go back to step 2 before looking for exotic exploits.
- **Read the output.** The answer is usually already on your screen.
- **One thing at a time**, and note what you tried so you don't loop.
- **Timebox** and rotate targets; a fresh box resets your brain.
- **Keep it simple** — the intended path is rarely a 0-day. It's a missed cred or an obvious misconfig.
- **Document as you go.** If it's not screenshotted, it didn't happen (for the exam).

➡️ Print the [checklist](checklist.md) and keep it beside you.
