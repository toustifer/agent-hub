package service

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stifer/agent-hub/ent"
)

type Service struct {
	Client *ent.Client
	Pool   *pgxpool.Pool
}

func New(client *ent.Client, pool *pgxpool.Pool) *Service {
	return &Service{Client: client, Pool: pool}
}
