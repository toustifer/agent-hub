package mcp

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stifer/agent-hub/internal/hub/service"
)

func (h *Hub) registerTools(s *mcpsdk.Server) {
	// Auth / account
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "hub_login",
		Description: "Step 1: hub_login() to get a browser URL. Open it, log in, click Approve. Step 2: hub_login({code}) to finish. Returns a JWT for subsequent Authorization: Bearer calls.",
	}, h.toolLogin)
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "hub_list_my_businesses",
		Description: "List businesses I belong to (JWT user)",
	}, h.toolListMyBusinesses)

	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name: "hub_export_soft_sync_config",
		Description: "Export soft-sync config for local agentflow: returns JSON for ~/.agent-hub/config.json using the current OAuth/Bearer JWT. Claude should write the file with Write/Bash. This bridges Hub MCP login → agentflow auto progress sync.",
	}, h.toolExportSoftSyncConfig)

	// Machine tools (API key or JWT membership)
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "hub_heartbeat",
		Description: "Worker heartbeat",
	}, h.toolHeartbeat)
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "hub_acquire_lock",
		Description: "Acquire distributed lock",
	}, h.toolAcquireLock)
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "hub_release_lock",
		Description: "Release lock",
	}, h.toolReleaseLock)
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "hub_renew_lock",
		Description: "Renew lock TTL",
	}, h.toolRenewLock)
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "hub_append_event",
		Description: "Record event",
	}, h.toolAppendEvent)
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "hub_create_playbook",
		Description: "Create playbook",
	}, h.toolCreatePlaybook)
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "hub_search_playbooks",
		Description: "Search playbooks",
	}, h.toolSearchPlaybooks)

	// List tools
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "hub_list_workers",
		Description: "List workers",
	}, h.toolListWorkers)
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "hub_list_locks",
		Description: "List active locks",
	}, h.toolListLocks)
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "hub_list_events",
		Description: "List events",
	}, h.toolListEvents)

	// Team docs + worker templates
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "hub_upsert_doc",
		Description: "Create or update a team document (visibility=team is shared)",
	}, h.toolUpsertDoc)
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "hub_list_docs",
		Description: "List team documents for a business",
	}, h.toolListDocs)
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "hub_get_doc",
		Description: "Get one team document by doc_key",
	}, h.toolGetDoc)
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "hub_publish_worker_template",
		Description: "Publish a reusable worker template to the team (not runtime)",
	}, h.toolPublishWorkerTemplate)
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "hub_list_worker_templates",
		Description: "List worker templates colleagues can reuse",
	}, h.toolListWorkerTemplates)

	// Repo / DAG
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "hub_add_repo",
		Description: "Bind GitHub repo",
	}, h.toolAddRepo)
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "hub_sync_dag",
		Description: "Sync DAG task",
	}, h.toolSyncDAG)
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "hub_get_dag",
		Description: "Get DAG tasks",
	}, h.toolGetDAG)

	// Invite / link requests
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "hub_invite_member",
		Description: "Invite a member by email (returns invite_url with one-time token)",
	}, h.toolInviteMember)
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "hub_accept_invitation",
		Description: "Accept invite by token (preferred) or join open business by code",
	}, h.toolAcceptInvitation)
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "hub_create_link_request",
		Description: "Request to join a business (admin approval)",
	}, h.toolCreateLinkRequest)
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "hub_list_link_requests",
		Description: "List pending link requests (admin only)",
	}, h.toolListLinkRequests)
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "hub_review_link_request",
		Description: "Approve or reject a link request (admin only)",
	}, h.toolReviewLinkRequest)

	// Branches
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "hub_report_branches",
		Description: "Report local git branch tips (+ optional task/dag/worker bindings) to Hub branch index. Soft-fail friendly; use after prepare/submit.",
	}, h.toolReportBranches)
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "hub_list_branches",
		Description: "List Hub branch index + who/what is bound to each branch for a business",
	}, h.toolListBranches)
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "hub_bind_branch",
		Description: "Bind a dag|task|worker|user to a branch name on Hub",
	}, h.toolBindBranch)
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "hub_refresh_branches",
		Description: "Refresh branch tips from GitHub API (GITHUB_TOKEN) or git ls-remote; source wins over report",
	}, h.toolRefreshBranches)
}

// ── shared arg structs ──────────────────────────────────────────────

type bcArgs struct {
	BusinessCode string `json:"business_code,omitempty"`
}

type exportSoftSyncArgs struct {
	BusinessCode string `json:"business_code,omitempty" jsonschema:"Team code to bind (e.g. zhiji). Optional if only one team."`
}

type loginArgs struct {
	Code string `json:"code,omitempty" jsonschema:"Device code to exchange (only needed in step 2)"`
}

type heartbeatArgs struct {
	BusinessCode string `json:"business_code,omitempty"`
	WorkerID     string `json:"worker_id"`
	Version      string `json:"version"`
	Host         string `json:"host,omitempty"`
	Pid          int    `json:"pid,omitempty"`
}

type acquireLockArgs struct {
	BusinessCode string `json:"business_code,omitempty"`
	ResourceKey  string `json:"resource_key"`
	WorkerID     string `json:"worker_id"`
	TTLSeconds   int    `json:"ttl_seconds,omitempty"`
}

type holderTokenArgs struct {
	HolderToken  string `json:"holder_token"`
	TTLSeconds   int    `json:"ttl_seconds,omitempty"`
	BusinessCode string `json:"business_code,omitempty"`
}

type appendEventArgs struct {
	BusinessCode  string                 `json:"business_code,omitempty"`
	Actor         string                 `json:"actor,omitempty"`
	EventType     string                 `json:"event_type"`
	Payload       map[string]interface{} `json:"payload,omitempty"`
	ActorRole     string                 `json:"actor_role,omitempty"`
	ActorWorkerID string                 `json:"actor_worker_id,omitempty"`
}

type upsertDocArgs struct {
	BusinessCode string `json:"business_code,omitempty"`
	DocKey       string `json:"doc_key"`
	Title        string `json:"title"`
	Content      string `json:"content,omitempty"`
	Category     string `json:"category,omitempty"`
	Visibility   string `json:"visibility,omitempty"`
}

type getDocArgs struct {
	BusinessCode string `json:"business_code,omitempty"`
	DocKey       string `json:"doc_key"`
}

type publishWorkerTemplateArgs struct {
	BusinessCode  string                 `json:"business_code,omitempty"`
	WorkerID      string                 `json:"worker_id"`
	DisplayName   string                 `json:"display_name,omitempty"`
	Description   string                 `json:"description,omitempty"`
	Skills        []string               `json:"skills,omitempty"`
	Handbook      map[string]interface{} `json:"handbook,omitempty"`
	PromptSummary string                 `json:"prompt_summary,omitempty"`
	Visibility    string                 `json:"visibility,omitempty"`
}

type createPlaybookArgs struct {
	BusinessCode string   `json:"business_code,omitempty"`
	Category     string   `json:"category"`
	Title        string   `json:"title"`
	Content      string   `json:"content"`
	Tags         []string `json:"tags,omitempty"`
	WorkerID     string   `json:"worker_id"`
}

type searchPlaybookArgs struct {
	BusinessCode string `json:"business_code,omitempty"`
	Query        string `json:"query"`
	Category     string `json:"category,omitempty"`
	Limit        int    `json:"limit,omitempty"`
}

type listWorkersArgs struct {
	BusinessCode string `json:"business_code,omitempty"`
	Status       string `json:"status,omitempty"`
}

type listEventsArgs struct {
	BusinessCode string `json:"business_code,omitempty"`
	EventType    string `json:"event_type,omitempty"`
	Limit        int    `json:"limit,omitempty"`
}

type addRepoArgs struct {
	BusinessCode  string `json:"business_code,omitempty"`
	RepoURL       string `json:"repo_url"`
	DefaultBranch string `json:"default_branch,omitempty"`
}

type syncDAGArgs struct {
	BusinessCode   string `json:"business_code,omitempty"`
	TaskID         string `json:"task_id"`
	Title          string `json:"title"`
	Status         string `json:"status"`
	AssignedWorker string `json:"assigned_worker,omitempty"`
}

type inviteArgs struct {
	BusinessCode string `json:"business_code,omitempty"`
	Email        string `json:"email"`
	Role         string `json:"role,omitempty"`
}

type acceptInviteArgs struct {
	Token        string `json:"token,omitempty" jsonschema:"Invite token from invite_url"`
	BusinessCode string `json:"business_code,omitempty" jsonschema:"Only works when join_policy=open"`
}

type linkReqArgs struct {
	BusinessCode string `json:"business_code"`
	DeviceInfo   string `json:"device_info,omitempty"`
}

type reviewLinkArgs struct {
	BusinessCode string `json:"business_code"`
	RequestID    int64  `json:"request_id"`
	Action       string `json:"action"`
}

type branchItem struct {
	Name   string `json:"name"`
	TipSHA string `json:"tip_sha,omitempty"`
	Source string `json:"source,omitempty"`
}

type bindItem struct {
	BindType     string `json:"bind_type"`
	BindID       string `json:"bind_id"`
	BranchName   string `json:"branch_name"`
	HeadSHA      string `json:"head_sha,omitempty"`
	WorktreeHost string `json:"worktree_host,omitempty"`
	Status       string `json:"status,omitempty"`
}

type reportBranchesArgs struct {
	BusinessCode string       `json:"business_code,omitempty"`
	RepoURL      string       `json:"repo_url,omitempty"`
	Reporter     string       `json:"reporter,omitempty"`
	Branches     []branchItem `json:"branches,omitempty"`
	Bindings     []bindItem   `json:"bindings,omitempty"`
}

type bindBranchArgs struct {
	BusinessCode string `json:"business_code,omitempty"`
	BindType     string `json:"bind_type"`
	BindID       string `json:"bind_id"`
	BranchName   string `json:"branch_name"`
	HeadSHA      string `json:"head_sha,omitempty"`
	WorktreeHost string `json:"worktree_host,omitempty"`
	Status       string `json:"status,omitempty"`
}

type refreshBranchesArgs struct {
	BusinessCode string `json:"business_code,omitempty"`
	RepoURL      string `json:"repo_url,omitempty"`
	DefaultOnly  bool   `json:"default_only,omitempty"`
}

// ── helpers ─────────────────────────────────────────────────────────

func jsonText(v any) (*mcpsdk.CallToolResult, any, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return errResult(err)
	}
	return textResult(string(b))
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func genRawToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func randomDeviceCode(n int) string {
	const letters = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, n)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = letters[int(b[i])%len(letters)]
	}
	return string(b)
}

func (h *Hub) genJWT(userID int64, email string) string {
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

func detectProvider(url string) string {
	if strings.Contains(strings.ToLower(url), "github.com") {
		return "github"
	}
	return "generic"
}

func (h *Hub) resolveBizRepo(ctx context.Context, bizID int64, repoURL string) (repoID int64, err error) {
	if repoURL != "" {
		err = h.Svc.Pool.QueryRow(ctx,
			`SELECT id FROM hub.hub_repos WHERE business_id=$1 AND repo_url=$2 ORDER BY id LIMIT 1`,
			bizID, repoURL,
		).Scan(&repoID)
		if err == nil && repoID > 0 {
			return repoID, nil
		}
		err = h.Svc.Pool.QueryRow(ctx,
			`INSERT INTO hub.hub_repos (business_id, repo_url, default_branch, provider)
			 VALUES ($1,$2,'main',$3) RETURNING id`,
			bizID, repoURL, detectProvider(repoURL),
		).Scan(&repoID)
		return repoID, err
	}
	err = h.Svc.Pool.QueryRow(ctx,
		`SELECT id FROM hub.hub_repos WHERE business_id=$1 ORDER BY id LIMIT 1`, bizID,
	).Scan(&repoID)
	if err != nil {
		var br string
		if e2 := h.Svc.Pool.QueryRow(ctx,
			`SELECT COALESCE(repo_url,'') FROM hub.hub_businesses WHERE id=$1`, bizID,
		).Scan(&br); e2 == nil && br != "" {
			err = h.Svc.Pool.QueryRow(ctx,
				`INSERT INTO hub.hub_repos (business_id, repo_url, default_branch, provider)
				 VALUES ($1,$2,'main',$3) RETURNING id`,
				bizID, br, detectProvider(br),
			).Scan(&repoID)
			return repoID, err
		}
	}
	return repoID, err
}

func (h *Hub) upsertBinding(ctx context.Context, bizID int64, b bindItem) error {
	bt := strings.ToLower(strings.TrimSpace(b.BindType))
	switch bt {
	case "dag", "task", "worker", "user":
	default:
		return errors.New("bind_type must be dag|task|worker|user")
	}
	status := b.Status
	if status == "" {
		status = "active"
	}
	host := b.WorktreeHost
	if host == "" {
		host, _ = os.Hostname()
	}
	_, err := h.Svc.Pool.Exec(ctx,
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

// ── tool implementations ────────────────────────────────────────────

func (h *Hub) toolLogin(ctx context.Context, req *mcpsdk.CallToolRequest, args loginArgs) (*mcpsdk.CallToolResult, any, error) {
	if args.Code != "" {
		var token *string
		var confirmed bool
		err := h.Svc.Pool.QueryRow(ctx,
			"SELECT token, confirmed FROM hub.hub_device_codes WHERE code=$1 AND expires_at > now()", args.Code,
		).Scan(&token, &confirmed)
		if err != nil {
			return errResult(errors.New("invalid or expired code"))
		}
		if !confirmed || token == nil || *token == "" {
			return errResult(errors.New("code not yet approved; open the verification URL, log in, and click Approve"))
		}
		_, _ = h.Svc.Pool.Exec(ctx, "DELETE FROM hub.hub_device_codes WHERE code=$1", args.Code)
		return textResult(fmt.Sprintf("Logged in. Use Authorization: Bearer %s on subsequent MCP requests (or configure OAuth).", *token))
	}
	code := randomDeviceCode(8)
	_, err := h.Svc.Pool.Exec(ctx,
		`INSERT INTO hub.hub_device_codes (code, token, user_id, confirmed, expires_at)
		 VALUES ($1, NULL, NULL, false, now() + interval '10 minutes')`, code)
	if err != nil {
		return errResult(err)
	}
	url := h.PublicBaseURL + "/auth/device?code=" + code
	return textResult(fmt.Sprintf("Open this URL in browser (you must be logged in as a real user):\n\n  %s\n\nThen call hub_login({ code: \"%s\" }) to finish.", url, code))
}


func (h *Hub) toolExportSoftSyncConfig(ctx context.Context, req *mcpsdk.CallToolRequest, args exportSoftSyncArgs) (*mcpsdk.CallToolResult, any, error) {
	id, err := h.identityFromRequest(ctx, req)
	if err != nil {
		return errResult(err)
	}
	if id.IsAPIKey || id.BearerToken == "" {
		return errResult(errors.New("need human OAuth/JWT session (not API key). Complete /mcp login, then call again"))
	}
	email := id.Email
	bc := strings.TrimSpace(args.BusinessCode)
	// list memberships
	rows, err := h.Svc.Pool.Query(ctx,
		`SELECT b.code, b.name, m.role
		 FROM hub.hub_businesses b
		 JOIN hub.hub_memberships m ON m.business_id = b.id
		 WHERE m.user_id = $1 ORDER BY b.code`, id.UserID)
	if err != nil {
		return errResult(err)
	}
	defer rows.Close()
	type team struct{ Code, Name, Role string }
	var teams []team
	for rows.Next() {
		var t team
		if rows.Scan(&t.Code, &t.Name, &t.Role) == nil {
			teams = append(teams, t)
		}
	}
	if bc == "" {
		if len(teams) == 1 {
			bc = teams[0].Code
		} else if len(teams) == 0 {
			return errResult(errors.New("you have no team memberships; create/join a team first"))
		} else {
			var lines []string
			for _, t := range teams {
				lines = append(lines, fmt.Sprintf("%s — %s [%s]", t.Code, t.Name, t.Role))
			}
			return textResult("Multiple teams — call again with business_code set to one of:\n" + strings.Join(lines, "\n") +
				"\n\nExample: hub_export_soft_sync_config({ business_code: \"zhiji\" })")
		}
	}
	// verify membership
	ok := false
	for _, t := range teams {
		if t.Code == bc {
			ok = true
			break
		}
	}
	if !ok {
		// still allow if requireMembership passes
		if _, _, _, err := h.requireMembership(ctx, req, bc); err != nil {
			return errResult(fmt.Errorf("not a member of %s: %w", bc, err))
		}
	}
	cfg := map[string]string{
		"hub_url":        h.PublicBaseURL,
		"token":          id.BearerToken,
		"business_code":  bc,
		"email":          email,
		"login_at":       time.Now().UTC().Format(time.RFC3339),
	}
	raw, _ := json.MarshalIndent(cfg, "", "  ")
	pathHint := `~/.agent-hub/config.json`
	winHint := `%USERPROFILE%\.agent-hub\config.json`
	msg := "Soft-sync config for local agentflow (same JWT as this MCP session).\n\n" +
		"Claude/Cursor: write this exact JSON to " + pathHint + " (Windows: " + winHint + ").\n" +
		"Create the .agent-hub directory if missing. Mode 600 preferred.\n\n" +
		"After writing, restart agentflow MCP (or next task_create/start will pick it up).\n" +
		"Disable anytime: HUB_SYNC=0\n\n" +
		"```json\n" + string(raw) + "\n```\n"
	return textResult(msg)
}

func (h *Hub) toolListMyBusinesses(ctx context.Context, req *mcpsdk.CallToolRequest, _ struct{}) (*mcpsdk.CallToolResult, any, error) {
	id, err := h.requireUser(ctx, req)
	if err != nil {
		return errResult(err)
	}
	rows, err := h.Svc.Pool.Query(ctx,
		`SELECT b.code, b.name, m.role
		 FROM hub.hub_businesses b
		 JOIN hub.hub_memberships m ON m.business_id = b.id
		 WHERE m.user_id = $1`, id.UserID)
	if err != nil {
		return errResult(err)
	}
	defer rows.Close()
	var lines []string
	for rows.Next() {
		var code, name, role string
		if err := rows.Scan(&code, &name, &role); err != nil {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s — %s [%s]", code, name, role))
	}
	if len(lines) == 0 {
		return textResult("No businesses.")
	}
	return textResult(strings.Join(lines, "\n"))
}

func (h *Hub) toolHeartbeat(ctx context.Context, req *mcpsdk.CallToolRequest, args heartbeatArgs) (*mcpsdk.CallToolResult, any, error) {
	if args.WorkerID == "" || args.Version == "" {
		return errResult(errors.New("worker_id and version required"))
	}
	if args.BusinessCode == "" {
		return errResult(errors.New("business_code required"))
	}
	if _, _, _, err := h.requireMembership(ctx, req, args.BusinessCode); err != nil {
		return errResult(err)
	}
	host := args.Host
	if host == "" {
		host = "mcp"
	}
	w, err := h.Svc.Heartbeat(ctx, args.BusinessCode, args.WorkerID, args.Version, host, args.Pid)
	if err != nil {
		return errResult(err)
	}
	return textResult(fmt.Sprintf("Worker %s online (status=%s).", w.WorkerID, w.Status))
}

func (h *Hub) toolAcquireLock(ctx context.Context, req *mcpsdk.CallToolRequest, args acquireLockArgs) (*mcpsdk.CallToolResult, any, error) {
	if args.ResourceKey == "" || args.WorkerID == "" {
		return errResult(errors.New("resource_key and worker_id required"))
	}
	if args.BusinessCode == "" {
		return errResult(errors.New("business_code required"))
	}
	if _, _, _, err := h.requireMembership(ctx, req, args.BusinessCode); err != nil {
		return errResult(err)
	}
	ttl := args.TTLSeconds
	if ttl <= 0 {
		ttl = 300
	}
	token, expires, err := h.Svc.AcquireLock(ctx, args.BusinessCode, args.ResourceKey, args.WorkerID, ttl)
	if err != nil {
		return errResult(err)
	}
	return textResult(fmt.Sprintf("Lock acquired. Token: %s (expires %s)", token, expires.Format(time.RFC3339)))
}

func (h *Hub) toolReleaseLock(ctx context.Context, req *mcpsdk.CallToolRequest, args holderTokenArgs) (*mcpsdk.CallToolResult, any, error) {
	if args.HolderToken == "" {
		return errResult(errors.New("holder_token required"))
	}
	if _, err := h.identityFromRequest(ctx, req); err != nil {
		return errResult(err)
	}
	if err := h.Svc.ReleaseLock(ctx, args.HolderToken); err != nil {
		return errResult(err)
	}
	return textResult("Released.")
}

func (h *Hub) toolRenewLock(ctx context.Context, req *mcpsdk.CallToolRequest, args holderTokenArgs) (*mcpsdk.CallToolResult, any, error) {
	if args.HolderToken == "" {
		return errResult(errors.New("holder_token required"))
	}
	if _, err := h.identityFromRequest(ctx, req); err != nil {
		return errResult(err)
	}
	ttl := args.TTLSeconds
	if ttl <= 0 {
		ttl = 300
	}
	if err := h.Svc.RenewLock(ctx, args.HolderToken, ttl); err != nil {
		return errResult(err)
	}
	return textResult("Renewed.")
}

func (h *Hub) toolAppendEvent(ctx context.Context, req *mcpsdk.CallToolRequest, args appendEventArgs) (*mcpsdk.CallToolResult, any, error) {
	if args.EventType == "" {
		return errResult(errors.New("event_type required"))
	}
	if args.BusinessCode == "" {
		return errResult(errors.New("business_code required"))
	}
	_, _, id, err := h.requireMembership(ctx, req, args.BusinessCode)
	if err != nil {
		return errResult(err)
	}
	payload := args.Payload
	if payload == nil {
		payload = map[string]interface{}{}
	}
	role := strings.ToLower(strings.TrimSpace(args.ActorRole))
	switch role {
	case "leader", "worker", "reviewer", "system":
	default:
		role = "worker"
	}
	actor := strings.TrimSpace(args.Actor)
	var uid *int64
	email := ""
	if id != nil && id.UserID > 0 {
		u := id.UserID
		uid = &u
		email = id.Email
		if actor == "" {
			actor = email
		}
	}
	if actor == "" {
		actor = strings.TrimSpace(args.ActorWorkerID)
	}
	if actor == "" {
		actor = "unknown"
	}
	evID, err := h.Svc.AppendEventFull(ctx, service.AppendEventInput{
		BusinessCode:  args.BusinessCode,
		Actor:         actor,
		EventType:     args.EventType,
		Payload:       payload,
		ActorUserID:   uid,
		ActorEmail:    email,
		ActorRole:     role,
		ActorWorkerID: strings.TrimSpace(args.ActorWorkerID),
	})
	if err != nil {
		return errResult(err)
	}
	return textResult(fmt.Sprintf("Event %d recorded. actor=%s role=%s worker=%s", evID, actor, role, args.ActorWorkerID))
}

func (h *Hub) toolCreatePlaybook(ctx context.Context, req *mcpsdk.CallToolRequest, args createPlaybookArgs) (*mcpsdk.CallToolResult, any, error) {
	if args.Category == "" || args.Title == "" || args.Content == "" || args.WorkerID == "" {
		return errResult(errors.New("category, title, content, worker_id required"))
	}
	if args.BusinessCode == "" {
		return errResult(errors.New("business_code required"))
	}
	if _, _, _, err := h.requireMembership(ctx, req, args.BusinessCode); err != nil {
		return errResult(err)
	}
	tags := args.Tags
	if tags == nil {
		tags = []string{}
	}
	pb, err := h.Svc.CreatePlaybook(ctx, args.BusinessCode, args.Category, args.Title, args.Content, tags, args.WorkerID)
	if err != nil {
		return errResult(err)
	}
	return textResult(fmt.Sprintf("Playbook %d created.", pb.ID))
}

func (h *Hub) toolSearchPlaybooks(ctx context.Context, req *mcpsdk.CallToolRequest, args searchPlaybookArgs) (*mcpsdk.CallToolResult, any, error) {
	if args.Query == "" {
		return errResult(errors.New("query required"))
	}
	if args.BusinessCode != "" {
		if _, _, _, err := h.requireMembership(ctx, req, args.BusinessCode); err != nil {
			return errResult(err)
		}
	} else if _, err := h.identityFromRequest(ctx, req); err != nil {
		return errResult(err)
	}
	limit := args.Limit
	if limit <= 0 {
		limit = 20
	}
	list, err := h.Svc.SearchPlaybooks(ctx, args.Query, args.BusinessCode, args.Category, limit, 0)
	if err != nil {
		return errResult(err)
	}
	if len(list) == 0 {
		return textResult("No results.")
	}
	var lines []string
	for _, p := range list {
		lines = append(lines, fmt.Sprintf("[%s] %s", p.Category, p.Title))
	}
	return textResult(strings.Join(lines, "\n"))
}

func (h *Hub) toolListWorkers(ctx context.Context, req *mcpsdk.CallToolRequest, args listWorkersArgs) (*mcpsdk.CallToolResult, any, error) {
	if args.BusinessCode == "" {
		return errResult(errors.New("business_code required"))
	}
	if _, _, _, err := h.requireMembership(ctx, req, args.BusinessCode); err != nil {
		return errResult(err)
	}
	list, err := h.Svc.ListWorkersByBusiness(ctx, args.BusinessCode, args.Status)
	if err != nil {
		return errResult(err)
	}
	if len(list) == 0 {
		return textResult("No workers.")
	}
	var lines []string
	for _, w := range list {
		lines = append(lines, fmt.Sprintf("%s %s", w.WorkerID, w.Status))
	}
	return textResult(strings.Join(lines, "\n"))
}

func (h *Hub) toolListLocks(ctx context.Context, req *mcpsdk.CallToolRequest, args bcArgs) (*mcpsdk.CallToolResult, any, error) {
	if args.BusinessCode == "" {
		return errResult(errors.New("business_code required"))
	}
	if _, _, _, err := h.requireMembership(ctx, req, args.BusinessCode); err != nil {
		return errResult(err)
	}
	list, err := h.Svc.ListActiveLocks(ctx, args.BusinessCode)
	if err != nil {
		return errResult(err)
	}
	if len(list) == 0 {
		return textResult("No locks.")
	}
	var lines []string
	for _, l := range list {
		lines = append(lines, fmt.Sprintf("%s %s", l.ResourceKey, l.HolderWorkerID))
	}
	return textResult(strings.Join(lines, "\n"))
}

func (h *Hub) toolListEvents(ctx context.Context, req *mcpsdk.CallToolRequest, args listEventsArgs) (*mcpsdk.CallToolResult, any, error) {
	if args.BusinessCode == "" {
		return errResult(errors.New("business_code required"))
	}
	if _, _, _, err := h.requireMembership(ctx, req, args.BusinessCode); err != nil {
		return errResult(err)
	}
	limit := args.Limit
	if limit <= 0 {
		limit = 20
	}
	list, err := h.Svc.ListEventRows(ctx, args.BusinessCode, args.EventType, limit, 0)
	if err != nil {
		return errResult(err)
	}
	if len(list) == 0 {
		return textResult("No events.")
	}
	var lines []string
	for _, e := range list {
		who := e.ActorEmail
		if who == "" {
			who = e.Actor
		}
		extra := ""
		if e.ActorRole != "" {
			extra += " [" + e.ActorRole + "]"
		}
		if e.ActorWorkerID != "" {
			extra += " " + e.ActorWorkerID
		}
		lines = append(lines, fmt.Sprintf("%s %s %s%s", e.CreatedAt.Format("2006-01-02T15:04:05"), e.EventType, who, extra))
	}
	return textResult(strings.Join(lines, "\n"))
}

func (h *Hub) toolAddRepo(ctx context.Context, req *mcpsdk.CallToolRequest, args addRepoArgs) (*mcpsdk.CallToolResult, any, error) {
	if args.RepoURL == "" {
		return errResult(errors.New("repo_url required"))
	}
	if args.BusinessCode == "" {
		return errResult(errors.New("business_code required"))
	}
	bizID, _, _, err := h.requireMembership(ctx, req, args.BusinessCode)
	if err != nil {
		return errResult(err)
	}
	branch := args.DefaultBranch
	if branch == "" {
		branch = "main"
	}
	var id int64
	err = h.Svc.Pool.QueryRow(ctx,
		`SELECT id FROM hub.hub_repos WHERE business_id=$1 AND repo_url=$2 ORDER BY id LIMIT 1`,
		bizID, args.RepoURL,
	).Scan(&id)
	if err != nil {
		err = h.Svc.Pool.QueryRow(ctx,
			`INSERT INTO hub.hub_repos (business_id, repo_url, default_branch, provider)
			 VALUES ($1,$2,$3,$4) RETURNING id`,
			bizID, args.RepoURL, branch, detectProvider(args.RepoURL),
		).Scan(&id)
	} else {
		_, _ = h.Svc.Pool.Exec(ctx,
			`UPDATE hub.hub_repos SET default_branch=$1, provider=$2, updated_at=now() WHERE id=$3`,
			branch, detectProvider(args.RepoURL), id)
	}
	if err != nil {
		return errResult(err)
	}
	_, _ = h.Svc.Pool.Exec(ctx,
		`INSERT INTO hub.hub_branches (business_id, repo_id, name, tip_sha, is_default, source, last_seen_at)
		 VALUES ($1,$2,$3,'',true,'report',now())
		 ON CONFLICT (repo_id, name) DO UPDATE SET is_default=true, updated_at=now()`,
		bizID, id, branch)
	return textResult(fmt.Sprintf("Repo %s bound (id=%d).", args.RepoURL, id))
}

func (h *Hub) toolSyncDAG(ctx context.Context, req *mcpsdk.CallToolRequest, args syncDAGArgs) (*mcpsdk.CallToolResult, any, error) {
	if args.TaskID == "" || args.Title == "" || args.Status == "" {
		return errResult(errors.New("task_id, title, status required"))
	}
	if args.BusinessCode == "" {
		return errResult(errors.New("business_code required"))
	}
	bizID, _, _, err := h.requireMembership(ctx, req, args.BusinessCode)
	if err != nil {
		return errResult(err)
	}
	_, err = h.Svc.Pool.Exec(ctx,
		`INSERT INTO hub.hub_dag_state (business_id, task_id, title, status, assigned_worker, depends_on, output_files, branch, head_sha, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,'','',now())
		 ON CONFLICT (business_id, task_id) DO UPDATE SET
		   title=$3, status=$4, assigned_worker=$5, updated_at=now()`,
		bizID, args.TaskID, args.Title, args.Status, args.AssignedWorker, []string{}, []string{})
	if err != nil {
		return errResult(err)
	}
	return textResult(fmt.Sprintf("DAG %s → %s", args.TaskID, args.Status))
}

func (h *Hub) toolGetDAG(ctx context.Context, req *mcpsdk.CallToolRequest, args bcArgs) (*mcpsdk.CallToolResult, any, error) {
	if args.BusinessCode == "" {
		return errResult(errors.New("business_code required"))
	}
	if _, _, _, err := h.requireMembership(ctx, req, args.BusinessCode); err != nil {
		return errResult(err)
	}
	rows, err := h.Svc.Pool.Query(ctx,
		`SELECT task_id, title, status, COALESCE(assigned_worker,''), COALESCE(branch,''),
		        COALESCE(assignee_email,'')
		 FROM hub.hub_dag_state WHERE business_id=(SELECT id FROM hub.hub_businesses WHERE code=$1)
		 ORDER BY task_id`, args.BusinessCode)
	// If assignee_email column missing, fall back
	if err != nil {
		rows, err = h.Svc.Pool.Query(ctx,
			`SELECT task_id, title, status, COALESCE(assigned_worker,''), COALESCE(branch,''), ''
			 FROM hub.hub_dag_state WHERE business_id=(SELECT id FROM hub.hub_businesses WHERE code=$1)
			 ORDER BY task_id`, args.BusinessCode)
	}
	if err != nil {
		return errResult(err)
	}
	defer rows.Close()
	type row struct {
		id, title, status, worker, branch, assignee string
	}
	var tasks []row
	var ids []string
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.title, &r.status, &r.worker, &r.branch, &r.assignee); err != nil {
			continue
		}
		tasks = append(tasks, r)
		ids = append(ids, r.id)
	}
	// last actors
	actors := map[string]string{}
	if len(ids) > 0 {
		arows, err := h.Svc.Pool.Query(ctx, `
			SELECT DISTINCT ON (payload->>'task_id')
			  payload->>'task_id',
			  COALESCE(NULLIF(actor_email,''), actor, '')
			FROM hub.hub_events
			WHERE business_id=(SELECT id FROM hub.hub_businesses WHERE code=$1)
			  AND payload ? 'task_id'
			  AND payload->>'task_id' = ANY($2::text[])
			ORDER BY payload->>'task_id', created_at DESC`, args.BusinessCode, ids)
		if err == nil {
			defer arows.Close()
			for arows.Next() {
				var tid, email string
				if arows.Scan(&tid, &email) == nil {
					actors[tid] = email
				}
			}
		}
	}
	if len(tasks) == 0 {
		return textResult("No tasks.")
	}
	var lines []string
	for _, tsk := range tasks {
		mark := "⏳"
		if tsk.status == "completed" || tsk.status == "done" {
			mark = "✓"
		}
		line := fmt.Sprintf("%s %s %s [%s]", mark, tsk.id, tsk.title, tsk.status)
		if tsk.worker != "" {
			line += " worker=" + tsk.worker
		}
		if tsk.assignee != "" {
			line += " assignee=" + tsk.assignee
		}
		if em := actors[tsk.id]; em != "" {
			line += " last_by=" + em
		}
		if tsk.branch != "" {
			line += " @" + tsk.branch
		}
		lines = append(lines, line)
	}
	return textResult(strings.Join(lines, "\n"))
}

func (h *Hub) toolInviteMember(ctx context.Context, req *mcpsdk.CallToolRequest, args inviteArgs) (*mcpsdk.CallToolResult, any, error) {
	if args.Email == "" {
		return errResult(errors.New("email required"))
	}
	if args.BusinessCode == "" {
		return errResult(errors.New("business_code required"))
	}
	bizID, id, err := h.requireAdmin(ctx, req, args.BusinessCode)
	if err != nil {
		return errResult(err)
	}
	role := args.Role
	if role == "" {
		role = "member"
	}
	rawToken, err := genRawToken()
	if err != nil {
		return errResult(err)
	}
	th := hashToken(rawToken)
	var inviteID int64
	err = h.Svc.Pool.QueryRow(ctx,
		`INSERT INTO hub.hub_invites
		   (business_id, email, role, token_hash, invited_by, status, expires_at)
		 VALUES ($1, $2, $3, $4, $5, 'pending', now() + interval '7 days')
		 ON CONFLICT DO NOTHING
		 RETURNING id`,
		bizID, args.Email, role, th, id.UserID,
	).Scan(&inviteID)
	if err != nil {
		_, _ = h.Svc.Pool.Exec(ctx,
			`UPDATE hub.hub_invites SET status='revoked', updated_at=now()
			 WHERE business_id=$1 AND lower(email)=lower($2) AND status='pending'`,
			bizID, args.Email)
		err = h.Svc.Pool.QueryRow(ctx,
			`INSERT INTO hub.hub_invites
			   (business_id, email, role, token_hash, invited_by, status, expires_at)
			 VALUES ($1, $2, $3, $4, $5, 'pending', now() + interval '7 days')
			 RETURNING id`,
			bizID, args.Email, role, th, id.UserID,
		).Scan(&inviteID)
		if err != nil {
			return errResult(err)
		}
	}
	inviteURL := h.PublicBaseURL + "/invite/accept?token=" + rawToken
	return textResult(fmt.Sprintf("Invitation for %s. Share (token once): %s", args.Email, inviteURL))
}

func (h *Hub) toolAcceptInvitation(ctx context.Context, req *mcpsdk.CallToolRequest, args acceptInviteArgs) (*mcpsdk.CallToolResult, any, error) {
	id, err := h.requireUser(ctx, req)
	if err != nil {
		return errResult(err)
	}
	if args.Token != "" {
		if err := h.acceptInviteToken(ctx, id.UserID, args.Token); err != nil {
			return errResult(err)
		}
		return textResult("Invite accepted.")
	}
	if args.BusinessCode == "" {
		return errResult(errors.New("provide token (preferred) or business_code (only if join_policy=open)"))
	}
	var bizID int64
	var policy string
	err = h.Svc.Pool.QueryRow(ctx,
		`SELECT id, COALESCE(join_policy,'invite') FROM hub.hub_businesses WHERE code=$1`,
		args.BusinessCode,
	).Scan(&bizID, &policy)
	if err != nil {
		return errResult(fmt.Errorf("business not found: %s", args.BusinessCode))
	}
	if policy != "open" {
		return errResult(errors.New("business is not open; use invite token or create a link request"))
	}
	_, err = h.Svc.Pool.Exec(ctx,
		`INSERT INTO hub.hub_memberships (user_id, business_id, role, created_at)
		 VALUES ($1,$2,'member',now()) ON CONFLICT DO NOTHING`,
		id.UserID, bizID)
	if err != nil {
		return errResult(err)
	}
	return textResult("Joined business: " + args.BusinessCode)
}

func (h *Hub) acceptInviteToken(ctx context.Context, userID int64, rawToken string) error {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return errors.New("token required")
	}
	th := hashToken(rawToken)
	var inviteID, businessID int64
	var email, role, status string
	var expiresAt time.Time
	err := h.Svc.Pool.QueryRow(ctx,
		`SELECT id, business_id, email, role, status, expires_at
		 FROM hub.hub_invites WHERE token_hash=$1`, th,
	).Scan(&inviteID, &businessID, &email, &role, &status, &expiresAt)
	if err != nil {
		return errors.New("invalid invite token")
	}
	if status != "pending" {
		return fmt.Errorf("invite is %s", status)
	}
	if time.Now().After(expiresAt) {
		_, _ = h.Svc.Pool.Exec(ctx,
			`UPDATE hub.hub_invites SET status='expired', updated_at=now() WHERE id=$1`, inviteID)
		return errors.New("invite expired")
	}
	var userEmail string
	if err := h.Svc.Pool.QueryRow(ctx,
		"SELECT email FROM hub.hub_users WHERE id=$1", userID).Scan(&userEmail); err != nil {
		return errors.New("user not found")
	}
	if !strings.EqualFold(userEmail, email) {
		return errors.New("invite email does not match current user")
	}
	if role == "" {
		role = "member"
	}
	if _, err := h.Svc.Pool.Exec(ctx,
		`INSERT INTO hub.hub_memberships (user_id, business_id, role, created_at)
		 VALUES ($1, $2, $3, now()) ON CONFLICT DO NOTHING`,
		userID, businessID, role); err != nil {
		return err
	}
	_, err = h.Svc.Pool.Exec(ctx,
		`UPDATE hub.hub_invites
		 SET status='accepted', accepted_by=$1, accepted_at=now(), updated_at=now()
		 WHERE id=$2`, userID, inviteID)
	return err
}

func (h *Hub) toolCreateLinkRequest(ctx context.Context, req *mcpsdk.CallToolRequest, args linkReqArgs) (*mcpsdk.CallToolResult, any, error) {
	if args.BusinessCode == "" {
		return errResult(errors.New("business_code required"))
	}
	id, err := h.requireUser(ctx, req)
	if err != nil {
		return errResult(err)
	}
	var bizID int64
	err = h.Svc.Pool.QueryRow(ctx,
		"SELECT id FROM hub.hub_businesses WHERE code=$1", args.BusinessCode,
	).Scan(&bizID)
	if err != nil {
		return errResult(fmt.Errorf("business not found: %s", args.BusinessCode))
	}
	var role string
	err = h.Svc.Pool.QueryRow(ctx,
		"SELECT role FROM hub.hub_memberships WHERE user_id=$1 AND business_id=$2",
		id.UserID, bizID,
	).Scan(&role)
	if err == nil {
		return errResult(errors.New("already a member"))
	}
	var existingID int64
	err = h.Svc.Pool.QueryRow(ctx,
		"SELECT id FROM hub.hub_link_requests WHERE business_id=$1 AND user_id=$2 AND status='pending'",
		bizID, id.UserID,
	).Scan(&existingID)
	if err == nil {
		return errResult(errors.New("link request already pending"))
	}
	var linkID int64
	err = h.Svc.Pool.QueryRow(ctx,
		`INSERT INTO hub.hub_link_requests (business_id, user_id, device_info, status)
		 VALUES ($1, $2, $3, 'pending') RETURNING id`,
		bizID, id.UserID, args.DeviceInfo,
	).Scan(&linkID)
	if err != nil {
		return errResult(err)
	}
	return textResult(fmt.Sprintf("Link request created for %s (id=%d). An admin must approve it.", args.BusinessCode, linkID))
}

func (h *Hub) toolListLinkRequests(ctx context.Context, req *mcpsdk.CallToolRequest, args bcArgs) (*mcpsdk.CallToolResult, any, error) {
	if args.BusinessCode == "" {
		return errResult(errors.New("business_code required"))
	}
	bizID, _, err := h.requireAdmin(ctx, req, args.BusinessCode)
	if err != nil {
		return errResult(err)
	}
	rows, err := h.Svc.Pool.Query(ctx,
		`SELECT r.id, u.email, COALESCE(u.name,''), r.created_at
		 FROM hub.hub_link_requests r
		 JOIN hub.hub_users u ON u.id = r.user_id
		 WHERE r.business_id = $1 AND r.status = 'pending'
		 ORDER BY r.created_at DESC`, bizID)
	if err != nil {
		return errResult(err)
	}
	defer rows.Close()
	var lines []string
	for rows.Next() {
		var id int64
		var email, name string
		var created time.Time
		if err := rows.Scan(&id, &email, &name, &created); err != nil {
			continue
		}
		label := name
		if label == "" {
			label = email
		}
		lines = append(lines, fmt.Sprintf("[%d] %s — %s", id, label, created.Format("2006-01-02 15:04")))
	}
	if len(lines) == 0 {
		return textResult("No pending link requests.")
	}
	return textResult(strings.Join(lines, "\n"))
}

func (h *Hub) toolReviewLinkRequest(ctx context.Context, req *mcpsdk.CallToolRequest, args reviewLinkArgs) (*mcpsdk.CallToolResult, any, error) {
	if args.BusinessCode == "" || args.RequestID == 0 {
		return errResult(errors.New("business_code and request_id required"))
	}
	if args.Action != "approve" && args.Action != "reject" {
		return errResult(errors.New("action must be 'approve' or 'reject'"))
	}
	bizID, id, err := h.requireAdmin(ctx, req, args.BusinessCode)
	if err != nil {
		return errResult(err)
	}
	var requestUserID int64
	var status string
	err = h.Svc.Pool.QueryRow(ctx,
		"SELECT user_id, status FROM hub.hub_link_requests WHERE id=$1 AND business_id=$2",
		args.RequestID, bizID,
	).Scan(&requestUserID, &status)
	if err != nil {
		return errResult(errors.New("link request not found"))
	}
	if status != "pending" {
		return errResult(fmt.Errorf("link request is already %s", status))
	}
	if requestUserID == id.UserID {
		return errResult(errors.New("cannot review your own link request"))
	}
	if args.Action == "approve" {
		_, err = h.Svc.Pool.Exec(ctx,
			"INSERT INTO hub.hub_memberships (user_id, business_id, role, created_at) VALUES ($1, $2, 'member', now()) ON CONFLICT DO NOTHING",
			requestUserID, bizID)
		if err != nil {
			return errResult(err)
		}
	}
	_, err = h.Svc.Pool.Exec(ctx,
		"UPDATE hub.hub_link_requests SET status=$1, reviewed_by=$2, reviewed_at=now(), updated_at=now() WHERE id=$3",
		args.Action+"d", id.UserID, args.RequestID)
	if err != nil {
		return errResult(err)
	}
	return textResult(fmt.Sprintf("Link request %d %sd.", args.RequestID, args.Action))
}

func (h *Hub) toolReportBranches(ctx context.Context, req *mcpsdk.CallToolRequest, args reportBranchesArgs) (*mcpsdk.CallToolResult, any, error) {
	if args.BusinessCode == "" {
		return errResult(errors.New("business_code required"))
	}
	if len(args.Branches) == 0 && len(args.Bindings) == 0 {
		return errResult(errors.New("branches or bindings required"))
	}
	bizID, _, id, err := h.requireMembership(ctx, req, args.BusinessCode)
	if err != nil {
		return errResult(err)
	}
	repoID, err := h.resolveBizRepo(ctx, bizID, args.RepoURL)
	if err != nil || repoID == 0 {
		return errResult(errors.New("business or repo not found; bind a repo first or pass repo_url"))
	}
	reporter := args.Reporter
	if reporter == "" && id != nil {
		reporter = id.Email
	}
	if reporter == "" {
		reporter = "worker"
	}
	upserted := 0
	for _, b := range args.Branches {
		name := strings.TrimSpace(b.Name)
		if name == "" {
			continue
		}
		src := b.Source
		if src == "" {
			src = "report"
		}
		_, err := h.Svc.Pool.Exec(ctx,
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
			bizID, repoID, name, strings.TrimSpace(b.TipSHA), reporter, src,
		)
		if err != nil {
			return errResult(err)
		}
		upserted++
	}
	bound := 0
	for _, b := range args.Bindings {
		if err := h.upsertBinding(ctx, bizID, b); err != nil {
			return errResult(err)
		}
		bound++
	}
	return textResult(fmt.Sprintf("Branches upserted=%d bindings=%d repo_id=%d", upserted, bound, repoID))
}

func (h *Hub) toolListBranches(ctx context.Context, req *mcpsdk.CallToolRequest, args bcArgs) (*mcpsdk.CallToolResult, any, error) {
	if args.BusinessCode == "" {
		return errResult(errors.New("business_code required"))
	}
	bizID, _, _, err := h.requireMembership(ctx, req, args.BusinessCode)
	if err != nil {
		return errResult(err)
	}
	rows, err := h.Svc.Pool.Query(ctx,
		`SELECT b.name, COALESCE(b.tip_sha,''), b.is_default, COALESCE(b.source,'')
		 FROM hub.hub_branches b
		 WHERE b.business_id=$1
		 ORDER BY b.name`, bizID)
	if err != nil {
		return errResult(err)
	}
	defer rows.Close()
	var lines []string
	for rows.Next() {
		var name, tip, src string
		var isDef bool
		if err := rows.Scan(&name, &tip, &isDef, &src); err != nil {
			continue
		}
		sha := tip
		if len(sha) > 8 {
			sha = sha[:8]
		}
		if sha == "" {
			sha = "--------"
		}
		line := fmt.Sprintf("%s @ %s [%s]", name, sha, src)
		if isDef {
			line += " (default)"
		}
		lines = append(lines, line)
	}
	brows, err := h.Svc.Pool.Query(ctx,
		`SELECT branch_name, bind_type, bind_id, status
		 FROM hub.hub_branch_bindings WHERE business_id=$1 ORDER BY branch_name, bind_type`, bizID)
	if err != nil {
		return errResult(err)
	}
	defer brows.Close()
	var bindLines []string
	for brows.Next() {
		var branch, bt, bid, status string
		if err := brows.Scan(&branch, &bt, &bid, &status); err != nil {
			continue
		}
		bindLines = append(bindLines, fmt.Sprintf("  bind %s:%s → %s [%s]", bt, bid, branch, status))
	}
	if len(lines) == 0 && len(bindLines) == 0 {
		return textResult("No branches reported.")
	}
	out := strings.Join(lines, "\n")
	if len(bindLines) > 0 {
		if out != "" {
			out += "\n"
		}
		out += strings.Join(bindLines, "\n")
	}
	return textResult(out)
}

func (h *Hub) toolBindBranch(ctx context.Context, req *mcpsdk.CallToolRequest, args bindBranchArgs) (*mcpsdk.CallToolResult, any, error) {
	if args.BusinessCode == "" {
		return errResult(errors.New("business_code required"))
	}
	if args.BindType == "" || args.BindID == "" || args.BranchName == "" {
		return errResult(errors.New("bind_type, bind_id, branch_name required"))
	}
	bizID, _, _, err := h.requireMembership(ctx, req, args.BusinessCode)
	if err != nil {
		return errResult(err)
	}
	if err := h.upsertBinding(ctx, bizID, bindItem{
		BindType: args.BindType, BindID: args.BindID, BranchName: args.BranchName,
		HeadSHA: args.HeadSHA, WorktreeHost: args.WorktreeHost, Status: args.Status,
	}); err != nil {
		return errResult(err)
	}
	return textResult(fmt.Sprintf("Bound %s:%s → %s", args.BindType, args.BindID, args.BranchName))
}

func (h *Hub) toolRefreshBranches(ctx context.Context, req *mcpsdk.CallToolRequest, args refreshBranchesArgs) (*mcpsdk.CallToolResult, any, error) {
	if args.BusinessCode == "" {
		return errResult(errors.New("business_code required"))
	}
	bizID, _, _, err := h.requireMembership(ctx, req, args.BusinessCode)
	if err != nil {
		return errResult(err)
	}
	repoID, err := h.resolveBizRepo(ctx, bizID, args.RepoURL)
	if err != nil || repoID == 0 {
		return errResult(errors.New("business or repo not found; bind a repo first"))
	}
	var repoURL, defaultBranch string
	if err := h.Svc.Pool.QueryRow(ctx,
		`SELECT repo_url, COALESCE(default_branch,'main') FROM hub.hub_repos WHERE id=$1`, repoID,
	).Scan(&repoURL, &defaultBranch); err != nil {
		return errResult(err)
	}
	// Prefer git ls-remote (no GitHub token required for public repos).
	tips, src, err := lsRemoteTips(ctx, repoURL, defaultBranch, args.DefaultOnly)
	if err != nil {
		return errResult(fmt.Errorf("refresh failed: %w", err))
	}
	upserted := 0
	for _, t := range tips {
		name := strings.TrimSpace(t.name)
		if name == "" {
			continue
		}
		_, err := h.Svc.Pool.Exec(ctx,
			`INSERT INTO hub.hub_branches
			   (business_id, repo_id, name, tip_sha, is_default, last_reporter, source, last_seen_at, updated_at)
			 VALUES ($1,$2,$3,$4,$5,'hub-refresh',$6,now(),now())
			 ON CONFLICT (repo_id, name) DO UPDATE SET
			   tip_sha = EXCLUDED.tip_sha,
			   is_default = EXCLUDED.is_default,
			   last_reporter = EXCLUDED.last_reporter,
			   source = EXCLUDED.source,
			   last_seen_at = now(),
			   updated_at = now()`,
			bizID, repoID, name, t.sha, t.isDefault, src,
		)
		if err != nil {
			return errResult(err)
		}
		upserted++
	}
	return textResult(fmt.Sprintf("Refreshed branches_upserted=%d source=%s repo=%s", upserted, src, repoURL))
}

type tip struct {
	name      string
	sha       string
	isDefault bool
}

func lsRemoteTips(ctx context.Context, repoURL, defaultBranch string, defaultOnly bool) ([]tip, string, error) {
	cmd := exec.CommandContext(ctx, "git", "ls-remote", "--heads", repoURL)
	out, err := cmd.Output()
	if err != nil {
		return nil, "", fmt.Errorf("git ls-remote: %w", err)
	}
	var tips []tip
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		sha, ref := parts[0], parts[1]
		if !strings.HasPrefix(ref, "refs/heads/") {
			continue
		}
		name := strings.TrimPrefix(ref, "refs/heads/")
		isDef := name == defaultBranch
		if defaultOnly && !isDef {
			continue
		}
		tips = append(tips, tip{name: name, sha: sha, isDefault: isDef})
	}
	if len(tips) == 0 {
		return nil, "", errors.New("no heads returned by ls-remote")
	}
	return tips, "ls_remote", nil
}

func (h *Hub) toolUpsertDoc(ctx context.Context, req *mcpsdk.CallToolRequest, args upsertDocArgs) (*mcpsdk.CallToolResult, any, error) {
	if args.BusinessCode == "" || args.DocKey == "" || args.Title == "" {
		return errResult(errors.New("business_code, doc_key, title required"))
	}
	if len(args.Content) > 64*1024 {
		return errResult(errors.New("content too large (max 64KB)"))
	}
	_, _, id, err := h.requireMembership(ctx, req, args.BusinessCode)
	if err != nil {
		return errResult(err)
	}
	vis := strings.ToLower(strings.TrimSpace(args.Visibility))
	if vis == "" {
		vis = "team"
	}
	if vis != "team" && vis != "private" {
		return errResult(errors.New("visibility must be team or private"))
	}
	cat := strings.TrimSpace(args.Category)
	if cat == "" {
		cat = "general"
	}
	var bizID int64
	if err := h.Svc.Pool.QueryRow(ctx, `SELECT id FROM hub.hub_businesses WHERE code=$1`, args.BusinessCode).Scan(&bizID); err != nil {
		return errResult(err)
	}
	var uid *int64
	email := ""
	if id != nil && id.UserID > 0 {
		u := id.UserID
		uid = &u
		email = id.Email
	}
	var docID int64
	err = h.Svc.Pool.QueryRow(ctx, `
		INSERT INTO hub.hub_team_docs (
		  business_id, doc_key, title, content, category, visibility,
		  created_by_user_id, updated_by_user_id, created_by_email, updated_by_email, source,
		  created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$7,$8,$8,'mcp', now(), now())
		ON CONFLICT (business_id, doc_key) DO UPDATE SET
		  title=EXCLUDED.title, content=EXCLUDED.content, category=EXCLUDED.category,
		  visibility=EXCLUDED.visibility, updated_by_user_id=EXCLUDED.updated_by_user_id,
		  updated_by_email=EXCLUDED.updated_by_email, source='mcp', updated_at=now()
		RETURNING id`,
		bizID, strings.TrimSpace(args.DocKey), strings.TrimSpace(args.Title), args.Content, cat, vis, uid, email,
	).Scan(&docID)
	if err != nil {
		return errResult(err)
	}
	actor := email
	if actor == "" {
		actor = "unknown"
	}
	_, _ = h.Svc.AppendEventFull(ctx, service.AppendEventInput{
		BusinessCode: args.BusinessCode,
		Actor:        actor,
		EventType:    "doc.updated",
		Payload:      map[string]interface{}{"doc_key": args.DocKey, "title": args.Title},
		ActorUserID:  uid,
		ActorEmail:   email,
		ActorRole:    "worker",
	})
	return textResult(fmt.Sprintf("Doc %s upserted (id=%d) by %s", args.DocKey, docID, email))
}

func (h *Hub) toolListDocs(ctx context.Context, req *mcpsdk.CallToolRequest, args bcArgs) (*mcpsdk.CallToolResult, any, error) {
	if args.BusinessCode == "" {
		return errResult(errors.New("business_code required"))
	}
	_, _, id, err := h.requireMembership(ctx, req, args.BusinessCode)
	if err != nil {
		return errResult(err)
	}
	var uid int64
	if id != nil {
		uid = id.UserID
	}
	rows, err := h.Svc.Pool.Query(ctx, `
		SELECT d.doc_key, d.title, d.category, d.visibility, COALESCE(d.updated_by_email,''), d.updated_at
		FROM hub.hub_team_docs d
		JOIN hub.hub_businesses b ON b.id=d.business_id
		WHERE b.code=$1 AND (d.visibility='team' OR d.created_by_user_id=$2)
		ORDER BY d.updated_at DESC LIMIT 50`, args.BusinessCode, uid)
	if err != nil {
		return errResult(err)
	}
	defer rows.Close()
	var lines []string
	for rows.Next() {
		var key, title, cat, vis, by string
		var updated time.Time
		if err := rows.Scan(&key, &title, &cat, &vis, &by, &updated); err != nil {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s — %s [%s] by %s @ %s", key, title, cat, by, updated.Format("2006-01-02 15:04")))
	}
	if len(lines) == 0 {
		return textResult("No team docs.")
	}
	return textResult(strings.Join(lines, "\n"))
}

func (h *Hub) toolGetDoc(ctx context.Context, req *mcpsdk.CallToolRequest, args getDocArgs) (*mcpsdk.CallToolResult, any, error) {
	if args.BusinessCode == "" || args.DocKey == "" {
		return errResult(errors.New("business_code and doc_key required"))
	}
	_, _, id, err := h.requireMembership(ctx, req, args.BusinessCode)
	if err != nil {
		return errResult(err)
	}
	var uid int64
	if id != nil {
		uid = id.UserID
	}
	var title, content, cat, vis, by string
	err = h.Svc.Pool.QueryRow(ctx, `
		SELECT d.title, d.content, d.category, d.visibility, COALESCE(d.updated_by_email,'')
		FROM hub.hub_team_docs d
		JOIN hub.hub_businesses b ON b.id=d.business_id
		WHERE b.code=$1 AND d.doc_key=$2 AND (d.visibility='team' OR d.created_by_user_id=$3)`,
		args.BusinessCode, args.DocKey, uid,
	).Scan(&title, &content, &cat, &vis, &by)
	if err != nil {
		return errResult(errors.New("doc not found"))
	}
	return textResult(fmt.Sprintf("# %s\n\ncategory=%s visibility=%s updated_by=%s\n\n%s", title, cat, vis, by, content))
}

func (h *Hub) toolPublishWorkerTemplate(ctx context.Context, req *mcpsdk.CallToolRequest, args publishWorkerTemplateArgs) (*mcpsdk.CallToolResult, any, error) {
	if args.BusinessCode == "" || args.WorkerID == "" {
		return errResult(errors.New("business_code and worker_id required"))
	}
	if len(args.PromptSummary) > 8000 {
		return errResult(errors.New("prompt_summary too large"))
	}
	_, _, id, err := h.requireMembership(ctx, req, args.BusinessCode)
	if err != nil {
		return errResult(err)
	}
	vis := strings.ToLower(strings.TrimSpace(args.Visibility))
	if vis == "" {
		vis = "team"
	}
	if vis != "team" && vis != "private" {
		return errResult(errors.New("visibility must be team or private"))
	}
	if args.Skills == nil {
		args.Skills = []string{}
	}
	if args.Handbook == nil {
		args.Handbook = map[string]interface{}{}
	}
	hb, err := json.Marshal(args.Handbook)
	if err != nil {
		return errResult(errors.New("invalid handbook"))
	}
	display := strings.TrimSpace(args.DisplayName)
	if display == "" {
		display = args.WorkerID
	}
	var bizID int64
	if err := h.Svc.Pool.QueryRow(ctx, `SELECT id FROM hub.hub_businesses WHERE code=$1`, args.BusinessCode).Scan(&bizID); err != nil {
		return errResult(err)
	}
	var uid *int64
	email := ""
	if id != nil && id.UserID > 0 {
		u := id.UserID
		uid = &u
		email = id.Email
	}
	var tid int64
	err = h.Svc.Pool.QueryRow(ctx, `
		INSERT INTO hub.hub_worker_templates (
		  business_id, worker_id, display_name, description, skills, handbook, prompt_summary,
		  published_by_user_id, published_by_email, visibility, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5::text[],$6::jsonb,$7,$8,$9,$10, now(), now())
		ON CONFLICT (business_id, worker_id) DO UPDATE SET
		  display_name=EXCLUDED.display_name, description=EXCLUDED.description, skills=EXCLUDED.skills,
		  handbook=EXCLUDED.handbook, prompt_summary=EXCLUDED.prompt_summary,
		  published_by_user_id=EXCLUDED.published_by_user_id, published_by_email=EXCLUDED.published_by_email,
		  visibility=EXCLUDED.visibility, updated_at=now()
		RETURNING id`,
		bizID, strings.TrimSpace(args.WorkerID), display, args.Description, args.Skills, hb, args.PromptSummary,
		uid, email, vis,
	).Scan(&tid)
	if err != nil {
		return errResult(err)
	}
	actor := email
	if actor == "" {
		actor = "unknown"
	}
	_, _ = h.Svc.AppendEventFull(ctx, service.AppendEventInput{
		BusinessCode:  args.BusinessCode,
		Actor:         actor,
		EventType:     "worker.template_published",
		Payload:       map[string]interface{}{"worker_id": args.WorkerID, "display_name": display},
		ActorUserID:   uid,
		ActorEmail:    email,
		ActorRole:     "leader",
		ActorWorkerID: args.WorkerID,
	})
	return textResult(fmt.Sprintf("Worker template %s published (id=%d) by %s", args.WorkerID, tid, email))
}

func (h *Hub) toolListWorkerTemplates(ctx context.Context, req *mcpsdk.CallToolRequest, args bcArgs) (*mcpsdk.CallToolResult, any, error) {
	if args.BusinessCode == "" {
		return errResult(errors.New("business_code required"))
	}
	_, _, id, err := h.requireMembership(ctx, req, args.BusinessCode)
	if err != nil {
		return errResult(err)
	}
	var uid int64
	if id != nil {
		uid = id.UserID
	}
	rows, err := h.Svc.Pool.Query(ctx, `
		SELECT t.worker_id, COALESCE(t.display_name,''), COALESCE(t.published_by_email,''), t.visibility
		FROM hub.hub_worker_templates t
		JOIN hub.hub_businesses b ON b.id=t.business_id
		WHERE b.code=$1 AND (t.visibility='team' OR t.published_by_user_id=$2)
		ORDER BY t.updated_at DESC LIMIT 50`, args.BusinessCode, uid)
	if err != nil {
		return errResult(err)
	}
	defer rows.Close()
	var lines []string
	for rows.Next() {
		var wid, name, by, vis string
		if err := rows.Scan(&wid, &name, &by, &vis); err != nil {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s — %s (by %s, %s)", wid, name, by, vis))
	}
	if len(lines) == 0 {
		return textResult("No worker templates.")
	}
	return textResult(strings.Join(lines, "\n"))
}
