package handlers

import (
"encoding/json"
"net/http"
"os"
"strconv"

"go-tecdoc-api/internal/database"
"github.com/gorilla/mux"
)

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

func getPathInt(r *http.Request, key string) (int, error) {
vars := mux.Vars(r)
return strconv.Atoi(vars[key])
}

func (h *Handler) getLanguageID(r *http.Request) int {
return getIntParam(r, "language_id", h.defaultLanguageID)
}

func (h *Handler) getCountryID(r *http.Request) int {
return getIntParam(r, "country_id", h.defaultCountryID)
}
