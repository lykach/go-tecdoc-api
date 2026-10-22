package handlers

import (
	"context"
	"net/http"
	"strings"

	"go-tecdoc-api/internal/models"
)

const (
	defaultArticleApplicabilityLimit = 100
	maxArticleApplicabilityLimit     = 500
)

// SearchArticles - пошук запчастин за номером
// GET /api/v1/articles/search?number=0001108466&limit=200&offset=0&language_id=48&country_id=62
func (h *Handler) SearchArticles(w http.ResponseWriter, r *http.Request) {
	searchNumber := getStringParam(r, "number", "")
	if searchNumber == "" {
		h.respondError(w, http.StatusBadRequest, "Search number is required")
		return
	}

	page, err := parsePageParams(r, defaultSearchLimit, maxSearchLimit)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	limit, offset := page.Limit, page.Offset
	languageID := h.getLanguageID(r)
	countryID := h.getCountryID(r)

	ctx, cancel := context.WithTimeout(r.Context(), searchQueryTimeout)
	defer cancel()

	results, err := h.queries.SearchArticlesByNumber(ctx, searchNumber, languageID, countryID, limit, offset)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to search articles: "+err.Error())
		return
	}

	response := models.SearchResponse{
		Results: results,
	}

	h.respondJSON(w, http.StatusOK, response)
}

// GetArticleDetails - отримати детальну інформацію про запчастину
// GET /api/v1/articles/{id}?language_id=48&country_id=62
func (h *Handler) GetArticleDetails(w http.ResponseWriter, r *http.Request) {
	artID, err := getPathInt(r, "id")
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid article ID")
		return
	}

	languageID := h.getLanguageID(r)
	countryID := h.getCountryID(r)
	relatedLimit, err := parsePositiveQueryInt(r, "related_limit", 50)
	if err != nil || relatedLimit > 100 {
		h.respondError(w, http.StatusBadRequest, "related_limit must be an integer between 1 and 100")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), searchQueryTimeout)
	defer cancel()

	articleDetail, err := h.queries.GetArticleByID(ctx, artID, languageID, countryID, relatedLimit)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to fetch article details: "+err.Error())
		return
	}

	if articleDetail == nil {
		h.respondError(w, http.StatusNotFound, "Article not found")
		return
	}

	response := models.ArticleDetailResponse{
		Article: *articleDetail,
	}

	h.respondJSON(w, http.StatusOK, response)
}

// GetArticleApplicability - отримати застосовність запчастини до автомобілів
// GET /api/v1/articles/{id}/applicability?language_id=48&country_id=62

// GetArticleCrossReferences - отримати крос-референси запчастини
// GET /api/v1/articles/{id}/cross-references?language_id=48
func (h *Handler) GetArticleCrossReferences(w http.ResponseWriter, r *http.Request) {
	artID, err := getPathInt(r, "id")
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid article ID")
		return
	}

	languageID := h.getLanguageID(r)
	countryID := h.getCountryID(r)

	ctx, cancel := context.WithTimeout(r.Context(), searchQueryTimeout)
	defer cancel()

	exists, err := h.queries.ArticleExistsForCountry(ctx, artID, countryID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to validate article: "+err.Error())
		return
	}
	if !exists {
		h.respondError(w, http.StatusNotFound, "Article not found in the requested country context")
		return
	}

	crosses, err := h.queries.GetArticleCrossReferences(ctx, artID, languageID, countryID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to fetch cross references: "+err.Error())
		return
	}

	response := models.CrossReferencesResponse{
		Crosses: crosses,
	}

	h.respondJSON(w, http.StatusOK, response)
}

// GetArticleApplicability keeps the legacy detailed PC list and adds a
// normalized, paged linkage view for PC/CV/Motorcycle/Engine/Axle targets.
func (h *Handler) GetArticleApplicability(w http.ResponseWriter, r *http.Request) {
	id, err := getPathInt(r, "id")
	if err != nil || id <= 0 {
		h.respondError(w, http.StatusBadRequest, "Invalid article ID")
		return
	}

	langID := h.getLanguageID(r)
	countryID := h.getCountryID(r)
	targetType := strings.TrimSpace(getStringParam(r, "target_type", ""))
	if targetType != "" && !isSupportedApplicabilityTargetType(targetType) {
		h.respondError(w, http.StatusBadRequest, "Invalid target_type; allowed: PC, CV, Motorcycle, Engine, Axle")
		return
	}
	pageParams, err := parsePageParams(r, defaultArticleApplicabilityLimit, maxArticleApplicabilityLimit)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	limit, offset := pageParams.Limit, pageParams.Offset

	ctx, cancel := context.WithTimeout(r.Context(), searchQueryTimeout)
	defer cancel()

	vehicles := make([]models.Applicability, 0)
	legacyHasMore := false
	// Backward compatibility: old clients received passenger-car details under
	// "vehicles". It is now bounded by the same page parameters as linkages so
	// a single article cannot produce an unbounded legacy payload.
	if targetType == "" || targetType == "PC" {
		vehicles, legacyHasMore, err = h.queries.GetArticleApplicability(ctx, id, langID, countryID, limit, offset)
		if err != nil {
			h.respondError(w, http.StatusInternalServerError, "Failed to fetch legacy passenger-car applicability: "+err.Error())
			return
		}
	}

	page, err := h.queries.GetArticleApplicabilityLinkagesPage(ctx, id, langID, countryID, targetType, limit, offset)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to fetch structured applicability: "+err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"vehicles":            vehicles,
		"total":               len(vehicles),
		"legacy_returned":     len(vehicles),
		"legacy_has_more":     legacyHasMore,
		"linkages":            page.Linkages,
		"total_targets":       page.TotalTargets,
		"returned_targets":    page.Returned,
		"truncated":           page.Truncated,
		"total_targets_exact": page.TotalTargetsExact,
		"limit":               limit,
		"offset":              offset,
		"target_type":         targetType,
	})
}

// GetArticleMedia повертає медіа-файли запчастини (зображення, PDF, відео)
func (h *Handler) GetArticleMedia(w http.ResponseWriter, r *http.Request) {
	id, err := getPathInt(r, "id")
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid article ID")
		return
	}

	langID := h.getLanguageID(r)

	mediaList, err := h.queries.GetArticleMedia(id, langID, h.getCountryID(r))
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to fetch media")
		return
	}

	// Розділяємо на зображення та документи
	var images []models.ArticleMedia
	var documents []models.ArticleMedia
	var videos []models.ArticleMedia

	for _, media := range mediaList {
		switch media.Type {
		case "JPEG", "PNG", "GIF", "BMP", "TIFF":
			images = append(images, media)
		case "PDF":
			documents = append(documents, media)
		case "URL":
			videos = append(videos, media)
		default:
			// Невідомий тип - додаємо до зображень
			images = append(images, media)
		}
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"images":    images,
		"documents": documents,
		"videos":    videos,
		"total":     len(mediaList),
	})
}

// GetArticleComponents повертає список компонентів запчастини
func (h *Handler) GetArticleComponents(w http.ResponseWriter, r *http.Request) {
	id, err := getPathInt(r, "id")
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid article ID")
		return
	}

	langID := h.getLanguageID(r)
	countryID := h.getCountryID(r)

	components, err := h.queries.GetArticleComponents(id, langID, countryID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to fetch components")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"components": components,
		"total":      len(components),
	})
}

// GetArticleAccessories повертає список аксесуарів запчастини
func (h *Handler) GetArticleAccessories(w http.ResponseWriter, r *http.Request) {
	id, err := getPathInt(r, "id")
	if err != nil || id <= 0 {
		h.respondError(w, http.StatusBadRequest, "Invalid article ID")
		return
	}
	page, err := parsePageParams(r, 100, 500)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), searchQueryTimeout)
	defer cancel()
	accessories, total, err := h.queries.GetArticleAccessoriesPage(ctx, id, h.getLanguageID(r), h.getCountryID(r), page.Limit, page.Offset)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to fetch accessories: "+err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"accessories":       accessories,
		"total":             total,
		"limit":             page.Limit,
		"offset":            page.Offset,
		"returned":          len(accessories),
		"has_more":          page.Offset+len(accessories) < total,
		"criteria_included": true,
	})
}

// GetArticleOEMNumbers повертає список OEM номерів запчастини
func (h *Handler) GetArticleOEMNumbers(w http.ResponseWriter, r *http.Request) {
	id, err := getPathInt(r, "id")
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid article ID")
		return
	}

	oemNumbers, err := h.queries.GetArticleOEMNumbers(id)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to fetch OEM numbers")
		return
	}

	h.respondJSON(w, http.StatusOK, models.OEMNumbersResponse{
		OEMNumbers: oemNumbers,
		Total:      len(oemNumbers),
	})
}

// GetArticleCoordinates повертає координати запчастини на зображенні
func (h *Handler) GetArticleCoordinates(w http.ResponseWriter, r *http.Request) {
	articleID, err := getPathInt(r, "id")
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid article ID")
		return
	}

	coordinates, err := h.queries.GetArticleCoordinates(articleID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to fetch coordinates")
		return
	}

	h.respondJSON(w, http.StatusOK, models.ArticleCoordinatesResponse{
		Coordinates: coordinates,
		Total:       len(coordinates),
	})
}

// GetArticleCriteria returns storefront-visible criteria by default. The
// enrichment worker can request the full TecDoc criterion set with
// include_hidden=true while preserving the legacy response fields.
func (h *Handler) GetArticleCriteria(w http.ResponseWriter, r *http.Request) {
	articleID, err := getPathInt(r, "id")
	if err != nil || articleID <= 0 {
		h.respondError(w, http.StatusBadRequest, "Invalid article ID")
		return
	}

	languageID := h.getLanguageID(r)
	countryID := h.getCountryID(r)
	includeHidden, err := parseBoolQuery(r, "include_hidden", false)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), searchQueryTimeout)
	defer cancel()

	criteria, err := h.queries.GetArticleCriteria(ctx, articleID, languageID, countryID, includeHidden)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to fetch article criteria: "+err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, models.ArticleCriteriaResponse{
		Criteria: criteria,
		Total:    len(criteria),
	})
}

// GetArticleClassification returns the raw TecDoc article -> Product Type ->
// Search Tree relation plus linkage evidence. It deliberately does not choose
// one category for the caller.
func (h *Handler) GetArticleClassification(w http.ResponseWriter, r *http.Request) {
	articleID, err := getPathInt(r, "id")
	if err != nil || articleID <= 0 {
		h.respondError(w, http.StatusBadRequest, "Invalid article ID")
		return
	}
	languageID := h.getLanguageID(r)
	countryID := h.getCountryID(r)
	ctx, cancel := context.WithTimeout(r.Context(), searchQueryTimeout)
	defer cancel()

	exists, err := h.queries.ArticleExistsForCountry(ctx, articleID, countryID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to validate article: "+err.Error())
		return
	}
	if !exists {
		h.respondError(w, http.StatusNotFound, "Article not found in the requested country context")
		return
	}

	items, err := h.queries.GetArticleClassification(ctx, articleID, languageID, countryID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to fetch article classification: "+err.Error())
		return
	}
	h.respondJSON(w, http.StatusOK, models.ArticleClassificationResponse{
		ArticleID:    articleID,
		ProductTypes: items,
		Total:        len(items),
	})
}
