package handler

import "github.com/stifer/agent-hub/internal/hub/service"

type Handler struct {
	Svc       *service.Service
	JWTSecret string
}

func New(svc *service.Service, jwtSecret string) *Handler {
	return &Handler{Svc: svc, JWTSecret: jwtSecret}
}
