package gameteams

import "testing"

func TestNormalizeIPAddress(t *testing.T) {
	ptr := func(s string) *string { return &s }

	valid := map[string]string{
		"ipv4":            "10.10.1.10",
		"ipv6":            "fe80::1",
		"hostname":        "team1.vuln.ctf",
		"single label":    "vulnbox",
		"trims surround":  " 10.10.7.7 ",
		"empty clears":    "",
		"spaces to empty": "   ",
	}
	want := map[string]string{
		"trims surround":  "10.10.7.7",
		"spaces to empty": "",
	}
	for name, in := range valid {
		t.Run("valid/"+name, func(t *testing.T) {
			got, err := normalizeIPAddress(ptr(in))
			if err != nil {
				t.Fatalf("normalizeIPAddress(%q) unexpected error: %v", in, err)
			}
			if got == nil {
				t.Fatalf("normalizeIPAddress(%q) returned nil", in)
			}
			exp, ok := want[name]
			if !ok {
				exp = in
			}
			if *got != exp {
				t.Errorf("normalizeIPAddress(%q) = %q, want %q", in, *got, exp)
			}
		})
	}

	invalid := map[string]string{
		"cyrillic":     "13а3у",
		"octet range":  "10.10.1.999",
		"space inside": "bad host",
		"bang":         "bad!host",
		"underscore":   "team_1",
	}
	for name, in := range invalid {
		t.Run("invalid/"+name, func(t *testing.T) {
			if _, err := normalizeIPAddress(ptr(in)); err == nil {
				t.Errorf("normalizeIPAddress(%q) expected validation error, got nil", in)
			}
		})
	}

	t.Run("nil stays nil", func(t *testing.T) {
		got, err := normalizeIPAddress(nil)
		if err != nil || got != nil {
			t.Errorf("normalizeIPAddress(nil) = %v, %v; want nil, nil", got, err)
		}
	})
}
