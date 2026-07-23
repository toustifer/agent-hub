package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func signToken(t *testing.T, secret string, claims jwt.MapClaims) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := tok.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return s
}

func TestRejectUidZeroJWT(t *testing.T) {
	secret := "test-secret"
	mw := &Middleware{}
	r := gin.New()
	r.GET("/protected", mw.JWT(secret), func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	token := signToken(t, secret, jwt.MapClaims{
		"sub": "device",
		"uid": 0,
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for uid=0, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestAcceptPositiveUidJWT(t *testing.T) {
	secret := "test-secret"
	mw := &Middleware{}
	var gotUID int64
	r := gin.New()
	r.GET("/protected", mw.JWT(secret), func(c *gin.Context) {
		if v, ok := c.Get("user_id"); ok {
			gotUID = v.(int64)
		}
		c.JSON(200, gin.H{"ok": true})
	})

	token := signToken(t, secret, jwt.MapClaims{
		"sub": "a@example.com",
		"uid": float64(42),
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if gotUID != 42 {
		t.Fatalf("user_id want 42 got %d", gotUID)
	}
}

func TestRejectMissingToken(t *testing.T) {
	mw := &Middleware{}
	r := gin.New()
	r.GET("/protected", mw.JWT("sec"), func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestRejectWrongAlg(t *testing.T) {
	secret := "test-secret"
	// Craft a token claiming none — Parse with method check should fail
	tok := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"sub": "x",
		"uid": 1,
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	s, err := tok.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Skip("none alg unavailable")
	}
	mw := &Middleware{}
	r := gin.New()
	r.GET("/protected", mw.JWT(secret), func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+s)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for none alg, got %d", w.Code)
	}
}
