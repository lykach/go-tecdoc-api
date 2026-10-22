package handlers

import (
	"net/http"
	"strconv"
	"strings"
)

var importVehicleTypeOrder = []string{"PC", "LCV", "E-PC", "E-LCV", "CV", "Motorcycle", "Engine"}

func normalizeImportVehicleTypes(raw string) ([]string, error) {
	selected := make(map[string]struct{}, len(importVehicleTypeOrder))
	if strings.TrimSpace(raw) == "" {
		return append([]string(nil), importVehicleTypeOrder...), nil
	}
	for _, token := range strings.Split(raw, ",") {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		valid := false
		for _, supported := range importVehicleTypeOrder {
			if token == supported {
				valid = true
				break
			}
		}
		if !valid {
			return nil, errInvalidQueryParameterWithMessage("vehicle_types must contain only PC, LCV, E-PC, E-LCV, CV, Motorcycle, Engine")
		}
		selected[token] = struct{}{}
	}
	if len(selected) == 0 {
		return append([]string(nil), importVehicleTypeOrder...), nil
	}
	out := make([]string, 0, len(selected))
	for _, supported := range importVehicleTypeOrder {
		if _, ok := selected[supported]; ok {
			out = append(out, supported)
		}
	}
	return out, nil
}

func errInvalidQueryParameterWithMessage(message string) error {
	return &queryParamError{message: message}
}

type queryParamError struct{ message string }

func (e *queryParamError) Error() string { return "invalid query parameter: " + e.message }

func (h *Handler) GetVehicleImportStats(w http.ResponseWriter, r *http.Request) {
	vehicleTypes, err := normalizeImportVehicleTypes(r.URL.Query().Get("vehicle_types"))
	if err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	stats, err := h.queries.GetVehicleImportStats(vehicleTypes)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to fetch vehicle import stats: "+err.Error())
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]any{
		"vehicle_types":       vehicleTypes,
		"manufacturers_total": stats.ManufacturersTotal,
		"models_total":        stats.ModelsTotal,
		"vehicles_total":      stats.VehiclesTotal,
		"engines_total":       stats.EnginesTotal,
		"by_type":             stats.ByType,
	})
}

func normalizeImportTreeTypes(raw string) ([]string, error) {
	vehicleTypes, err := normalizeImportVehicleTypes(raw)
	if err != nil {
		return nil, err
	}
	selected := make(map[string]struct{}, 4)
	for _, vehicleType := range vehicleTypes {
		switch vehicleType {
		case "PC", "LCV", "E-PC", "E-LCV":
			selected["PC"] = struct{}{}
		case "CV", "Motorcycle", "Engine":
			selected[vehicleType] = struct{}{}
		}
	}
	out := make([]string, 0, len(selected))
	for _, vehicleType := range []string{"PC", "CV", "Motorcycle", "Engine"} {
		if _, ok := selected[vehicleType]; ok {
			out = append(out, vehicleType)
		}
	}
	return out, nil
}

func (h *Handler) GetImportProductGroups(w http.ResponseWriter, r *http.Request) {
	vehicleTypes, err := normalizeImportTreeTypes(r.URL.Query().Get("vehicle_types"))
	if err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	page, err := parsePageParams(r, 500, 500)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	languageID := h.getLanguageID(r)
	categories, total, err := h.queries.GetImportProductGroups(vehicleTypes, languageID, page.Limit, page.Offset)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to fetch import product groups: "+err.Error())
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]any{
		"vehicle_types": vehicleTypes,
		"categories":    categories,
		"total":         total,
		"limit":         page.Limit,
		"offset":        page.Offset,
	})
}

func normalizeImportProductGroupVehicleType(raw string) (string, error) {
	return normalizeVehicleProductGroupTargetType(raw)
}

func parseImportVehicleIDs(raw string) ([]int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, errInvalidQueryParameterWithMessage("vehicle_ids is required")
	}
	parts := strings.Split(raw, ",")
	if len(parts) > 500 {
		return nil, errInvalidQueryParameterWithMessage("vehicle_ids must contain at most 500 IDs")
	}
	seen := make(map[int]struct{}, len(parts))
	ids := make([]int, 0, len(parts))
	for _, rawID := range parts {
		value, err := strconv.Atoi(strings.TrimSpace(rawID))
		if err != nil || value <= 0 {
			return nil, errInvalidQueryParameterWithMessage("vehicle_ids must contain positive integers")
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		ids = append(ids, value)
	}
	if len(ids) == 0 {
		return nil, errInvalidQueryParameterWithMessage("vehicle_ids is required")
	}
	return ids, nil
}

// GetImportVehicleProductGroups returns applicable root product groups for up
// to 500 vehicle IDs in one request. Passenger subtypes intentionally use the
// PC target namespace because LINK_LA_TYP/SEARCH_TREE are family-level.
func (h *Handler) GetImportVehicleProductGroups(w http.ResponseWriter, r *http.Request) {
	vehicleType, err := normalizeImportProductGroupVehicleType(r.URL.Query().Get("vehicle_type"))
	if err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	vehicleIDs, err := parseImportVehicleIDs(r.URL.Query().Get("vehicle_ids"))
	if err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	idsOnly, err := parseBoolQuery(r, "ids_only", false)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if idsOnly {
		items, err := h.queries.GetImportVehicleProductGroupIDs(r.Context(), vehicleIDs, vehicleType)
		if err != nil {
			h.respondError(w, http.StatusInternalServerError, "Failed to fetch import vehicle product group ids: "+err.Error())
			return
		}
		h.respondJSON(w, http.StatusOK, map[string]any{
			"vehicle_type": vehicleType,
			"ids_only":     true,
			"items":        items,
		})
		return
	}

	languageID := h.getLanguageID(r)
	items, err := h.queries.GetImportVehicleProductGroups(r.Context(), vehicleIDs, vehicleType, languageID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to fetch import vehicle product groups: "+err.Error())
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]any{
		"vehicle_type": vehicleType,
		"ids_only":     false,
		"items":        items,
	})
}
