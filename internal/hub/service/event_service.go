package service

import (
	"context"
	"fmt"
	"time"

	"github.com/stifer/agent-hub/ent"
	"github.com/stifer/agent-hub/ent/hubevent"
	"github.com/stifer/agent-hub/ent/hubbusiness"
)

func (s *Service) AppendEvent(ctx context.Context, businessCode, actor, eventType string, payload map[string]interface{}) (*ent.HubEvent, error) {
	biz, err := s.Client.HubBusiness.Query().Where(hubbusiness.CodeEQ(businessCode)).First(ctx)
	if err != nil {
		return nil, fmt.Errorf("find business %s: %w", businessCode, err)
	}

	ev, err := s.Client.HubEvent.Create().
		SetBusinessID(biz.ID).
		SetActor(actor).
		SetEventType(eventType).
		SetPayload(payload).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("append event: %w", err)
	}
	return ev, nil
}

func (s *Service) ListEvents(ctx context.Context, businessCode string, eventType string, limit, offset int) ([]*ent.HubEvent, error) {
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
