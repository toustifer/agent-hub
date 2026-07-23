package mcp_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stifer/agent-hub/internal/mcp"
)

func TestAcceptInvitationOptionalFields(t *testing.T) {
	h := mcp.New(nil, "test-secret")
	srv := httptest.NewServer(h.HTTPHandler())
	defer srv.Close()
	client := srv.Client()
	initBody := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"t","version":"0"}}}`
	req, _ := http.NewRequest(http.MethodPost, srv.URL, strings.NewReader(initBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	res, err := client.Do(req)
	if err != nil { t.Fatal(err) }
	sid := res.Header.Get("Mcp-Session-Id")
	res.Body.Close()
	listBody := `{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`
	req2, _ := http.NewRequest(http.MethodPost, srv.URL, strings.NewReader(listBody))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Accept", "application/json, text/event-stream")
	if sid != "" { req2.Header.Set("Mcp-Session-Id", sid) }
	res2, err := client.Do(req2)
	if err != nil { t.Fatal(err) }
	b, _ := io.ReadAll(res2.Body)
	res2.Body.Close()
	text := string(b)
	// Find hub_accept_invitation block
	i := strings.Index(text, "hub_accept_invitation")
	if i < 0 { t.Fatal("tool missing") }
	snippet := text[i:min(i+500, len(text))]
	t.Log(snippet)
	// required should not force both if omitempty worked - empty required array or only none
	var env map[string]any
	json.Unmarshal(b, &env)
	// soft assert: if required present with both, warn
	if strings.Contains(snippet, `"required":["token","business_code"]`) {
		t.Fatalf("accept invitation still requires both fields: %s", snippet)
	}
}
func min(a,b int) int { if a<b {return a}; return b }
