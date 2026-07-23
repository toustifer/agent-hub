// Package mcp implements the official Streamable HTTP MCP server for Agent Hub.
// Claude (and other hosts) connect to https://hub.stifer.xyz/mcp with OAuth/JWT.
package mcp

import (
	"net/http"
	"os"
	"strings"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stifer/agent-hub/internal/hub/service"
	"github.com/stifer/agent-hub/internal/version"
)

// Hub is the MCP tool surface over the existing Hub service layer.
type Hub struct {
	Svc          *service.Service
	JWTSecret    string
	PublicBaseURL string
}

// New builds a Hub MCP surface.
func New(svc *service.Service, jwtSecret string) *Hub {
	base := strings.TrimRight(os.Getenv("HUB_PUBLIC_URL"), "/")
	if base == "" {
		base = "https://hub.stifer.xyz"
	}
	return &Hub{Svc: svc, JWTSecret: jwtSecret, PublicBaseURL: base}
}

// Server returns a configured MCP server with all Hub tools registered.
func (h *Hub) Server() *mcpsdk.Server {
	s := mcpsdk.NewServer(&mcpsdk.Implementation{
		Name:    "agent-hub",
		Version: version.Version,
	}, &mcpsdk.ServerOptions{
		Instructions: "Agent Hub remote MCP. Authenticate with OAuth (preferred) or Authorization: Bearer <jwt>. Machine tools also accept X-API-Key + X-Business-Code.",
	})
	h.registerTools(s)
	return s
}

// HTTPHandler returns a Streamable HTTP handler for /mcp.
// JSONResponse keeps Inspector/curl simple; sessions remain stateful by default.
//
// DisableLocalhostProtection is required when Hub sits behind nginx (or any
// reverse proxy) that dials 127.0.0.1 while forwarding Host: hub.stifer.xyz —
// the go-sdk otherwise rejects those requests as DNS-rebinding (403).
func (h *Hub) HTTPHandler() http.Handler {
	server := h.Server()
	return mcpsdk.NewStreamableHTTPHandler(func(r *http.Request) *mcpsdk.Server {
		return server
	}, &mcpsdk.StreamableHTTPOptions{
		JSONResponse:               true,
		DisableLocalhostProtection: true,
	})
}

func textResult(s string) (*mcpsdk.CallToolResult, any, error) {
	return &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: s}},
	}, nil, nil
}

func errResult(err error) (*mcpsdk.CallToolResult, any, error) {
	if err == nil {
		return textResult("ok")
	}
	// Tool-level error (visible to the model), not protocol error.
	return &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: err.Error()}},
		IsError: true,
	}, nil, nil
}
