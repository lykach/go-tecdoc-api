package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"go-tecdoc-api/internal/models"
)

func normalizeProductGroupTreeType(raw string) (string, error) {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "PC", "LCV", "E-PC", "E-LCV":
		return "PC", nil
	case "CV":
		return "CV", nil
	case "MOTORCYCLE", "MC", "MTB":
		return "Motorcycle", nil
	case "ENGINE", "ENGINES":
		return "Engine", nil
	case "AXLE", "AXLES":
		return "Axle", nil
	case "UNIVERSAL", "UNIVERSAL PARTS", "UNIVERSAL_PARTS":
		return "Universal Parts", nil
	default:
		return "", errInvalidQueryParameterWithMessage("vehicle_type must resolve to one of PC, CV, Motorcycle, Engine, Axle, Universal Parts")
	}
}

func normalizeVehicleProductGroupTargetType(raw string) (string, error) {
	targetType, err := normalizeProductGroupTreeType(raw)
	if err != nil {
		return "", err
	}
	if targetType != "PC" && targetType != "CV" && targetType != "Motorcycle" {
		return "", errInvalidQueryParameterWithMessage("vehicle_type must resolve to one of PC, CV, Motorcycle")
	}
	return targetType, nil
}

// GetProductGroups - отримати список категорій запчастин (корені дерева)
func (h *Handler) GetProductGroups(w http.ResponseWriter, r *http.Request) {
	vehicleType, err := normalizeProductGroupTreeType(getStringParam(r, "vehicle_type", "PC"))
	if err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	page, err := parsePageParams(r, 100, 500)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	limit, offset := page.Limit, page.Offset
	languageID := h.getLanguageID(r)

	categories, err := h.queries.GetProductGroups(vehicleType, languageID, limit, offset)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to fetch product groups: "+err.Error())
		return
	}

	response := models.CategoriesResponse{
		Categories: categories,
	}

	h.respondJSON(w, http.StatusOK, response)
}

// GetCategoryChildren - отримати дочірні категорії
func (h *Handler) GetCategoryChildren(w http.ResponseWriter, r *http.Request) {
	parentID, err := getPathInt(r, "id")
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid category ID")
		return
	}

	vehicleType, err := normalizeProductGroupTreeType(getStringParam(r, "vehicle_type", "PC"))
	if err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	languageID := h.getLanguageID(r)

	categories, err := h.queries.GetCategoryChildren(parentID, vehicleType, languageID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to fetch category children: "+err.Error())
		return
	}

	response := models.CategoriesResponse{
		Categories: categories,
	}

	h.respondJSON(w, http.StatusOK, response)
}

// GetCarProductGroups - отримати категорії запчастин для конкретного автомобіля
func (h *Handler) GetCarProductGroups(w http.ResponseWriter, r *http.Request) {
	pcID, err := getPathInt(r, "id")
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid car ID")
		return
	}

	vehicleType, err := normalizeVehicleProductGroupTargetType(getStringParam(r, "vehicle_type", "PC"))
	if err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	page, err := parsePageParams(r, 100, 500)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	limit, offset := page.Limit, page.Offset
	languageID := h.getLanguageID(r)

	categories, err := h.queries.GetCarProductGroups(pcID, vehicleType, languageID, limit, offset)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to fetch car product groups: "+err.Error())
		return
	}

	response := models.CategoriesResponse{
		Categories: categories,
	}

	h.respondJSON(w, http.StatusOK, response)
}

// GetProductGroupArticles - отримати запчастини для категорії з підтримкою фільтрації
func (h *Handler) GetProductGroupArticles(w http.ResponseWriter, r *http.Request) {
	strID, err := getPathInt(r, "id")
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid product group ID")
		return
	}

	pcID, err := parsePositiveQueryInt(r, "car_id", 0)
	if err != nil || pcID <= 0 {
		h.respondError(w, http.StatusBadRequest, "car_id parameter is required and must be a positive integer")
		return
	}

	vehicleType, err := normalizeVehicleProductGroupTargetType(getStringParam(r, "vehicle_type", "PC"))
	if err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
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

	// Перевіряємо чи є фільтри по критеріям
	criteriaStr := r.URL.Query().Get("criteria")
	var articles []models.Article
	var criteriaFilters map[int]string

	if criteriaStr != "" {
		// Використовуємо фільтрацію за критеріями
		var parseErr error
		criteriaFilters, parseErr = parseCriteriaFilters(criteriaStr)
		if parseErr != nil {
			h.respondError(w, http.StatusBadRequest, parseErr.Error())
			return
		}
		articles, err = h.queries.GetProductGroupArticlesWithCriteria(
			strID, pcID, vehicleType, languageID, countryID,
			criteriaFilters, limit, offset,
		)
	} else {
		// Використовуємо звичайний метод без фільтрації
		articles, err = h.queries.GetProductGroupArticles(
			strID, pcID, vehicleType, languageID, countryID,
			limit, offset,
		)
	}

	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to fetch articles: "+err.Error())
		return
	}

	// Отримати загальну кількість
	total, err := h.queries.CountProductGroupArticles(strID, pcID, vehicleType, countryID, criteriaFilters)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to count articles: "+err.Error())
		return
	}

	response := map[string]interface{}{
		"articles": articles,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	}

	h.respondJSON(w, http.StatusOK, response)
}

// parseCriteriaFilters parses criteria filters from "6:12,92:9". Malformed
// filters are rejected instead of being silently ignored, because silently
// dropping a restriction can turn a narrow catalogue query into a broad one.
func parseCriteriaFilters(criteriaStr string) (map[int]string, error) {
	filters := make(map[int]string)
	criteriaStr = strings.TrimSpace(criteriaStr)
	if criteriaStr == "" {
		return filters, nil
	}

	pairs := strings.Split(criteriaStr, ",")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		parts := strings.SplitN(pair, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid criteria filter %q; expected CRITERIA_ID:value", pair)
		}
		criID, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil || criID <= 0 {
			return nil, fmt.Errorf("invalid criteria id in filter %q", pair)
		}
		value := strings.TrimSpace(parts[1])
		if value == "" {
			return nil, fmt.Errorf("criteria value must not be empty for id %d", criID)
		}
		filters[criID] = value
	}
	return filters, nil
}
