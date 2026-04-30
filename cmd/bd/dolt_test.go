package main

import "testing"

func TestExtractSSHHost(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"ssh://user@host.example.com:22/repo", "host.example.com"},
		{"git+ssh://user@host.example.com/repo", "host.example.com"},
		{"user@host.example.com:repo.git", "host.example.com"},
		{"host.example.com", "host.example.com"},
		{"ssh://user@[::1]:22/repo", "::1"},
		{"ssh://user@[2001:db8::1]:2222/x", "2001:db8::1"},
		{"ssh://[fe80::1]/repo", "fe80::1"},
	}
	for _, c := range cases {
		got := extractSSHHost(c.in)
		if got != c.want {
			t.Errorf("extractSSHHost(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
