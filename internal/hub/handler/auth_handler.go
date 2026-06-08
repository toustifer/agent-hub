package handler

import (
	"crypto/rand"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) Register(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if req.Email == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "email and password required"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "hash failed"})
		return
	}

	var userID int64
	err = h.Svc.Pool.QueryRow(c.Request.Context(),
		"INSERT INTO hub.hub_users (email, password_hash, name) VALUES ($1, $2, $3) ON CONFLICT (email) DO UPDATE SET password_hash=$2 RETURNING id",
		req.Email, string(hash), req.Email,
	).Scan(&userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	token := h.genToken(userID, req.Email)
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"token": token, "user_id": userID, "email": req.Email}})
}

func (h *Handler) Login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	// admin password fallback — look up real user ID for admin
	adminPassword := os.Getenv("HUB_ADMIN_PASSWORD")
	if req.Email == "admin" && adminPassword != "" && req.Password == adminPassword {
		var adminID int64
		h.Svc.Pool.QueryRow(c.Request.Context(), "SELECT id FROM hub.hub_users WHERE email='admin@stifer.xyz' LIMIT 1").Scan(&adminID)
		token := h.genToken(adminID, "admin")
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"token": token, "user_id": adminID, "email": "admin", "role": "admin"}})
		return
	}

	var userID int64
	var email, hash string
	err := h.Svc.Pool.QueryRow(c.Request.Context(),
		"SELECT id, email, password_hash FROM hub.hub_users WHERE email = $1", req.Email,
	).Scan(&userID, &email, &hash)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "invalid email or password"})
		return
	}

	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "invalid email or password"})
		return
	}

	token := h.genToken(userID, email)
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"token": token, "user_id": userID, "email": email}})
}

func (h *Handler) OAuthAuthorizeRedirect(c *gin.Context) {
	// Authorization code flow: generate a code, store it, redirect to approval page.
	code := randomCode(8)
	token := h.genToken(0, "device")
	_, err := h.Svc.Pool.Exec(c.Request.Context(),
		"INSERT INTO hub.hub_device_codes (code, token, expires_at) VALUES ($1, $2, now() + interval '10 minutes')",
		code, token)
	if err != nil {
		c.JSON(500, gin.H{"error": "server_error"})
		return
	}
	dest := "/auth/device?code=" + code
	if r := c.Query("redirect_uri"); r != "" {
		dest += "&redirect_uri=" + url.QueryEscape(r)
	}
	if s := c.Query("state"); s != "" {
		dest += "&state=" + url.QueryEscape(s)
	}
	c.Redirect(http.StatusFound, dest)
}

func (h *Handler) OAuthDeviceAuthorize(c *gin.Context) {
	code := randomCode(8)
	token := h.genToken(0, "device")
	_, err := h.Svc.Pool.Exec(c.Request.Context(),
		"INSERT INTO hub.hub_device_codes (code, token, expires_at) VALUES ($1, $2, now() + interval '10 minutes')",
		code, token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error", "error_description": err.Error()})
		return
	}
	verifyURL := "https://hub.stifer.xyz/auth/device?code=" + code
	c.JSON(http.StatusOK, gin.H{
		"device_code":               code,
		"user_code":                  code,
		"verification_uri":           verifyURL,
		"verification_uri_complete":  verifyURL,
		"expires_in":                 600,
		"interval":                   2,
	})
}

func (h *Handler) OAuthDeviceToken(c *gin.Context) {
	// Accept both form-urlencoded (OAuth standard) and JSON
	ct := c.ContentType()
	grantType := c.PostForm("grant_type")
	deviceCode := c.PostForm("device_code")
	code := c.PostForm("code")

	// Log for debugging
	log.Printf("[OAuth Token] Content-Type=%s grant_type=%s code=%s device_code=%s",
		ct, grantType, code, deviceCode)

	if grantType == "" {
		var req struct {
			GrantType  string `json:"grant_type"`
			DeviceCode string `json:"device_code"`
			Code       string `json:"code"`
		}
		if err := c.ShouldBindJSON(&req); err == nil {
			grantType = req.GrantType
			deviceCode = req.DeviceCode
			code = req.Code
			log.Printf("[OAuth Token] JSON fallback: grant_type=%s code=%s", grantType, code)
		}
	}

	// Support both device_code and authorization_code grant types
	if grantType == "authorization_code" || grantType == "code" {
		if code == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "error_description": "code required"})
			return
		}
		var token string
		var confirmed bool
		err := h.Svc.Pool.QueryRow(c.Request.Context(),
			"SELECT token, confirmed FROM hub.hub_device_codes WHERE code=$1 AND expires_at > now()",
			code,
		).Scan(&token, &confirmed)
		if err != nil || !confirmed {
			c.JSON(http.StatusBadRequest, gin.H{"error": "authorization_pending", "error_description": "Code not yet authorized"})
			return
		}
		h.Svc.Pool.Exec(c.Request.Context(), "DELETE FROM hub.hub_device_codes WHERE code=$1", code)
		c.JSON(http.StatusOK, gin.H{"access_token": token, "token_type": "Bearer", "expires_in": 259200})
		return
	}

	if grantType != "urn:ietf:params:oauth:grant-type:device_code" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported_grant_type"})
		return
	}
	var token string
	var confirmed bool
	err := h.Svc.Pool.QueryRow(c.Request.Context(),
		"SELECT token, confirmed FROM hub.hub_device_codes WHERE code=$1 AND expires_at > now()",
		deviceCode,
	).Scan(&token, &confirmed)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "authorization_pending", "error_description": "Code not found or expired"})
		return
	}
	if !confirmed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "authorization_pending", "error_description": "User has not yet authorized"})
		return
	}
	h.Svc.Pool.Exec(c.Request.Context(), "DELETE FROM hub.hub_device_codes WHERE code=$1", deviceCode)
	c.JSON(http.StatusOK, gin.H{"access_token": token, "token_type": "Bearer", "expires_in": 259200})
}

func (h *Handler) genToken(userID int64, email string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   email,
		"uid":   userID,
		"role":  "user",
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(72 * time.Hour).Unix(),
	})
	t, err := token.SignedString([]byte(h.JWTSecret))
		if err != nil {
			return ""
		}
	return t
}

func (h *Handler) GetMyBusinesses(c *gin.Context) {
	userIDRaw, ok := c.Get("user_id"); if !ok { c.JSON(http.StatusUnauthorized, gin.H{"code":401,"message":"unauthorized"}); return }; userID := userIDRaw
	rows, err := h.Svc.Pool.Query(c.Request.Context(),
		`SELECT b.id, b.code, b.name, b.description, b.status, m.role
		 FROM hub.hub_businesses b
		 JOIN hub.hub_memberships m ON m.business_id = b.id
		 WHERE m.user_id = $1`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	defer rows.Close()

	type biz struct {
		ID          int64  `json:"id"`
		Code        string `json:"code"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Status      string `json:"status"`
		Role        string `json:"role"`
	}
	var list []biz
	for rows.Next() {
		var b biz
		if err := rows.Scan(&b.ID, &b.Code, &b.Name, &b.Description, &b.Status, &b.Role); err != nil { continue }
		list = append(list, b)
	}
	if list == nil {
		list = []biz{}
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}

func (h *Handler) JoinBusiness(c *gin.Context) {
	code := c.Param("code")
	userIDRaw, ok := c.Get("user_id"); if !ok { c.JSON(http.StatusUnauthorized, gin.H{"code":401,"message":"unauthorized"}); return }; userID := userIDRaw

	var bizID int64
	err := h.Svc.Pool.QueryRow(c.Request.Context(),
		"SELECT id FROM hub.hub_businesses WHERE code = $1", code,
	).Scan(&bizID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "business not found"})
		return
	}

	_, err = h.Svc.Pool.Exec(c.Request.Context(),
		"INSERT INTO hub.hub_memberships (user_id, business_id, role) VALUES ($1, $2, 'member') ON CONFLICT DO NOTHING",
		userID, bizID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": "joined"})
}

func (h *Handler) DeviceAuth(c *gin.Context) {
	code := randomCode(8)
	token := h.genToken(0, "device")
	_, err := h.Svc.Pool.Exec(c.Request.Context(),
		"INSERT INTO hub.hub_device_codes (code, token, expires_at) VALUES ($1, $2, now() + interval '10 minutes')",
		code, token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"code":             code,
		"verification_url": "https://hub.stifer.xyz/auth/device?code=" + code,
		"expires_in":       600,
	}})
}

func (h *Handler) DeviceConfirm(c *gin.Context) {
	code := c.Query("code")
	_, err := h.Svc.Pool.Exec(c.Request.Context(),
		"UPDATE hub.hub_device_codes SET confirmed=true WHERE code=$1 AND expires_at > now()", code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": "confirmed"})
}

func (h *Handler) DeviceToken(c *gin.Context) {
	code := c.Query("code")
	var token string
	var confirmed bool
	err := h.Svc.Pool.QueryRow(c.Request.Context(),
		"SELECT token, confirmed FROM hub.hub_device_codes WHERE code=$1 AND expires_at > now()", code,
	).Scan(&token, &confirmed)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "invalid code"})
		return
	}
	if !confirmed {
		c.JSON(http.StatusAccepted, gin.H{"data": gin.H{"status": "pending"}})
		return
	}
	if _, err := h.Svc.Pool.Exec(c.Request.Context(), "DELETE FROM hub.hub_device_codes WHERE code=$1", code); err != nil { log.Printf("device code cleanup failed: %v", err) }
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"token": token}})
}

func randomCode(n int) string {
	const letters = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, n)
	rand.Read(b)
	for i := range b {
		b[i] = letters[int(b[i])%len(letters)]
	}
	return string(b)
}
