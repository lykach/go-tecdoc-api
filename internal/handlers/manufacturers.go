package handlers

import (
	"go-tecdoc-api/internal/models"
	"net/http"
)

func (h *Handler) GetManufacturers(w http.ResponseWriter, r *http.Request) {
	vehicleType := getStringParam(r, "vehicle_type", "PC")
	page, err := parsePageParams(r, 50, 500)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	limit, offset := page.Limit, page.Offset

	manufacturers, err := h.queries.GetManufacturers(vehicleType, limit, offset)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to fetch manufacturers: "+err.Error())
		return
	}

	total, err := h.queries.CountManufacturers(vehicleType)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to count manufacturers: "+err.Error())
		return
	}

	response := map[string]interface{}{
		"manufacturers": manufacturers,
		"total":         total,
		"limit":         limit,
		"offset":        offset,
	}

	h.respondJSON(w, http.StatusOK, response)
}

func (h *Handler) GetManufacturerByID(w http.ResponseWriter, r *http.Request) {
	id, err := getPathInt(r, "id")
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid manufacturer ID")
		return
	}

	manufacturer, err := h.queries.GetManufacturerByID(id)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to fetch manufacturer: "+err.Error())
		return
	}

	if manufacturer == nil {
		h.respondError(w, http.StatusNotFound, "Manufacturer not found")
		return
	}

	response := models.ManufacturerDetailResponse{
		Manufacturer: *manufacturer,
	}

	h.respondJSON(w, http.StatusOK, response)
}
