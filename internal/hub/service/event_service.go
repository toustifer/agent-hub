package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/stifer/agent-hub/ent"
	"github.com/stifer/agent-hub/ent/hubbusiness"
	"github.com/stifer/agent-hub/ent/hubevent"
)

// AppendEventInput is the full work-log write path (v0.2 attribution).
type AppendEventInput struct {
	BusinessCode  string
	Actor         string // legacy column
	EventType     string
	Payload       map[string]interface{}
	ActorUserID   *int64
	ActorEmail    string
	ActorRole     string
	ActorWorkerID string
}

// EventRow is a list DTO including attribution fields.
type EventRow struct {
	ID            int64                  `json:"id"`
	BusinessID    int64                  `json:"business_id,omitempty"`
	Actor         string                 `json:"actor"`
	EventType     string                 `json:"event_type"`
	Payload       map[string]interface{} `json:"payload,omitempty"`
	ActorUserID   *int64                 `json:"actor_user_id,omitempty"`
	ActorEmail    string                 `json:"actor_email,omitempty"`
	ActorRole     string                 `json:"actor_role,omitempty"`
	ActorWorkerID string                 `json:"actor_worker_id,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
}

// AppendEvent keeps the old signature for lock/playbook call sites.
func (s *Service) AppendEvent(ctx context.Context, businessCode, actor, eventType string, payload map[string]interface{}) (*ent.HubEvent, error) {
	id, err := s.AppendEventFull(ctx, AppendEventInput{
		BusinessCode: businessCode,
		Actor:        actor,
		EventType:    eventType,
		Payload:      payload,
		ActorRole:    "system",
	})
	if err != nil {
		return nil, err
	}
	// Minimal ent-shaped return for older handlers that only need ID.
	return &ent.HubEvent{ID: id, Actor: actor, EventType: eventType, Payload: payload}, nil
}

// AppendEventFull inserts a work log with optional user attribution.
func (s *Service) AppendEventFull(ctx context.Context, in AppendEventInput) (int64, error) {
	if in.BusinessCode == "" {
		return 0, fmt.Errorf("business_code required")
	}
	if in.EventType == "" {
		return 0, fmt.Errorf("event_type required")
	}
	if in.Actor == "" {
		in.Actor = "unknown"
	}
	if in.Payload == nil {
		in.Payload = map[string]interface{}{}
	}
	payloadBytes, err := json.Marshal(in.Payload)
	if err != nil {
		return 0, fmt.Errorf("marshal payload: %w", err)
	}
	if len(payloadBytes) > 4096 {
		return 0, fmt.Errorf("payload too large (max 4KB)")
	}

	var bizID int64
	if err := s.Pool.QueryRow(ctx,
		`SELECT id FROM hub.hub_businesses WHERE code=$1`, in.BusinessCode,
	).Scan(&bizID); err != nil {
		return 0, fmt.Errorf("find business %s: %w", in.BusinessCode, err)
	}

	var id int64
	err = s.Pool.QueryRow(ctx, `
		INSERT INTO hub.hub_events (
		  business_id, actor, event_type, payload,
		  actor_user_id, actor_email, actor_role, actor_worker_id,
		  created_at, updated_at
		) VALUES (
		  $1, $2, $3, $4::jsonb,
		  $5, NULLIF($6,''), NULLIF($7,''), NULLIF($8,''),
		  now(), now()
		) RETURNING id`,
		bizID, in.Actor, in.EventType, payloadBytes,
		in.ActorUserID, in.ActorEmail, in.ActorRole, in.ActorWorkerID,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("append event: %w", err)
	}
	return id, nil
}

// ListEventRows returns events with attribution columns (Pool SQL).
func (s *Service) ListEventRows(ctx context.Context, businessCode, eventType string, limit, offset int) ([]EventRow, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	sql := `
		SELECT e.id, e.business_id, e.actor, e.event_type, e.payload,
		       e.actor_user_id, COALESCE(e.actor_email,''), COALESCE(e.actor_role,''), COALESCE(e.actor_worker_id,''),
		       e.created_at
		FROM hub.hub_events e
		JOIN hub.hub_businesses b ON b.id = e.business_id
		WHERE b.code = $1`
	args := []interface{}{businessCode}
	if eventType != "" {
		sql += ` AND e.event_type = $2`
		args = append(args, eventType)
		sql += fmt.Sprintf(` ORDER BY e.created_at DESC LIMIT $%d OFFSET $%d`, len(args)+1, len(args)+2)
	} else {
		sql += ` ORDER BY e.created_at DESC LIMIT $2 OFFSET $3`
	}
	args = append(args, limit, offset)

	rows, err := s.Pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	defer rows.Close()

	var list []EventRow
	for rows.Next() {
		var r EventRow
		var payload []byte
		var uid *int64
		if err := rows.Scan(&r.ID, &r.BusinessID, &r.Actor, &r.EventType, &payload,
			&uid, &r.ActorEmail, &r.ActorRole, &r.ActorWorkerID, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		r.ActorUserID = uid
		if len(payload) > 0 {
			_ = json.Unmarshal(payload, &r.Payload)
		}
		list = append(list, r)
	}
	if list == nil {
		list = []EventRow{}
	}
	return list, rows.Err()
}

func (s *Service) ListEvents(ctx context.Context, businessCode string, eventType string, limit, offset int) ([]*ent.HubEvent, error) {
	// Legacy ent path still used by some callers; prefer ListEventRows for new code.
	q := s.Client.HubEvent.Query().
		Where(hubevent.HasBusinessWith(hubbusiness.CodeEQ(businessCode))).
		Order(ent.Desc(hubevent.FieldCreatedAt))
	if eventType != "" {
		q = q.Where(hubevent.EventTypeEQ(eventType))
	}
	list, err := q.Limit(limit).Offset(offset).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	return list, nil
}

func (s *Service) StreamEvents(ctx context.Context, businessCode string, since time.Time) (<-chan *ent.HubEvent, error) {
	biz, err := s.Client.HubBusiness.Query().Where(hubbusiness.CodeEQ(businessCode)).First(ctx)
	if err != nil {
		return nil, fmt.Errorf("find business %s: %w", businessCode, err)
	}

	ch := make(chan *ent.HubEvent)
	go func() {
		defer close(ch)
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		cursor := since
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				events, err := s.Client.HubEvent.Query().
					Where(
						hubevent.BusinessIDEQ(biz.ID),
						hubevent.CreatedAtGT(cursor),
					).
					Order(ent.Asc(hubevent.FieldCreatedAt)).
					All(ctx)
				if err != nil {
					continue
				}
				for _, ev := range events {
					select {
					case ch <- ev:
						cursor = ev.CreatedAt
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()

	return ch, nil
}
