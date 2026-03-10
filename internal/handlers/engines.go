package handlers

import (
	"net/http"

	"go-tecdoc-api/internal/models"
)

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