package handlers

import (
	"net/http"
	"go-tecdoc-api/internal/models"
)

// GetModelSeries - отримати серії моделей для виробника
// GET /api/v1/manufacturers/{id}/models?vehicle_type=PC&limit=50&offset=0&language_id=48&country_id=62
func (h *Handler) GetModelSeries(w http.ResponseWriter, r *http.Request) {
	mfaID, err := getPathInt(r, "id")
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid manufacturer ID")
		return
	}

	vehicleType := getStringParam(r, "vehicle_type", "PC")
	limit := getIntParam(r, "limit", 100)
	offset := getIntParam(r, "offset", 0)
	languageID := h.getLanguageID(r)
	countryID := h.getCountryID(r)

	// Валідація
	if limit > 500 {
		limit = 500
	}
	if limit < 1 {
		limit = 100
	}

	modelSeries, err := h.queries.GetModelSeries(mfaID, vehicleType, languageID, countryID, limit, offset)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to fetch model series: "+err.Error())
		return
	}

	// Отримати загальну кількість
	total, err := h.queries.CountModelSeries(mfaID, vehicleType)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to count model series: "+err.Error())
		return
	}

	response := map[string]interface{}{
		"models": modelSeries,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	}

	h.respondJSON(w, http.StatusOK, response)
}

// GetModelSeriesDetails - отримати детальну інформацію про серію моделі
// GET /api/v1/models/{id}?language_id=48&country_id=62
func (h *Handler) GetModelSeriesDetails(w http.ResponseWriter, r *http.Request) {
	msID, err := getPathInt(r, "id")
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid model series ID")
		return
	}

	languageID := h.getLanguageID(r)
	countryID := h.getCountryID(r)

	modelDetail, err := h.queries.GetModelSeriesByID(msID, languageID, countryID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to fetch model series: "+err.Error())
		return
	}

	if modelDetail == nil {
		h.respondError(w, http.StatusNotFound, "Model series not found")
		return
	}

	response := models.ModelDetailResponse{
		Model: *modelDetail,
	}

	h.respondJSON(w, http.StatusOK, response)
}