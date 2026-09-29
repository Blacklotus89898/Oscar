# Curated OSCP Box List — Organized for Learning

The goal: practice **by skill, in order of difficulty**, instead of picking boxes at random. Work each phase until the technique is automatic, then move on. Active Directory (Phase 6) is 40% of the exam — weight your time accordingly.

## Sources & honesty note

- **Primary list:** [TJnull's NetSecFocus Trophy Room](https://docs.google.com/spreadsheets/d/1dwSMIAPIam0PuRBkCiDI88pU3yzrqqHkDtBngUHNCw8/htmlview) (PWK V3, last updated **July 2026**) — the canonical community list. Reproduced faithfully in the [Full reference](#full-reference--tjnull-list-current) at the bottom.
- **Also great:** LainKusanagi's OSCP-like list.
- The **Phase** grouping below is *my* re-organization by learning objective to make practice progressive. **Classic starters** (Phase 1) are universally-recommended warm-ups that have rotated off the current TJnull sheet but remain the best on-ramp.
- **This list is not exhaustive and does not guarantee a pass.** Difficulty is approximate — check the live platform rating. Per-box technique tags are for the well-documented boxes; where a box isn't tagged, read its writeup.
- **Per-box writeups:** [0xdf](https://0xdf.gitlab.io/cheatsheets/offsec) (best written) · [IppSec](https://ippsec.rocks/) (best video). Use them **only after** an honest ~90-minute attempt.

## Legend

| Tag | Meaning |
|---|---|
| **HTB** | Hack The Box (retired boxes need VIP) |
| **PGP** | Proving Grounds *Practice* (OffSec, paid — closest to exam) |
| **PGPlay** | Proving Grounds *Play* (free tier) |
| **VL** | Vulnlab (now merged into HTB) |
| 🟢 | Warm-up / easy · 🟡 core exam-level · 🔴 hard / post-OSCP stretch |
| 📄 | Related doc in this repo |

## How to run a box (every time)

1. Follow the [checklist](../01-methodology/checklist.md); log it in [`box-tracker.md`](box-tracker.md) or the [dashboard](../dashboard.html).
2. **Manual only** — no Metasploit (that's the exam reality; you get it on one exam box only).
3. Stuck 90 min → one nudge from a writeup, then keep going yourself.
4. Rooted → read the writeup anyway to learn the *intended* path and what you missed.
5. Screenshot the flag **with the IP**, every time, to build the exam habit.

---

## Phase 1 — Foundations (classic warm-ups) 🟢

Learn the loop: enumerate → foothold → shell → escalate. Do most of these. Fast wins to build confidence and muscle memory.

| Box | Plat | OS | Teaches | 📄 |
|---|---|---|---|---|
| Lame | HTB | Linux | Ancient Samba RCE — pure enumeration→exploit | [enum](../02-enumeration/enumeration.md) |
| Blue | HTB | Win | EternalBlue (MS17-010) — the canonical Windows exploit | [enum](../02-enumeration/enumeration.md) |
| Legacy | HTB | Win | MS08-067 / MS17-010 — SMB exploitation | [enum](../02-enumeration/enumeration.md) |
| Devel | HTB | Win | FTP→webroot webshell → kernel privesc | [win-pe](../05-privesc/windows-privesc.md) |
| Jerry | HTB | Win | Tomcat manager default creds → WAR shell | [web](../03-web/web-attacks.md) |
| Netmon | HTB | Win | PRTG creds → RCE | [win-pe](../05-privesc/windows-privesc.md) |
| Optimum | HTB | Win | HFS RCE → Windows kernel privesc | [win-pe](../05-privesc/windows-privesc.md) |
| Bashed | HTB | Linux | phpbash webshell → sudo misconfig | [lin-pe](../05-privesc/linux-privesc.md) |
| Shocker | HTB | Linux | Shellshock → sudo | [lin-pe](../05-privesc/linux-privesc.md) |
| Nibbles | HTB/PGP | Linux | Nibbleblog upload → sudo | [lin-pe](../05-privesc/linux-privesc.md) |
| Twiggy | PGP | Linux | Salt-API RCE — friendly PGP starter | [enum](../02-enumeration/enumeration.md) |
| Astronaut | PGP | Linux | Web (PHP/GLPI) foothold | [web](../03-web/web-attacks.md) |

---

## Phase 2 — Web-app footholds 🟡

The #1 foothold surface on the exam. Drill SQLi, file upload, LFI/RFI, command injection, and CMS CVEs. Pair with [`03-web/web-attacks.md`](../03-web/web-attacks.md).

| Box | Plat | OS | Focus |
|---|---|---|---|
| Networked | HTB | Linux | File-upload filter bypass → cron |
| CozyHosting | HTB | Linux | Spring Boot actuator → cred → RCE |
| BoardLight | HTB | Linux | Dolibarr CMS CVE |
| Editorial | HTB | Linux | SSRF → git creds |
| Usage | HTB | Linux | SQLi (Laravel) → admin |
| Builder / LinkVortex / Dog | HTB | Linux | Jenkins / Ghost CMS / Backdrop CMS CVEs |
| Sau / Broker | HTB | Linux | Request-baskets SSRF / ActiveMQ CVE |
| PC / Levram / Extplorer | PGP | Linux | gRPC / eLabFTW / eXtplorer web CVEs |
| Blogger / Loly | PGPlay | Linux | WordPress enumeration → upload shell |
| ExploreThe CMS: Magic | HTB | Linux | SQLi auth bypass + file upload |

---

## Phase 3 — Linux privilege escalation 🟡

Foothold is often easy; the *lesson* is the privesc. Run linpeas but **read it**. Pair with [`05-privesc/linux-privesc.md`](../05-privesc/linux-privesc.md) + [GTFOBins](https://gtfobins.github.io/).

| Box | Plat | Privesc lesson |
|---|---|---|
| Busqueda | HTB | Sudo script + Gitea |
| UpDown | HTB | Git dev vhost → sudo binary |
| Help | HTB | Kernel / sudo |
| Monitored | HTB | SNMP creds → sudo |
| Keeper | HTB | KeePass dump → SSH key |
| Titanic / Editor | HTB | Gitea / XWiki path → root process |
| Exfiltrated | PGP | Subrion CMS → cron |
| Pelican | PGP | Exiftool / process abuse |
| Boolean / Codo / Clue | PGP | Capabilities / sudo / info-leak chains |
| Scrutiny / Lavita / Hub | PGP | Cred reuse & sudo paths |
| SoSimple / Seppuku / Tre | PGPlay | Classic sudo/SUID/cron practice |

---

## Phase 4 — Windows privilege escalation 🟡

Focus on token privileges (`SeImpersonate` → Potato), service misconfigs, and stored creds. Pair with [`05-privesc/windows-privesc.md`](../05-privesc/windows-privesc.md) + [LOLBAS](https://lolbas-project.github.io/).

| Box | Plat | Privesc lesson |
|---|---|---|
| Servmon | HTB | NVMS/NSClient++ → service |
| Support | HTB | LDAP cred → resource-based delegation |
| StreamIO | HTB | SQLi → LAPS read |
| Jeeves | HTB | Jenkins → token / KeePass |
| Access / Heist | HTB | Stored creds, runas / cmdkey |
| Algernon | PGP | Smasher service RCE |
| Kevin / Authby | PGP | Gom/HP service, FTP creds |
| Craft / Jacko | PGP | Service binary hijack |
| Squid / Slort / MedJed | PGP | Proxy pivot / unquoted-path / service |
| Nickel / Shenzi / Hutch | PGP | Cred reuse, XAMPP, LAPS |
| DVR4 | PGP | Argus service exploit |

---

## Phase 5 — Buffer overflow (optional) 🟡

De-emphasized on OSCP+ but mechanical points if it appears. One box is enough to lock in the workflow. Pair with [`10-buffer-overflow/buffer-overflow.md`](../10-buffer-overflow/buffer-overflow.md).

| Box | Plat | Notes |
|---|---|---|
| Brainpan | HTB/VulnHub | The classic 32-bit Windows BOF teaching box |
| Buff | HTB | CloudMe BOF (+ Gym web foothold) |
| Kyoto | PGP | Tagged "Windows Buffer Overflow" on TJnull list |
| Slort | PGP | Includes a BOF component |

---

## Phase 6 — Active Directory ⭐ (40% of the exam) 🟡→🔴

**The highest-priority skill.** Start with single AD boxes (foothold → DA on one host set), then do multi-host chains. Pair with [`06-active-directory/active-directory.md`](../06-active-directory/active-directory.md) + [The Hacker Recipes](https://www.thehacker.recipes/).

### Single-box AD (start here)

| Box | Plat | AD lesson |
|---|---|---|
| Active | HTB | GPP password (SYSVOL) + Kerberoasting — *the* AD starter |
| Forest | HTB | AS-REP roast → DCSync |
| Sauna | HTB | AS-REP roast → autologon → DCSync |
| Monteverde | HTB | Azure AD Connect cred |
| Cascade | HTB | LDAP recon → AD Recycle Bin |
| Blackfield | HTB | AS-REP → Backup Operators → NTDS |
| Timelapse | HTB | PFX cert → LAPS |
| Return | HTB | Printer creds → Server Operators |
| Intelligence | HTB | DNS + gMSA + RBCD |
| Escape / Certified | HTB | AD CS (ESC1/ESC-*) certificate abuse |
| Cicada / Manager / Administrator | HTB | Recent exam-like single AD sets |
| Access / Heist / Vault / Resourced | PGP | PGP's AD practice boxes |
| Nagoya / Hutch | PGP | AD foothold → DA |

### Multi-host AD chains (exam-realistic) 🔴

| Set | Plat | Why |
|---|---|---|
| Baby / Baby2 / Breach / Sweep / Sendai | VL | Bite-sized AD chains |
| Hybrid / Trusted / Lustrous / Reflection | VL | Multi-box AD chains (pivoting required) |
| GOAD | Self-host | Free, deliberately-vulnerable AD lab — endless practice |

---

## Phase 7 — Pivoting & full networks 🔴

Chain hosts, tunnel into hidden subnets. Pair with [`07-pivoting/pivoting-tunneling.md`](../07-pivoting/pivoting-tunneling.md). The Vulnlab **[Chain]** sets above and the Pro Labs below are the best practice.

| Lab | Plat | Notes |
|---|---|---|
| Dante | HTB Pro Lab | Beginner-friendly multi-host network — great OSCP pivoting prep |
| Zephyr | HTB Pro Lab | AD-heavy network (post-OSCP / OSEP-leaning) |
| RastaLabs / Alchemy | HTB Pro Lab | Larger red-team networks (stretch) |

---

## Phase 8 — Exam simulation 🔴

Closest to the real thing. Do these last, **timed**, then write a full [report](../09-reporting/report-template.md).

- **PEN-200 Challenge Labs (OSCP-A / -B / -C)** — the official, most exam-like simulation. Do all three if enrolled.
- **PGP "Post-OSCP / Challenging" boxes** (see reference below): Osaka, ProStore, RPC1, Symbolic, Validator, Marshalled, Educated, Nara.
- **HTB "Challenging yourself" list:** Mentor, Absolute, Outdated, Atom, APT, Multimaster, Authority, Rebound, Vintage, EscapeTwo, Tombwatcher.
- Run a self-made **24h simulation**: 3 standalones + 1 AD set, no help, then report.

---

## Suggested schedule

| Weeks | Focus | Target |
|---|---|---|
| 1–2 | Phase 1 foundations | ~10 boxes, get the loop automatic |
| 3–4 | Phase 2 web + Phase 3 Linux PE | ~15 boxes |
| 5–6 | Phase 4 Windows PE (+ Phase 5 BOF once) | ~15 boxes |
| 6–9 | **Phase 6 Active Directory** (overlap) | all single AD boxes + 2–3 chains |
| 9–10 | Phase 7 pivoting (Dante) | full lab |
| 10–12 | Phase 8 simulations + reports | Challenge Labs + timed runs |

---

# Full reference — TJnull list (current)

> Faithful reproduction of the NetSecFocus Trophy Room (PWK V3, updated **July 2026**). Do not request edit access to the sheet — make a copy. Availability/difficulty change; verify on the platform.

### Hack The Box

**Linux:** Busqueda · UpDown · Sau · Help · Broker · Intentions · Soccer · Keeper · Monitored · BoardLight · Networked · CozyHosting · Editorial · Magic · Pandora · Builder · LinkVortex · Dog · Markup · Editor · Usage · Titanic · Outbound *(Assumed Breach)* · Expressway · Browsed

**Windows:** Escape · Servmon · Support · StreamIO · Blackfield · Intelligence · Jeeves · Manager · Access · Aero · Mailing · Administrator · Certified · Heist · Tombwatcher *(Assumed Breach)*

**Windows Active Directory:** Active · Forest · Sauna · Monteverde · Timelapse · Return · Cascade · Flight · Blackfield · Cicada · Escape · Adagio *(Enterprise)* · TheFrizz · Fluffy · Puppy · Voleur · Signed *(Assumed Breach)* · Eighteen *(Assumed Breach)*

**Post-OSCP / challenge:** Mentor · Absolute · Outdated · Atom · APT · Aero · Cerberus · Multimaster · Cereal · Quick · Authority · Clicker · Rebound · Mailing · Vintage · EscapeTwo · Tombwatcher · Rustykey *(Timeroasting)* · DarkZero *(Assumed Breach)*

**Pro Labs:** Dante · RastaLabs · Zephyr · Alchemy

### Proving Grounds Practice (PGP)

**Linux:** Twiggy · Exfiltrated · Pelican · Astronaut · Blackgate · Boolean · Clue · Cockpit · Codo · Crane · Levram · Extplorer · Hub · Image · Law · Lavita · PC · Fired · Press · Scrutiny · RubyDome · Zipper · Flu · Workaholic · PyLoader · Plum · SPX · Jordak · BitForge · Vmdak · Ochima · Nibbles · CVE-2023-6019 · Sea · Payday · Snookums · SpiderSociety

**Windows:** Algernon · Authby · Craft · Hutch · Internal · Jacko · Kevin · Resourced · Squid · DVR4 · Hepet · Shenzi · Nickel · Slort · MedJed · Monster · Mice

**Windows Active Directory:** Access · Heist · Vault · Nagoya · Resourced · Hutch

**Post-OSCP / challenge:** Nagoya · Osaka · ProStore · RPC1 · Symbolic · Upsploit · Validator · GLPI · Marshalled · Educated · Kyoto *(Win BOF)* · Nara *(Win AD)*

### Proving Grounds Play (free)

Election 1 · Stapler · Monitoring · InsanityHosting · Vegeta 1 · SoSimple · Gaara · Amaterasu · Blogger · Potato · DC-9 · Tre · Seppuku · Funbox · Katana · DriftingBlue6 · Loly · Sams *(Win)* · BTRSys2.1

### Vulnlab (now in HTB)

**Linux:** Data · Feedback · Sync · Forgotten · Build · Bamboo · Dump
**Windows:** Media · Job · Job2 · Lock · Escape · Bruno
**Windows AD:** Baby · Baby2 · Breach · Sweep · Sendai · Phantom · Hybrid *(Chain)* · Trusted *(Chain)* · Lustrous *(Chain)* · Reflection *(Chain)*

### Self-hosted AD labs (free)

- **GOAD** — https://github.com/Orange-Cyberdefense/GOAD
- **VulnAD / TJnull AD boxes** — https://github.com/tjnull/OSCP-Stuff/tree/master/Active-Directory
- **Ludus** — https://gitlab.com/badsectorlabs/ludus
