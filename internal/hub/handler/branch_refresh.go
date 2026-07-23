package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type remoteTip struct {
	Name      string
	SHA       string
	Source    string
	IsDefault bool
	PRNumber  *int
	PRURL     *string
	PRState   *string
}

// RefreshBranches pulls tip SHAs (and PR metadata for github.com) and upserts
// hub_branches with source=github_api or ls_remote (wins over report for tip).
//
// Body optional: { "repo_url": "...", "default_only": false }
func (h *Handler) RefreshBranches(c *gin.Context) {
	code := c.Param("code")
	if _, _, ok := h.RequireMembership(c, code); !ok {
		return
	}
	var req struct {
		RepoURL     string `json:"repo_url"`
		DefaultOnly bool   `json:"default_only"`
	}
	_ = c.ShouldBindJSON(&req)

	bizID, repoID, err := h.resolveBizAndRepo(c, code, req.RepoURL)
	if err != nil || repoID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "business or repo not found; bind a repo first"})
		return
	}

	var repoURL, defaultBranch, provider string
	if err := h.Svc.Pool.QueryRow(c.Request.Context(),
		`SELECT repo_url, COALESCE(default_branch,'main'), COALESCE(provider,'generic')
		 FROM hub.hub_repos WHERE id=$1`, repoID,
	).Scan(&repoURL, &defaultBranch, &provider); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN"))
	if token == "" {
		token = strings.TrimSpace(os.Getenv("HUB_GITHUB_TOKEN"))
	}

	owner, repo, isGH := parseGitHubOwnerRepo(repoURL)
	var tips []remoteTip
	if isGH && token != "" {
		tips, err = fetchGitHubBranches(c, token, owner, repo, defaultBranch, req.DefaultOnly)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"code": 502, "message": "github api: " + err.Error()})
			return
		}
	} else {
		tips, err = fetchLSRemoteBranches(repoURL, defaultBranch, req.DefaultOnly)
		if err != nil {
			msg := err.Error()
			if !isGH {
				msg = "not a github url and ls-remote failed: " + msg
			} else if token == "" {
				msg = "GITHUB_TOKEN unset and ls-remote failed: " + msg
			}
			c.JSON(http.StatusBadGateway, gin.H{"code": 502, "message": msg})
			return
		}
	}

	if isGH && token != "" {
		for i := range tips {
			if tips[i].IsDefault {
				continue
			}
			n, url, state, e := fetchOpenPR(c, token, owner, repo, tips[i].Name)
			if e == nil && n > 0 {
				tips[i].PRNumber = &n
				tips[i].PRURL = &url
				tips[i].PRState = &state
			}
		}
	}

	srcUsed := "ls_remote"
	if len(tips) > 0 && tips[0].Source != "" {
		srcUsed = tips[0].Source
	}

	upserted := 0
	for _, t := range tips {
		name := strings.TrimSpace(t.Name)
		if name == "" {
			continue
		}
		src := t.Source
		if src == "" {
			src = "ls_remote"
		}
		_, err := h.Svc.Pool.Exec(c.Request.Context(),
			`INSERT INTO hub.hub_branches
			   (business_id, repo_id, name, tip_sha, is_default, pr_number, pr_url, pr_state,
			    last_reporter, source, last_seen_at, updated_at)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,'hub-refresh',$9,now(),now())
			 ON CONFLICT (repo_id, name) DO UPDATE SET
			   tip_sha = EXCLUDED.tip_sha,
			   is_default = EXCLUDED.is_default,
			   pr_number = COALESCE(EXCLUDED.pr_number, hub.hub_branches.pr_number),
			   pr_url = COALESCE(EXCLUDED.pr_url, hub.hub_branches.pr_url),
			   pr_state = COALESCE(EXCLUDED.pr_state, hub.hub_branches.pr_state),
			   last_reporter = EXCLUDED.last_reporter,
			   source = EXCLUDED.source,
			   last_seen_at = now(),
			   updated_at = now()`,
			bizID, repoID, name, t.SHA, t.IsDefault, t.PRNumber, t.PRURL, t.PRState, src,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
			return
		}
		upserted++
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"branches_upserted": upserted,
		"repo_url":          repoURL,
		"provider":          provider,
		"source":            srcUsed,
	}})
}

func parseGitHubOwnerRepo(repoURL string) (owner, repo string, ok bool) {
	u := strings.TrimSpace(repoURL)
	u = strings.TrimSuffix(u, ".git")
	if strings.HasPrefix(u, "git@github.com:") {
		rest := strings.TrimPrefix(u, "git@github.com:")
		parts := strings.Split(rest, "/")
		if len(parts) >= 2 {
			return parts[0], parts[1], true
		}
	}
	low := strings.ToLower(u)
	if i := strings.Index(low, "github.com/"); i >= 0 {
		rest := u[i+len("github.com/"):]
		parts := strings.Split(strings.Trim(rest, "/"), "/")
		if len(parts) >= 2 {
			return parts[0], strings.TrimSuffix(parts[1], ".git"), true
		}
	}
	if i := strings.Index(low, "github.com:"); i >= 0 {
		rest := u[i+len("github.com:"):]
		parts := strings.Split(strings.Trim(rest, "/"), "/")
		if len(parts) >= 2 {
			return parts[0], strings.TrimSuffix(parts[1], ".git"), true
		}
	}
	return "", "", false
}

func fetchGitHubBranches(c *gin.Context, token, owner, repo, defaultBranch string, defaultOnly bool) ([]remoteTip, error) {
	type ghBranch struct {
		Name   string `json:"name"`
		Commit struct {
			SHA string `json:"sha"`
		} `json:"commit"`
	}
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/branches?per_page=100", owner, repo)
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, truncate(string(body), 200))
	}
	var list []ghBranch
	if err := json.Unmarshal(body, &list); err != nil {
		return nil, err
	}
	out := make([]remoteTip, 0, len(list))
	for _, b := range list {
		if defaultOnly && b.Name != defaultBranch {
			continue
		}
		out = append(out, remoteTip{
			Name:      b.Name,
			SHA:       b.Commit.SHA,
			Source:    "github_api",
			IsDefault: b.Name == defaultBranch,
		})
	}
	return out, nil
}

func fetchOpenPR(c *gin.Context, token, owner, repo, branch string) (number int, prURL, state string, err error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls?state=open&head=%s:%s&per_page=1",
		owner, repo, owner, branch)
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, url, nil)
	if err != nil {
		return 0, "", "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return 0, "", "", fmt.Errorf("pr status %d", resp.StatusCode)
	}
	var prs []struct {
		Number  int    `json:"number"`
		HTMLURL string `json:"html_url"`
		State   string `json:"state"`
	}
	if err := json.Unmarshal(body, &prs); err != nil {
		return 0, "", "", err
	}
	if len(prs) == 0 {
		return 0, "", "", nil
	}
	return prs[0].Number, prs[0].HTMLURL, prs[0].State, nil
}

func fetchLSRemoteBranches(repoURL, defaultBranch string, defaultOnly bool) ([]remoteTip, error) {
	if repoURL == "" {
		return nil, fmt.Errorf("empty repo url")
	}
	cmd := exec.Command("git", "ls-remote", "--heads", repoURL)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%v: %s", err, truncate(string(out), 200))
	}
	lines := strings.Split(string(out), "\n")
	result := make([]remoteTip, 0)
	for _, line := range lines {
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
		if defaultOnly && name != defaultBranch {
			continue
		}
		result = append(result, remoteTip{
			Name:      name,
			SHA:       sha,
			Source:    "ls_remote",
			IsDefault: name == defaultBranch,
		})
	}
	return result, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
