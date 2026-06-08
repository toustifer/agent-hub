package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/stifer/agent-hub/ent"
	"github.com/stifer/agent-hub/ent/hubbusiness"
	"github.com/stifer/agent-hub/ent/hublock"
)

var ErrLockHeld = errors.New("lock is held by another worker")

type LockHeldError struct {
	ResourceKey    string
	HolderWorkerID string
}

func (e *LockHeldError) Error() string {
	return fmt.Sprintf("lock %s is held by worker %s", e.ResourceKey, e.HolderWorkerID)
}

func (e *LockHeldError) Unwrap() error {
	return ErrLockHeld
}

func newHolderToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}

func (s *Service) AcquireLock(ctx context.Context, businessCode, resourceKey, workerID string, ttlSeconds int) (string, time.Time, error) {
	biz, err := s.Client.HubBusiness.Query().Where(hubbusiness.CodeEQ(businessCode)).First(ctx)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("find business %s: %w", businessCode, err)
	}

	cleanSQL := "UPDATE hub.hub_locks SET released_at = now() WHERE released_at IS NULL AND expires_at < now() AND resource_key = $1"
	_, err = s.Pool.Exec(ctx, cleanSQL, resourceKey)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("clean expired locks: %w", err)
	}

	token, err := newHolderToken()
	if err != nil {
		return "", time.Time{}, fmt.Errorf("generate token: %w", err)
	}

	var expiresAt time.Time
		insertSQL := "INSERT INTO hub.hub_locks (business_id, resource_key, holder_token, holder_worker_id, acquired_at, expires_at, heartbeat_at) VALUES ($1, $2, $3, $4, now(), now() + make_interval(secs => $5), now()) ON CONFLICT (resource_key) WHERE released_at IS NULL AND expires_at > now() DO NOTHING RETURNING expires_at"
	err = s.Pool.QueryRow(ctx, insertSQL, biz.ID, resourceKey, token, workerID, ttlSeconds).Scan(&expiresAt)

	if errors.Is(err, pgx.ErrNoRows) {
		var holderWorkerID string
		holderSQL := "SELECT holder_worker_id FROM hub.hub_locks WHERE resource_key = $1 AND released_at IS NULL AND expires_at > now() LIMIT 1"
		s.Pool.QueryRow(ctx, holderSQL, resourceKey).Scan(&holderWorkerID)
		return "", time.Time{}, &LockHeldError{ResourceKey: resourceKey, HolderWorkerID: holderWorkerID}
	}
	if err != nil {
		return "", time.Time{}, fmt.Errorf("acquire lock: %w", err)
	}

	if err := s.Client.HubEvent.Create().
		SetBusinessID(biz.ID).
		SetActor(workerID).
		SetEventType("lock.acquired").
		SetPayload(map[string]interface{}{"resource_key": resourceKey, "token": token}).
		Exec(ctx); err != nil {
		log.Printf("lock.acquired event failed: %v", err)
	}

	return token, expiresAt, nil
}

func (s *Service) RenewLock(ctx context.Context, holderToken string, ttlSeconds int) error {
	renewSQL := "UPDATE hub.hub_locks SET expires_at = now() + make_interval(secs => $2), heartbeat_at = now() WHERE holder_token = $1 AND released_at IS NULL AND expires_at > now()"
	tag, err := s.Pool.Exec(ctx, renewSQL, holderToken, ttlSeconds)
	if err != nil {
		return fmt.Errorf("renew lock: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("lock not found or already expired")
	}
	return nil
}

func (s *Service) ReleaseLock(ctx context.Context, holderToken string) error {
	var resourceKey string
	var businessID int64
	var workerID string

	releaseSQL := "UPDATE hub.hub_locks SET released_at = now() WHERE holder_token = $1 AND released_at IS NULL RETURNING resource_key, business_id, holder_worker_id"
	err := s.Pool.QueryRow(ctx, releaseSQL, holderToken).Scan(&resourceKey, &businessID, &workerID)

	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("lock not found or already released: %s", holderToken)
	}
	if err != nil {
		return fmt.Errorf("release lock: %w", err)
	}

	if err := s.Client.HubEvent.Create().
		SetBusinessID(businessID).
		SetActor(workerID).
		SetEventType("lock.released").
		SetPayload(map[string]interface{}{"resource_key": resourceKey, "token": holderToken}).
		Exec(ctx); err != nil {
		log.Printf("lock.released event failed: %v", err)
	}

	return nil
}

func (s *Service) CleanupExpiredLocks(ctx context.Context) (int64, error) {
	tag, err := s.Pool.Exec(ctx,
		"UPDATE hub.hub_locks SET released_at = now() WHERE released_at IS NULL AND expires_at < now()")
	if err != nil {
		return 0, fmt.Errorf("cleanup expired locks: %w", err)
	}
	return tag.RowsAffected(), nil
}

func (s *Service) ListActiveLocks(ctx context.Context, businessCode string) ([]*ent.HubLock, error) {
	locks, err := s.Client.HubLock.Query().
		Where(hublock.HasBusinessWith(hubbusiness.CodeEQ(businessCode))).
		Where(hublock.ReleasedAtIsNil()).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list active locks: %w", err)
	}
	return locks, nil
}
