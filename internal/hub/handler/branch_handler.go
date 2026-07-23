package handler

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type branchReportItem struct {
	Name   string `json:"name" binding:"required"`
	TipSHA string `json:"tip_sha"`
	Source string `json:"source"`
}

type branchBindItem struct {
	BindType    string `json:"bind_type" binding:"required"` // dag|task|worker|user
	BindID      string `json:"bind_id" binding:"required"`
	BranchName  string `json:"branch_name" binding:"required"`
	HeadSHA     string `json:"head_sha"`
	WorktreeHost string `json:"worktree_host"`
	Status      string `json:"status"`
}

type branchReportReq struct {
	RepoURL    string             `json:"repo_url"` // optional; uses first repo or business.repo_url
	Branches   []branchReportItem `json:"branches"`
	Bindings   []branchBindItem   `json:"bindings"`
	Reporter   string             `json:"reporter"`
}

func (h *Handler) resolveBizAndRepo(c *gin.Context, code, repoURL string) (bizID, repoID int64, err error) {
	err = h.Svc.Pool.QueryRow(c.Request.Context(),
		"SELECT id FROM hub.hub_businesses WHERE code=$1", code,
	).Scan(&bizID)
	if err != nil {
		return 0, 0, err
	}
	if repoURL != "" {
		err = h.Svc.Pool.QueryRow(c.Request.Context(),
			`SELECT id FROM hub.hub_repos WHERE business_id=$1 AND repo_url=$2`,
			bizID, repoURL,
		).Scan(&repoID)
		if err != nil {
			// auto-create repo row
			_ = h.Svc.Pool.QueryRow(c.Request.Context(),
				`INSERT INTO hub.hub_repos (business_id, repo_url, default_branch, provider)
				 VALUES ($1,$2,'main',$3) RETURNING id`,
				bizID, repoURL, detectProvider(repoURL),
			).Scan(&repoID)
		}
		return bizID, repoID, nil
	}
	err = h.Svc.Pool.QueryRow(c.Request.Context(),
		`SELECT id FROM hub.hub_repos WHERE business_id=$1 ORDER BY id LIMIT 1`, bizID,
	).Scan(&repoID)
	if err != nil {
		// fallback: business.repo_url
		var br string
		if e2 := h.Svc.Pool.QueryRow(c.Request.Context(),
			`SELECT COALESCE(repo_url,'') FROM hub.hub_businesses WHERE id=$1`, bizID,
		).Scan(&br); e2 == nil && br != "" {
			_ = h.Svc.Pool.QueryRow(c.Request.Context(),
				`INSERT INTO hub.hub_repos (business_id, repo_url, default_branch, provider)
				 VALUES ($1,$2,'main',$3)
				 ON CONFLICT (business_id, repo_url) DO UPDATE SET updated_at=now()
				 RETURNING id`,
				bizID, br, detectProvider(br),
			).Scan(&repoID)
			if repoID > 0 {
				return bizID, repoID, nil
			}
		}
		return bizID, 0, err
	}
	return bizID, repoID, nil
}

func detectProvider(url string) string {
	u := strings.ToLower(url)
	if strings.Contains(u, "github.com") {
		return "github"
	}
	return "generic"
}

// ReportBranches upserts branch tips and optional bindings (worker API key or JWT).
func (h *Handler) ReportBranches(c *gin.Context) {
	code := c.Param("code")
	if _, _, ok := h.RequireMembership(c, code); !ok {
		return
	}
	var req branchReportReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if len(req.Branches) == 0 && len(req.Bindings) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "branches or bindings required"})
		return
	}

	bizID, repoID, err := h.resolveBizAndRepo(c, code, req.RepoURL)
	if err != nil || repoID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "business or repo not found; bind a repo first or pass repo_url"})
		return
	}

	// F.2: JWT email is source of truth for who reported (GitHub-commit style).
	reporter := ""
	if v, ok := c.Get("email"); ok {
		if s, castOK := v.(string); castOK {
			reporter = strings.TrimSpace(s)
		}
	}
	if reporter == "" {
		reporter = strings.TrimSpace(req.Reporter)
	}
	if reporter == "" {
		reporter = "system"
	}

	upserted := 0
	for _, b := range req.Branches {
		name := strings.TrimSpace(b.Name)
		if name == "" {
			continue
		}
		src := b.Source
		if src == "" {
			src = "report"
		}
		tip := strings.TrimSpace(b.TipSHA)
		// Prefer github_api/ls_remote over report when existing source is stronger
		_, err := h.Svc.Pool.Exec(c.Request.Context(),
			`INSERT INTO hub.hub_branches (business_id, repo_id, name, tip_sha, last_reporter, source, last_seen_at, updated_at)
			 VALUES ($1,$2,$3,$4,$5,$6,now(),now())
			 ON CONFLICT (repo_id, name) DO UPDATE SET
			   tip_sha = CASE
			     WHEN EXCLUDED.source IN ('github_api','ls_remote') THEN EXCLUDED.tip_sha
			     WHEN hub.hub_branches.source IN ('github_api','ls_remote') AND EXCLUDED.source = 'report'
			          AND hub.hub_branches.tip_sha <> '' THEN hub.hub_branches.tip_sha
			     ELSE EXCLUDED.tip_sha
			   END,
			   last_reporter = EXCLUDED.last_reporter,
			   source = CASE
			     WHEN EXCLUDED.source IN ('github_api','ls_remote') THEN EXCLUDED.source
			     WHEN hub.hub_branches.source IN ('github_api','ls_remote') THEN hub.hub_branches.source
			     ELSE EXCLUDED.source
			   END,
			   last_seen_at = now(),
			   updated_at = now()`,
			bizID, repoID, name, tip, reporter, src,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
			return
		}
		upserted++
	}

	bound := 0
	for _, b := range req.Bindings {
		if err := h.upsertBinding(c, bizID, b); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
			return
		}
		bound++
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"branches_upserted": upserted,
		"bindings_upserted": bound,
		"repo_id":           repoID,
	}})
}

func (h *Handler) upsertBinding(c *gin.Context, bizID int64, b branchBindItem) error {
	bt := strings.ToLower(strings.TrimSpace(b.BindType))
	switch bt {
	case "dag", "task", "worker", "user":
	default:
		return ginError("bind_type must be dag|task|worker|user")
	}
	status := b.Status
	if status == "" {
		status = "active"
	}
	host := b.WorktreeHost
	if host == "" {
		host, _ = os.Hostname()
	}
	_, err := h.Svc.Pool.Exec(c.Request.Context(),
		`INSERT INTO hub.hub_branch_bindings
		   (business_id, branch_name, bind_type, bind_id, worktree_host, head_sha, status, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,now())
		 ON CONFLICT (business_id, bind_type, bind_id) DO UPDATE SET
		   branch_name = EXCLUDED.branch_name,
		   worktree_host = EXCLUDED.worktree_host,
		   head_sha = EXCLUDED.head_sha,
		   status = EXCLUDED.status,
		   updated_at = now()`,
		bizID, strings.TrimSpace(b.BranchName), bt, strings.TrimSpace(b.BindID),
		host, strings.TrimSpace(b.HeadSHA), status,
	)
	return err
}

type ginError string

func (e ginError) Error() string { return string(e) }

// ListBranches returns branch index + bindings for a business.
func (h *Handler) ListBranches(c *gin.Context) {
	code := c.Param("code")
	bizID, _, ok := h.RequireMembership(c, code)
	if !ok {
		return
	}

	type branchRow struct {
		Name         string     `json:"name"`
		TipSHA       string     `json:"tip_sha"`
		IsDefault    bool       `json:"is_default"`
		PRNumber     *int       `json:"pr_number,omitempty"`
		PRURL        *string    `json:"pr_url,omitempty"`
		PRState      *string    `json:"pr_state,omitempty"`
		LastReporter *string    `json:"last_reporter,omitempty"`
		Source       string     `json:"source"`
		LastSeenAt   time.Time  `json:"last_seen_at"`
		RepoURL      string     `json:"repo_url"`
	}
	rows, err := h.Svc.Pool.Query(c.Request.Context(),
		`SELECT b.name, b.tip_sha, b.is_default, b.pr_number, b.pr_url, b.pr_state,
		        b.last_reporter, b.source, b.last_seen_at, r.repo_url
		 FROM hub.hub_branches b
		 JOIN hub.hub_repos r ON r.id = b.repo_id
		 WHERE b.business_id=$1
		 ORDER BY b.name`, bizID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	defer rows.Close()
	var branches []branchRow
	for rows.Next() {
		var br branchRow
		if err := rows.Scan(&br.Name, &br.TipSHA, &br.IsDefault, &br.PRNumber, &br.PRURL, &br.PRState,
			&br.LastReporter, &br.Source, &br.LastSeenAt, &br.RepoURL); err != nil {
			continue
		}
		branches = append(branches, br)
	}
	if branches == nil {
		branches = []branchRow{}
	}

	type bindRow struct {
		BranchName   string  `json:"branch_name"`
		BindType     string  `json:"bind_type"`
		BindID       string  `json:"bind_id"`
		WorktreeHost *string `json:"worktree_host,omitempty"`
		HeadSHA      *string `json:"head_sha,omitempty"`
		Status       string  `json:"status"`
		UpdatedAt    time.Time `json:"updated_at"`
	}
	brows, err := h.Svc.Pool.Query(c.Request.Context(),
		`SELECT branch_name, bind_type, bind_id, worktree_host, head_sha, status, updated_at
		 FROM hub.hub_branch_bindings WHERE business_id=$1 ORDER BY branch_name, bind_type`, bizID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	defer brows.Close()
	var bindings []bindRow
	for brows.Next() {
		var b bindRow
		if err := brows.Scan(&b.BranchName, &b.BindType, &b.BindID, &b.WorktreeHost, &b.HeadSHA, &b.Status, &b.UpdatedAt); err != nil {
			continue
		}
		// stale if binding head differs from branch tip
		for _, br := range branches {
			if br.Name == b.BranchName && b.HeadSHA != nil && *b.HeadSHA != "" && br.TipSHA != "" && *b.HeadSHA != br.TipSHA {
				b.Status = "stale"
			}
		}
		bindings = append(bindings, b)
	}
	if bindings == nil {
		bindings = []bindRow{}
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"branches":  branches,
		"bindings":  bindings,
		"business":  code,
	}})
}

// BindBranch upserts a single binding.
func (h *Handler) BindBranch(c *gin.Context) {
	code := c.Param("code")
	bizID, _, ok := h.RequireMembership(c, code)
	if !ok {
		return
	}
	var req branchBindItem
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if err := h.upsertBinding(c, bizID, req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": "bound"})
}

// UnbindBranch removes a binding by type+id.
func (h *Handler) UnbindBranch(c *gin.Context) {
	code := c.Param("code")
	bizID, _, ok := h.RequireMembership(c, code)
	if !ok {
		return
	}
	var req struct {
		BindType string `json:"bind_type" binding:"required"`
		BindID   string `json:"bind_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	tag, err := h.Svc.Pool.Exec(c.Request.Context(),
		`DELETE FROM hub.hub_branch_bindings
		 WHERE business_id=$1 AND bind_type=$2 AND bind_id=$3`,
		bizID, strings.ToLower(req.BindType), req.BindID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"deleted": tag.RowsAffected()}})
}
