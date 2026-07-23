package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type syncReq struct {
	BusinessCode string        `json:"business_code" binding:"required"`
	Workers      []syncWorker  `json:"workers" binding:"required,min=1"`
}

type syncWorker struct {
	WorkerID    string        `json:"worker_id" binding:"required,max=128"`
	Version     string        `json:"version"`
	Host        string        `json:"host"`
	Pid         int           `json:"pid"`
	Owner       string                 `json:"owner"`
	Scope       string                 `json:"scope"`
	Handbook    map[string]interface{} `json:"handbook"`
	Patterns    []syncPlaybook         `json:"patterns"`
	Gotchas     []syncPlaybook `json:"gotchas"`
	Decisions   []syncPlaybook `json:"decisions"`
}

type syncPlaybook struct {
	Title   string   `json:"title" binding:"required,max=256"`
	Content string   `json:"content" binding:"required"`
	Tags    []string `json:"tags"`
}

func (h *Handler) SyncWorkers(c *gin.Context) {
	var req syncReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	var bizID int64
	err := h.Svc.Pool.QueryRow(c.Request.Context(),
		"SELECT id FROM hub.hub_businesses WHERE code=$1", req.BusinessCode,
	).Scan(&bizID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "business not found: " + req.BusinessCode})
		return
	}

	errors := []string{}
	workersSynced := 0
	playbooksSynced := 0

	for _, w := range req.Workers {
		_, err := h.Svc.Pool.Exec(c.Request.Context(),
			`INSERT INTO hub.hub_workers (business_id, worker_id, version, last_heartbeat_at, status, host, pid, owner, handbook, created_at, updated_at)
			 VALUES ($1,$2,$3,now(),'offline',$4,$5,$6,$7::jsonb,now(),now())
			 ON CONFLICT (business_id, worker_id) DO UPDATE SET version=$3, host=$4, pid=$5, owner=$6, handbook=$7::jsonb, updated_at=now()`,
			bizID, w.WorkerID, w.Version, w.Host, w.Pid, w.Owner, w.Handbook,
		)
		if err != nil {
			errors = append(errors, "worker "+w.WorkerID+": "+err.Error())
			continue
		}
		workersSynced++

		for _, p := range w.Patterns {
			err := insertPlaybook(c, h, bizID, w.WorkerID, "patterns", p)
			if err != nil {
				errors = append(errors, "worker "+w.WorkerID+" pattern '"+p.Title+"': "+err.Error())
			} else {
				playbooksSynced++
			}
		}
		for _, p := range w.Gotchas {
			err := insertPlaybook(c, h, bizID, w.WorkerID, "gotchas", p)
			if err != nil {
				errors = append(errors, "worker "+w.WorkerID+" gotcha '"+p.Title+"': "+err.Error())
			} else {
				playbooksSynced++
			}
		}
		for _, p := range w.Decisions {
			err := insertPlaybook(c, h, bizID, w.WorkerID, "decisions", p)
			if err != nil {
				errors = append(errors, "worker "+w.WorkerID+" decision '"+p.Title+"': "+err.Error())
			} else {
				playbooksSynced++
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"workers_synced":   workersSynced,
			"playbooks_synced": playbooksSynced,
			"errors":           errors,
		},
	})
}

type addRepoReq struct {
	RepoURL       string `json:"repo_url"`
	DefaultBranch string `json:"default_branch"`
}

func (h *Handler) AddRepo(c *gin.Context) {
	code := c.Param("code")
	bizID, _, ok := h.RequireMembership(c, code)
	if !ok {
		return
	}
	var req addRepoReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if req.DefaultBranch == "" {
		req.DefaultBranch = "main"
	}
	if req.RepoURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "repo_url required"})
		return
	}
	provider := "generic"
	if strings.Contains(strings.ToLower(req.RepoURL), "github.com") {
		provider = "github"
	}
	var id int64
	err := h.Svc.Pool.QueryRow(c.Request.Context(),
		`INSERT INTO hub.hub_repos (business_id, repo_url, default_branch, provider)
		 VALUES ($1,$2,$3,$4)
		 ON CONFLICT (business_id, repo_url) DO UPDATE SET
		   default_branch = EXCLUDED.default_branch,
		   provider = EXCLUDED.provider,
		   updated_at = now()
		 RETURNING id`,
		bizID, req.RepoURL, req.DefaultBranch, provider).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	// seed default branch row
	_, _ = h.Svc.Pool.Exec(c.Request.Context(),
		`INSERT INTO hub.hub_branches (business_id, repo_id, name, tip_sha, is_default, source, last_seen_at)
		 VALUES ($1,$2,$3,'',true,'report',now())
		 ON CONFLICT (repo_id, name) DO UPDATE SET is_default=true, updated_at=now()`,
		bizID, id, req.DefaultBranch)
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"id": id, "repo_url": req.RepoURL, "default_branch": req.DefaultBranch, "provider": provider}})
}

func (h *Handler) ListRepos(c *gin.Context) {
	code := c.Param("code")
	if _, _, ok := h.RequireMembership(c, code); !ok {
		return
	}
	rows, err := h.Svc.Pool.Query(c.Request.Context(),
		"SELECT r.id, r.repo_url, r.default_branch FROM hub.hub_repos r JOIN hub.hub_businesses b ON b.id=r.business_id WHERE b.code=$1 ORDER BY r.id", code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	defer rows.Close()
	type repo struct{ ID int64 `json:"id"`; RepoURL string `json:"repo_url"`; DefaultBranch string `json:"default_branch"` }
	var list []repo
	for rows.Next() {
		var r repo
		rows.Scan(&r.ID, &r.RepoURL, &r.DefaultBranch)
		list = append(list, r)
	}
	if list == nil { list = []repo{} }
	c.JSON(http.StatusOK, gin.H{"data": list})
}

func (h *Handler) DeleteRepo(c *gin.Context) {
	id := c.Param("id")
	_, err := h.Svc.Pool.Exec(c.Request.Context(), "DELETE FROM hub.hub_repos WHERE id=$1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": "ok"})
}

type syncDAGReq struct {
	TaskID         string   `json:"task_id"`
	Title          string   `json:"title"`
	Status         string   `json:"status"`
	AssignedWorker string   `json:"assigned_worker"`
	DependsOn      []string `json:"depends_on"`
	OutputFiles    []string `json:"output_files"`
	Branch         string   `json:"branch"`
	HeadSHA        string   `json:"head_sha"`
	// Optional explicit assignee; if empty and JWT present, stamp current user (F.1b).
	AssigneeEmail  string `json:"assignee_email"`
}

func (h *Handler) SyncDAG(c *gin.Context) {
	code := c.Param("code")
	bizID, _, ok := h.RequireMembership(c, code)
	if !ok {
		return
	}
	var req syncDAGReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	// F.1b assignee: body override, else JWT email/user
	assigneeEmail := strings.TrimSpace(req.AssigneeEmail)
	var assigneeUID *int64
	st := stampActorFromGin(c, "worker", req.AssignedWorker, "")
	if assigneeEmail == "" {
		assigneeEmail = st.Email
	}
	if st.UserID > 0 {
		assigneeUID = userIDPtr(st.UserID)
	}
	_, err := h.Svc.Pool.Exec(c.Request.Context(),
		`INSERT INTO hub.hub_dag_state (
		   business_id, task_id, title, status, assigned_worker, depends_on, output_files,
		   branch, head_sha, assignee_user_id, assignee_email, updated_at
		 ) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,NULLIF($11,''),now())
		 ON CONFLICT (business_id, task_id) DO UPDATE SET
		   title=$3, status=$4, assigned_worker=$5, depends_on=$6, output_files=$7,
		   branch=COALESCE(NULLIF($8,''), hub.hub_dag_state.branch),
		   head_sha=COALESCE(NULLIF($9,''), hub.hub_dag_state.head_sha),
		   assignee_user_id=COALESCE($10, hub.hub_dag_state.assignee_user_id),
		   assignee_email=COALESCE(NULLIF($11,''), hub.hub_dag_state.assignee_email),
		   updated_at=now()`,
		bizID, req.TaskID, req.Title, req.Status, req.AssignedWorker, req.DependsOn, req.OutputFiles,
		req.Branch, req.HeadSHA, assigneeUID, assigneeEmail)
	if err != nil {
		// fallback without assignee columns if migration not applied
		_, err2 := h.Svc.Pool.Exec(c.Request.Context(),
			`INSERT INTO hub.hub_dag_state (business_id, task_id, title, status, assigned_worker, depends_on, output_files, branch, head_sha, updated_at)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,now())
			 ON CONFLICT (business_id, task_id) DO UPDATE SET
			   title=$3, status=$4, assigned_worker=$5, depends_on=$6, output_files=$7,
			   branch=COALESCE(NULLIF($8,''), hub.hub_dag_state.branch),
			   head_sha=COALESCE(NULLIF($9,''), hub.hub_dag_state.head_sha),
			   updated_at=now()`,
			bizID, req.TaskID, req.Title, req.Status, req.AssignedWorker, req.DependsOn, req.OutputFiles, req.Branch, req.HeadSHA)
		if err2 != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"ok": true, "assignee_email": assigneeEmail}})
}

func (h *Handler) GetDAG(c *gin.Context) {
	code := c.Param("code")
	bizID, _, ok := h.RequireMembership(c, code)
	if !ok {
		return
	}
	rows, err := h.Svc.Pool.Query(c.Request.Context(),
		`SELECT task_id, title, status, COALESCE(assigned_worker,''), depends_on, output_files,
		        COALESCE(branch,''), COALESCE(head_sha,'')
		 FROM hub.hub_dag_state WHERE business_id=$1
		 ORDER BY task_id`, bizID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	defer rows.Close()
	type dagTask struct {
		TaskID         string   `json:"task_id"`
		Title          string   `json:"title"`
		Status         string   `json:"status"`
		AssignedWorker string   `json:"assigned_worker"`
		DependsOn      []string `json:"depends_on"`
		OutputFiles    []string `json:"output_files"`
		Branch         string   `json:"branch,omitempty"`
		HeadSHA        string   `json:"head_sha,omitempty"`
		LastActorEmail string   `json:"last_actor_email,omitempty"`
		LastActorRole  string   `json:"last_actor_role,omitempty"`
		LastEventType  string   `json:"last_event_type,omitempty"`
		LastEventAt    string   `json:"last_event_at,omitempty"`
		AssigneeEmail  string   `json:"assignee_email,omitempty"`
		AssigneeUserID *int64   `json:"assignee_user_id,omitempty"`
	}
	var list []dagTask
	var taskIDs []string
	for rows.Next() {
		var t dagTask
		_ = rows.Scan(&t.TaskID, &t.Title, &t.Status, &t.AssignedWorker, &t.DependsOn, &t.OutputFiles, &t.Branch, &t.HeadSHA)
		list = append(list, t)
		taskIDs = append(taskIDs, t.TaskID)
	}
	if list == nil {
		list = []dagTask{}
	}
	if len(taskIDs) > 0 {
		actors := h.lastActorsForTasks(c, bizID, taskIDs)
		for i := range list {
			if a, ok := actors[list[i].TaskID]; ok {
				list[i].LastActorEmail = a.Email
				list[i].LastActorRole = a.Role
				list[i].LastEventType = a.EventType
				list[i].LastEventAt = a.At
			}
		}
		assignees := h.assigneesForTasks(c, bizID, taskIDs)
		for i := range list {
			if a, ok := assignees[list[i].TaskID]; ok {
				list[i].AssigneeEmail = a.Email
				list[i].AssigneeUserID = a.UserID
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}

type lastActorInfo struct {
	Email     string
	Role      string
	EventType string
	At        string
}

func (h *Handler) lastActorsForTasks(c *gin.Context, bizID int64, taskIDs []string) map[string]lastActorInfo {
	out := map[string]lastActorInfo{}
	if len(taskIDs) == 0 {
		return out
	}
	rows, err := h.Svc.Pool.Query(c.Request.Context(), `
		SELECT DISTINCT ON (payload->>'task_id')
		  payload->>'task_id' AS task_id,
		  COALESCE(NULLIF(actor_email,''), actor, '') AS email,
		  COALESCE(actor_role, '') AS role,
		  event_type,
		  created_at
		FROM hub.hub_events
		WHERE business_id = $1
		  AND payload ? 'task_id'
		  AND payload->>'task_id' = ANY($2::text[])
		ORDER BY payload->>'task_id', created_at DESC`,
		bizID, taskIDs,
	)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var tid, email, role, et string
		var at interface{}
		if err := rows.Scan(&tid, &email, &role, &et, &at); err != nil {
			continue
		}
		out[tid] = lastActorInfo{Email: email, Role: role, EventType: et, At: stringifyTime(at)}
	}
	return out
}

type assigneeInfo struct {
	Email  string
	UserID *int64
}

func (h *Handler) assigneesForTasks(c *gin.Context, bizID int64, taskIDs []string) map[string]assigneeInfo {
	out := map[string]assigneeInfo{}
	rows, err := h.Svc.Pool.Query(c.Request.Context(), `
		SELECT task_id, COALESCE(assignee_email,''), assignee_user_id
		FROM hub.hub_dag_state
		WHERE business_id=$1 AND task_id = ANY($2::text[])`,
		bizID, taskIDs,
	)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var tid, email string
		var uid *int64
		if err := rows.Scan(&tid, &email, &uid); err != nil {
			continue
		}
		if email != "" || uid != nil {
			out[tid] = assigneeInfo{Email: email, UserID: uid}
		}
	}
	return out
}

func insertPlaybook(c *gin.Context, h *Handler, bizID int64, workerID, category string, p syncPlaybook) error {
	tags := p.Tags
	if tags == nil {
		tags = []string{}
	}
	content := strings.TrimSpace(p.Content)
	if content == "" {
		content = "(empty)"
	}
	_, err := h.Svc.Pool.Exec(c.Request.Context(),
		`INSERT INTO hub.hub_playbooks (business_id, category, title, content, tags, created_by_worker_id, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5::text[],$6,now(),now())
		 ON CONFLICT (business_id, category, title) DO UPDATE SET content=$4, tags=$5::text[], updated_at=now()`,
		bizID, category, p.Title, content, tags, workerID,
	)
	return err
}
