package handlers

import (
	"net/http"
	"go-tecdoc-api/internal/models"
)

// SearchArticles - пошук запчастин за номером
// GET /api/v1/articles/search?number=0001108466&limit=20&offset=0&language_id=48&country_id=62
func (h *Handler) SearchArticles(w http.ResponseWriter, r *http.Request) {
	searchNumber := getStringParam(r, "number", "")
	if searchNumber == "" {
		h.respondError(w, http.StatusBadRequest, "Search number is required")
		return
	}

	limit := getIntParam(r, "limit", 50)
	offset := getIntParam(r, "offset", 0)
	languageID := h.getLanguageID(r)
	countryID := h.getCountryID(r)

	// Валідація
	if limit > 500 {
		limit = 500
	}
	if limit < 1 {
		limit = 50
	}

	results, err := h.queries.SearchArticlesByNumber(searchNumber, languageID, countryID, limit, offset)
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

	articleDetail, err := h.queries.GetArticleByID(artID, languageID, countryID)
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

	crosses, err := h.queries.GetArticleCrossReferences(artID, languageID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to fetch cross references: "+err.Error())
		return
	}

	response := models.CrossReferencesResponse{
		Crosses: crosses,
	}

	h.respondJSON(w, http.StatusOK, response)
}
// GetArticleApplicability повертає список автомобілів для запчастини
func (h *Handler) GetArticleApplicability(w http.ResponseWriter, r *http.Request) {
id, err := getPathInt(r, "id")
if err != nil {
h.respondError(w, http.StatusBadRequest, "Invalid article ID")
return
}

langID := h.getLanguageID(r)
countryID := h.getCountryID(r)

applicabilities, err := h.queries.GetArticleApplicability(id, langID, countryID)
if err != nil {
h.respondError(w, http.StatusInternalServerError, "Failed to fetch applicability")
return
}

h.respondJSON(w, http.StatusOK, map[string]interface{}{
"vehicles": applicabilities,
"total":    len(applicabilities),
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

mediaList, err := h.queries.GetArticleMedia(id, langID)
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
if err != nil {
h.respondError(w, http.StatusBadRequest, "Invalid article ID")
return
}

langID := h.getLanguageID(r)
countryID := h.getCountryID(r)

accessories, err := h.queries.GetArticleAccessories(id, langID, countryID)
if err != nil {
h.respondError(w, http.StatusInternalServerError, "Failed to fetch accessories")
return
}

h.respondJSON(w, http.StatusOK, map[string]interface{}{
"accessories": accessories,
"total":       len(accessories),
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

// GetArticleCriteria повертає критерії (характеристики) запчастини
func (h *Handler) GetArticleCriteria(w http.ResponseWriter, r *http.Request) {
articleID, err := getPathInt(r, "id")
if err != nil {
h.respondError(w, http.StatusBadRequest, "Invalid article ID")
return
}

languageID := h.getLanguageID(r)

criteria, err := h.queries.GetArticleCriteria(articleID, languageID)
if err != nil {
h.respondError(w, http.StatusInternalServerError, "Failed to fetch article criteria")
return
}

h.respondJSON(w, http.StatusOK, models.ArticleCriteriaResponse{
Criteria: criteria,
Total:    len(criteria),
})
}
