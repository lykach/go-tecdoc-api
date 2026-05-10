package auth

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"go-tecdoc-api/internal/database"
)

type contextKey string

const APIKeyContextKey contextKey = "api_key"

type APIKeyMiddleware struct {
	queries *database.Queries
	limiter *MemoryRateLimiter
}

func NewAPIKeyMiddleware(queries *database.Queries) *APIKeyMiddleware {
	return &APIKeyMiddleware{
		queries: queries,
		limiter: NewMemoryRateLimiter(time.Minute),
	}
}

func (m *APIKeyMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		plainKey := strings.TrimSpace(r.Header.Get("X-API-Key"))
		if plainKey == "" {
			m.reject(w, r, nil, "", http.StatusUnauthorized, "missing_api_key", "X-API-Key header is required")
			return
		}

		keyPrefix := database.APIKeyPublicPrefix(plainKey)
		apiKey, err := m.queries.GetAPIKeyByHash(database.HashAPIKey(plainKey))
		if err != nil {
			if errors.Is(err, database.ErrAPIKeyNotFound) {
				m.reject(w, r, nil, keyPrefix, http.StatusUnauthorized, "invalid_api_key", "API key is invalid")
				return
			}
			log.Printf("api key lookup failed: %v", err)
			m.reject(w, r, nil, keyPrefix, http.StatusInternalServerError, "api_key_lookup_failed", "API key verification failed")
			return
		}

		if !apiKey.IsActive {
			m.reject(w, r, &apiKey.ID, keyPrefix, http.StatusForbidden, "api_key_inactive", "API key is inactive")
			return
		}

		if apiKey.ExpiresAt.Valid && time.Now().After(apiKey.ExpiresAt.Time) {
			m.reject(w, r, &apiKey.ID, keyPrefix, http.StatusForbidden, "api_key_expired", "API key has expired")
			return
		}

		clientIP := getClientIP(r)
		if !isIPAllowed(clientIP, apiKey.AllowedIPs.String) {
			m.reject(w, r, &apiKey.ID, keyPrefix, http.StatusForbidden, "ip_not_allowed", "Client IP is not allowed for this API key")
			return
		}

		if !isRouteAllowed(r.URL.Path, apiKey.AllowedRoutes.String) {
			m.reject(w, r, &apiKey.ID, keyPrefix, http.StatusForbidden, "route_not_allowed", "This API key is not allowed to access the requested route")
			return
		}

		if apiKey.RateLimitPerMinute > 0 && !m.limiter.Allow(apiKey.ID, apiKey.RateLimitPerMinute) {
			m.reject(w, r, &apiKey.ID, keyPrefix, http.StatusTooManyRequests, "rate_limit_exceeded", "API key rate limit exceeded")
			return
		}

		if err := m.queries.TouchAPIKeyLastUsed(apiKey.ID); err != nil {
			log.Printf("failed to update api key last_used_at: %v", err)
		}

		m.queries.LogAPIKeyUsage(&apiKey.ID, apiKey.KeyPrefix, r.Method, r.URL.Path, clientIP, r.UserAgent(), "allowed", "")

		ctx := context.WithValue(r.Context(), APIKeyContextKey, apiKey)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *APIKeyMiddleware) reject(w http.ResponseWriter, r *http.Request, apiKeyID *uint64, keyPrefix string, httpStatus int, code string, message string) {
	clientIP := getClientIP(r)
	m.queries.LogAPIKeyUsage(apiKeyID, keyPrefix, r.Method, r.URL.Path, clientIP, r.UserAgent(), code, message)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":   code,
		"message": message,
	})
}

func getClientIP(r *http.Request) string {
	forwardedFor := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if forwardedFor != "" {
		parts := strings.Split(forwardedFor, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}

	realIP := strings.TrimSpace(r.Header.Get("X-Real-IP"))
	if realIP != "" {
		return realIP
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func isIPAllowed(clientIP string, allowedRaw string) bool {
	allowedRaw = strings.TrimSpace(allowedRaw)
	if allowedRaw == "" {
		return true
	}

	client := net.ParseIP(clientIP)
	if client == nil {
		return false
	}

	for _, item := range splitCSV(allowedRaw) {
		if item == "" {
			continue
		}
		if strings.Contains(item, "/") {
			_, cidr, err := net.ParseCIDR(item)
			if err == nil && cidr.Contains(client) {
				return true
			}
			continue
		}
		if ip := net.ParseIP(item); ip != nil && ip.Equal(client) {
			return true
		}
	}
	return false
}

func isRouteAllowed(path string, allowedRaw string) bool {
	allowedRaw = strings.TrimSpace(allowedRaw)
	if allowedRaw == "" {
		return true
	}

	for _, pattern := range splitCSV(allowedRaw) {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		if pattern == "*" || path == pattern {
			return true
		}
		if strings.HasSuffix(pattern, "*") {
			prefix := strings.TrimSuffix(pattern, "*")
			if strings.HasPrefix(path, prefix) {
				return true
			}
		}
	}
	return false
}

func splitCSV(raw string) []string {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '\n' || r == ';'
	})
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

type MemoryRateLimiter struct {
	mu     sync.Mutex
	window time.Duration
	items  map[uint64]*rateLimitBucket
}

type rateLimitBucket struct {
	windowStartedAt time.Time
	count           int
}

func NewMemoryRateLimiter(window time.Duration) *MemoryRateLimiter {
	return &MemoryRateLimiter{
		window: window,
		items:  make(map[uint64]*rateLimitBucket),
	}
}

func (l *MemoryRateLimiter) Allow(apiKeyID uint64, limit int) bool {
	if limit <= 0 {
		return true
	}

	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()

	bucket, ok := l.items[apiKeyID]
	if !ok || now.Sub(bucket.windowStartedAt) >= l.window {
		l.items[apiKeyID] = &rateLimitBucket{windowStartedAt: now, count: 1}
		return true
	}

	if bucket.count >= limit {
		return false
	}

	bucket.count++
	return true
}
