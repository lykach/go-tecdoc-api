package handlers

import (
	"net/http"
	"go-tecdoc-api/internal/models"
)

// GetPassengerCars - отримати список легкових автомобілів для серії моделі
// GET /api/v1/models/{id}/cars?limit=50&offset=0&language_id=48&country_id=62
func (h *Handler) GetPassengerCars(w http.ResponseWriter, r *http.Request) {
	msID, err := getPathInt(r, "id")
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid model series ID")
		return
	}

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

	cars, err := h.queries.GetPassengerCars(msID, languageID, countryID, limit, offset)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to fetch passenger cars: "+err.Error())
		return
	}

	// Отримати загальну кількість
	total, err := h.queries.CountPassengerCars(msID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to count passenger cars: "+err.Error())
		return
	}

	response := map[string]interface{}{
		"vehicles": cars,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	}

	h.respondJSON(w, http.StatusOK, response)
}

// GetCarDetails - отримати детальну інформацію про автомобіль
// GET /api/v1/cars/{id}?language_id=48&country_id=62
func (h *Handler) GetCarDetails(w http.ResponseWriter, r *http.Request) {
	pcID, err := getPathInt(r, "id")
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid car ID")
		return
	}

	languageID := h.getLanguageID(r)
	countryID := h.getCountryID(r)

	carDetail, err := h.queries.GetPassengerCarByID(pcID, languageID, countryID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to fetch car details: "+err.Error())
		return
	}

	if carDetail == nil {
		h.respondError(w, http.StatusNotFound, "Car not found")
		return
	}

	response := models.VehicleDetailResponse{
		Vehicle: *carDetail,
	}

	h.respondJSON(w, http.StatusOK, response)
}