package database

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

const apiKeyPrefix = "td_live_"

var ErrAPIKeyNotFound = errors.New("api key not found")

type APIKey struct {
	ID                 uint64
	Name               string
	KeyPrefix          string
	KeyHash            string
	OwnerName          sql.NullString
	OwnerEmail         sql.NullString
	IsActive           bool
	ExpiresAt          sql.NullTime
	AllowedIPs         sql.NullString
	AllowedRoutes      sql.NullString
	RateLimitPerMinute int
	LastUsedAt         sql.NullTime
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type CreateAPIKeyParams struct {
	Name               string
	OwnerName          string
	OwnerEmail         string
	ExpiresAt          *time.Time
	AllowedIPs         string
	AllowedRoutes      string
	RateLimitPerMinute int
}

type CreatedAPIKey struct {
	Record *APIKey
	Plain  string
}

func GenerateAPIKey() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate random api key: %w", err)
	}
	return apiKeyPrefix + hex.EncodeToString(raw), nil
}

func HashAPIKey(plain string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(plain)))
	return hex.EncodeToString(sum[:])
}

func APIKeyPublicPrefix(plain string) string {
	plain = strings.TrimSpace(plain)
	if len(plain) <= 16 {
		return plain
	}
	return plain[:16]
}

func (q *Queries) CreateAPIKey(params CreateAPIKeyParams) (*CreatedAPIKey, error) {
	name := strings.TrimSpace(params.Name)
	if name == "" {
		return nil, errors.New("api key name is required")
	}

	rateLimit := params.RateLimitPerMinute
	if rateLimit <= 0 {
		rateLimit = 120
	}

	plain, err := GenerateAPIKey()
	if err != nil {
		return nil, err
	}

	keyHash := HashAPIKey(plain)
	keyPrefix := APIKeyPublicPrefix(plain)

	res, err := q.db.Exec(`
INSERT INTO api_keys (
    name,
    key_prefix,
    key_hash,
    owner_name,
    owner_email,
    is_active,
    expires_at,
    allowed_ips,
    allowed_routes,
    rate_limit_per_minute
) VALUES (?, ?, ?, ?, ?, 1, ?, ?, ?, ?)
`,
		name,
		keyPrefix,
		keyHash,
		toNullString(strings.TrimSpace(params.OwnerName)),
		toNullString(strings.TrimSpace(params.OwnerEmail)),
		toNullTime(params.ExpiresAt),
		toNullString(strings.TrimSpace(params.AllowedIPs)),
		toNullString(strings.TrimSpace(params.AllowedRoutes)),
		rateLimit,
	)
	if err != nil {
		return nil, fmt.Errorf("create api key: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("read created api key id: %w", err)
	}

	record, err := q.GetAPIKeyByID(uint64(id))
	if err != nil {
		return nil, err
	}

	return &CreatedAPIKey{Record: record, Plain: plain}, nil
}

func (q *Queries) GetAPIKeyByHash(hash string) (*APIKey, error) {
	return q.scanAPIKeyRow(q.db.QueryRow(`
SELECT id, name, key_prefix, key_hash, owner_name, owner_email, is_active, expires_at,
       allowed_ips, allowed_routes, rate_limit_per_minute, last_used_at, created_at, updated_at
FROM api_keys
WHERE key_hash = ?
LIMIT 1
`, hash))
}

func (q *Queries) GetAPIKeyByID(id uint64) (*APIKey, error) {
	return q.scanAPIKeyRow(q.db.QueryRow(`
SELECT id, name, key_prefix, key_hash, owner_name, owner_email, is_active, expires_at,
       allowed_ips, allowed_routes, rate_limit_per_minute, last_used_at, created_at, updated_at
FROM api_keys
WHERE id = ?
LIMIT 1
`, id))
}

func (q *Queries) TouchAPIKeyLastUsed(id uint64) error {
	_, err := q.db.Exec(`UPDATE api_keys SET last_used_at = CURRENT_TIMESTAMP WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("touch api key last used: %w", err)
	}
	return nil
}

func (q *Queries) LogAPIKeyUsage(apiKeyID *uint64, keyPrefix, method, path, clientIP, userAgent, status, message string) {
	var id sql.NullInt64
	if apiKeyID != nil {
		id = sql.NullInt64{Int64: int64(*apiKeyID), Valid: true}
	}
	_, _ = q.db.Exec(`
INSERT INTO api_key_usage_logs (api_key_id, key_prefix, request_method, request_path, client_ip, user_agent, status, message)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
`,
		id,
		toNullString(keyPrefix),
		method,
		path,
		clientIP,
		toNullString(userAgent),
		status,
		toNullString(message),
	)
}

func (q *Queries) scanAPIKeyRow(row *sql.Row) (*APIKey, error) {
	var item APIKey
	if err := row.Scan(
		&item.ID,
		&item.Name,
		&item.KeyPrefix,
		&item.KeyHash,
		&item.OwnerName,
		&item.OwnerEmail,
		&item.IsActive,
		&item.ExpiresAt,
		&item.AllowedIPs,
		&item.AllowedRoutes,
		&item.RateLimitPerMinute,
		&item.LastUsedAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAPIKeyNotFound
		}
		return nil, err
	}
	return &item, nil
}

func toNullTime(t *time.Time) sql.NullTime {
	if t == nil || t.IsZero() {
		return sql.NullTime{Valid: false}
	}
	return sql.NullTime{Time: *t, Valid: true}
}
