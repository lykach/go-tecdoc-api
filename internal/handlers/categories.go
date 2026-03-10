package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"go-tecdoc-api/internal/models"
)

// GetProductGroups - отримати список категорій запчастин (корені дерева)
func (h *Handler) GetProductGroups(w http.ResponseWriter, r *http.Request) {
	vehicleType := getStringParam(r, "vehicle_type", "PC")
	limit := getIntParam(r, "limit", 100)
	offset := getIntParam(r, "offset", 0)
	languageID := h.getLanguageID(r)

	if limit > 500 {
		limit = 500
	}
	if limit < 1 {
		limit = 100
	}

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

	vehicleType := getStringParam(r, "vehicle_type", "PC")
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

	vehicleType := getStringParam(r, "vehicle_type", "PC")
	limit := getIntParam(r, "limit", 100)
	offset := getIntParam(r, "offset", 0)
	languageID := h.getLanguageID(r)

	if limit > 500 {
		limit = 500
	}
	if limit < 1 {
		limit = 100
	}

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

	pcID := getIntParam(r, "car_id", 0)
	if pcID == 0 {
		h.respondError(w, http.StatusBadRequest, "car_id parameter is required")
		return
	}

	vehicleType := getStringParam(r, "vehicle_type", "PC")
	limit := getIntParam(r, "limit", 50)
	offset := getIntParam(r, "offset", 0)
	languageID := h.getLanguageID(r)
	countryID := h.getCountryID(r)

	if limit > 500 {
		limit = 500
	}
	if limit < 1 {
		limit = 50
	}

	// Перевіряємо чи є фільтри по критеріям
	criteriaStr := r.URL.Query().Get("criteria")
	var articles []models.Article

	if criteriaStr != "" {
		// Використовуємо фільтрацію за критеріями
		criteriaFilters := parseCriteriaFilters(criteriaStr)
		articles, err = h.queries.GetProductGroupArticlesWithCriteria(
			strID, pcID, languageID, countryID,
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
	total, err := h.queries.CountProductGroupArticles(strID, pcID, vehicleType)
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

// parseCriteriaFilters парсить criteria фільтри з формату "6:12,92:9"
func parseCriteriaFilters(criteriaStr string) map[int]string {
	filters := make(map[int]string)
	if criteriaStr == "" {
		return filters
	}

	pairs := strings.Split(criteriaStr, ",")
	for _, pair := range pairs {
		parts := strings.Split(pair, ":")
		if len(parts) == 2 {
			criID, err := strconv.Atoi(strings.TrimSpace(parts[0]))
			if err == nil {
				filters[criID] = strings.TrimSpace(parts[1])
			}
		}
	}
	return filters
}