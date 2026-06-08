package service

import (
	"context"
	"fmt"
	"regexp"
	"strconv"

	"github.com/stifer/agent-hub/ent"
	"github.com/stifer/agent-hub/ent/hubbusiness"
	"github.com/stifer/agent-hub/ent/hubcommunityreview"
)

func (s *Service) PublishWorker(ctx context.Context, publisherUserID int64, title, description, domain, scope string, handbook, playbooks map[string]interface{}, tags []string) (*ent.HubCommunityWorker, error) {
	if len(tags) == 0 {
		tags = []string{}
	}
	if handbook == nil {
		handbook = map[string]interface{}{}
	}
	if playbooks == nil {
		playbooks = map[string]interface{}{}
	}

	sql := "INSERT INTO hub.hub_community_workers (publisher_user_id, title, description, domain, scope, handbook, playbooks, tags, install_count, status, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8::text[], 0, 'published', now(), now()) RETURNING id"
	var id int64
	err := s.Pool.QueryRow(ctx, sql, publisherUserID, title, description, domain, scope, handbook, playbooks, tags).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("publish community worker: %w", err)
	}
	return s.getCommunityWorkerByIDRaw(ctx, id)
}

func (s *Service) getCommunityWorkerByIDRaw(ctx context.Context, id int64) (*ent.HubCommunityWorker, error) {
	sql := "SELECT id, publisher_user_id, title, description, domain, scope, handbook, playbooks, tags, install_count, status, created_at, updated_at FROM hub.hub_community_workers WHERE id = $1"
	row := s.Pool.QueryRow(ctx, sql, id)
	w := &ent.HubCommunityWorker{}
	err := row.Scan(&w.ID, &w.PublisherUserID, &w.Title, &w.Description, &w.Domain, &w.Scope, &w.Handbook, &w.Playbooks, &w.Tags, &w.InstallCount, &w.Status, &w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get community worker %d: %w", id, err)
	}
	return w, nil
}

func (s *Service) ListCommunityWorkers(ctx context.Context, search, domain, sort string, page, limit int) ([]*ent.HubCommunityWorker, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit

	args := []interface{}{}
	argIdx := 1
	conditions := ""

	if search != "" {
		conditions += fmt.Sprintf(" AND (title ILIKE $%d OR description ILIKE $%d)", argIdx, argIdx+1)
		pattern := "%" + search + "%"
		args = append(args, pattern, pattern)
		argIdx += 2
	}
	if domain != "" {
		conditions += fmt.Sprintf(" AND domain = $%d", argIdx)
		args = append(args, domain)
		argIdx++
	}

	var orderClause string
	switch sort {
	case "popular":
		orderClause = " ORDER BY install_count DESC, created_at DESC"
	default:
		orderClause = " ORDER BY created_at DESC"
	}

	tableName := "hub.hub_community_workers"

	countSQL := "SELECT COUNT(*) FROM " + tableName + " WHERE 1=1" + conditions
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	var total int
	if err := s.Pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count community workers: %w", err)
	}

	idSQL := "SELECT id FROM " + tableName + " WHERE 1=1" + conditions + orderClause + fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, limit, offset)
	rows, err := s.Pool.Query(ctx, idSQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list community workers: %w", err)
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, 0, fmt.Errorf("scan community worker id: %w", err)
		}
		ids = append(ids, id)
	}
	if rows.Err() != nil {
		return nil, 0, fmt.Errorf("iterate community worker rows: %w", rows.Err())
	}

	if len(ids) == 0 {
		return []*ent.HubCommunityWorker{}, total, nil
	}

	list, err := s.getCommunityWorkersByIDs(ctx, ids, sort)
	if err != nil {
		return nil, 0, fmt.Errorf("fetch community workers by ids: %w", err)
	}
	return list, total, nil
}

func (s *Service) GetCommunityWorker(ctx context.Context, id int64) (*ent.HubCommunityWorker, error) {
	return s.getCommunityWorkerByIDRaw(ctx, id)
}

func (s *Service) getCommunityWorkersByIDs(ctx context.Context, ids []int64, sort string) ([]*ent.HubCommunityWorker, error) {
	if len(ids) == 0 {
		return []*ent.HubCommunityWorker{}, nil
	}
	order := " ORDER BY created_at DESC"
	if sort == "popular" {
		order = " ORDER BY install_count DESC, created_at DESC"
	}
	rows, err := s.Pool.Query(ctx,
		"SELECT id, publisher_user_id, title, description, domain, scope, handbook, playbooks, tags, install_count, status, created_at, updated_at FROM hub.hub_community_workers WHERE id = ANY($1)"+order,
		ids)
	if err != nil {
		return nil, fmt.Errorf("batch fetch community workers: %w", err)
	}
	defer rows.Close()
	var list []*ent.HubCommunityWorker
	for rows.Next() {
		w := &ent.HubCommunityWorker{}
		if err := rows.Scan(&w.ID, &w.PublisherUserID, &w.Title, &w.Description, &w.Domain, &w.Scope, &w.Handbook, &w.Playbooks, &w.Tags, &w.InstallCount, &w.Status, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan community worker: %w", err)
		}
		list = append(list, w)
	}
	if list == nil {
		list = []*ent.HubCommunityWorker{}
	}
	return list, nil
}

func (s *Service) InstallWorker(ctx context.Context, communityWorkerID int64, targetBusinessCode string) error {
	cw, err := s.getCommunityWorkerByIDRaw(ctx, communityWorkerID)
	if err != nil {
		return fmt.Errorf("find community worker %d: %w", communityWorkerID, err)
	}

	biz, err := s.Client.HubBusiness.Query().
		Where(hubbusiness.CodeEQ(targetBusinessCode)).
		First(ctx)
	if err != nil {
		return fmt.Errorf("find business %s: %w", targetBusinessCode, err)
	}

	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	workerID := "community-" + strconv.FormatInt(cw.ID, 10)

	_, err = tx.Exec(ctx,
		"INSERT INTO hub.hub_workers (business_id, worker_id, version, status, handbook, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, now(), now()) ON CONFLICT (business_id, worker_id) DO UPDATE SET handbook=$5, updated_at=now()",
		biz.ID, workerID, "1.0.0", "imported", cw.Handbook)
	if err != nil {
		return fmt.Errorf("insert hub worker: %w", err)
	}

	if cw.Playbooks != nil {
		for key, val := range cw.Playbooks {
			pb, ok := val.(map[string]interface{})
			if !ok {
				continue
			}
			category, _ := pb["category"].(string)
			title, _ := pb["title"].(string)
			content, _ := pb["content"].(string)
			var tags []string
			if t, ok := pb["tags"].([]interface{}); ok {
				for _, tag := range t {
					if s, ok := tag.(string); ok {
						tags = append(tags, s)
					}
				}
			}
			if len(tags) == 0 {
				tags = []string{}
			}

			_, err = tx.Exec(ctx,
				"INSERT INTO hub.hub_playbooks (business_id, category, title, content, tags, created_by_worker_id, created_at, updated_at) VALUES ($1, $2, $3, $4, $5::text[], $6, now(), now())",
				biz.ID, category, title, content, tags, workerID)
			if err != nil {
				return fmt.Errorf("import playbook %s: %w", key, err)
			}
		}
	}

	_, err = tx.Exec(ctx,
		"UPDATE hub.hub_community_workers SET install_count = install_count + 1, updated_at = now() WHERE id = $1",
		cw.ID)
	if err != nil {
		return fmt.Errorf("increment install count: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

func (s *Service) GetCommunityWorkerReviews(ctx context.Context, workerID int64) ([]*ent.HubCommunityReview, error) {
	list, err := s.Client.HubCommunityReview.Query().
		Where(hubcommunityreview.WorkerIDEQ(workerID)).
		Order(ent.Desc(hubcommunityreview.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("get reviews for worker %d: %w", workerID, err)
	}
	if list == nil {
		list = []*ent.HubCommunityReview{}
	}
	return list, nil
}

func (s *Service) AddCommunityWorkerReview(ctx context.Context, workerID, userID int64, rating int, comment string) (*ent.HubCommunityReview, error) {
	create := s.Client.HubCommunityReview.Create().
		SetWorkerID(workerID).
		SetUserID(userID).
		SetRating(rating)

	if comment != "" {
		create.SetComment(comment)
	}

	r, err := create.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("add review: %w", err)
	}
	return r, nil
}

func deidentify(text string) string {
	reFilepath := regexp.MustCompile(`(?:[A-Za-z]:\\[\w\-. \\]+|\/[\w\-. \/]+)\.\w{1,6}`)
	text = reFilepath.ReplaceAllString(text, "[文件路径]")

	reURL := regexp.MustCompile(`https?:\/\/[^\s]+`)
	text = reURL.ReplaceAllString(text, "[URL]")

	reIP := regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`)
	text = reIP.ReplaceAllString(text, "[IP地址]")

	reAPIKey := regexp.MustCompile(`(?:sk-[a-zA-Z0-9]+|Bearer\s+[a-zA-Z0-9\-_\.]+)`)
	text = reAPIKey.ReplaceAllString(text, "[密钥]")

	return text
}
