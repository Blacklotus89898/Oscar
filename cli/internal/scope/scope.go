// Package scope decides whether a host is in an engagement's rules of
// engagement. Every action oscar runs against a target goes through Check,
// so a typo in an IP can't send a scan to someone else's network.
//
// Scope lines (scope.txt, one per line, # comments):
//
//	10.10.10.0/24          CIDR
//	192.168.50.10          single address
//	10.11.1.5-20           last-octet range
//	app.example.com        hostname (exact, case-insensitive)
//	*.corp.example.com     any subdomain
//	!10.10.10.1            exclusion: always wins (gateways, out-of-scope hosts)
package scope

import (
	"bufio"
	"fmt"
	"io"
	"net/netip"
	"strconv"
	"strings"
)

type rule struct {
	raw    string
	prefix netip.Prefix // CIDR or single address (/32, /128)
	lo, hi netip.Addr   // range
	host   string       // exact hostname, lower case
	suffix string       // "*.x" → ".x"
}

func (r rule) match(host string) bool {
	if a, err := netip.ParseAddr(host); err == nil {
		a = a.Unmap()
		switch {
		case r.prefix.IsValid():
			return r.prefix.Contains(a)
		case r.lo.IsValid():
			return a.Compare(r.lo) >= 0 && a.Compare(r.hi) <= 0
		}
		return false
	}
	h := strings.TrimSuffix(strings.ToLower(host), ".")
	switch {
	case r.host != "":
		return h == r.host
	case r.suffix != "":
		return strings.HasSuffix(h, r.suffix)
	}
	return false
}

// Scope is a parsed rules-of-engagement list.
type Scope struct {
	include []rule
	exclude []rule
}

// Parse reads scope lines; it rejects anything it can't interpret rather than
// guessing, because a silently dropped line widens or narrows the scope.
func Parse(r io.Reader) (*Scope, error) {
	s := &Scope{}
	sc := bufio.NewScanner(r)
	n := 0
	for sc.Scan() {
		n++
		line := sc.Text()
		if i := strings.IndexByte(line, '#'); i >= 0 {
			line = line[:i]
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		neg := strings.HasPrefix(line, "!")
		line = strings.TrimSpace(strings.TrimPrefix(line, "!"))
		ru, err := parseRule(line)
		if err != nil {
			return nil, fmt.Errorf("scope line %d %q: %w", n, line, err)
		}
		if neg {
			s.exclude = append(s.exclude, ru)
		} else {
			s.include = append(s.include, ru)
		}
	}
	return s, sc.Err()
}

// ParseString is Parse for inline scope (tests, `oscar new --scope a,b`).
func ParseString(s string) (*Scope, error) {
	return Parse(strings.NewReader(strings.ReplaceAll(s, ",", "\n")))
}

func parseRule(s string) (rule, error) {
	r := rule{raw: s}
	if strings.Contains(s, "/") {
		p, err := netip.ParsePrefix(s)
		if err != nil {
			return r, err
		}
		r.prefix = p.Masked()
		return r, nil
	}
	if a, err := netip.ParseAddr(s); err == nil {
		r.prefix = netip.PrefixFrom(a.Unmap(), a.Unmap().BitLen())
		return r, nil
	}
	if i := strings.LastIndexByte(s, '-'); i > 0 {
		if lo, err := netip.ParseAddr(s[:i]); err == nil && lo.Is4() {
			end, err := strconv.Atoi(s[i+1:])
			b := lo.As4()
			if err != nil || end < int(b[3]) || end > 255 {
				return r, fmt.Errorf("range end must be %d-255", b[3])
			}
			b[3] = byte(end)
			r.lo, r.hi = lo, netip.AddrFrom4(b)
			return r, nil
		}
	}
	h := strings.TrimSuffix(strings.ToLower(s), ".")
	if strings.HasPrefix(h, "*.") {
		if !validHost(h[2:]) {
			return r, fmt.Errorf("not a hostname")
		}
		r.suffix = h[1:]
		return r, nil
	}
	if !validHost(h) {
		return r, fmt.Errorf("not an address, CIDR, range or hostname")
	}
	r.host = h
	return r, nil
}

func validHost(h string) bool {
	if h == "" || len(h) > 253 {
		return false
	}
	for _, label := range strings.Split(h, ".") {
		if label == "" || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, c := range label {
			if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
				return false
			}
		}
	}
	return true
}

// Check returns nil if host is in scope, or an error saying why not.
func (s *Scope) Check(host string) error {
	host = strings.TrimSpace(host)
	if host == "" {
		return fmt.Errorf("no target")
	}
	for _, r := range s.exclude {
		if r.match(host) {
			return fmt.Errorf("%s is excluded by scope rule !%s", host, r.raw)
		}
	}
	for _, r := range s.include {
		if r.match(host) {
			return nil
		}
	}
	if len(s.include) == 0 {
		return fmt.Errorf("scope is empty: add targets to scope.txt first")
	}
	return fmt.Errorf("%s is OUT OF SCOPE (not in scope.txt)", host)
}

// Contains reports whether host is in scope.
func (s *Scope) Contains(host string) bool { return s.Check(host) == nil }

// Rules lists the scope as written, exclusions prefixed with "!".
func (s *Scope) Rules() []string {
	out := make([]string, 0, len(s.include)+len(s.exclude))
	for _, r := range s.include {
		out = append(out, r.raw)
	}
	for _, r := range s.exclude {
		out = append(out, "!"+r.raw)
	}
	return out
}
