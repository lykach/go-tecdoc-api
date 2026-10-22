package handlers

import (
	"database/sql"
	"net/http"

	"go-tecdoc-api/internal/models"
)

// GetSuppliers повертає список постачальників
func (h *Handler) GetSuppliers(w http.ResponseWriter, r *http.Request) {
	pageNumber, err := parsePositiveQueryInt(r, "page", 1)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	pageParams, err := parsePageParams(r, 50, 100)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	limit := pageParams.Limit

	// Якщо є параметр brand - використовуємо пошук
	if brand := r.URL.Query().Get("brand"); brand != "" {
		suppliers, err := h.queries.SearchSuppliersByBrand(brand, limit)
		if err != nil {
			h.respondError(w, http.StatusInternalServerError, "Failed to search suppliers")
			return
		}

		h.respondJSON(w, http.StatusOK, map[string]interface{}{
			"suppliers": suppliers,
		})
		return
	}

	// Інакше - звичайний список з пагінацією
	suppliers, total, err := h.queries.GetSuppliers(pageNumber, limit)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to fetch suppliers")
		return
	}

	totalPages := (total + limit - 1) / limit

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"suppliers": suppliers,
		"pagination": map[string]interface{}{
			"page":        pageNumber,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

// GetSupplierByID повертає деталі постачальника за ID
func (h *Handler) GetSupplierByID(w http.ResponseWriter, r *http.Request) {
	id, err := getPathInt(r, "id")
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid supplier ID")
		return
	}

	supplier, err := h.queries.GetSupplierByID(id)
	if err != nil {
		if err == sql.ErrNoRows {
			h.respondError(w, http.StatusNotFound, "Supplier not found")
			return
		}
		h.respondError(w, http.StatusInternalServerError, "Failed to fetch supplier")
		return
	}

	h.respondJSON(w, http.StatusOK, supplier)
}

// GetSupplierProducts повертає товарні групи для постачальника
func (h *Handler) GetSupplierProducts(w http.ResponseWriter, r *http.Request) {
	supplierID, err := getPathInt(r, "id")
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid supplier ID")
		return
	}

	pageParams, err := parsePageParams(r, 100, 500)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	limit, offset := pageParams.Limit, pageParams.Offset
	languageID := h.getLanguageID(r)

	products, err := h.queries.GetSupplierProducts(supplierID, languageID, limit, offset)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to fetch supplier products")
		return
	}

	total, err := h.queries.CountSupplierProducts(supplierID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to count supplier products")
		return
	}

	h.respondJSON(w, http.StatusOK, models.SupplierProductsResponse{
		Products: products,
		Total:    total,
		Limit:    limit,
		Offset:   offset,
	})
}
