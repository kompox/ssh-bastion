package forwarding

import "testing"

func TestValidateRule(t *testing.T) {
	cases := []struct {
		rule string
		ok   bool
	}{
		{"db.example.com:5432", true},
		{"10.0.0.10:22", true},
		{"[fd00::1]:5432", true},
		{"*.example.com:5432", true},
		{"db.example.com:*", true},
		{"*:*", true},
		{"any", false},
		{"none", false},
		{"", false},
		{"db.example.com", false},
		{"db.example.com:0", false},
		{"db.example.com:70000", false},
		{"[fd00::1]5432", false},
		{"fd00::1:5432", false},
		{"db.example.com:54 32", false},
		{"db.example.com:5432#x", false},
	}

	for _, tc := range cases {
		err := ValidateRule(tc.rule)
		if tc.ok && err != nil {
			t.Fatalf("expected ok for %q; got err=%v", tc.rule, err)
		}
		if !tc.ok && err == nil {
			t.Fatalf("expected error for %q", tc.rule)
		}
	}
}
