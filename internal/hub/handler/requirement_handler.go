package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type createRequirementReq struct {
	Title       string `json:"title" binding:"required,max=512"`
	Description string `json:"description"`
}

type requirementTransitionMeta struct {
	submitted bool
	accepted  bool
	decision  string
}

func requirementTransitionMetadata(status string) requirementTransitionMeta {
	switch status {
	case "submitted":
		return requirementTransitionMeta{submitted: true}
	case "accepted":
		return requirementTransitionMeta{accepted: true, decision: "accept"}
	case "rejected":
		return requirementTransitionMeta{decision: "reject"}
	default:
		return requirementTransitionMeta{}
	}
}

func canManageRequirement(role string) bool {
	return role == "admin" || role == "owner" || role == "leader"
}

func canCancelRequirement(role string) bool {
	return canManageRequirement(role)
}

func (h *Handler) CreateRequirement(c *gin.Context) {
	code := c.Param("code")
	bizID, _, ok := h.RequireMembership(c, code)
	if !ok {
		return
	}
	var req createRequirementReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	st := stampActorFromGin(c, "member", "", "")
	var id int64
	err := h.Svc.Pool.QueryRow(c.Request.Context(),
		`INSERT INTO hub.hub_requirements (business_id, title, description, status, created_by_user_id, created_by_email, created_at, updated_at)
		 VALUES ($1,$2,$3,'draft',$4,NULLIF($5,''),now(),now()) RETURNING id`,
		bizID, req.Title, req.Description, userIDPtr(st.UserID), st.Email,
	).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"id": id}})
}

type requirementItem struct {
	ID             int64      `json:"id"`
	Title          string     `json:"title"`
	Description    string     `json:"description"`
	Status         string     `json:"status"`
	CreatedByEmail string     `json:"created_by_email"`
	SubmittedAt    *time.Time `json:"submitted_at,omitempty"`
	AcceptedAt     *time.Time `json:"accepted_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	TaskCount      int        `json:"task_count"`
	TasksDone      int        `json:"tasks_done"`
	TaskIDs        []string   `json:"task_ids,omitempty"`
}

func (h *Handler) ListRequirements(c *gin.Context) {
	code := c.Param("code")
	bizID, _, ok := h.RequireMembership(c, code)
	if !ok {
		return
	}
	statusFilter := strings.TrimSpace(c.Query("status"))

	query := `SELECT r.id, r.title, r.description, r.status,
		COALESCE(NULLIF(r.created_by_email,''),''),
		r.submitted_at, r.accepted_at, r.created_at, r.updated_at,
		(SELECT count(*) FROM hub.hub_requirement_links l WHERE l.business_id = r.business_id AND l.requirement_id = r.id) AS task_count,
		(SELECT count(*) FROM hub.hub_requirement_links l
			JOIN hub.hub_dag_state d ON d.business_id=l.business_id AND d.task_id=l.task_id
			WHERE l.business_id = r.business_id AND l.requirement_id = r.id AND d.status = 'completed') AS tasks_done
		,	COALESCE((SELECT array_agg(l.task_id ORDER BY l.task_id) FROM hub.hub_requirement_links l WHERE l.business_id=r.business_id AND l.requirement_id=r.id), '{}') AS task_ids
	 FROM hub.hub_requirements r
	 WHERE r.business_id = $1`
	args := []interface{}{bizID}
	if statusFilter != "" {
		query += ` AND r.status = $2`
		args = append(args, statusFilter)
	}
	query += ` ORDER BY r.updated_at DESC`

	rows, err := h.Svc.Pool.Query(c.Request.Context(), query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	defer rows.Close()

	var list []requirementItem
	for rows.Next() {
		var item requirementItem
		if err := rows.Scan(&item.ID, &item.Title, &item.Description, &item.Status,
			&item.CreatedByEmail, &item.SubmittedAt, &item.AcceptedAt,
			&item.CreatedAt, &item.UpdatedAt, &item.TaskCount, &item.TasksDone, &item.TaskIDs); err != nil {
			continue
		}
		list = append(list, item)
	}
	if list == nil {
		list = []requirementItem{}
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}

func (h *Handler) GetRequirement(c *gin.Context) {
	code := c.Param("code")
	bizID, _, ok := h.RequireMembership(c, code)
	if !ok {
		return
	}
	id := c.Param("id")
	var item requirementItem
	err := h.Svc.Pool.QueryRow(c.Request.Context(),
		`SELECT r.id, r.title, r.description, r.status,
			COALESCE(NULLIF(r.created_by_email,''),''),
			r.submitted_at, r.accepted_at, r.created_at, r.updated_at,
			(SELECT count(*) FROM hub.hub_requirement_links l WHERE l.business_id = r.business_id AND l.requirement_id = r.id) AS task_count,
			(SELECT count(*) FROM hub.hub_requirement_links l
				JOIN hub.hub_dag_state d ON d.business_id=l.business_id AND d.task_id=l.task_id
				WHERE l.business_id = r.business_id AND l.requirement_id = r.id AND d.status = 'completed') AS tasks_done
			,COALESCE((SELECT array_agg(l.task_id ORDER BY l.task_id) FROM hub.hub_requirement_links l WHERE l.business_id=r.business_id AND l.requirement_id=r.id), '{}') AS task_ids
		 FROM hub.hub_requirements r
		 WHERE r.id = $1 AND r.business_id = $2`, id, bizID,
	).Scan(&item.ID, &item.Title, &item.Description, &item.Status,
		&item.CreatedByEmail, &item.SubmittedAt, &item.AcceptedAt,
		&item.CreatedAt, &item.UpdatedAt, &item.TaskCount, &item.TasksDone, &item.TaskIDs)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "requirement not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": item})
}

type updateRequirementReq struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
}

func (h *Handler) UpdateRequirement(c *gin.Context) {
	code := c.Param("code")
	bizID, role, ok := h.RequireMembership(c, code)
	if !ok {
		return
	}
	id := c.Param("id")
	st := stampActorFromGin(c, "member", "", "")

	// Only author (by email) or admin may edit
	var creatorEmail string
	var currentStatus string
	err := h.Svc.Pool.QueryRow(c.Request.Context(),
		`SELECT COALESCE(NULLIF(created_by_email,''),''), status FROM hub.hub_requirements WHERE id=$1 AND business_id=$2`,
		id, bizID,
	).Scan(&creatorEmail, &currentStatus)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "requirement not found"})
		return
	}
	if st.Email != creatorEmail && !canManageRequirement(role) {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "only author or admin may edit"})
		return
	}
	if currentStatus != "draft" && !canManageRequirement(role) {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "only draft requirements can be edited"})
		return
	}

	var req updateRequirementReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if req.Title == nil && req.Description == nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "no fields to update"})
		return
	}

	var sets []string
	args := []interface{}{id, bizID}
	if req.Title != nil {
		sets = append(sets, "title = $3")
		args = append(args, *req.Title)
	}
	if req.Description != nil {
		idx := len(args) + 1
		sets = append(sets, "description = $"+string(rune('0'+idx)))
		args = append(args, *req.Description)
	}
	sets = append(sets, "updated_at = now()")

	_, err = h.Svc.Pool.Exec(c.Request.Context(),
		`UPDATE hub.hub_requirements SET `+strings.Join(sets, ", ")+` WHERE id=$1 AND business_id=$2`,
		args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": "ok"})
}

type transitionReq struct {
	Comment string `json:"comment"`
}

func (h *Handler) SubmitRequirement(c *gin.Context) {
	h.transitionRequirement(c, "draft", "submitted")
}

func (h *Handler) AcceptRequirement(c *gin.Context) {
	h.transitionRequirement(c, "in_review", "accepted")
}

func (h *Handler) RejectRequirement(c *gin.Context) {
	h.transitionRequirement(c, "in_review", "rejected")
}

func (h *Handler) CancelRequirement(c *gin.Context) {
	h.transitionRequirementAny(c, "cancelled")
}

func (h *Handler) transitionRequirement(c *gin.Context, fromStatus, toStatus string) {
	code := c.Param("code")
	bizID, role, ok := h.RequireMembership(c, code)
	if !ok {
		return
	}
	id := c.Param("id")
	st := stampActorFromGin(c, "member", "", "")
	var req transitionReq
	_ = c.ShouldBindJSON(&req)

	var creatorEmail string
	var currentStatus string
	err := h.Svc.Pool.QueryRow(c.Request.Context(),
		`SELECT COALESCE(NULLIF(created_by_email,''),''), status FROM hub.hub_requirements WHERE id=$1 AND business_id=$2`,
		id, bizID,
	).Scan(&creatorEmail, &currentStatus)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "requirement not found"})
		return
	}
	if currentStatus != fromStatus {
		c.JSON(http.StatusConflict, gin.H{"code": 409, "message": "requirement is in status " + currentStatus + ", expected " + fromStatus})
		return
	}

	// Accept/reject requires admin or author
	if toStatus == "accepted" || toStatus == "rejected" {
		if st.Email != creatorEmail && !canManageRequirement(role) {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "only author or admin may " + toStatus})
			return
		}
	}

	now := time.Now()
	actorID := userIDPtr(st.UserID)
	meta := requirementTransitionMetadata(toStatus)
	_, err = h.Svc.Pool.Exec(c.Request.Context(),
		`UPDATE hub.hub_requirements SET status=$1, updated_at=$2,
		 submitted_at=CASE WHEN $1='submitted' THEN $2 ELSE submitted_at END,
		 accepted_at=CASE WHEN $1='accepted' THEN $2 WHEN $1='rejected' THEN NULL ELSE accepted_at END,
		 accepted_by_user_id=CASE WHEN $1='accepted' THEN $3 WHEN $1='rejected' THEN NULL ELSE accepted_by_user_id END
		 WHERE id=$4 AND business_id=$5`,
		toStatus, now, actorID, id, bizID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	// Write comment for the transition
	if req.Comment != "" {
		h.Svc.Pool.Exec(c.Request.Context(),
			`INSERT INTO hub.hub_requirement_comments (business_id, requirement_id, author_user_id, author_email, body, decision, created_at)
				 VALUES ($1,$2,$3,$4,$5,$6,now())`,
			bizID, id, actorID, st.Email, req.Comment, meta.decision)
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"status": toStatus}})
}

// transitionRequirementAny restricts cancellation to the author or an administrator role.
func (h *Handler) transitionRequirementAny(c *gin.Context, toStatus string) {
	code := c.Param("code")
	bizID, role, ok := h.RequireMembership(c, code)
	if !ok {
		return
	}
	id := c.Param("id")
	st := stampActorFromGin(c, "member", "", "")
	var creatorEmail, currentStatus string
	if err := h.Svc.Pool.QueryRow(c.Request.Context(), `SELECT COALESCE(NULLIF(created_by_email,''),''), status FROM hub.hub_requirements WHERE id=$1 AND business_id=$2`, id, bizID).Scan(&creatorEmail, &currentStatus); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "requirement not found"})
		return
	}
	if currentStatus == "accepted" || currentStatus == "rejected" || currentStatus == "cancelled" {
		c.JSON(http.StatusConflict, gin.H{"code": 409, "message": "requirement cannot be cancelled from status " + currentStatus})
		return
	}
	if !canCancelRequirement(role) && st.Email != creatorEmail {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "only author or admin may cancel"})
		return
	}
	_, err := h.Svc.Pool.Exec(c.Request.Context(),
		`UPDATE hub.hub_requirements SET status=$1, updated_at=now() WHERE id=$2 AND business_id=$3`,
		toStatus, id, bizID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"status": toStatus}})
}

type linkTaskReq struct {
	TaskID string `json:"task_id" binding:"required"`
}

func (h *Handler) LinkRequirementTask(c *gin.Context) {
	code := c.Param("code")
	bizID, _, ok := h.RequireMembership(c, code)
	if !ok {
		return
	}
	id := c.Param("id")
	var req linkTaskReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	var exists bool
	if err := h.Svc.Pool.QueryRow(c.Request.Context(), `SELECT EXISTS (SELECT 1 FROM hub.hub_requirements WHERE id=$1 AND business_id=$2)`, id, bizID).Scan(&exists); err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "requirement not found"})
		return
	}
	_, err := h.Svc.Pool.Exec(c.Request.Context(),
		`INSERT INTO hub.hub_requirement_links (business_id, requirement_id, task_id)
		 VALUES ($1,$2,$3) ON CONFLICT DO NOTHING`,
		bizID, id, req.TaskID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": "ok"})
}

func (h *Handler) UnlinkRequirementTask(c *gin.Context) {
	code := c.Param("code")
	bizID, _, ok := h.RequireMembership(c, code)
	if !ok {
		return
	}
	id := c.Param("id")
	taskID := c.Param("task_id")
	_, err := h.Svc.Pool.Exec(c.Request.Context(),
		`DELETE FROM hub.hub_requirement_links WHERE business_id=$1 AND requirement_id=$2 AND task_id=$3`,
		bizID, id, taskID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": "ok"})
}
