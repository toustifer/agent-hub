package handler

import "testing"

func TestNormalizeActorRole(t *testing.T) {
	cases := map[string]string{
		"leader":   "leader",
		"Worker":   "worker",
		"REVIEWER": "reviewer",
		"system":   "system",
		"":         "worker",
		"nope":     "worker",
	}
	for in, want := range cases {
		if got := normalizeActorRole(in); got != want {
			t.Fatalf("normalizeActorRole(%q)=%q want %q", in, got, want)
		}
	}
}
