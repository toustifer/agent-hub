package service

import (
	"context"
	"fmt"

	"github.com/stifer/agent-hub/ent"
	"github.com/stifer/agent-hub/ent/hubbusiness"
	"github.com/stifer/agent-hub/ent/hubplaybook"
)

func (s *Service) CreatePlaybook(ctx context.Context, businessCode, category, title, content string, tags []string, workerID string) (*ent.HubPlaybook, error) {
	var businessID *int64
	if businessCode != "" {
		biz, err := s.Client.HubBusiness.Query().Where(hubbusiness.CodeEQ(businessCode)).First(ctx)
		if err != nil {
			return nil, fmt.Errorf("find business %s: %w", businessCode, err)
		}
		businessID = &biz.ID
	}

	sql := "INSERT INTO hub.hub_playbooks (business_id, category, title, content, tags, created_by_worker_id, created_at, updated_at) VALUES ($1, $2, $3, $4, $5::text[], $6, now(), now()) ON CONFLICT (business_id, category, title) DO UPDATE SET content=$4, tags=$5::text[], updated_at=now() RETURNING id"
	var id int64
	err := s.Pool.QueryRow(ctx, sql, businessID, category, title, content, tags, workerID).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("create playbook: %w", err)
	}
	return s.getPlaybookByIDRaw(ctx, id)
}

func (s *Service) SearchPlaybooks(ctx context.Context, query string, businessCode string, category string, limit, offset int) ([]*ent.HubPlaybook, error) {
	if query == "" {
		return s.ListPlaybooksByCategory(ctx, businessCode, category, limit, offset)
	}

	sql := "SELECT id FROM hub.hub_playbooks WHERE deleted_at IS NULL AND tsv @@ plainto_tsquery('simple', $1)"
	args := []interface{}{query}
	argIdx := 2

	if businessCode != "" {
		sql += fmt.Sprintf(" AND business_id = (SELECT id FROM hub.hub_businesses WHERE code = $%d)", argIdx)
		args = append(args, businessCode)
		argIdx++
	}

	if category != "" {
		sql += fmt.Sprintf(" AND category = $%d", argIdx)
		args = append(args, category)
		argIdx++
	}

	sql += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := s.Pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("search playbooks: %w", err)
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan playbook id: %w", err)
		}
		ids = append(ids, id)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("iterate playbook rows: %w", rows.Err())
	}

	if len(ids) == 0 {
		return []*ent.HubPlaybook{}, nil
	}

	list, err := s.Client.HubPlaybook.Query().Where(hubplaybook.IDIn(ids...)).Order(ent.Desc(hubplaybook.FieldCreatedAt)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch playbooks by ids: %w", err)
	}
	return list, nil
}

func (s *Service) GetPlaybookByID(ctx context.Context, id int64) (*ent.HubPlaybook, error) {
	return s.getPlaybookByIDRaw(ctx, id)
}

func (s *Service) getPlaybookByIDRaw(ctx context.Context, id int64) (*ent.HubPlaybook, error) {
	sql := "SELECT id, business_id, category, title, content, tags, created_by_worker_id, created_at, updated_at FROM hub.hub_playbooks WHERE id = $1 AND deleted_at IS NULL"
	row := s.Pool.QueryRow(ctx, sql, id)
	pb := &ent.HubPlaybook{}
	err := row.Scan(&pb.ID, &pb.BusinessID, &pb.Category, &pb.Title, &pb.Content, &pb.Tags, &pb.CreatedByWorkerID, &pb.CreatedAt, &pb.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get playbook %d: %w", id, err)
	}
	return pb, nil
}

func (s *Service) DeletePlaybook(ctx context.Context, id int64) error {
	err := s.Client.HubPlaybook.DeleteOneID(id).Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete playbook %d: %w", id, err)
	}
	return nil
}

func (s *Service) ListPlaybooksByCategory(ctx context.Context, businessCode, category string, limit, offset int) ([]*ent.HubPlaybook, error) {
	sql := "SELECT id FROM hub.hub_playbooks WHERE deleted_at IS NULL"
	args := []interface{}{}
	argIdx := 1

	if category != "" {
		sql += fmt.Sprintf(" AND category = $%d", argIdx)
		args = append(args, category)
		argIdx++
	}
	if businessCode != "" {
		sql += fmt.Sprintf(" AND business_id = (SELECT id FROM hub.hub_businesses WHERE code = $%d)", argIdx)
		args = append(args, businessCode)
		argIdx++
	}
	sql += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := s.Pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("list playbooks: %w", err)
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan playbook id: %w", err)
		}
		ids = append(ids, id)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("iterate playbook rows: %w", rows.Err())
	}
	if len(ids) == 0 {
		return []*ent.HubPlaybook{}, nil
	}

	return s.getPlaybooksByIDs(ctx, ids)
}

func (s *Service) getPlaybooksByIDs(ctx context.Context, ids []int64) ([]*ent.HubPlaybook, error) {
	if len(ids) == 0 {
		return []*ent.HubPlaybook{}, nil
	}
	rows, err := s.Pool.Query(ctx,
		"SELECT id, business_id, category, title, content, tags, created_by_worker_id, created_at, updated_at FROM hub.hub_playbooks WHERE id = ANY($1) AND deleted_at IS NULL ORDER BY created_at DESC",
		ids)
	if err != nil {
		return nil, fmt.Errorf("batch fetch playbooks: %w", err)
	}
	defer rows.Close()
	var list []*ent.HubPlaybook
	for rows.Next() {
		pb := &ent.HubPlaybook{}
		if err := rows.Scan(&pb.ID, &pb.BusinessID, &pb.Category, &pb.Title, &pb.Content, &pb.Tags, &pb.CreatedByWorkerID, &pb.CreatedAt, &pb.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan playbook: %w", err)
		}
		list = append(list, pb)
	}
	if list == nil {
		list = []*ent.HubPlaybook{}
	}
	return list, nil
}
