package scope

import "testing"

func TestCheck(t *testing.T) {
	s, err := ParseString("10.10.10.0/24\n!10.10.10.1 # gateway\n192.168.50.10\n10.11.1.5-20\napp.example.com\n*.corp.example.com\nfd00::/64")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		host string
		in   bool
	}{
		{"10.10.10.5", true},
		{"10.10.10.1", false}, // excluded
		{"10.10.11.5", false},
		{"192.168.50.10", true},
		{"192.168.50.11", false},
		{"10.11.1.5", true},
		{"10.11.1.20", true},
		{"10.11.1.21", false},
		{"10.11.1.4", false},
		{"APP.example.com", true},
		{"app.example.com.", true},
		{"evil-app.example.com", false},
		{"dc01.corp.example.com", true},
		{"corp.example.com", false},
		{"fd00::5", true},
		{"::ffff:10.10.10.7", true}, // v4-mapped
		{"", false},
	}
	for _, c := range cases {
		if got := s.Contains(c.host); got != c.in {
			t.Errorf("Contains(%q) = %v, want %v (%v)", c.host, got, c.in, s.Check(c.host))
		}
	}
}

func TestParseRejectsGarbage(t *testing.T) {
	for _, bad := range []string{"10.10.10.0/33", "10.0.0.9-3", "10.0.0.1-300", "not a host!", "-bad.com", "*."} {
		if _, err := ParseString(bad); err == nil {
			t.Errorf("ParseString(%q) accepted", bad)
		}
	}
}

func TestEmptyScopeRefusesEverything(t *testing.T) {
	s, _ := ParseString("# nothing yet\n")
	if err := s.Check("10.0.0.1"); err == nil {
		t.Fatal("empty scope allowed a host")
	}
}

func TestRules(t *testing.T) {
	s, _ := ParseString("10.0.0.0/8,!10.0.0.1")
	got := s.Rules()
	if len(got) != 2 || got[0] != "10.0.0.0/8" || got[1] != "!10.0.0.1" {
		t.Fatalf("Rules() = %v", got)
	}
}
