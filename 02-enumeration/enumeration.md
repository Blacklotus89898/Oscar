# Enumeration

> Set a target variable to keep commands copy-pasteable:
> ```bash
> export IP=10.10.10.10
> export DOMAIN=example.local
> ```

## Nmap — the standard flow

```bash
# 1) Fast full TCP port discovery (all 65535 ports)
nmap -p- --min-rate 5000 -T4 -Pn $IP -oN nmap-allports.txt

# 2) Deep scan on the open ports only (version + default scripts)
nmap -p 22,80,445 -sC -sV -O -Pn $IP -oN nmap-deep.txt
#   -sC default scripts   -sV version   -O OS guess   -Pn skip host discovery

# 3) UDP top ports (slow — run in background)
sudo nmap -sU --top-ports 100 $IP -oN nmap-udp.txt

# Vuln scripts (use judiciously; NOT an automated scanner ban issue, but noisy)
nmap -p 445 --script "vuln" $IP
```

**Tips**
- `--min-rate 5000` speeds up full scans dramatically; drop it if results look unreliable.
- Grep open ports for the next command: `grep -oP '^\d+' nmap-allports.txt` (from `-oG`/`-oN` output).
- Always add discovered hostnames to `/etc/hosts`: `echo "$IP box.htb DC01.example.local" | sudo tee -a /etc/hosts`.
- Re-scan if a box was reverted or acts differently.

`autorecon` (allowed, it's just automation of enumeration, not exploitation) is a great time-saver:
```bash
autorecon $IP
```

---

## HTTP / HTTPS (80, 443, 8080, 8000, 8443, ...)

**The #1 foothold surface. Spend the most time here.**

```bash
# Fingerprint
whatweb http://$IP
curl -sI http://$IP                       # headers, server, redirects
curl -s http://$IP | head -50             # peek at the body

# Directory / file brute force
feroxbuster -u http://$IP -w /usr/share/seclists/Discovery/Web-Content/raft-medium-directories.txt -x php,txt,html,bak -t 50
gobuster dir -u http://$IP -w /usr/share/seclists/Discovery/Web-Content/directory-list-2.3-medium.txt -x php,txt,html -t 50
ffuf -u http://$IP/FUZZ -w /usr/share/seclists/Discovery/Web-Content/raft-medium-directories.txt -e .php,.txt,.bak

# Virtual host / subdomain brute force (when a domain is known)
ffuf -u http://$IP -H "Host: FUZZ.$DOMAIN" -w /usr/share/seclists/Discovery/DNS/subdomains-top1million-5000.txt -fs <size-of-default-response>
gobuster vhost -u http://$DOMAIN -w /usr/share/seclists/Discovery/DNS/subdomains-top1million-5000.txt --append-domain
```

**Always check by hand:** `robots.txt`, `sitemap.xml`, `/.git/` (dump with `git-dumper`), `/backup`, `.bak`/`.old`/`~` files, JS files for endpoints & secrets, HTML comments, error messages (version leaks), login pages (default creds / SQLi), any upload feature.

CMS-specific:
```bash
wpscan --url http://$IP --enumerate ap,u --api-token <token>   # WordPress
droopescan scan drupal -u http://$IP                            # Drupal
# Joomla: joomscan; identify version → searchsploit
```
→ Attacks in [`../03-web/web-attacks.md`](../03-web/web-attacks.md).

---

## SMB (139, 445)

```bash
# Enumerate shares (null/anonymous, then with creds)
netexec smb $IP -u '' -p '' --shares          # NetExec (formerly crackmapexec)
netexec smb $IP -u 'user' -p 'pass' --shares
smbclient -L //$IP/ -N                         # list shares, no password
smbclient //$IP/ShareName -N                   # connect to a share
smbmap -H $IP -u null                           # readable/writable overview
smbmap -H $IP -u user -p pass -R ShareName      # recursive listing

enum4linux-ng -A $IP                            # all-in-one (users, shares, policy, OS)

# Version → EternalBlue etc.
nmap -p445 --script smb-vuln-* $IP

# RID cycling / user enumeration
netexec smb $IP -u '' -p '' --rid-brute
```
Look inside shares for: creds in configs/scripts, `.kdbx` (KeePass), SSH keys, `web.config`, backup files, Groups.xml (GPP passwords).

---

## FTP (21)

```bash
nmap -p21 -sC -sV $IP
ftp $IP            # try user "anonymous", password anything
# inside: binary; ls -la; get <file>; put <file> (writable? maybe web-root!)
# grab everything anonymous:
wget -m --no-passive ftp://anonymous:anonymous@$IP
```
Note the FTP software+version → `searchsploit`. Anonymous **write** into a web root = instant webshell.

---

## SSH (22)

```bash
nmap -p22 -sV $IP                 # version → any auth-bypass CVE (rare)
ssh user@$IP                      # try found creds
ssh-keyscan $IP                   # host key
# Weak-key / user enum are rare on OSCP; SSH is mostly a spray target for creds you found elsewhere.
```

---

## DNS (53)

```bash
dig axfr @$IP $DOMAIN             # zone transfer — can dump all records
dnsenum --dnsserver $IP $DOMAIN
nslookup; > server $IP; > $DOMAIN
# subdomain brute:
ffuf -u http://$IP -H "Host: FUZZ.$DOMAIN" -w <wordlist>
```

---

## SNMP (161/udp)

```bash
snmpwalk -v2c -c public $IP                     # whole tree
snmpwalk -v2c -c public $IP 1.3.6.1.4.1.77.1.2.25   # Windows users
snmpwalk -v2c -c public $IP 1.3.6.1.2.1.25.4.2.1.2  # running processes
snmpwalk -v2c -c public $IP 1.3.6.1.2.1.6.13.1.3    # listening TCP ports
onesixtyone -c /usr/share/seclists/Discovery/SNMP/common-snmp-community-strings.txt $IP  # find community strings
snmp-check $IP -c public
```
SNMP leaks processes (with **command-line args → creds!**), users, installed software, and ports.

---

## LDAP (389, 636)

```bash
nmap -p389 --script "ldap*" $IP
ldapsearch -x -H ldap://$IP -s base namingcontexts     # find base DN
ldapsearch -x -H ldap://$IP -b "DC=example,DC=local"   # anonymous dump
netexec ldap $IP -u '' -p '' --users                   # user enum
```
See [`../06-active-directory/active-directory.md`](../06-active-directory/active-directory.md) for the full AD flow.

---

## NFS (111 rpcbind, 2049)

```bash
showmount -e $IP                  # list exports
mkdir /mnt/nfs && sudo mount -t nfs $IP:/export /mnt/nfs -o nolock
# no_root_squash + you create a SUID binary as root locally → privesc vector
```

---

## RPC (111, 135)

```bash
rpcclient -U "" -N $IP
# then: enumdomusers ; enumdomgroups ; querydominfo ; lsaquery ; getdompwinfo
rpcinfo -p $IP
```

---

## Mail (25 SMTP, 110 POP3, 143 IMAP)

```bash
nmap -p25 --script smtp-* $IP
# user enumeration
smtp-user-enum -M VRFY -U /usr/share/seclists/Usernames/names.txt -t $IP
# manual:  telnet $IP 25  →  VRFY root
```

---

## Databases

```bash
# MySQL 3306
mysql -h $IP -u root -p
# MSSQL 1433
netexec mssql $IP -u sa -p 'pass'          # then --local-auth / xp_cmdshell
impacket-mssqlclient user:pass@$IP -windows-auth
# PostgreSQL 5432
psql -h $IP -U postgres
# Redis 6379  (often unauthenticated!)
redis-cli -h $IP
# MongoDB 27017
mongo mongodb://$IP:27017
# Oracle 1521
odat all -s $IP
```
MSSQL `sa` + `xp_cmdshell` = RCE. Redis unauth can write SSH keys / webshells.

---

## RDP (3389) / WinRM (5985/5986)

```bash
netexec rdp $IP -u user -p pass            # validate RDP creds
xfreerdp /u:user /p:pass /v:$IP /cert:ignore /dynamic-resolution
netexec winrm $IP -u user -p pass          # validate WinRM creds
evil-winrm -i $IP -u user -p pass          # shell over WinRM (great for AD)
```

---

## Quick "what do I do with port X" table

| Port | Service | First move |
|---|---|---|
| 21 | FTP | anon login, version exploit |
| 22 | SSH | spray found creds |
| 25/110/143 | Mail | user enum (VRFY), version |
| 53 | DNS | zone transfer, subdomain enum |
| 80/443/8080 | HTTP | **fingerprint + dirbust + attack inputs** |
| 88 | Kerberos | AD! → AS-REP/kerberoast |
| 111/2049 | NFS | `showmount -e`, mount |
| 135/139/445 | SMB/RPC | shares, null session, version exploit |
| 161/udp | SNMP | `snmpwalk public` |
| 389/636 | LDAP | anonymous bind dump |
| 1433 | MSSQL | `sa` creds → xp_cmdshell |
| 3306 | MySQL | default creds |
| 3389 | RDP | validate creds, connect |
| 5432 | PostgreSQL | default creds |
| 5985/5986 | WinRM | evil-winrm with creds |
| 6379 | Redis | unauth → write keys |
