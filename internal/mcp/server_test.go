package mcp_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stifer/agent-hub/internal/mcp"
)

// Smoke: Streamable HTTP initialize + tools/list without DB (server construction only).
func TestMCPInitializeAndToolsList(t *testing.T) {
	// Hub with nil Svc is fine for protocol surface; tool calls that hit DB would panic.
	h := mcp.New(nil, "test-secret")
	handler := h.HTTPHandler()
	srv := httptest.NewServer(handler)
	defer srv.Close()

	// initialize
	initBody := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test","version":"0"}}}`
	resp := mustPost(t, srv.URL, initBody, "")
	if !strings.Contains(resp, `"serverInfo"`) && !strings.Contains(resp, `"result"`) {
		t.Fatalf("initialize response unexpected: %s", resp)
	}
	// extract session id if any
	// tools/list
	listBody := `{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`
	listResp := mustPost(t, srv.URL, listBody, extractSession(resp))
	// Session may be required; if 400, retry with Mcp-Session-Id from initialize headers via helper
	if strings.Contains(listResp, "hub_login") {
		// good path
	} else {
		// Try using session from Set-Cookie / header by re-posting through shared client
		t.Logf("tools/list first response: %s", truncate(listResp, 400))
	}

	// Count tools via a fresh session-aware client
	client := srv.Client()
	req, _ := http.NewRequest(http.MethodPost, srv.URL, strings.NewReader(initBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	sessionID := res.Header.Get("Mcp-Session-Id")
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	t.Logf("init status=%d session=%s body=%s", res.StatusCode, sessionID, truncate(string(body), 200))

	req2, _ := http.NewRequest(http.MethodPost, srv.URL, strings.NewReader(listBody))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Accept", "application/json, text/event-stream")
	if sessionID != "" {
		req2.Header.Set("Mcp-Session-Id", sessionID)
	}
	res2, err := client.Do(req2)
	if err != nil {
		t.Fatal(err)
	}
	body2, _ := io.ReadAll(res2.Body)
	res2.Body.Close()
	text := string(body2)
	t.Logf("tools/list status=%d body=%s", res2.StatusCode, truncate(text, 500))

	// Parse tools
	var envelope struct {
		Result struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		} `json:"result"`
	}
	// May be SSE or plain JSON
	jsonPart := text
	if i := strings.Index(text, "{"); i >= 0 {
		// find last complete JSON object roughly
		jsonPart = text[i:]
	}
	if err := json.Unmarshal([]byte(jsonPart), &envelope); err != nil {
		// try line by line for SSE data:
		for _, line := range strings.Split(text, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "data:") {
				payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
				if json.Unmarshal([]byte(payload), &envelope) == nil && len(envelope.Result.Tools) > 0 {
					break
				}
			}
		}
	}
	if len(envelope.Result.Tools) == 0 {
		// Fallback: substring count of known tools
		names := []string{
			"hub_login", "hub_list_my_businesses", "hub_heartbeat", "hub_acquire_lock",
			"hub_release_lock", "hub_renew_lock", "hub_append_event", "hub_create_playbook",
			"hub_search_playbooks", "hub_list_workers", "hub_list_locks", "hub_list_events",
			"hub_add_repo", "hub_sync_dag", "hub_get_dag", "hub_invite_member",
			"hub_accept_invitation", "hub_create_link_request", "hub_list_link_requests",
			"hub_review_link_request", "hub_report_branches", "hub_list_branches",
			"hub_bind_branch", "hub_refresh_branches",
		}
		missing := []string{}
		for _, n := range names {
			if !strings.Contains(text, n) {
				missing = append(missing, n)
			}
		}
		if len(missing) > 0 {
			t.Fatalf("expected 24 tools in tools/list; missing %v; body=%s", missing, truncate(text, 800))
		}
		return
	}
	if len(envelope.Result.Tools) < 24 {
		t.Fatalf("expected >=24 tools, got %d", len(envelope.Result.Tools))
	}
}

func mustPost(t *testing.T, url, body, session string) string {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if session != "" {
		req.Header.Set("Mcp-Session-Id", session)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	return string(b)
}

func extractSession(_ string) string { return "" }

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
