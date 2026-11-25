package nmap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const quick = `<?xml version="1.0"?>
<nmaprun scanner="nmap">
<host><status state="up"/><address addr="10.10.10.5" addrtype="ipv4"/>
<address addr="00:50:56:AA:BB:CC" addrtype="mac"/>
<hostnames><hostname name="dc01.htb" type="PTR"/></hostnames>
<ports>
<port protocol="tcp" portid="445"><state state="open"/><service name="microsoft-ds"/></port>
<port protocol="tcp" portid="80"><state state="open"/><service name="http" product="Microsoft IIS httpd" version="10.0"/>
  <script id="http-title" output="IIS Windows Server"/></port>
<port protocol="tcp" portid="23"><state state="closed"/><service name="telnet"/></port>
</ports>
<os><osmatch name="Windows Server 2019" accuracy="96"/><osmatch name="Windows 10" accuracy="90"/></os>
</host>
<host><status state="down"/><address addr="10.10.10.6" addrtype="ipv4"/></host>
</nmaprun>`

const full = `<?xml version="1.0"?>
<nmaprun>
<host><status state="up"/><address addr="10.10.10.5" addrtype="ipv4"/>
<ports>
<port protocol="tcp" portid="445"><state state="open"/><service name="microsoft-ds" product="Windows Server 2019 microsoft-ds"/></port>
<port protocol="tcp" portid="5985"><state state="open"/><service name="wsman"/></port>
<port protocol="tcp" portid="443"><state state="open"/><service name="http" tunnel="ssl"/></port>
</ports></host>` // interrupted: no </nmaprun>

func TestParse(t *testing.T) {
	hosts, err := Parse(strings.NewReader(quick))
	if err != nil {
		t.Fatal(err)
	}
	if len(hosts) != 1 {
		t.Fatalf("want 1 up host, got %d", len(hosts))
	}
	h := hosts[0]
	if h.Addr != "10.10.10.5" || h.OS != "Windows Server 2019" || h.Hostnames[0] != "dc01.htb" {
		t.Fatalf("host = %+v", h)
	}
	if len(h.Ports) != 2 {
		t.Fatalf("closed port not dropped: %+v", h.Ports)
	}
	if got := h.Ports[1].Label(); got != "80/tcp http Microsoft IIS httpd 10.0" {
		t.Errorf("label = %q", got)
	}
	if h.Ports[1].Scripts["http-title"] != "IIS Windows Server" {
		t.Errorf("scripts = %v", h.Ports[1].Scripts)
	}
}

func TestParseInterrupted(t *testing.T) {
	hosts, err := Parse(strings.NewReader(full))
	if err != nil {
		t.Fatal(err)
	}
	if len(hosts) != 1 || len(hosts[0].Ports) != 3 {
		t.Fatalf("hosts = %+v", hosts)
	}
}

func TestLoadDirMerges(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string, age time.Duration) {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		ts := time.Now().Add(-age)
		_ = os.Chtimes(p, ts, ts)
	}
	write("quick.xml", quick, 2*time.Hour)
	write("full.xml", full, time.Hour)
	write("junk.nmap", "not xml", 0)

	h, err := LoadDir(dir, "10.10.10.5")
	if err != nil {
		t.Fatal(err)
	}
	if got := h.PortNumbers("tcp"); got != "80,443,445,5985" {
		t.Fatalf("ports = %s", got)
	}
	for _, p := range h.Ports {
		switch p.Number {
		case 445:
			if p.Product != "Windows Server 2019 microsoft-ds" {
				t.Errorf("445 product not upgraded: %q", p.Product)
			}
		case 80:
			if p.Product == "" || p.Scripts["http-title"] == "" {
				t.Errorf("80 lost detail: %+v", p)
			}
		case 443:
			if p.Label() != "443/tcp ssl/http" {
				t.Errorf("443 label = %q", p.Label())
			}
		}
	}
}

func TestLoadDirEmpty(t *testing.T) {
	h, err := LoadDir(t.TempDir(), "10.0.0.1")
	if err != nil || len(h.Ports) != 0 {
		t.Fatalf("h=%+v err=%v", h, err)
	}
}
