package service

import (
	"context"
	"fmt"

	"github.com/stifer/agent-hub/ent"
	"github.com/stifer/agent-hub/ent/hubbusiness"
)

func (s *Service) CreateBusiness(ctx context.Context, code, name, repoURL string, ownerUserID int64, description string) (*ent.HubBusiness, error) {
	biz, err := s.Client.HubBusiness.Create().
		SetCode(code).
		SetName(name).
		SetRepoURL(repoURL).
		SetNillableOwnerUserID(&ownerUserID).
		SetDescription(description).
		SetStatus("active").
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("create business: %w", err)
	}
	return biz, nil
}

func (s *Service) GetBusinessByID(ctx context.Context, id int64) (*ent.HubBusiness, error) {
	biz, err := s.Client.HubBusiness.Query().
		Where(hubbusiness.IDEQ(id)).
		First(ctx)
	if err != nil {
		return nil, fmt.Errorf("get business by id %d: %w", id, err)
	}
	return biz, nil
}

func (s *Service) GetBusinessByCode(ctx context.Context, code string) (*ent.HubBusiness, error) {
	biz, err := s.Client.HubBusiness.Query().
		Where(hubbusiness.CodeEQ(code)).
		First(ctx)
	if err != nil {
		return nil, fmt.Errorf("get business by code %s: %w", code, err)
	}
	return biz, nil
}

func (s *Service) ListBusinesses(ctx context.Context, status string, limit, offset int) ([]*ent.HubBusiness, error) {
	q := s.Client.HubBusiness.Query()
	if status != "" {
		q = q.Where(hubbusiness.StatusEQ(status))
	}
	list, err := q.Limit(limit).Offset(offset).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list businesses: %w", err)
	}
	return list, nil
}

func (s *Service) UpdateBusinessStatus(ctx context.Context, id int64, status string) error {
	_, err := s.Client.HubBusiness.UpdateOneID(id).
		SetStatus(status).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("update business status: %w", err)
	}
	return nil
}

func (s *Service) DeleteBusiness(ctx context.Context, id int64) error {
	err := s.Client.HubBusiness.DeleteOneID(id).Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete business %d: %w", id, err)
	}
	return nil
}
