package handler

import (
	"crypto/rand"
	"errors"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/stifer/agent-hub/internal/hub/mailer"
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
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "email and password required"})
		return
	}
	if len(req.Password) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "password must be at least 6 characters"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "hash failed"})
		return
	}

	ctx := c.Request.Context()
	var userID int64
	err = h.Svc.Pool.QueryRow(ctx,
		`INSERT INTO hub.hub_users (email, password_hash, name, email_verified_at)
		 VALUES ($1, $2, $3, NULL)
		 ON CONFLICT (email) DO NOTHING
		 RETURNING id`,
		req.Email, string(hash), req.Email,
	).Scan(&userID)

	if err != nil {
		// Email already registered — if still unverified, allow resend + password update
		var existingID int64
		var verified *time.Time
		qerr := h.Svc.Pool.QueryRow(ctx,
			`SELECT id, email_verified_at FROM hub.hub_users WHERE email = $1`,
			req.Email,
		).Scan(&existingID, &verified)
		if qerr != nil {
			c.JSON(http.StatusConflict, gin.H{"code": 409, "message": "email already registered"})
			return
		}
		if verified != nil {
			c.JSON(http.StatusConflict, gin.H{"code": 409, "message": "email already registered"})
			return
		}
		// Unverified re-register: update password and resend
		userID = existingID
		_, _ = h.Svc.Pool.Exec(ctx,
			`UPDATE hub.hub_users SET password_hash = $1 WHERE id = $2`,
			string(hash), userID,
		)
	}

	if err := h.issueAndSendVerification(c, userID, req.Email); err != nil {
		log.Printf("[register] send verification to %s: %v", req.Email, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "account created but failed to send verification email; try resend",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"email":              req.Email,
		"needs_verification": true,
		"message":            "verification email sent",
	}})
}

func (h *Handler) Login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	adminPassword := os.Getenv("HUB_ADMIN_PASSWORD")
	if (req.Email == "admin" || req.Email == "admin@stifer.xyz") && adminPassword != "" && req.Password == adminPassword {
		var adminID int64
		err := h.Svc.Pool.QueryRow(c.Request.Context(),
			"SELECT id FROM hub.hub_users WHERE email IN ('admin@stifer.xyz','admin') ORDER BY id LIMIT 1").Scan(&adminID)
		if err != nil || adminID <= 0 {
			hash, _ := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
			_ = h.Svc.Pool.QueryRow(c.Request.Context(),
				`INSERT INTO hub.hub_users (email, password_hash, name, email_verified_at)
				 VALUES ('admin@stifer.xyz', $1, 'admin', now())
				 ON CONFLICT (email) DO UPDATE SET password_hash = EXCLUDED.password_hash, email_verified_at = COALESCE(hub.hub_users.email_verified_at, now())
				 RETURNING id`, string(hash),
			).Scan(&adminID)
		} else {
			// Ensure admin is verified
			_, _ = h.Svc.Pool.Exec(c.Request.Context(),
				`UPDATE hub.hub_users SET email_verified_at = COALESCE(email_verified_at, now()) WHERE id = $1`, adminID)
		}
		if adminID <= 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "admin user unavailable"})
			return
		}
		token := h.genToken(adminID, "admin@stifer.xyz")
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"token": token, "user_id": adminID, "email": "admin@stifer.xyz", "role": "admin"}})
		return
	}

	var userID int64
	var email, hash string
	var verifiedAt *time.Time
	err := h.Svc.Pool.QueryRow(c.Request.Context(),
		"SELECT id, email, password_hash, email_verified_at FROM hub.hub_users WHERE email = $1", req.Email,
	).Scan(&userID, &email, &hash, &verifiedAt)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "invalid email or password"})
		return
	}

	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "invalid email or password"})
		return
	}

	if verifiedAt == nil {
		c.JSON(http.StatusForbidden, gin.H{
			"code":    403,
			"error":   "email_not_verified",
			"message": "email not verified; check your inbox or resend verification",
			"email":   email,
		})
		return
	}

	token := h.genToken(userID, email)
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"token": token, "user_id": userID, "email": email}})
}

// VerifyEmail marks a user verified using the raw token from the email link.
// Accepts GET ?token= or POST {"token":"..."}.
func (h *Handler) VerifyEmail(c *gin.Context) {
	raw := strings.TrimSpace(c.Query("token"))
	if raw == "" {
		var body struct {
			Token string `json:"token"`
		}
		_ = c.ShouldBindJSON(&body)
		raw = strings.TrimSpace(body.Token)
	}
	if raw == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "token required"})
		return
	}

	hash := mailer.HashToken(raw)
	ctx := c.Request.Context()

	var userID int64
	var email string
	var expiresAt time.Time
	var usedAt *time.Time
	err := h.Svc.Pool.QueryRow(ctx,
		`SELECT user_id, email, expires_at, used_at
		 FROM hub.hub_email_verifications
		 WHERE token_hash = $1`,
		hash,
	).Scan(&userID, &email, &expiresAt, &usedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid or expired token"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	if usedAt != nil {
		// Idempotent: already used → still issue token if user is verified
		var verified *time.Time
		_ = h.Svc.Pool.QueryRow(ctx,
			`SELECT email_verified_at FROM hub.hub_users WHERE id = $1`, userID,
		).Scan(&verified)
		if verified != nil {
			token := h.genToken(userID, email)
			c.JSON(http.StatusOK, gin.H{"data": gin.H{
				"token": token, "user_id": userID, "email": email, "already_verified": true,
			}})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "token already used"})
		return
	}
	if time.Now().After(expiresAt) {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "token expired; request a new verification email"})
		return
	}

	tx, err := h.Svc.Pool.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx,
		`UPDATE hub.hub_email_verifications SET used_at = now()
		 WHERE token_hash = $1 AND used_at IS NULL`, hash)
	if err != nil || tag.RowsAffected() == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "token already used"})
		return
	}
	_, err = tx.Exec(ctx,
		`UPDATE hub.hub_users SET email_verified_at = now() WHERE id = $1`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	// Invalidate other open tokens for this user
	_, _ = tx.Exec(ctx,
		`UPDATE hub.hub_email_verifications SET used_at = now()
		 WHERE user_id = $1 AND used_at IS NULL AND token_hash <> $2`, userID, hash)
	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	token := h.genToken(userID, email)
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"token": token, "user_id": userID, "email": email, "verified": true,
	}})
}

// ResendVerification sends a new verification email if the account exists and is unverified.
func (h *Handler) ResendVerification(c *gin.Context) {
	var req struct {
		Email string `json:"email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Email) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "email required"})
		return
	}
	email := strings.TrimSpace(strings.ToLower(req.Email))
	ctx := c.Request.Context()

	var userID int64
	var verified *time.Time
	err := h.Svc.Pool.QueryRow(ctx,
		`SELECT id, email_verified_at FROM hub.hub_users WHERE email = $1`, email,
	).Scan(&userID, &verified)
	// Always return the same message to avoid email enumeration
	okMsg := gin.H{"data": gin.H{"message": "if the account exists and is unverified, a new email was sent"}}
	if err != nil {
		c.JSON(http.StatusOK, okMsg)
		return
	}
	if verified != nil {
		c.JSON(http.StatusOK, okMsg)
		return
	}
	if err := h.issueAndSendVerification(c, userID, email); err != nil {
		log.Printf("[resend] %s: %v", email, err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "failed to send email"})
		return
	}
	c.JSON(http.StatusOK, okMsg)
}

func (h *Handler) issueAndSendVerification(c *gin.Context, userID int64, email string) error {
	raw, hash, err := mailer.NewToken()
	if err != nil {
		return err
	}
	ctx := c.Request.Context()
	// Invalidate previous unused tokens
	_, _ = h.Svc.Pool.Exec(ctx,
		`UPDATE hub.hub_email_verifications SET used_at = now()
		 WHERE user_id = $1 AND used_at IS NULL`, userID)

	_, err = h.Svc.Pool.Exec(ctx,
		`INSERT INTO hub.hub_email_verifications (user_id, email, token_hash, expires_at)
		 VALUES ($1, $2, $3, now() + interval '24 hours')`,
		userID, email, hash)
	if err != nil {
		return err
	}

	cfg := mailer.Load()
	subject, text, html := mailer.VerifyEmailContent(cfg.PublicURL, raw)
	return cfg.Send(email, subject, text, html)
}

func (h *Handler) OAuthAuthorizeRedirect(c *gin.Context) {
	code := randomCode(8)
	_, err := h.Svc.Pool.Exec(c.Request.Context(),
		`INSERT INTO hub.hub_device_codes (code, token, user_id, confirmed, expires_at)
		 VALUES ($1, NULL, NULL, false, now() + interval '10 minutes')`,
		code)
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
	_, err := h.Svc.Pool.Exec(c.Request.Context(),
		`INSERT INTO hub.hub_device_codes (code, token, user_id, confirmed, expires_at)
		 VALUES ($1, NULL, NULL, false, now() + interval '10 minutes')`,
		code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error", "error_description": err.Error()})
		return
	}
	verifyURL := publicBaseURL() + "/auth/device?code=" + code
	c.JSON(http.StatusOK, gin.H{
		"device_code":               code,
		"user_code":                 code,
		"verification_uri":          verifyURL,
		"verification_uri_complete": verifyURL,
		"expires_in":                600,
		"interval":                  2,
	})
}

func (h *Handler) OAuthDeviceToken(c *gin.Context) {
	ct := c.ContentType()
	grantType := c.PostForm("grant_type")
	deviceCode := c.PostForm("device_code")
	code := c.PostForm("code")

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
		}
	}

	lookupCode := deviceCode
	if grantType == "authorization_code" || grantType == "code" {
		if code == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "error_description": "code required"})
			return
		}
		lookupCode = code
	} else if grantType != "urn:ietf:params:oauth:grant-type:device_code" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported_grant_type"})
		return
	} else if deviceCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "error_description": "device_code required"})
		return
	}

	var token *string
	var confirmed bool
	err := h.Svc.Pool.QueryRow(c.Request.Context(),
		"SELECT token, confirmed FROM hub.hub_device_codes WHERE code=$1 AND expires_at > now()",
		lookupCode,
	).Scan(&token, &confirmed)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "authorization_pending", "error_description": "Code not found or expired"})
		return
	}
	if !confirmed || token == nil || *token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "authorization_pending", "error_description": "User has not yet authorized"})
		return
	}
	h.Svc.Pool.Exec(c.Request.Context(), "DELETE FROM hub.hub_device_codes WHERE code=$1", lookupCode)
	c.JSON(http.StatusOK, gin.H{"access_token": *token, "token_type": "Bearer", "expires_in": 259200})
}

func (h *Handler) genToken(userID int64, email string) string {
	if userID <= 0 {
		return ""
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  email,
		"uid":  userID,
		"role": "user",
		"iat":  time.Now().Unix(),
		"exp":  time.Now().Add(72 * time.Hour).Unix(),
	})
	t, err := token.SignedString([]byte(h.JWTSecret))
	if err != nil {
		return ""
	}
	return t
}

func (h *Handler) GetMyBusinesses(c *gin.Context) {
	userIDRaw, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "unauthorized"})
		return
	}
	userID := userIDRaw
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
		if err := rows.Scan(&b.ID, &b.Code, &b.Name, &b.Description, &b.Status, &b.Role); err != nil {
			continue
		}
		list = append(list, b)
	}
	if list == nil {
		list = []biz{}
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}

type joinBusinessReq struct {
	Token string `json:"token"`
}

func (h *Handler) JoinBusiness(c *gin.Context) {
	code := c.Param("code")
	userIDRaw, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "unauthorized"})
		return
	}
	userID, ok := userIDRaw.(int64)
	if !ok || userID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "invalid user"})
		return
	}

	var req joinBusinessReq
	_ = c.ShouldBindJSON(&req)

	var bizID int64
	var joinPolicy string
	err := h.Svc.Pool.QueryRow(c.Request.Context(),
		"SELECT id, COALESCE(join_policy, 'invite_only') FROM hub.hub_businesses WHERE code = $1", code,
	).Scan(&bizID, &joinPolicy)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "business not found"})
		return
	}

	if req.Token != "" {
		if err := h.acceptInviteToken(c, userID, req.Token); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": "joined via invite"})
		return
	}

	switch joinPolicy {
	case "open":
		_, err = h.Svc.Pool.Exec(c.Request.Context(),
			"INSERT INTO hub.hub_memberships (user_id, business_id, role) VALUES ($1, $2, 'member') ON CONFLICT DO NOTHING",
			userID, bizID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": "joined"})
	case "approval":
		c.JSON(http.StatusConflict, gin.H{
			"code":    409,
			"message": "this business requires approval; create a link request instead",
		})
	default:
		c.JSON(http.StatusConflict, gin.H{
			"code":    409,
			"message": "this business is invite-only; provide an invite token or use link-request",
		})
	}
}

func (h *Handler) DeviceAuth(c *gin.Context) {
	code := randomCode(8)
	_, err := h.Svc.Pool.Exec(c.Request.Context(),
		`INSERT INTO hub.hub_device_codes (code, token, user_id, confirmed, expires_at)
		 VALUES ($1, NULL, NULL, false, now() + interval '10 minutes')`,
		code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"code":             code,
		"verification_url": publicBaseURL() + "/auth/device?code=" + code,
		"expires_in":       600,
	}})
}

func (h *Handler) DeviceConfirm(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		var body struct {
			Code string `json:"code"`
		}
		_ = c.ShouldBindJSON(&body)
		code = body.Code
	}
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "code required"})
		return
	}

	userIDRaw, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "login required"})
		return
	}
	userID, ok := userIDRaw.(int64)
	if !ok || userID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "invalid user"})
		return
	}
	emailRaw, _ := c.Get("email")
	email, _ := emailRaw.(string)
	if email == "" {
		_ = h.Svc.Pool.QueryRow(c.Request.Context(),
			"SELECT email FROM hub.hub_users WHERE id=$1", userID).Scan(&email)
	}
	if email == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "user email missing"})
		return
	}

	token := h.genToken(userID, email)
	if token == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "token issue failed"})
		return
	}

	tag, err := h.Svc.Pool.Exec(c.Request.Context(),
		`UPDATE hub.hub_device_codes
		 SET confirmed=true, user_id=$1, token=$2
		 WHERE code=$3 AND expires_at > now() AND confirmed=false`,
		userID, token, code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "invalid, expired, or already used code"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": "confirmed"})
}

func (h *Handler) DeviceToken(c *gin.Context) {
	code := c.Query("code")
	var token *string
	var confirmed bool
	err := h.Svc.Pool.QueryRow(c.Request.Context(),
		"SELECT token, confirmed FROM hub.hub_device_codes WHERE code=$1 AND expires_at > now()", code,
	).Scan(&token, &confirmed)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "invalid code"})
		return
	}
	if !confirmed || token == nil || *token == "" {
		c.JSON(http.StatusAccepted, gin.H{"data": gin.H{"status": "pending"}})
		return
	}
	if _, err := h.Svc.Pool.Exec(c.Request.Context(), "DELETE FROM hub.hub_device_codes WHERE code=$1", code); err != nil {
		log.Printf("device code cleanup failed: %v", err)
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"token": *token}})
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

func publicBaseURL() string {
	base := strings.TrimRight(os.Getenv("HUB_PUBLIC_URL"), "/")
	if base == "" {
		base = "https://hub.stifer.xyz"
	}
	return base
}

// OAuthRegister implements RFC 7591 Dynamic Client Registration for MCP hosts (Claude).
// Clients are persisted in hub.hub_oauth_clients when the table exists; otherwise a
// stable ephemeral response is returned so hosts can still complete the flow.
func (h *Handler) OAuthRegister(c *gin.Context) {
	var req struct {
		ClientName              string   `json:"client_name"`
		RedirectURIs            []string `json:"redirect_uris"`
		GrantTypes              []string `json:"grant_types"`
		ResponseTypes           []string `json:"response_types"`
		TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	}
	_ = c.ShouldBindJSON(&req)
	if req.ClientName == "" {
		req.ClientName = "MCP Client"
	}
	if len(req.RedirectURIs) == 0 {
		req.RedirectURIs = []string{"http://localhost/callback", "http://127.0.0.1/callback"}
	}
	if len(req.GrantTypes) == 0 {
		req.GrantTypes = []string{"authorization_code", "urn:ietf:params:oauth:grant-type:device_code"}
	}
	if len(req.ResponseTypes) == 0 {
		req.ResponseTypes = []string{"code"}
	}
	if req.TokenEndpointAuthMethod == "" {
		req.TokenEndpointAuthMethod = "none"
	}

	clientID := "ah-" + strings.ToLower(randomCode(12))
	// Try persist; soft-fail if migration not applied yet
	_, err := h.Svc.Pool.Exec(c.Request.Context(),
		`INSERT INTO hub.hub_oauth_clients
		   (client_id, client_name, redirect_uris, grant_types, response_types, token_endpoint_auth_method)
		 VALUES ($1,$2,$3,$4,$5,$6)
		 ON CONFLICT (client_id) DO NOTHING`,
		clientID, req.ClientName, req.RedirectURIs, req.GrantTypes, req.ResponseTypes, req.TokenEndpointAuthMethod,
	)
	if err != nil {
		log.Printf("[OAuthRegister] persist failed (using ephemeral client): %v", err)
		clientID = "agent-hub-mcp"
	}

	c.JSON(http.StatusCreated, gin.H{
		"client_id":                  clientID,
		"client_name":                req.ClientName,
		"redirect_uris":              req.RedirectURIs,
		"grant_types":                req.GrantTypes,
		"response_types":             req.ResponseTypes,
		"token_endpoint_auth_method": req.TokenEndpointAuthMethod,
		"client_id_issued_at":        time.Now().Unix(),
	})
}
