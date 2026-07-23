package handler

import (
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

func TestGenTokenRejectsNonPositiveUID(t *testing.T) {
	h := &Handler{JWTSecret: "unit-test-secret"}
	if tok := h.genToken(0, "device"); tok != "" {
		t.Fatalf("genToken(0) must return empty, got %q", tok)
	}
	if tok := h.genToken(-1, "x"); tok != "" {
		t.Fatalf("genToken(-1) must return empty, got %q", tok)
	}
}

func TestGenTokenBindsRealUser(t *testing.T) {
	h := &Handler{JWTSecret: "unit-test-secret"}
	tok := h.genToken(7, "user@example.com")
	if tok == "" {
		t.Fatal("expected non-empty token")
	}
	parsed, err := jwt.Parse(tok, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			t.Fatalf("unexpected method %v", token.Method)
		}
		return []byte(h.JWTSecret), nil
	})
	if err != nil || !parsed.Valid {
		t.Fatalf("parse: %v", err)
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatal("claims type")
	}
	uid, _ := claims["uid"].(float64)
	if int64(uid) != 7 {
		t.Fatalf("uid want 7 got %v", claims["uid"])
	}
	if claims["sub"] != "user@example.com" {
		t.Fatalf("sub want user@example.com got %v", claims["sub"])
	}
}

func TestHashTokenDeterministic(t *testing.T) {
	a := hashToken("raw-invite-token")
	b := hashToken("raw-invite-token")
	c := hashToken("other")
	if a != b {
		t.Fatal("hashToken not stable")
	}
	if a == c {
		t.Fatal("different inputs should hash differently")
	}
	if len(a) != 64 {
		t.Fatalf("sha256 hex length want 64 got %d", len(a))
	}
}

func TestGenInviteTokenLength(t *testing.T) {
	tok, err := genInviteToken()
	if err != nil {
		t.Fatal(err)
	}
	if len(tok) != 64 { // 32 bytes hex
		t.Fatalf("invite token len want 64 got %d", len(tok))
	}
}
