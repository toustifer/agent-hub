package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/stifer/agent-hub/internal/hub/service"
)

const maxTeamDocBytes = 64 * 1024

type upsertDocReq struct {
	DocKey     string `json:"doc_key" binding:"required"`
	Title      string `json:"title" binding:"required"`
	Content    string `json:"content"`
	Category   string `json:"category"`
	Visibility string `json:"visibility"`
	Source     string `json:"source"`
}

// UpsertTeamDoc POST /v1/hub/businesses/:code/docs
func (h *Handler) UpsertTeamDoc(c *gin.Context) {
	code := c.Param("code")
	if _, _, ok := h.RequireMembership(c, code); !ok {
		return
	}
	var req upsertDocReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	req.DocKey = strings.TrimSpace(req.DocKey)
	req.Title = strings.TrimSpace(req.Title)
	if req.DocKey == "" || req.Title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "doc_key and title required"})
		return
	}
	if len(req.Content) > maxTeamDocBytes {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "content too large (max 64KB)"})
		return
	}
	vis := strings.ToLower(strings.TrimSpace(req.Visibility))
	if vis == "" {
		vis = "team"
	}
	if vis != "team" && vis != "private" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "visibility must be team or private"})
		return
	}
	cat := strings.TrimSpace(req.Category)
	if cat == "" {
		cat = "general"
	}
	src := strings.TrimSpace(req.Source)
	if src == "" {
		src = "api"
	}

	st := stampActorFromGin(c, "worker", "", "")
	var bizID int64
	if err := h.Svc.Pool.QueryRow(c.Request.Context(),
		`SELECT id FROM hub.hub_businesses WHERE code=$1`, code,
	).Scan(&bizID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "business not found"})
		return
	}

	var id int64
	err := h.Svc.Pool.QueryRow(c.Request.Context(), `
		INSERT INTO hub.hub_team_docs (
		  business_id, doc_key, title, content, category, visibility,
		  created_by_user_id, updated_by_user_id, created_by_email, updated_by_email, source,
		  created_at, updated_at
		) VALUES (
		  $1,$2,$3,$4,$5,$6,$7,$7,$8,$8,$9, now(), now()
		)
		ON CONFLICT (business_id, doc_key) DO UPDATE SET
		  title = EXCLUDED.title,
		  content = EXCLUDED.content,
		  category = EXCLUDED.category,
		  visibility = EXCLUDED.visibility,
		  updated_by_user_id = EXCLUDED.updated_by_user_id,
		  updated_by_email = EXCLUDED.updated_by_email,
		  source = EXCLUDED.source,
		  updated_at = now()
		RETURNING id`,
		bizID, req.DocKey, req.Title, req.Content, cat, vis,
		userIDPtr(st.UserID), st.Email, src,
	).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	// short activity event (no full content)
	_, _ = h.Svc.AppendEventFull(c.Request.Context(), service.AppendEventInput{
		BusinessCode: code,
		Actor:        st.LegacyActor,
		EventType:    "doc.updated",
		Payload: map[string]interface{}{
			"doc_key": req.DocKey,
			"title":   req.Title,
		},
		ActorUserID: userIDPtr(st.UserID),
		ActorEmail:  st.Email,
		ActorRole:   "worker",
	})

	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"id":               id,
		"doc_key":          req.DocKey,
		"title":            req.Title,
		"visibility":       vis,
		"updated_by_email": st.Email,
	}})
}

// ListTeamDocs GET /v1/hub/businesses/:code/docs
func (h *Handler) ListTeamDocs(c *gin.Context) {
	code := c.Param("code")
	if _, _, ok := h.RequireMembership(c, code); !ok {
		return
	}
	st := stampActorFromGin(c, "worker", "", "")
	rows, err := h.Svc.Pool.Query(c.Request.Context(), `
		SELECT d.doc_key, d.title, d.category, d.visibility,
		       COALESCE(d.updated_by_email,''), d.updated_at, length(d.content)
		FROM hub.hub_team_docs d
		JOIN hub.hub_businesses b ON b.id = d.business_id
		WHERE b.code = $1
		  AND (d.visibility = 'team' OR d.created_by_user_id = $2)
		ORDER BY d.updated_at DESC
		LIMIT 100`, code, st.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	defer rows.Close()
	type item struct {
		DocKey         string `json:"doc_key"`
		Title          string `json:"title"`
		Category       string `json:"category"`
		Visibility     string `json:"visibility"`
		UpdatedByEmail string `json:"updated_by_email"`
		UpdatedAt      string `json:"updated_at"`
		ContentBytes   int    `json:"content_bytes"`
	}
	var list []item
	for rows.Next() {
		var it item
		var updatedAt interface{}
		if err := rows.Scan(&it.DocKey, &it.Title, &it.Category, &it.Visibility, &it.UpdatedByEmail, &updatedAt, &it.ContentBytes); err != nil {
			continue
		}
		it.UpdatedAt = stringifyTime(updatedAt)
		list = append(list, it)
	}
	if list == nil {
		list = []item{}
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}

// GetTeamDoc GET /v1/hub/businesses/:code/docs/:key
func (h *Handler) GetTeamDoc(c *gin.Context) {
	code := c.Param("code")
	key := c.Param("key")
	if _, _, ok := h.RequireMembership(c, code); !ok {
		return
	}
	st := stampActorFromGin(c, "worker", "", "")
	var (
		title, content, category, visibility string
		updatedBy, createdBy                 string
		updatedAt                            interface{}
	)
	err := h.Svc.Pool.QueryRow(c.Request.Context(), `
		SELECT d.title, d.content, d.category, d.visibility,
		       COALESCE(d.updated_by_email,''), COALESCE(d.created_by_email,''), d.updated_at
		FROM hub.hub_team_docs d
		JOIN hub.hub_businesses b ON b.id = d.business_id
		WHERE b.code=$1 AND d.doc_key=$2
		  AND (d.visibility='team' OR d.created_by_user_id=$3)`,
		code, key, st.UserID,
	).Scan(&title, &content, &category, &visibility, &updatedBy, &createdBy, &updatedAt)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "doc not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"doc_key":            key,
		"title":              title,
		"content":            content,
		"category":           category,
		"visibility":         visibility,
		"updated_by_email":   updatedBy,
		"created_by_email":   createdBy,
		"updated_at":         stringifyTime(updatedAt),
	}})
}

func stringifyTime(v interface{}) string {
	switch t := v.(type) {
	case interface{ Format(string) string }:
		return t.Format("2006-01-02T15:04:05Z07:00")
	default:
		return ""
	}
}
