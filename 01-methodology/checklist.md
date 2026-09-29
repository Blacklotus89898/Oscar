# Per-Target Checklist

A no-panic checklist to run against every box. When stuck, come back here and find the box you *haven't* ticked.

## 🔍 Recon
- [ ] Full TCP port scan (all 65535 ports)
- [ ] Version/script scan on open ports
- [ ] UDP scan of common ports (53, 69, 161, 123, 500)
- [ ] Noted every open port + service + version in notes
- [ ] Added hostname(s) to `/etc/hosts` (from TLS certs, redirects, SMB)

## 🌐 Web (per web port)
- [ ] Visited the site; read the page + **view-source** + JS files
- [ ] Fingerprinted tech (whatweb / Wappalyzer / headers)
- [ ] Directory brute force (feroxbuster/gobuster/ffuf) with a good wordlist + extensions
- [ ] Vhost / subdomain brute force (if a domain is in play)
- [ ] Checked `robots.txt`, `sitemap.xml`, `/.git`, backup files (`.bak`, `~`, `.old`)
- [ ] Identified the CMS/framework + exact version → `searchsploit`
- [ ] Tested every input: login (SQLi/default creds), search, upload, URL params (LFI/RFI/cmd inj)
- [ ] Looked for default creds on any admin panel
- [ ] Any file upload → tried getting a web shell

## 📁 SMB / 445
- [ ] Null session / anonymous listing of shares
- [ ] Listed + read readable shares (look for creds, configs, keys)
- [ ] SMB version → known exploit? (EternalBlue etc.)
- [ ] Enumerated users (RID cycling) if allowed
- [ ] Tried any creds found elsewhere here

## 🗂️ Other services
- [ ] FTP (21): anonymous login? readable/writable? version exploit?
- [ ] SSH (22): version; any user list; password spray with found creds
- [ ] DNS (53): zone transfer attempt; subdomain enum
- [ ] SNMP (161/udp): `snmpwalk` with `public` → processes, users, ports, software
- [ ] LDAP (389/636): anonymous bind → users/structure
- [ ] NFS (2049): `showmount -e` → mount exports
- [ ] RPC (111/135): `rpcclient` enum
- [ ] Mail (25/110/143): user enumeration (VRFY), version
- [ ] Databases (3306/1433/5432/1521/27017/6379): default creds, version, access

## 🐚 Foothold
- [ ] Got a shell OR valid credentials
- [ ] **Screenshotted** initial access
- [ ] Upgraded to a stable TTY (Linux) / verified shell type (Windows)
- [ ] Grabbed `local.txt` / user flag + screenshot with IP

## ⬆️ Privilege escalation
- [ ] `whoami` / `id`, groups, hostname, OS + kernel/build version
- [ ] Ran linpeas/winpeas **and read it**
- [ ] `sudo -l` (Linux) / privileges via `whoami /priv` (Windows)
- [ ] SUID/SGID (Linux) / service & registry misconfigs (Windows)
- [ ] Cron jobs / scheduled tasks
- [ ] Writable files/paths owned by root/SYSTEM; PATH hijack
- [ ] Config files & histories for creds; other users' dirs
- [ ] Internal-only listening ports (`ss -tlnp` / `netstat -ano`)
- [ ] Kernel/software version → local exploit (verify it fits!)
- [ ] Got root/SYSTEM → `proof.txt` + **screenshot with IP**

## 💰 Loot & pivot
- [ ] Dumped hashes / SSH keys / config secrets
- [ ] Sprayed found creds across all users/services/hosts
- [ ] Noted extra network interfaces / routes / neighbors
- [ ] Set up tunnel and scanned the new subnet

## 📸 Proof (do this the moment you get each flag)
- [ ] Screenshot: flag contents **+** `ip a`/`ipconfig` in the same shot
- [ ] Command log / notes updated for the report
