package handlers

import (
	"net/http"
	"go-tecdoc-api/internal/models"
)

// GetCommercialVehicles повертає список вантажівок для серії моделі
func (h *Handler) GetCommercialVehicles(w http.ResponseWriter, r *http.Request) {
	msID, err := getPathInt(r, "id")
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid model series ID")
		return
	}

	limit := getIntParam(r, "limit", 50)
	offset := getIntParam(r, "offset", 0)
	languageID := h.getLanguageID(r)
	countryID := h.getCountryID(r)

	vehicles, err := h.queries.GetCommercialVehicles(msID, languageID, countryID, limit, offset)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to fetch commercial vehicles")
		return
	}

	total, _ := h.queries.CountCommercialVehicles(msID)

	h.respondJSON(w, http.StatusOK, models.CommercialVehiclesResponse{
		Vehicles: vehicles,
		Total:    total,
	})
}
// GetCommercialVehicleDetails повертає детальну інформацію про вантажівку
func (h *Handler) GetCommercialVehicleDetails(w http.ResponseWriter, r *http.Request) {
cvID, err := getPathInt(r, "id")
if err != nil {
h.respondError(w, http.StatusBadRequest, "Invalid commercial vehicle ID")
return
}

languageID := h.getLanguageID(r)
countryID := h.getCountryID(r)

vehicleDetail, err := h.queries.GetCommercialVehicleByID(cvID, languageID, countryID)
if err != nil {
h.respondError(w, http.StatusInternalServerError, "Failed to fetch commercial vehicle details")
return
}

if vehicleDetail == nil {
h.respondError(w, http.StatusNotFound, "Commercial vehicle not found")
return
}

h.respondJSON(w, http.StatusOK, models.VehicleDetailResponse{
Vehicle: *vehicleDetail,
})
}
