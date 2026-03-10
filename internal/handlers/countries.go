package handlers

import (
"database/sql"
"net/http"
)

// GetCountries повертає список країн
func (h *Handler) GetCountries(w http.ResponseWriter, r *http.Request) {
langID := h.getLanguageID(r)
includeGroups := r.URL.Query().Get("include_groups") == "true"

countries, err := h.queries.GetCountries(langID, includeGroups)
if err != nil {
h.respondError(w, http.StatusInternalServerError, "Failed to fetch countries")
return
}

h.respondJSON(w, http.StatusOK, map[string]interface{}{
"countries": countries,
})
}

// GetCountryByID повертає деталі країни за ID
func (h *Handler) GetCountryByID(w http.ResponseWriter, r *http.Request) {
id, err := getPathInt(r, "id")
if err != nil {
h.respondError(w, http.StatusBadRequest, "Invalid country ID")
return
}

langID := h.getLanguageID(r)

country, err := h.queries.GetCountryByID(id, langID)
if err != nil {
if err == sql.ErrNoRows {
h.respondError(w, http.StatusNotFound, "Country not found")
return
}
h.respondError(w, http.StatusInternalServerError, "Failed to fetch country")
return
}

h.respondJSON(w, http.StatusOK, country)
}
