package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"go-tecdoc-api/internal/database"
)

var errInvalidQueryParameter = errors.New("invalid query parameter")

type pageParams struct {
	Limit  int
	Offset int
}

// parsePageParams validates limit/offset strictly instead of silently coercing
// malformed or negative values. This keeps client bugs visible and prevents
// unexpectedly large queries from reaching MariaDB.
func parsePageParams(r *http.Request, defaultLimit, maxLimit int) (pageParams, error) {
	if defaultLimit < 1 {
		defaultLimit = 100
	}
	if maxLimit < defaultLimit {
		maxLimit = defaultLimit
	}
	limit := defaultLimit
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 1 || v > maxLimit {
			return pageParams{}, fmt.Errorf("%w: limit must be an integer between 1 and %d", errInvalidQueryParameter, maxLimit)
		}
		limit = v
	}
	offset := 0
	if raw := strings.TrimSpace(r.URL.Query().Get("offset")); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 0 {
			return pageParams{}, fmt.Errorf("%w: offset must be a non-negative integer", errInvalidQueryParameter)
		}
		offset = v
	}
	return pageParams{Limit: limit, Offset: offset}, nil
}

func parsePositiveQueryInt(r *http.Request, key string, defaultValue int) (int, error) {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return defaultValue, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return 0, fmt.Errorf("%w: %s must be a positive integer", errInvalidQueryParameter, key)
	}
	return v, nil
}

func parseOptionalNonNegativeQueryInt(r *http.Request, key string) (int, error) {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return 0, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < 0 {
		return 0, fmt.Errorf("%w: %s must be a non-negative integer", errInvalidQueryParameter, key)
	}
	return v, nil
}

func parseBoolQuery(r *http.Request, key string, defaultValue bool) (bool, error) {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return defaultValue, nil
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("%w: %s must be true or false", errInvalidQueryParameter, key)
	}
	return v, nil
}

func parseResponseFormat(r *http.Request, allowed ...string) (string, error) {
	raw := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
	if raw == "" {
		raw = "json"
	}
	for _, v := range allowed {
		if raw == strings.ToLower(v) {
			return raw, nil
		}
	}
	return "", fmt.Errorf("%w: unsupported format %q", errInvalidQueryParameter, raw)
}

type Handler struct {
	queries           *database.Queries
	defaultLanguageID int
	defaultCountryID  int
}

func New(queries *database.Queries) *Handler {
	langID := 48 // Ukrainian by default
	if envLangID := os.Getenv("DEFAULT_LANGUAGE_ID"); envLangID != "" {
		if id, err := strconv.Atoi(envLangID); err == nil {
			langID = id
		}
	}

	countryID := 258 // Ukraine by default
	if envCountryID := os.Getenv("DEFAULT_COUNTRY_ID"); envCountryID != "" {
		if id, err := strconv.Atoi(envCountryID); err == nil {
			countryID = id
		}
	}

	return &Handler{
		queries:           queries,
		defaultLanguageID: langID,
		defaultCountryID:  countryID,
	}
}

// Helper functions
func (h *Handler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *Handler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]string{"error": message})
}

func getIntParam(r *http.Request, key string, defaultValue int) int {
	if val := r.URL.Query().Get(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getStringParam(r *http.Request, key string, defaultValue string) string {
	if val := r.URL.Query().Get(key); val != "" {
		return val
	}
	return defaultValue
}

func getBoolParam(r *http.Request, key string, defaultValue bool) bool {
	val := strings.TrimSpace(r.URL.Query().Get(key))
	if val == "" {
		return defaultValue
	}
	parsed, err := strconv.ParseBool(val)
	if err != nil {
		return defaultValue
	}
	return parsed
}

func getPathInt(r *http.Request, key string) (int, error) {
	vars := mux.Vars(r)
	v, err := strconv.Atoi(vars[key])
	if err != nil || v <= 0 {
		return 0, fmt.Errorf("%w: %s must be a positive integer", errInvalidQueryParameter, key)
	}
	return v, nil
}

func (h *Handler) getLanguageID(r *http.Request) int {
	return getIntParam(r, "language_id", h.defaultLanguageID)
}

func (h *Handler) getCountryID(r *http.Request) int {
	return getIntParam(r, "country_id", h.defaultCountryID)
}

// validateCommonQueryContext is used by the API middleware to reject malformed
// localization parameters before a handler can silently fall back to defaults.
func (h *Handler) validateCommonQueryContext(r *http.Request) error {
	if _, err := parsePositiveQueryInt(r, "language_id", h.defaultLanguageID); err != nil {
		return err
	}
	if _, err := parsePositiveQueryInt(r, "country_id", h.defaultCountryID); err != nil {
		return err
	}
	return nil
}

// ValidateCommonQueryMiddleware rejects malformed language_id/country_id on
// every authenticated API route. Authentication remains the outer gate, so an
// unauthenticated caller never receives validation details for protected APIs.
func (h *Handler) ValidateCommonQueryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := h.validateCommonQueryContext(r); err != nil {
			h.respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		next.ServeHTTP(w, r)
	})
}
