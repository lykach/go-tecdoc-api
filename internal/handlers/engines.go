package handlers

import (
	"net/http"
	"strings"

	"go-tecdoc-api/internal/models"
)

// GetEngines returns the TecDoc ENGINES master table using bounded pagination.
// GET /api/v1/engines?manufacturer_id=1505&limit=500&offset=0
func (h *Handler) GetEngines(w http.ResponseWriter, r *http.Request) {
	page, err := parsePageParams(r, 100, 500)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	var manufacturerID *int
	if strings.TrimSpace(r.URL.Query().Get("manufacturer_id")) != "" {
		id, err := parsePositiveQueryInt(r, "manufacturer_id", 0)
		if err != nil {
			h.respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		manufacturerID = &id
	}

	engines, err := h.queries.GetEngines(manufacturerID, h.getLanguageID(r), page.Limit, page.Offset)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to fetch engines")
		return
	}
	total, err := h.queries.CountEngines(manufacturerID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to count engines")
		return
	}

	h.respondJSON(w, http.StatusOK, models.EngineListResponse{
		Engines: engines,
		Total:   total,
		Limit:   page.Limit,
		Offset:  page.Offset,
	})
}

// GetEngineDetails повертає детальну інформацію про двигун
func (h *Handler) GetEngineDetails(w http.ResponseWriter, r *http.Request) {
	engID, err := getPathInt(r, "id")
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid engine ID")
		return
	}

	languageID := h.getLanguageID(r)

	engineDetail, err := h.queries.GetEngineByID(engID, languageID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to fetch engine details")
		return
	}

	if engineDetail == nil {
		h.respondError(w, http.StatusNotFound, "Engine not found")
		return
	}

	h.respondJSON(w, http.StatusOK, models.EngineDetailResponse{
		Engine: *engineDetail,
	})
}
