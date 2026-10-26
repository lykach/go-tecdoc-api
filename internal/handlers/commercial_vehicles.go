package handlers

import (
	"go-tecdoc-api/internal/models"
	"net/http"
)

// GetCommercialVehicles повертає список вантажівок для серії моделі
func (h *Handler) GetCommercialVehicles(w http.ResponseWriter, r *http.Request) {
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

	vehicles, err := h.queries.GetCommercialVehicles(msID, languageID, countryID, limit, offset)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to fetch commercial vehicles")
		return
	}

	total, err := h.queries.CountCommercialVehicles(msID, countryID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to count commercial vehicles")
		return
	}

	h.respondJSON(w, http.StatusOK, models.CommercialVehiclesResponse{
		Vehicles: vehicles,
		Total:    total,
		Limit:    limit,
		Offset:   offset,
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
