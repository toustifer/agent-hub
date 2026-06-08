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
	var bizID int64
	err := h.Svc.Pool.QueryRow(c.Request.Context(), "SELECT id FROM hub.hub_businesses WHERE code=$1", code).Scan(&bizID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "business not found"})
		return
	}
	var req addRepoReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if req.DefaultBranch == "" { req.DefaultBranch = "main" }
	var id int64
	err = h.Svc.Pool.QueryRow(c.Request.Context(),
		"INSERT INTO hub.hub_repos (business_id, repo_url, default_branch) VALUES ($1,$2,$3) RETURNING id",
		bizID, req.RepoURL, req.DefaultBranch).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"id": id, "repo_url": req.RepoURL, "default_branch": req.DefaultBranch}})
}

func (h *Handler) ListRepos(c *gin.Context) {
	code := c.Param("code")
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
	TaskID        string   `json:"task_id"`
	Title         string   `json:"title"`
	Status        string   `json:"status"`
	AssignedWorker string  `json:"assigned_worker"`
	DependsOn     []string `json:"depends_on"`
	OutputFiles   []string `json:"output_files"`
}

func (h *Handler) SyncDAG(c *gin.Context) {
	code := c.Param("code")
	var bizID int64
	err := h.Svc.Pool.QueryRow(c.Request.Context(), "SELECT id FROM hub.hub_businesses WHERE code=$1", code).Scan(&bizID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "business not found"})
		return
	}
	var req syncDAGReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	_, err = h.Svc.Pool.Exec(c.Request.Context(),
		`INSERT INTO hub.hub_dag_state (business_id, task_id, title, status, assigned_worker, depends_on, output_files, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,now())
		 ON CONFLICT (business_id, task_id) DO UPDATE SET status=$4, updated_at=now()`,
		bizID, req.TaskID, req.Title, req.Status, req.AssignedWorker, req.DependsOn, req.OutputFiles)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": "ok"})
}

func (h *Handler) GetDAG(c *gin.Context) {
	code := c.Param("code")
	rows, err := h.Svc.Pool.Query(c.Request.Context(),
		`SELECT task_id, title, status, assigned_worker, depends_on, output_files
		 FROM hub.hub_dag_state WHERE business_id=(SELECT id FROM hub.hub_businesses WHERE code=$1)
		 ORDER BY task_id`, code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	defer rows.Close()
	type dagTask struct {
		TaskID        string   `json:"task_id"`
		Title         string   `json:"title"`
		Status        string   `json:"status"`
		AssignedWorker string  `json:"assigned_worker"`
		DependsOn     []string `json:"depends_on"`
		OutputFiles   []string `json:"output_files"`
	}
	var list []dagTask
	for rows.Next() {
		var t dagTask
		rows.Scan(&t.TaskID, &t.Title, &t.Status, &t.AssignedWorker, &t.DependsOn, &t.OutputFiles)
		list = append(list, t)
	}
	if list == nil { list = []dagTask{} }
	c.JSON(http.StatusOK, gin.H{"data": list})
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
