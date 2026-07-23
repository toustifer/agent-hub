package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/jackc/pgx/v5"
	"github.com/stifer/agent-hub/ent"
	"github.com/stifer/agent-hub/ent/hubbusiness"
)

// Short business codes: 4 chars, no hyphen — last path segment of /team/{slug}-{code}.
const shortCodeLen = 4
const shortCodeAlphabet = "abcdefghjkmnpqrstuvwxyz23456789" // no i/l/o/0/1

var shortCodeRe = regexp.MustCompile(`^[a-z0-9]{4}$`)

// ParseTeamCodeInput extracts a lookup key from a URL segment or raw business_code.
// If input looks like "{slug}-{4char}", returns the trailing short code; otherwise returns input trimmed.
func ParseTeamCodeInput(input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}
	// URL may still be percent-encoded in rare cases
	parts := strings.Split(input, "-")
	if len(parts) >= 2 {
		last := parts[len(parts)-1]
		if shortCodeRe.MatchString(last) {
			return last
		}
	}
	return input
}

// GenerateShortBusinessCode returns a unique 4-char code not used in hub_businesses.
func (s *Service) GenerateShortBusinessCode(ctx context.Context) (string, error) {
	for i := 0; i < 32; i++ {
		code, err := randomShortCode(shortCodeLen)
		if err != nil {
			return "", err
		}
		var exists bool
		err = s.Pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM hub.hub_businesses WHERE code=$1)
			  OR EXISTS(SELECT 1 FROM hub.hub_business_code_aliases WHERE old_code=$1 AND expires_at > now())`,
			code,
		).Scan(&exists)
		if err != nil {
			return "", fmt.Errorf("check short code: %w", err)
		}
		if !exists {
			return code, nil
		}
	}
	return "", errors.New("unable to generate unique business code")
}

func randomShortCode(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	out := make([]byte, n)
	for i := range b {
		out[i] = shortCodeAlphabet[int(b[i])%len(shortCodeAlphabet)]
	}
	return string(out), nil
}

// ResolveBusinessCode maps input (short code, old alias, or name-slug-code path) to business.
func (s *Service) ResolveBusinessCode(ctx context.Context, input string) (bizID int64, canonicalCode, name string, err error) {
	key := ParseTeamCodeInput(input)
	if key == "" {
		return 0, "", "", errors.New("business code required")
	}

	err = s.Pool.QueryRow(ctx,
		`SELECT id, code, name FROM hub.hub_businesses WHERE code=$1`, key,
	).Scan(&bizID, &canonicalCode, &name)
	if err == nil {
		return bizID, canonicalCode, name, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, "", "", fmt.Errorf("lookup business: %w", err)
	}

	// Alias grace period
	err = s.Pool.QueryRow(ctx,
		`SELECT b.id, b.code, b.name
		 FROM hub.hub_business_code_aliases a
		 JOIN hub.hub_businesses b ON b.id = a.business_id
		 WHERE a.old_code = $1 AND a.expires_at > now()`,
		key,
	).Scan(&bizID, &canonicalCode, &name)
	if err == nil {
		return bizID, canonicalCode, name, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, "", "", fmt.Errorf("business not found: %s", key)
	}
	return 0, "", "", fmt.Errorf("lookup business alias: %w", err)
}

func (s *Service) CreateBusiness(ctx context.Context, code, name, repoURL string, ownerUserID int64, description string) (*ent.HubBusiness, error) {
	// Prefer server-generated short code when caller leaves code empty or non-short.
	if code == "" || !shortCodeRe.MatchString(code) {
		gen, err := s.GenerateShortBusinessCode(ctx)
		if err != nil {
			return nil, err
		}
		code = gen
	}
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
	bizID, _, _, err := s.ResolveBusinessCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return s.GetBusinessByID(ctx, bizID)
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

// UpdateBusinessProfile renames / updates description; code is never changed.
func (s *Service) UpdateBusinessProfile(ctx context.Context, id int64, name, description *string) error {
	u := s.Client.HubBusiness.UpdateOneID(id)
	if name != nil {
		n := strings.TrimSpace(*name)
		if n == "" {
			return errors.New("name cannot be empty")
		}
		if len(n) > 128 {
			return errors.New("name too long")
		}
		u = u.SetName(n)
	}
	if description != nil {
		u = u.SetDescription(*description)
	}
	_, err := u.Save(ctx)
	if err != nil {
		return fmt.Errorf("update business profile: %w", err)
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

// EnsureShortBusinessCodes reassigns any non-short code to a new 4-char code and stores alias.
// Idempotent: rows already matching shortCodeRe are skipped.
func (s *Service) EnsureShortBusinessCodes(ctx context.Context) (int, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id, code FROM hub.hub_businesses`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	type row struct {
		id   int64
		code string
	}
	var list []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.code); err != nil {
			return 0, err
		}
		list = append(list, r)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	n := 0
	for _, r := range list {
		if shortCodeRe.MatchString(r.code) {
			continue
		}
		newCode, err := s.GenerateShortBusinessCode(ctx)
		if err != nil {
			return n, err
		}
		tx, err := s.Pool.Begin(ctx)
		if err != nil {
			return n, err
		}
		_, err = tx.Exec(ctx,
			`INSERT INTO hub.hub_business_code_aliases (old_code, business_id, expires_at)
			 VALUES ($1, $2, now() + interval '90 days')
			 ON CONFLICT (old_code) DO UPDATE SET business_id = EXCLUDED.business_id, expires_at = EXCLUDED.expires_at`,
			r.code, r.id,
		)
		if err != nil {
			_ = tx.Rollback(ctx)
			return n, fmt.Errorf("alias %s: %w", r.code, err)
		}
		_, err = tx.Exec(ctx, `UPDATE hub.hub_businesses SET code=$1 WHERE id=$2`, newCode, r.id)
		if err != nil {
			_ = tx.Rollback(ctx)
			return n, fmt.Errorf("reassign id=%d: %w", r.id, err)
		}
		if err := tx.Commit(ctx); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

// NameSlug builds the URL-readable segment from a display name (keeps CJK).
func NameSlug(name string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range strings.TrimSpace(name) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			prevDash = false
			continue
		}
		if r == ' ' || r == '-' || r == '_' {
			if b.Len() > 0 && !prevDash {
				b.WriteByte('-')
				prevDash = true
			}
		}
		// drop other punctuation
	}
	s := strings.Trim(b.String(), "-")
	return s
}

// BuildTeamPath returns /team/{slug}-{code} or /team/{code}.
func BuildTeamPath(name, code string) string {
	slug := NameSlug(name)
	if slug == "" {
		return "/team/" + code
	}
	return "/team/" + slug + "-" + code
}
