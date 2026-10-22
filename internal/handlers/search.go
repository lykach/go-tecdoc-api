package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go-tecdoc-api/internal/models"
)

const (
	defaultSearchLimit  = 100
	maxSearchLimit      = 500
	searchQueryTimeout  = 15 * time.Second
	maxSourceArticleIDs = 200
)

func clampSearchLimit(limit int) int {
	if limit > maxSearchLimit {
		return maxSearchLimit
	}
	if limit < 1 {
		return defaultSearchLimit
	}
	return limit
}

// SearchArticleByNumber performs the broad legacy lookup across TecDoc lookup
// types. Analog resolution must use SearchExactArticleByNumber instead.
func (h *Handler) SearchArticleByNumber(w http.ResponseWriter, r *http.Request) {
	searchNumber := strings.TrimSpace(getStringParam(r, "number", ""))
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
	h.respondJSON(w, http.StatusOK, models.SearchResponse{Results: results})
}

// SearchExactArticleByNumber returns only supplier articles whose own
// ArticleNumber equals the requested number. This endpoint is the mandatory
// first step before a safe ART_ID cross-reference lookup.
func (h *Handler) SearchExactArticleByNumber(w http.ResponseWriter, r *http.Request) {
	searchNumber := strings.TrimSpace(getStringParam(r, "number", ""))
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

	results, err := h.queries.SearchExactArticlesByNumber(ctx, searchNumber, languageID, countryID, limit, offset)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to search exact articles: "+err.Error())
		return
	}
	h.respondJSON(w, http.StatusOK, models.SearchResponse{Results: results})
}

// SearchByOEM searches articles explicitly linked to an OE/OEM number.
func (h *Handler) SearchByOEM(w http.ResponseWriter, r *http.Request) {
	oemNumber := strings.TrimSpace(getStringParam(r, "number", ""))
	if oemNumber == "" {
		h.respondError(w, http.StatusBadRequest, "OEM number is required")
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

	results, err := h.queries.SearchByOEMNumber(ctx, oemNumber, languageID, countryID, limit, offset)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to search by OEM: "+err.Error())
		return
	}
	h.respondJSON(w, http.StatusOK, models.SearchResponse{Results: results})
}

func (h *Handler) SearchByKBA(w http.ResponseWriter, r *http.Request) {
	h.searchTypeNumber(w, r, "German KBA Number")
}

// SearchAnalogs resolves only explicit TecDoc IAM crosses. Safe contracts:
//   - ?art_id=<exact TecDoc article ID>
//   - ?search_number=<number>&supplier_id=<TecDoc supplier ID>
//
// A bare number is rejected because article numbers are not globally unique.
func (h *Handler) SearchAnalogs(w http.ResponseWriter, r *http.Request) {
	artIDRaw := strings.TrimSpace(getStringParam(r, "art_id", ""))
	searchNumber := strings.TrimSpace(getStringParam(r, "search_number", ""))
	supplierIDRaw := strings.TrimSpace(getStringParam(r, "supplier_id", ""))

	if artIDRaw != "" && (searchNumber != "" || supplierIDRaw != "") {
		h.respondError(w, http.StatusBadRequest, "Use either art_id or search_number with supplier_id, not both")
		return
	}
	if artIDRaw == "" && searchNumber == "" && supplierIDRaw == "" {
		h.respondError(w, http.StatusBadRequest, "art_id or search_number with supplier_id is required")
		return
	}
	if artIDRaw == "" && (searchNumber == "" || supplierIDRaw == "") {
		h.respondError(w, http.StatusBadRequest, "Number-only analog lookup is unsafe; search_number requires supplier_id")
		return
	}

	page, err := parsePageParams(r, defaultSearchLimit, maxSearchLimit)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	limit, offset := page.Limit, page.Offset
	if offset < 0 {
		offset = 0
	}
	languageID := h.getLanguageID(r)
	countryID := h.getCountryID(r)

	ctx, cancel := context.WithTimeout(r.Context(), searchQueryTimeout)
	defer cancel()

	sourceIDs := make([]int, 0, 1)
	if artIDRaw != "" {
		artID, err := strconv.Atoi(artIDRaw)
		if err != nil || artID <= 0 {
			h.respondError(w, http.StatusBadRequest, "Invalid art_id")
			return
		}
		exists, err := h.queries.ArticleExistsForCountry(ctx, artID, countryID)
		if err != nil {
			h.respondError(w, http.StatusInternalServerError, "Failed to validate source article: "+err.Error())
			return
		}
		if !exists {
			h.respondError(w, http.StatusNotFound, "Source article was not found in the requested country context")
			return
		}
		sourceIDs = append(sourceIDs, artID)
	} else {
		supplierID, err := strconv.Atoi(supplierIDRaw)
		if err != nil || supplierID <= 0 {
			h.respondError(w, http.StatusBadRequest, "Invalid supplier_id")
			return
		}
		ids, err := h.queries.FindExactArticleIDsBySupplier(ctx, searchNumber, supplierID, countryID, maxSourceArticleIDs+1)
		if err != nil {
			h.respondError(w, http.StatusInternalServerError, "Failed to resolve exact source article: "+err.Error())
			return
		}
		if len(ids) > maxSourceArticleIDs {
			h.respondError(w, http.StatusUnprocessableEntity, "Source article resolves to too many TecDoc technical rows; use an exact art_id")
			return
		}
		// Several ART_ID rows can represent one commercial supplier article.
		// supplier_id + own ArticleNumber already prove the source identity, so
		// merge their explicit IAM links instead of rejecting short/common numbers.
		sourceIDs = ids
	}

	results, err := h.queries.SearchAnalogsByArticleIDs(ctx, sourceIDs, languageID, countryID, limit, offset)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to search analogs: "+err.Error())
		return
	}
	h.respondJSON(w, http.StatusOK, models.SearchResponse{Results: results})
}

func (h *Handler) SearchOEMByOEM(w http.ResponseWriter, r *http.Request) {
	oemNumber := strings.TrimSpace(getStringParam(r, "oem_number", ""))
	if oemNumber == "" {
		h.respondError(w, http.StatusBadRequest, "OEM number is required")
		return
	}

	page, err := parsePageParams(r, defaultSearchLimit, maxSearchLimit)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	limit, offset := page.Limit, page.Offset

	ctx, cancel := context.WithTimeout(r.Context(), searchQueryTimeout)
	defer cancel()

	results, err := h.queries.SearchOEMByOEMNumber(ctx, oemNumber, limit, offset)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to search OEM cross-references: "+err.Error())
		return
	}
	h.respondJSON(w, http.StatusOK, models.OEMCrossReferenceResponse{References: results, Total: len(results)})
}
