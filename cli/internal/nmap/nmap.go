// Package nmap reads nmap's XML output (-oX / -oA) into the ports the
// dashboard shows and the playbook matches on. Several scans of one host
// (quick, full TCP, UDP, scripts) merge into one view; later, more detailed
// results (service versions) win over bare "open" lines.
package nmap

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Port is one open port.
type Port struct {
	Number  int
	Proto   string // tcp, udp
	Service string // nmap service name: http, microsoft-ds, ssh…
	Product string // "Apache httpd 2.4.41", "OpenSSH 8.2p1 Ubuntu"
	Tunnel  string // "ssl" when nmap detected TLS
	Scripts map[string]string
}

// Label is "80/tcp http Apache httpd 2.4.41".
func (p Port) Label() string {
	s := fmt.Sprintf("%d/%s %s", p.Number, p.Proto, p.Service)
	if p.Tunnel == "ssl" && !strings.Contains(p.Service, "https") {
		s = fmt.Sprintf("%d/%s ssl/%s", p.Number, p.Proto, p.Service)
	}
	if p.Product != "" {
		s += " " + p.Product
	}
	return s
}

// Host is everything known about one address.
type Host struct {
	Addr      string
	Hostnames []string
	OS        string // best osmatch, if -O ran
	Ports     []Port
}

// PortNumbers is the TCP port list "22,80,445" for a follow-up -sC -sV scan.
func (h Host) PortNumbers(proto string) string {
	var ps []string
	for _, p := range h.Ports {
		if p.Proto == proto {
			ps = append(ps, fmt.Sprint(p.Number))
		}
	}
	return strings.Join(ps, ",")
}

type xmlRun struct {
	Hosts []struct {
		Status struct {
			State string `xml:"state,attr"`
		} `xml:"status"`
		Addresses []struct {
			Addr     string `xml:"addr,attr"`
			AddrType string `xml:"addrtype,attr"`
		} `xml:"address"`
		Hostnames []struct {
			Name string `xml:"name,attr"`
		} `xml:"hostnames>hostname"`
		Ports []struct {
			Proto  string `xml:"protocol,attr"`
			PortID int    `xml:"portid,attr"`
			State  struct {
				State string `xml:"state,attr"`
			} `xml:"state"`
			Service struct {
				Name      string `xml:"name,attr"`
				Product   string `xml:"product,attr"`
				Version   string `xml:"version,attr"`
				ExtraInfo string `xml:"extrainfo,attr"`
				Tunnel    string `xml:"tunnel,attr"`
			} `xml:"service"`
			Scripts []struct {
				ID     string `xml:"id,attr"`
				Output string `xml:"output,attr"`
			} `xml:"script"`
		} `xml:"ports>port"`
		OSMatches []struct {
			Name     string `xml:"name,attr"`
			Accuracy int    `xml:"accuracy,attr"`
		} `xml:"os>osmatch"`
	} `xml:"host"`
}

// Parse reads one nmap XML document. An interrupted scan (no closing tag)
// still yields the hosts written so far.
func Parse(r io.Reader) ([]Host, error) {
	var run xmlRun
	err := xml.NewDecoder(r).Decode(&run)
	if err != nil && err != io.ErrUnexpectedEOF && !strings.Contains(err.Error(), "unexpected EOF") {
		return nil, err
	}
	var out []Host
	for _, xh := range run.Hosts {
		if xh.Status.State != "" && xh.Status.State != "up" {
			continue
		}
		h := Host{}
		for _, a := range xh.Addresses {
			if a.AddrType == "ipv4" || a.AddrType == "ipv6" {
				h.Addr = a.Addr
				break
			}
		}
		if h.Addr == "" {
			continue
		}
		for _, n := range xh.Hostnames {
			h.Hostnames = appendUnique(h.Hostnames, n.Name)
		}
		best := 0
		for _, m := range xh.OSMatches {
			if m.Accuracy > best {
				best, h.OS = m.Accuracy, m.Name
			}
		}
		for _, xp := range xh.Ports {
			if xp.State.State != "open" {
				continue
			}
			p := Port{Number: xp.PortID, Proto: xp.Proto, Service: xp.Service.Name, Tunnel: xp.Service.Tunnel}
			p.Product = strings.TrimSpace(strings.Join(nonEmpty(xp.Service.Product, xp.Service.Version, paren(xp.Service.ExtraInfo)), " "))
			for _, s := range xp.Scripts {
				if p.Scripts == nil {
					p.Scripts = map[string]string{}
				}
				p.Scripts[s.ID] = strings.TrimSpace(s.Output)
			}
			h.Ports = append(h.Ports, p)
		}
		out = append(out, h)
	}
	return out, nil
}

// Merge folds b into a (same address): ports are unioned, and a port's
// richer record (product, scripts) replaces a bare one.
func Merge(a, b Host) Host {
	if a.Addr == "" {
		a.Addr = b.Addr
	}
	for _, n := range b.Hostnames {
		a.Hostnames = appendUnique(a.Hostnames, n)
	}
	if b.OS != "" {
		a.OS = b.OS
	}
	idx := map[string]int{}
	for i, p := range a.Ports {
		idx[key(p)] = i
	}
	for _, p := range b.Ports {
		i, ok := idx[key(p)]
		if !ok {
			idx[key(p)] = len(a.Ports)
			a.Ports = append(a.Ports, p)
			continue
		}
		old := &a.Ports[i]
		if p.Service != "" && (old.Service == "" || p.Product != "") {
			old.Service = p.Service
		}
		if p.Product != "" {
			old.Product = p.Product
		}
		if p.Tunnel != "" {
			old.Tunnel = p.Tunnel
		}
		for k, v := range p.Scripts {
			if old.Scripts == nil {
				old.Scripts = map[string]string{}
			}
			old.Scripts[k] = v
		}
	}
	sort.Slice(a.Ports, func(i, j int) bool {
		if a.Ports[i].Proto != a.Ports[j].Proto {
			return a.Ports[i].Proto < a.Ports[j].Proto
		}
		return a.Ports[i].Number < a.Ports[j].Number
	})
	return a
}

// LoadDir merges every *.xml under dir (oldest first) into one Host for addr.
// Hosts with other addresses in those files are ignored.
func LoadDir(dir, addr string) (Host, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.xml"))
	if err != nil {
		return Host{}, err
	}
	sort.Slice(files, func(i, j int) bool { return modTime(files[i]) < modTime(files[j]) })
	h := Host{Addr: addr}
	var firstErr error
	for _, f := range files {
		fh, err := os.Open(f)
		if err != nil {
			continue
		}
		hosts, err := Parse(fh)
		fh.Close()
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("%s: %w", filepath.Base(f), err)
			}
			continue
		}
		for _, x := range hosts {
			if x.Addr == addr || addr == "" || contains(x.Hostnames, addr) {
				h = Merge(h, x)
			}
		}
	}
	return h, firstErr
}

func modTime(p string) int64 {
	if fi, err := os.Stat(p); err == nil {
		return fi.ModTime().UnixNano()
	}
	return 0
}

func key(p Port) string { return fmt.Sprintf("%s/%d", p.Proto, p.Number) }

func paren(s string) string {
	if s == "" {
		return ""
	}
	return "(" + s + ")"
}

func nonEmpty(xs ...string) []string {
	var out []string
	for _, x := range xs {
		if x != "" {
			out = append(out, x)
		}
	}
	return out
}

func appendUnique(xs []string, x string) []string {
	if x == "" || contains(xs, x) {
		return xs
	}
	return append(xs, x)
}

func contains(xs []string, x string) bool {
	for _, y := range xs {
		if strings.EqualFold(x, y) {
			return true
		}
	}
	return false
}
