package handlers

import (
	"go-tecdoc-api/internal/models"
	"net/http"
	"strings"
)

// GetPassengerCars - отримати список легкових автомобілів для серії моделі
// GET /api/v1/models/{id}/cars?limit=50&offset=0&language_id=48&country_id=62
func (h *Handler) GetPassengerCars(w http.ResponseWriter, r *http.Request) {
	msID, err := getPathInt(r, "id")
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid model series ID")
		return
	}

	page, err := parsePageParams(r, 100, 500)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	limit, offset := page.Limit, page.Offset
	vehicleType := strings.TrimSpace(r.URL.Query().Get("vehicle_type"))
	if vehicleType == "" {
		vehicleType = "PC"
	}
	switch vehicleType {
	case "PC", "LCV", "E-PC", "E-LCV":
	default:
		h.respondError(w, http.StatusBadRequest, "invalid query parameter: vehicle_type must be one of PC, LCV, E-PC, E-LCV")
		return
	}
	languageID := h.getLanguageID(r)
	countryID := h.getCountryID(r)

	cars, err := h.queries.GetPassengerCars(msID, vehicleType, languageID, countryID, limit, offset)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to fetch passenger cars: "+err.Error())
		return
	}

	// Отримати загальну кількість
	total, err := h.queries.CountPassengerCars(msID, vehicleType)
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
