package handlers

import (
"database/sql"
"net/http"
)

// GetLanguages повертає список всіх мов
func (h *Handler) GetLanguages(w http.ResponseWriter, r *http.Request) {
languages, err := h.queries.GetLanguages()
if err != nil {
h.respondError(w, http.StatusInternalServerError, "Failed to fetch languages")
return
}

h.respondJSON(w, http.StatusOK, map[string]interface{}{
"languages": languages,
})
}

// GetLanguageByID повертає деталі мови за ID
func (h *Handler) GetLanguageByID(w http.ResponseWriter, r *http.Request) {
id, err := getPathInt(r, "id")
if err != nil {
h.respondError(w, http.StatusBadRequest, "Invalid language ID")
return
}

language, err := h.queries.GetLanguageByID(id)
if err != nil {
if err == sql.ErrNoRows {
h.respondError(w, http.StatusNotFound, "Language not found")
return
}
h.respondError(w, http.StatusInternalServerError, "Failed to fetch language")
return
}

h.respondJSON(w, http.StatusOK, language)
}
