package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/stifer/agent-hub/internal/hub/service"
)

type publishWorkerTemplateReq struct {
	WorkerID      string                 `json:"worker_id" binding:"required"`
	DisplayName   string                 `json:"display_name"`
	Description   string                 `json:"description"`
	Skills        []string               `json:"skills"`
	Handbook      map[string]interface{} `json:"handbook"`
	PromptSummary string                 `json:"prompt_summary"`
	Visibility    string                 `json:"visibility"`
}

// PublishWorkerTemplate POST /v1/hub/businesses/:code/worker-templates
func (h *Handler) PublishWorkerTemplate(c *gin.Context) {
	code := c.Param("code")
	if _, _, ok := h.RequireMembership(c, code); !ok {
		return
	}
	var req publishWorkerTemplateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	req.WorkerID = strings.TrimSpace(req.WorkerID)
	if req.WorkerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "worker_id required"})
		return
	}
	if len(req.PromptSummary) > 8000 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "prompt_summary too large"})
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
	if req.Skills == nil {
		req.Skills = []string{}
	}
	if req.Handbook == nil {
		req.Handbook = map[string]interface{}{}
	}
	hb, err := json.Marshal(req.Handbook)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid handbook"})
		return
	}
	display := strings.TrimSpace(req.DisplayName)
	if display == "" {
		display = req.WorkerID
	}

	st := stampActorFromGin(c, "leader", req.WorkerID, "")
	var bizID int64
	if err := h.Svc.Pool.QueryRow(c.Request.Context(),
		`SELECT id FROM hub.hub_businesses WHERE code=$1`, code,
	).Scan(&bizID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "business not found"})
		return
	}

	var id int64
	err = h.Svc.Pool.QueryRow(c.Request.Context(), `
		INSERT INTO hub.hub_worker_templates (
		  business_id, worker_id, display_name, description, skills, handbook, prompt_summary,
		  published_by_user_id, published_by_email, visibility, created_at, updated_at
		) VALUES (
		  $1,$2,$3,$4,$5::text[],$6::jsonb,$7,$8,$9,$10, now(), now()
		)
		ON CONFLICT (business_id, worker_id) DO UPDATE SET
		  display_name = EXCLUDED.display_name,
		  description = EXCLUDED.description,
		  skills = EXCLUDED.skills,
		  handbook = EXCLUDED.handbook,
		  prompt_summary = EXCLUDED.prompt_summary,
		  published_by_user_id = EXCLUDED.published_by_user_id,
		  published_by_email = EXCLUDED.published_by_email,
		  visibility = EXCLUDED.visibility,
		  updated_at = now()
		RETURNING id`,
		bizID, req.WorkerID, display, req.Description, req.Skills, hb, req.PromptSummary,
		userIDPtr(st.UserID), st.Email, vis,
	).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	_, _ = h.Svc.AppendEventFull(c.Request.Context(), service.AppendEventInput{
		BusinessCode: code,
		Actor:        st.LegacyActor,
		EventType:    "worker.template_published",
		Payload: map[string]interface{}{
			"worker_id":    req.WorkerID,
			"display_name": display,
		},
		ActorUserID:   userIDPtr(st.UserID),
		ActorEmail:    st.Email,
		ActorRole:     "leader",
		ActorWorkerID: req.WorkerID,
	})

	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"id":                   id,
		"worker_id":            req.WorkerID,
		"display_name":         display,
		"published_by_email":   st.Email,
		"visibility":           vis,
	}})
}

// ListWorkerTemplates GET /v1/hub/businesses/:code/worker-templates
func (h *Handler) ListWorkerTemplates(c *gin.Context) {
	code := c.Param("code")
	if _, _, ok := h.RequireMembership(c, code); !ok {
		return
	}
	st := stampActorFromGin(c, "worker", "", "")
	rows, err := h.Svc.Pool.Query(c.Request.Context(), `
		SELECT t.worker_id, COALESCE(t.display_name,''), COALESCE(t.description,''),
		       t.skills, COALESCE(t.published_by_email,''), t.visibility, t.updated_at
		FROM hub.hub_worker_templates t
		JOIN hub.hub_businesses b ON b.id = t.business_id
		WHERE b.code = $1
		  AND (t.visibility = 'team' OR t.published_by_user_id = $2)
		ORDER BY t.updated_at DESC
		LIMIT 100`, code, st.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	defer rows.Close()
	type item struct {
		WorkerID           string   `json:"worker_id"`
		DisplayName        string   `json:"display_name"`
		Description        string   `json:"description"`
		Skills             []string `json:"skills"`
		PublishedByEmail   string   `json:"published_by_email"`
		Visibility         string   `json:"visibility"`
		UpdatedAt          string   `json:"updated_at"`
	}
	var list []item
	for rows.Next() {
		var it item
		var skills []string
		var updatedAt interface{}
		if err := rows.Scan(&it.WorkerID, &it.DisplayName, &it.Description, &skills,
			&it.PublishedByEmail, &it.Visibility, &updatedAt); err != nil {
			continue
		}
		if skills == nil {
			skills = []string{}
		}
		it.Skills = skills
		it.UpdatedAt = stringifyTime(updatedAt)
		list = append(list, it)
	}
	if list == nil {
		list = []item{}
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}

// GetWorkerTemplate GET /v1/hub/businesses/:code/worker-templates/:worker_id
func (h *Handler) GetWorkerTemplate(c *gin.Context) {
	code := c.Param("code")
	wid := c.Param("worker_id")
	if _, _, ok := h.RequireMembership(c, code); !ok {
		return
	}
	st := stampActorFromGin(c, "worker", "", "")
	var (
		display, desc, prompt, pubEmail, vis string
		skills                               []string
		handbook                             []byte
		updatedAt                            interface{}
	)
	err := h.Svc.Pool.QueryRow(c.Request.Context(), `
		SELECT COALESCE(t.display_name,''), COALESCE(t.description,''), t.skills, t.handbook,
		       COALESCE(t.prompt_summary,''), COALESCE(t.published_by_email,''), t.visibility, t.updated_at
		FROM hub.hub_worker_templates t
		JOIN hub.hub_businesses b ON b.id = t.business_id
		WHERE b.code=$1 AND t.worker_id=$2
		  AND (t.visibility='team' OR t.published_by_user_id=$3)`,
		code, wid, st.UserID,
	).Scan(&display, &desc, &skills, &handbook, &prompt, &pubEmail, &vis, &updatedAt)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "template not found"})
		return
	}
	if skills == nil {
		skills = []string{}
	}
	var hb map[string]interface{}
	_ = json.Unmarshal(handbook, &hb)
	if hb == nil {
		hb = map[string]interface{}{}
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"worker_id":            wid,
		"display_name":         display,
		"description":          desc,
		"skills":               skills,
		"handbook":             hb,
		"prompt_summary":       prompt,
		"published_by_email":   pubEmail,
		"visibility":           vis,
		"updated_at":           stringifyTime(updatedAt),
	}})
}
