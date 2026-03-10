package handlers

import (
"net/http"
"go-tecdoc-api/internal/models"
)

func (h *Handler) GetManufacturers(w http.ResponseWriter, r *http.Request) {
vehicleType := getStringParam(r, "vehicle_type", "PC")
limit := getIntParam(r, "limit", 50)
offset := getIntParam(r, "offset", 0)

if limit > 500 {
limit = 500
}
if limit < 1 {
limit = 50
}

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
"total":        total,
"limit":        limit,
"offset":       offset,
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
