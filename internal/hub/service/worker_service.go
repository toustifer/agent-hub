package service

import (
	"context"
	"fmt"
	"time"

	"github.com/stifer/agent-hub/ent"
	"github.com/stifer/agent-hub/ent/hubbusiness"
	"github.com/stifer/agent-hub/ent/hubworker"
)

func (s *Service) Heartbeat(ctx context.Context, businessCode, workerID, version, host string, pid int) (*ent.HubWorker, error) {
	biz, err := s.Client.HubBusiness.Query().
		Where(hubbusiness.CodeEQ(businessCode)).
		First(ctx)
	if err != nil {
		return nil, fmt.Errorf("find business %s: %w", businessCode, err)
	}

	existing, err := s.Client.HubWorker.Query().
		Where(hubworker.BusinessIDEQ(biz.ID), hubworker.WorkerIDEQ(workerID)).
		First(ctx)

	if err == nil {
		return s.Client.HubWorker.UpdateOne(existing).
			SetVersion(version).
			SetHost(host).
			SetPid(pid).
			SetLastHeartbeatAt(time.Now()).
			SetStatus("online").
			Save(ctx)
	}

	if !ent.IsNotFound(err) {
		return nil, fmt.Errorf("query worker: %w", err)
	}

	return s.Client.HubWorker.Create().
		SetBusinessID(biz.ID).
		SetWorkerID(workerID).
		SetVersion(version).
		SetHost(host).
		SetPid(pid).
		SetLastHeartbeatAt(time.Now()).
		SetStatus("online").
		Save(ctx)
}

func (s *Service) ListWorkersByBusiness(ctx context.Context, businessCode string, status string) ([]*ent.HubWorker, error) {
	q := s.Client.HubWorker.Query().
		Where(hubworker.HasBusinessWith(hubbusiness.CodeEQ(businessCode)))

	if status != "" {
		q = q.Where(hubworker.StatusEQ(status))
	}

	list, err := q.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list workers: %w", err)
	}
	return list, nil
}

func (s *Service) GetWorkerByID(ctx context.Context, id int64) (*ent.HubWorker, error) {
	w, err := s.Client.HubWorker.Query().
		Where(hubworker.IDEQ(id)).
		First(ctx)
	if err != nil {
		return nil, fmt.Errorf("get worker %d: %w", id, err)
	}
	return w, nil
}

func (s *Service) MarkOffline(ctx context.Context, timeoutSeconds int) error {
	cutoff := time.Now().Add(-time.Duration(timeoutSeconds) * time.Second)
	_, err := s.Client.HubWorker.Update().
		Where(
			hubworker.StatusEQ("online"),
			hubworker.LastHeartbeatAtLT(cutoff),
		).
		SetStatus("offline").
		Save(ctx)
	if err != nil {
		return fmt.Errorf("mark offline: %w", err)
	}
	return nil
}
