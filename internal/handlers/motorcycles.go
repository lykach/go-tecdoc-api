package handlers

import (
	"go-tecdoc-api/internal/models"
	"net/http"
)

// GetMotorcycles повертає список мотоциклів для серії моделі
func (h *Handler) GetMotorcycles(w http.ResponseWriter, r *http.Request) {
	msID, err := getPathInt(r, "id")
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid model series ID")
		return
	}

	page, err := parsePageParams(r, 50, 500)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	limit, offset := page.Limit, page.Offset
	languageID := h.getLanguageID(r)
	countryID := h.getCountryID(r)

	motorcycles, err := h.queries.GetMotorcycles(msID, languageID, countryID, limit, offset)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to fetch motorcycles")
		return
	}

	total, err := h.queries.CountMotorcycles(msID, countryID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to count motorcycles")
		return
	}

	h.respondJSON(w, http.StatusOK, models.MotorcyclesResponse{
		Motorcycles: motorcycles,
		Total:       total,
		Limit:       limit,
		Offset:      offset,
	})
}

// GetMotorcycleDetails повертає детальну інформацію про мотоцикл
func (h *Handler) GetMotorcycleDetails(w http.ResponseWriter, r *http.Request) {
	mtbID, err := getPathInt(r, "id")
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid motorcycle ID")
		return
	}

	languageID := h.getLanguageID(r)
	countryID := h.getCountryID(r)

	vehicleDetail, err := h.queries.GetMotorcycleByID(mtbID, languageID, countryID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to fetch motorcycle details")
		return
	}

	if vehicleDetail == nil {
		h.respondError(w, http.StatusNotFound, "Motorcycle not found")
		return
	}

	h.respondJSON(w, http.StatusOK, models.VehicleDetailResponse{
		Vehicle: *vehicleDetail,
	})
}
