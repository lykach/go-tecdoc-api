package handlers

import (
	"context"
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"go-tecdoc-api/internal/models"
)

func (h *Handler) searchTypeNumber(w http.ResponseWriter, r *http.Request, forcedType string) {
	number := strings.TrimSpace(getStringParam(r, "number", ""))
	if number == "" {
		h.respondError(w, http.StatusBadRequest, "number is required")
		return
	}
	numberType := strings.TrimSpace(getStringParam(r, "number_type", forcedType))
	vehicleType := strings.TrimSpace(getStringParam(r, "vehicle_type", ""))
	page, err := parsePageParams(r, 100, maxSearchLimit)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	limit, offset := page.Limit, page.Offset
	ctx, cancel := context.WithTimeout(r.Context(), searchQueryTimeout)
	defer cancel()
	items, err := h.queries.SearchTypeNumbers(ctx, number, numberType, vehicleType, limit, offset)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Failed to search type numbers: "+err.Error())
		return
	}
	h.respondJSON(w, http.StatusOK, models.TypeNumberMatchesResponse{Matches: items, Total: len(items)})
}

func (h *Handler) SearchByKenteken(w http.ResponseWriter, r *http.Request) {
	h.searchTypeNumber(w, r, "Netherlands Kenteken")
}

func (h *Handler) SearchByTypeNumber(w http.ResponseWriter, r *http.Request) {
	h.searchTypeNumber(w, r, "")
}

func (h *Handler) getVehicleTypeNumbers(w http.ResponseWriter, r *http.Request, vehicleType string) {
	vehicleID, err := getPathInt(r, "id")
	if err != nil || vehicleID <= 0 {
		h.respondError(w, http.StatusBadRequest, "Invalid vehicle ID")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), searchQueryTimeout)
	defer cancel()
	items, err := h.queries.GetVehicleTypeNumbers(ctx, vehicleID, vehicleType)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to fetch vehicle type numbers: "+err.Error())
		return
	}
	h.respondJSON(w, http.StatusOK, models.TypeNumberMatchesResponse{Matches: items, Total: len(items)})
}

func (h *Handler) GetCarTypeNumbers(w http.ResponseWriter, r *http.Request) {
	h.getVehicleTypeNumbers(w, r, "PC")
}
func (h *Handler) GetCommercialVehicleTypeNumbers(w http.ResponseWriter, r *http.Request) {
	h.getVehicleTypeNumbers(w, r, "CV")
}
func (h *Handler) GetMotorcycleTypeNumbers(w http.ResponseWriter, r *http.Request) {
	h.getVehicleTypeNumbers(w, r, "Motorcycle")
}

func (h *Handler) getVehicleCriteria(w http.ResponseWriter, r *http.Request, vehicleType string) {
	vehicleID, err := getPathInt(r, "id")
	if err != nil || vehicleID <= 0 {
		h.respondError(w, http.StatusBadRequest, "Invalid vehicle ID")
		return
	}
	languageID := h.getLanguageID(r)
	productTypeID, err := parseOptionalNonNegativeQueryInt(r, "product_type_id")
	if err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	supplierID, err := parseOptionalNonNegativeQueryInt(r, "supplier_id")
	if err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	suggestedOnly, err := parseBoolQuery(r, "suggested_only", false)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), searchQueryTimeout)
	defer cancel()
	criteria, err := h.queries.GetVehicleCriteria(ctx, vehicleID, vehicleType, languageID, productTypeID, supplierID, suggestedOnly)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to fetch vehicle criteria: "+err.Error())
		return
	}
	h.respondJSON(w, http.StatusOK, models.VehicleCriteriaResponse{VehicleID: vehicleID, VehicleType: vehicleType, Criteria: criteria, Total: len(criteria)})
}

func (h *Handler) GetCarCriteria(w http.ResponseWriter, r *http.Request) {
	h.getVehicleCriteria(w, r, "PC")
}
func (h *Handler) GetCommercialVehicleCriteria(w http.ResponseWriter, r *http.Request) {
	h.getVehicleCriteria(w, r, "CV")
}
func (h *Handler) GetMotorcycleCriteria(w http.ResponseWriter, r *http.Request) {
	h.getVehicleCriteria(w, r, "Motorcycle")
}

func (h *Handler) GetProductTypeCriteria(w http.ResponseWriter, r *http.Request) {
	productTypeID, err := getPathInt(r, "id")
	if err != nil || productTypeID <= 0 {
		h.respondError(w, http.StatusBadRequest, "Invalid product type ID")
		return
	}
	suggestedOnly, err := parseBoolQuery(r, "suggested_only", false)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), searchQueryTimeout)
	defer cancel()
	items, err := h.queries.GetProductTypeCriteria(ctx, productTypeID, h.getLanguageID(r), suggestedOnly)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to fetch product type criteria: "+err.Error())
		return
	}
	h.respondJSON(w, http.StatusOK, models.ProductTypeCriteriaResponse{Criteria: items, Total: len(items)})
}

func (h *Handler) SearchProductGroups(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(getStringParam(r, "q", ""))
	if query == "" {
		h.respondError(w, http.StatusBadRequest, "q is required")
		return
	}
	vehicleType := getStringParam(r, "vehicle_type", "PC")
	page, err := parsePageParams(r, 100, maxSearchLimit)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	limit, offset := page.Limit, page.Offset
	ctx, cancel := context.WithTimeout(r.Context(), searchQueryTimeout)
	defer cancel()
	items, err := h.queries.SearchProductGroups(ctx, query, vehicleType, h.getLanguageID(r), limit, offset)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Failed to search product groups: "+err.Error())
		return
	}
	h.respondJSON(w, http.StatusOK, models.ProductGroupSearchResponse{Results: items, Total: len(items)})
}

func (h *Handler) GetQuickStart(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), searchQueryTimeout)
	defer cancel()
	items, err := h.queries.GetQuickStart(ctx, getStringParam(r, "vehicle_type", ""), h.getLanguageID(r))
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Failed to fetch quick-start data: "+err.Error())
		return
	}
	h.respondJSON(w, http.StatusOK, models.QuickStartResponse{Items: items, Total: len(items)})
}

func (h *Handler) GetArticlePrices(w http.ResponseWriter, r *http.Request) {
	articleID, err := getPathInt(r, "id")
	if err != nil || articleID <= 0 {
		h.respondError(w, http.StatusBadRequest, "Invalid article ID")
		return
	}
	includeExpired, err := parseBoolQuery(r, "include_expired", false)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), searchQueryTimeout)
	defer cancel()
	items, err := h.queries.GetArticlePrices(ctx, articleID, h.getCountryID(r), h.getLanguageID(r), includeExpired)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to fetch article prices: "+err.Error())
		return
	}
	h.respondJSON(w, http.StatusOK, models.ArticlePricesResponse{Prices: items, Total: len(items)})
}

func (h *Handler) GetArticleInfo(w http.ResponseWriter, r *http.Request) {
	articleID, err := getPathInt(r, "id")
	if err != nil || articleID <= 0 {
		h.respondError(w, http.StatusBadRequest, "Invalid article ID")
		return
	}
	includeHidden, err := parseBoolQuery(r, "include_hidden", false)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), searchQueryTimeout)
	defer cancel()
	items, err := h.queries.GetArticleInfo(ctx, articleID, h.getLanguageID(r), h.getCountryID(r), includeHidden)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to fetch article info: "+err.Error())
		return
	}
	h.respondJSON(w, http.StatusOK, models.ArticleInfoResponse{Items: items, Total: len(items)})
}

func (h *Handler) GetSupplierAddresses(w http.ResponseWriter, r *http.Request) {
	supplierID, err := getPathInt(r, "id")
	if err != nil || supplierID <= 0 {
		h.respondError(w, http.StatusBadRequest, "Invalid supplier ID")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), searchQueryTimeout)
	defer cancel()
	items, err := h.queries.GetSupplierAddresses(ctx, supplierID, h.getLanguageID(r))
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to fetch supplier addresses: "+err.Error())
		return
	}
	h.respondJSON(w, http.StatusOK, models.SupplierAddressesResponse{Addresses: items, Total: len(items)})
}

func (h *Handler) GetSupplierLogos(w http.ResponseWriter, r *http.Request) {
	supplierID, err := getPathInt(r, "id")
	if err != nil || supplierID <= 0 {
		h.respondError(w, http.StatusBadRequest, "Invalid supplier ID")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), searchQueryTimeout)
	defer cancel()
	items, err := h.queries.GetSupplierLogos(ctx, supplierID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to fetch supplier logos: "+err.Error())
		return
	}
	h.respondJSON(w, http.StatusOK, models.SupplierLogosResponse{Logos: items, Total: len(items)})
}

func (h *Handler) GetCommercialVehicleChassis(w http.ResponseWriter, r *http.Request) {
	vehicleID, err := getPathInt(r, "id")
	if err != nil || vehicleID <= 0 {
		h.respondError(w, http.StatusBadRequest, "Invalid commercial vehicle ID")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), searchQueryTimeout)
	defer cancel()
	item, err := h.queries.GetCommercialVehicleChassis(ctx, vehicleID, h.getLanguageID(r))
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to fetch commercial vehicle chassis: "+err.Error())
		return
	}
	h.respondJSON(w, http.StatusOK, item)
}

func (h *Handler) GetDatasets(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), searchQueryTimeout)
	defer cancel()
	items, err := h.queries.ListDatasets(ctx)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to list datasets: "+err.Error())
		return
	}
	h.respondJSON(w, http.StatusOK, models.DatasetsResponse{Datasets: items, Total: len(items)})
}

func (h *Handler) GetDatasetSchema(w http.ResponseWriter, r *http.Request) {
	table := strings.TrimSpace(muxVar(r, "table"))
	ctx, cancel := context.WithTimeout(r.Context(), searchQueryTimeout)
	defer cancel()
	columns, err := h.queries.GetDatasetSchema(ctx, table)
	if err != nil {
		if err == sql.ErrNoRows {
			h.respondError(w, http.StatusNotFound, "Dataset not found or not exposed")
			return
		}
		h.respondError(w, http.StatusInternalServerError, "Failed to fetch dataset schema: "+err.Error())
		return
	}
	h.respondJSON(w, http.StatusOK, models.DatasetSchemaResponse{Table: table, Columns: columns})
}

func muxVar(r *http.Request, key string) string {
	return mux.Vars(r)[key]
}

func parseDatasetFilters(r *http.Request) (map[string][]string, error) {
	filters := make(map[string][]string)
	for _, item := range r.URL.Query()["filter"] {
		parts := strings.SplitN(item, ":", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" {
			return nil, strconv.ErrSyntax
		}
		name := strings.TrimSpace(parts[0])
		values := strings.Split(parts[1], ",")
		for _, value := range values {
			filters[name] = append(filters[name], strings.TrimSpace(value))
		}
	}
	return filters, nil
}

func (h *Handler) GetDatasetRows(w http.ResponseWriter, r *http.Request) {
	table := strings.TrimSpace(muxVar(r, "table"))
	page, err := parsePageParams(r, 100, 5000)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	limit, offset := page.Limit, page.Offset
	var selectColumns []string
	if raw := strings.TrimSpace(getStringParam(r, "select", "")); raw != "" {
		for _, name := range strings.Split(raw, ",") {
			if strings.TrimSpace(name) != "" {
				selectColumns = append(selectColumns, strings.TrimSpace(name))
			}
		}
	}
	filters, err := parseDatasetFilters(r)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid filter; expected filter=COLUMN:value1,value2")
		return
	}
	format, err := parseResponseFormat(r, "json", "csv")
	if err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	includeTotal, err := parseBoolQuery(r, "include_total", false)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), searchQueryTimeout)
	defer cancel()
	rows, total, err := h.queries.GetDatasetRows(ctx, table, selectColumns, filters, strings.TrimSpace(getStringParam(r, "order_by", "")), getStringParam(r, "order", "ASC"), limit, offset, includeTotal)
	if err != nil {
		if err == sql.ErrNoRows {
			h.respondError(w, http.StatusNotFound, "Dataset not found or not exposed")
			return
		}
		h.respondError(w, http.StatusBadRequest, "Failed to fetch dataset: "+err.Error())
		return
	}
	if format == "csv" {
		writeDatasetCSV(w, table, rows)
		return
	}
	h.respondJSON(w, http.StatusOK, models.DatasetRowsResponse{Table: table, Rows: rows, Limit: limit, Offset: offset, Total: total})
}

func writeDatasetCSV(w http.ResponseWriter, table string, rows []map[string]any) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+table+`.csv"`)
	writer := csv.NewWriter(w)
	defer writer.Flush()
	if len(rows) == 0 {
		return
	}
	columns := make([]string, 0, len(rows[0]))
	for name := range rows[0] {
		columns = append(columns, name)
	}
	sort.Strings(columns)
	_ = writer.Write(columns)
	for _, row := range rows {
		record := make([]string, len(columns))
		for i, name := range columns {
			if row[name] != nil {
				record[i] = strings.TrimSpace(toString(row[name]))
			}
		}
		_ = writer.Write(record)
	}
}

func toString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		return strings.ReplaceAll(fmt.Sprint(v), "\x00", "")
	}
}

func (h *Handler) GetAPIIndex(w http.ResponseWriter, r *http.Request) {
	h.respondJSON(w, http.StatusOK, map[string]any{
		"name":     "Go TecDoc API",
		"version":  "v1",
		"database": "TecDoc 2024Q1",
		"auth":     "X-API-Key",
		"docs":     []string{"README.md", "API_REFERENCE.md", "OPENAPI.yaml"},
		"capabilities": []string{
			"localization", "suppliers", "manufacturers", "model series", "passenger cars",
			"commercial vehicles", "motorcycles", "engines", "product groups", "product criteria",
			"articles", "applicability", "cross references", "OEM/EAN", "media", "prices",
			"type-number search", "KBA", "Kenteken", "catalog enrichment", "read-only datasets",
		},
	})
}

func (h *Handler) GetArticleSpecifications(w http.ResponseWriter, r *http.Request) {
	articleID, err := getPathInt(r, "id")
	if err != nil || articleID <= 0 {
		h.respondError(w, http.StatusBadRequest, "Invalid article ID")
		return
	}
	page, err := parsePageParams(r, 100, 500)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	includeHidden, err := parseBoolQuery(r, "include_hidden", true)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), searchQueryTimeout)
	defer cancel()
	items, total, err := h.queries.GetArticleSpecifications(ctx, articleID, h.getLanguageID(r), h.getCountryID(r), includeHidden, page.Limit, page.Offset)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to fetch article specifications: "+err.Error())
		return
	}
	h.respondJSON(w, http.StatusOK, models.ArticleSpecificationsResponse{ArticleID: articleID, Specifications: items, Total: total, Limit: page.Limit, Offset: page.Offset, Returned: len(items), HasMore: page.Offset+len(items) < total})
}

func (h *Handler) GetArticleLinkageInfo(w http.ResponseWriter, r *http.Request) {
	articleID, err := getPathInt(r, "id")
	if err != nil || articleID <= 0 {
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
	items, total, err := h.queries.GetArticleLinkageInfo(ctx, articleID, h.getLanguageID(r), h.getCountryID(r), page.Limit, page.Offset)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to fetch article linkage info: "+err.Error())
		return
	}
	h.respondJSON(w, http.StatusOK, models.ArticleLinkageInfoResponse{ArticleID: articleID, Items: items, Total: total, Limit: page.Limit, Offset: page.Offset, Returned: len(items), HasMore: page.Offset+len(items) < total})
}

func (h *Handler) GetPartsDiagram(w http.ResponseWriter, r *http.Request) {
	articleID, err := getPathInt(r, "id")
	if err != nil || articleID <= 0 {
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
	items, total, err := h.queries.GetPartsDiagramCoordinates(ctx, articleID, page.Limit, page.Offset)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to fetch parts diagram: "+err.Error())
		return
	}
	h.respondJSON(w, http.StatusOK, models.PartsDiagramResponse{ArticleID: articleID, Components: items, Total: total, Limit: page.Limit, Offset: page.Offset, Returned: len(items), HasMore: page.Offset+len(items) < total})
}

const exportQueryTimeout = 25 * time.Second

func respondExportError(h *Handler, w http.ResponseWriter, prefix string, err error) {
	if errors.Is(err, context.DeadlineExceeded) {
		h.respondError(w, http.StatusGatewayTimeout, prefix+": query timeout")
		return
	}
	h.respondError(w, http.StatusInternalServerError, prefix+": "+err.Error())
}

func (h *Handler) ExportCrossReferences(w http.ResponseWriter, r *http.Request) {
	page, err := parsePageParams(r, 1000, 5000)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	format, err := parseResponseFormat(r, "json", "csv")
	if err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	limit, offset := page.Limit, page.Offset
	ctx, cancel := context.WithTimeout(r.Context(), exportQueryTimeout)
	defer cancel()
	items, err := h.queries.ExportCrossReferences(ctx, limit, offset)
	if err != nil {
		respondExportError(h, w, "Failed to export cross references", err)
		return
	}
	if format == "csv" {
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="cross-references.csv"`)
		writer := csv.NewWriter(w)
		defer writer.Flush()
		_ = writer.Write([]string{"ART_NUMBER", "ART_BRAND", "CROSS_NUMBER", "CROSS_BRAND", "CROSS_TYPE"})
		for _, item := range items {
			_ = writer.Write([]string{item.ArticleNumber, item.ArticleBrand, item.CrossNumber, item.CrossBrand, item.CrossType})
		}
		return
	}
	h.respondJSON(w, http.StatusOK, models.CrossReferenceExportResponse{Rows: items, Limit: limit, Offset: offset})
}

func (h *Handler) ExportEAN(w http.ResponseWriter, r *http.Request) {
	page, err := parsePageParams(r, 1000, 5000)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	format, err := parseResponseFormat(r, "json", "csv")
	if err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	limit, offset := page.Limit, page.Offset
	ctx, cancel := context.WithTimeout(r.Context(), exportQueryTimeout)
	defer cancel()
	items, err := h.queries.ExportEAN(ctx, limit, offset)
	if err != nil {
		respondExportError(h, w, "Failed to export EAN", err)
		return
	}
	if format == "csv" {
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="ean.csv"`)
		writer := csv.NewWriter(w)
		defer writer.Flush()
		_ = writer.Write([]string{"ART_NUMBER", "ART_BRAND", "EAN_NUMBER"})
		for _, item := range items {
			_ = writer.Write([]string{item.ArticleNumber, item.ArticleBrand, item.EAN})
		}
		return
	}
	h.respondJSON(w, http.StatusOK, models.EANExportResponse{Rows: items, Limit: limit, Offset: offset})
}

func (h *Handler) GetCriterionKeyValues(w http.ResponseWriter, r *http.Request) {
	criteriaID, err := getPathInt(r, "id")
	if err != nil || criteriaID <= 0 {
		h.respondError(w, http.StatusBadRequest, "Invalid criteria ID")
		return
	}
	page, err := parsePageParams(r, 100, 1000)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), searchQueryTimeout)
	defer cancel()
	items, total, err := h.queries.GetCriterionKeyValues(ctx, criteriaID, h.getLanguageID(r), page.Limit, page.Offset)
	if err != nil {
		if err == sql.ErrNoRows {
			h.respondError(w, http.StatusNotFound, "Criteria not found")
			return
		}
		h.respondError(w, http.StatusInternalServerError, "Failed to fetch criterion key values: "+err.Error())
		return
	}
	h.respondJSON(w, http.StatusOK, models.CriterionKeyValuesResponse{CriteriaID: criteriaID, Values: items, Total: total, Limit: page.Limit, Offset: page.Offset, Returned: len(items), HasMore: page.Offset+len(items) < total})
}

func (h *Handler) GetAxleDetail(w http.ResponseWriter, r *http.Request) {
	axleID, err := getPathInt(r, "id")
	if err != nil || axleID <= 0 {
		h.respondError(w, http.StatusBadRequest, "Invalid axle ID")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), searchQueryTimeout)
	defer cancel()
	item, err := h.queries.GetAxleDetail(ctx, axleID, h.getLanguageID(r))
	if err != nil {
		if err == sql.ErrNoRows {
			h.respondError(w, http.StatusNotFound, "Axle not found")
			return
		}
		h.respondError(w, http.StatusInternalServerError, "Failed to fetch axle: "+err.Error())
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]any{"axle": item})
}
