package handler

import "testing"

func TestParseGitHubOwnerRepo(t *testing.T) {
	cases := []struct {
		in          string
		owner, repo string
		ok          bool
	}{
		{"https://github.com/acme/widget.git", "acme", "widget", true},
		{"https://github.com/acme/widget", "acme", "widget", true},
		{"git@github.com:acme/widget.git", "acme", "widget", true},
		{"https://gitlab.com/acme/widget", "", "", false},
		{"", "", "", false},
	}
	for _, c := range cases {
		o, r, ok := parseGitHubOwnerRepo(c.in)
		if ok != c.ok || o != c.owner || r != c.repo {
			t.Fatalf("%q → (%s,%s,%v) want (%s,%s,%v)", c.in, o, r, ok, c.owner, c.repo, c.ok)
		}
	}
}
