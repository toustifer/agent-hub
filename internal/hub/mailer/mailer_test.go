package mailer

import "testing"

func TestNewTokenAndHash(t *testing.T) {
	raw, hash, err := NewToken()
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) != 64 {
		t.Fatalf("raw len want 64 got %d", len(raw))
	}
	if len(hash) != 64 {
		t.Fatalf("hash len want 64 got %d", len(hash))
	}
	if HashToken(raw) != hash {
		t.Fatal("HashToken mismatch")
	}
	if HashToken("other") == hash {
		t.Fatal("different inputs must not collide")
	}
}

func TestVerifyEmailContent(t *testing.T) {
	sub, text, html := VerifyEmailContent("https://hub.stifer.xyz", "abc")
	if sub == "" || text == "" || html == "" {
		t.Fatal("empty content")
	}
	if !contains(text, "https://hub.stifer.xyz/verify-email?token=abc") {
		t.Fatalf("text missing link: %s", text)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		(func() bool {
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		})())
}
