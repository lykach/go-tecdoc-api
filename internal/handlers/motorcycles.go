package handlers

import (
	"net/http"
	"go-tecdoc-api/internal/models"
)

// GetMotorcycles повертає список мотоциклів для серії моделі
func (h *Handler) GetMotorcycles(w http.ResponseWriter, r *http.Request) {
	msID, err := getPathInt(r, "id")
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid model series ID")
		return
	}

	limit := getIntParam(r, "limit", 50)
	offset := getIntParam(r, "offset", 0)
	languageID := h.getLanguageID(r)
	countryID := h.getCountryID(r)

	motorcycles, err := h.queries.GetMotorcycles(msID, languageID, countryID, limit, offset)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to fetch motorcycles")
		return
	}

	total, _ := h.queries.CountMotorcycles(msID)

	h.respondJSON(w, http.StatusOK, models.MotorcyclesResponse{
		Motorcycles: motorcycles,
		Total:       total,
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
