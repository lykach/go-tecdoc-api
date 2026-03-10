package handlers

import (
	"net/http"
	"strconv"
	"go-tecdoc-api/internal/models"
)

// SearchArticleByNumber - універсальний пошук запчастин
// GET /api/v1/search/article?number=0001108466&limit=20&offset=0&language_id=48&country_id=62
func (h *Handler) SearchArticleByNumber(w http.ResponseWriter, r *http.Request) {
	searchNumber := getStringParam(r, "number", "")
	if searchNumber == "" {
		h.respondError(w, http.StatusBadRequest, "Search number is required")
		return
	}

	limit := getIntParam(r, "limit", 50)
	offset := getIntParam(r, "offset", 0)
	languageID := h.getLanguageID(r)
	countryID := h.getCountryID(r)

	// Валідація
	if limit > 500 {
		limit = 500
	}
	if limit < 1 {
		limit = 50
	}

	results, err := h.queries.SearchArticlesByNumber(searchNumber, languageID, countryID, limit, offset)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to search articles: "+err.Error())
		return
	}

	response := models.SearchResponse{
		Results: results,
	}

	h.respondJSON(w, http.StatusOK, response)
}

// SearchByOEM - пошук аналогів за OEM номером
// GET /api/v1/search/oem?number=7700115294&limit=20&offset=0&language_id=48&country_id=62
func (h *Handler) SearchByOEM(w http.ResponseWriter, r *http.Request) {
	oemNumber := getStringParam(r, "number", "")
	if oemNumber == "" {
		h.respondError(w, http.StatusBadRequest, "OEM number is required")
		return
	}

	limit := getIntParam(r, "limit", 50)
	offset := getIntParam(r, "offset", 0)
	languageID := h.getLanguageID(r)
	countryID := h.getCountryID(r)

	// Валідація
	if limit > 500 {
		limit = 500
	}
	if limit < 1 {
		limit = 50
	}

	results, err := h.queries.SearchByOEMNumber(oemNumber, languageID, countryID, limit, offset)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to search by OEM: "+err.Error())
		return
	}

	response := models.SearchResponse{
		Results: results,
	}

	h.respondJSON(w, http.StatusOK, response)
}

// SearchByKBA - пошук за німецьким номером KBA (TODO)
func (h *Handler) SearchByKBA(w http.ResponseWriter, r *http.Request) {
	h.respondError(w, http.StatusNotImplemented, "KBA search not implemented yet")
}
// SearchAnalogs - пошук IAM аналогів
// GET /api/v1/search/analog?art_id=29  OR  ?search_number=0001106017
func (h *Handler) SearchAnalogs(w http.ResponseWriter, r *http.Request) {
artIDStr := getStringParam(r, "art_id", "")
searchNumber := getStringParam(r, "search_number", "")

if artIDStr == "" && searchNumber == "" {
h.respondError(w, http.StatusBadRequest, "Either art_id or search_number is required")
return
}

limit := getIntParam(r, "limit", 50)
offset := getIntParam(r, "offset", 0)
languageID := h.getLanguageID(r)
countryID := h.getCountryID(r)

// Валідація
if limit > 500 {
limit = 500
}
if limit < 1 {
limit = 50
}

var results []models.SearchResult
var err error

if artIDStr != "" {
// Пошук за ID запчастини
artID, err := strconv.Atoi(artIDStr)
if err != nil {
h.respondError(w, http.StatusBadRequest, "Invalid art_id")
return
}
results, err = h.queries.SearchAnalogsByArticleID(artID, languageID, countryID, limit, offset)
} else {
// Пошук за номером
results, err = h.queries.SearchAnalogsByNumber(searchNumber, languageID, countryID, limit, offset)
}

if err != nil {
h.respondError(w, http.StatusInternalServerError, "Failed to search analogs: "+err.Error())
return
}

response := models.SearchResponse{
Results: results,
}

h.respondJSON(w, http.StatusOK, response)
}

// SearchOEMByOEM - пошук OEM аналогів для OEM номера
// GET /api/v1/search/oem-oem?oem_number=4853009T50
func (h *Handler) SearchOEMByOEM(w http.ResponseWriter, r *http.Request) {
oemNumber := getStringParam(r, "oem_number", "")
if oemNumber == "" {
h.respondError(w, http.StatusBadRequest, "OEM number is required")
return
}

limit := getIntParam(r, "limit", 50)
offset := getIntParam(r, "offset", 0)

// Валідація
if limit > 500 {
limit = 500
}
if limit < 1 {
limit = 50
}

results, err := h.queries.SearchOEMByOEMNumber(oemNumber, limit, offset)
if err != nil {
h.respondError(w, http.StatusInternalServerError, "Failed to search OEM cross-references: "+err.Error())
return
}

h.respondJSON(w, http.StatusOK, models.OEMCrossReferenceResponse{
References: results,
Total:      len(results),
})
}
