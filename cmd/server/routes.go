package main

import (
	"github.com/gorilla/mux"

	"go-tecdoc-api/internal/handlers"
)

// registerAPIRoutes is the single production route registry. Keeping route
// registration outside main makes the OpenAPI contract mechanically testable.
func registerAPIRoutes(api *mux.Router, h *handlers.Handler) {
	// Localization routes
	api.HandleFunc("/languages", h.GetLanguages).Methods("GET")
	api.HandleFunc("/languages/{id}", h.GetLanguageByID).Methods("GET")
	api.HandleFunc("/countries", h.GetCountries).Methods("GET")
	api.HandleFunc("/countries/{id}", h.GetCountryByID).Methods("GET")

	// API index / discovery
	api.HandleFunc("", h.GetAPIIndex).Methods("GET")
	api.HandleFunc("/", h.GetAPIIndex).Methods("GET")

	// Suppliers routes
	api.HandleFunc("/suppliers", h.GetSuppliers).Methods("GET")
	api.HandleFunc("/suppliers/{id}", h.GetSupplierByID).Methods("GET")
	api.HandleFunc("/suppliers/{id}/products", h.GetSupplierProducts).Methods("GET")
	api.HandleFunc("/suppliers/{id}/addresses", h.GetSupplierAddresses).Methods("GET")
	api.HandleFunc("/suppliers/{id}/logos", h.GetSupplierLogos).Methods("GET")

	// Bulk-import summary. One aggregate request replaces the old client-side
	// fan-out preview that enumerated every manufacturer/model/vehicle page.
	api.HandleFunc("/import/vehicle-stats", h.GetVehicleImportStats).Methods("GET")
	api.HandleFunc("/import/product-groups", h.GetImportProductGroups).Methods("GET")
	api.HandleFunc("/import/vehicle-product-groups", h.GetImportVehicleProductGroups).Methods("GET")

	// Manufacturers routes
	api.HandleFunc("/manufacturers", h.GetManufacturers).Methods("GET")
	api.HandleFunc("/manufacturers/{id}", h.GetManufacturerByID).Methods("GET")

	// Model series routes
	api.HandleFunc("/manufacturers/{id}/models", h.GetModelSeries).Methods("GET")
	api.HandleFunc("/models/{id}", h.GetModelSeriesDetails).Methods("GET")

	// Commercial Vehicles
	api.HandleFunc("/models/{id}/cv", h.GetCommercialVehicles).Methods("GET")
	api.HandleFunc("/cv/{id}", h.GetCommercialVehicleDetails).Methods("GET")
	api.HandleFunc("/cv/{id}/type-numbers", h.GetCommercialVehicleTypeNumbers).Methods("GET")
	api.HandleFunc("/cv/{id}/criteria", h.GetCommercialVehicleCriteria).Methods("GET")
	api.HandleFunc("/cv/{id}/chassis", h.GetCommercialVehicleChassis).Methods("GET")

	// Motorcycles
	api.HandleFunc("/models/{id}/mc", h.GetMotorcycles).Methods("GET")
	api.HandleFunc("/mc/{id}", h.GetMotorcycleDetails).Methods("GET")
	api.HandleFunc("/mc/{id}/type-numbers", h.GetMotorcycleTypeNumbers).Methods("GET")
	api.HandleFunc("/mc/{id}/criteria", h.GetMotorcycleCriteria).Methods("GET")

	// Passenger cars routes
	api.HandleFunc("/models/{id}/cars", h.GetPassengerCars).Methods("GET")
	api.HandleFunc("/cars/{id}", h.GetCarDetails).Methods("GET")
	api.HandleFunc("/cars/{id}/product-groups", h.GetCarProductGroups).Methods("GET")
	api.HandleFunc("/cars/{id}/type-numbers", h.GetCarTypeNumbers).Methods("GET")
	api.HandleFunc("/cars/{id}/criteria", h.GetCarCriteria).Methods("GET")

	// Engine and axle details
	api.HandleFunc("/engines", h.GetEngines).Methods("GET")
	api.HandleFunc("/engines/{id}", h.GetEngineDetails).Methods("GET")
	api.HandleFunc("/axles/{id}", h.GetAxleDetail).Methods("GET")

	// Articles routes
	api.HandleFunc("/articles/search", h.SearchArticles).Methods("GET")
	api.HandleFunc("/articles/{id}", h.GetArticleDetails).Methods("GET")
	api.HandleFunc("/articles/{id}/applicability", h.GetArticleApplicability).Methods("GET")
	api.HandleFunc("/articles/{id}/cross-references", h.GetArticleCrossReferences).Methods("GET")
	api.HandleFunc("/articles/{id}/media", h.GetArticleMedia).Methods("GET")
	api.HandleFunc("/articles/{id}/components", h.GetArticleComponents).Methods("GET")
	api.HandleFunc("/articles/{id}/accessories", h.GetArticleAccessories).Methods("GET")
	api.HandleFunc("/articles/{id}/oem", h.GetArticleOEMNumbers).Methods("GET")
	api.HandleFunc("/articles/{id}/coordinates", h.GetArticleCoordinates).Methods("GET")
	api.HandleFunc("/articles/{id}/criteria", h.GetArticleCriteria).Methods("GET")
	api.HandleFunc("/articles/{id}/classification", h.GetArticleClassification).Methods("GET")
	api.HandleFunc("/articles/{id}/info", h.GetArticleInfo).Methods("GET")
	api.HandleFunc("/articles/{id}/prices", h.GetArticlePrices).Methods("GET")
	api.HandleFunc("/articles/{id}/specifications", h.GetArticleSpecifications).Methods("GET")
	api.HandleFunc("/articles/{id}/linkage-info", h.GetArticleLinkageInfo).Methods("GET")
	api.HandleFunc("/articles/{id}/parts-diagram", h.GetPartsDiagram).Methods("GET")

	// Catalog enrichment (server-to-server batch).
	api.HandleFunc("/enrichment/articles/batch", h.EnrichArticlesBatch).Methods("POST")

	// Product groups and criteria routes
	api.HandleFunc("/quick-start", h.GetQuickStart).Methods("GET")
	api.HandleFunc("/product-types/{id}/criteria", h.GetProductTypeCriteria).Methods("GET")
	api.HandleFunc("/criteria/{id}/values", h.GetCriterionKeyValues).Methods("GET")
	api.HandleFunc("/product-groups/search", h.SearchProductGroups).Methods("GET")
	api.HandleFunc("/product-groups", h.GetProductGroups).Methods("GET")
	api.HandleFunc("/product-groups/{id}/children", h.GetCategoryChildren).Methods("GET")
	api.HandleFunc("/product-groups/{id}/articles", h.GetProductGroupArticles).Methods("GET")

	// Search routes
	api.HandleFunc("/search/kba", h.SearchByKBA).Methods("GET")
	api.HandleFunc("/search/kenteken", h.SearchByKenteken).Methods("GET")
	api.HandleFunc("/search/type-number", h.SearchByTypeNumber).Methods("GET")
	api.HandleFunc("/search/article", h.SearchArticleByNumber).Methods("GET")
	api.HandleFunc("/search/article-exact", h.SearchExactArticleByNumber).Methods("GET")
	// NOTE: this used to also register h.SearchArticles on the same path+method
	// as the line above. gorilla/mux matches exact routes in registration
	// order, so the second registration was silently unreachable dead code -
	// removed. h.SearchArticles is still live on its own path, /articles/search.
	api.HandleFunc("/search/oem", h.SearchByOEM).Methods("GET")
	api.HandleFunc("/search/analog", h.SearchAnalogs).Methods("GET")
	api.HandleFunc("/search/oem-oem", h.SearchOEMByOEM).Methods("GET")

	// Complete, read-only TecDoc source-table access. Internal auth and phpMyAdmin
	// tables are intentionally excluded. Dynamic identifiers are validated against
	// INFORMATION_SCHEMA before they are interpolated into SQL.
	api.HandleFunc("/datasets", h.GetDatasets).Methods("GET")
	api.HandleFunc("/datasets/{table}/schema", h.GetDatasetSchema).Methods("GET")
	api.HandleFunc("/datasets/{table}", h.GetDatasetRows).Methods("GET")

	// Streaming-friendly paged exports corresponding to the canonical TecDoc
	// 2024Q1 CROSS and EAN SQL examples. JSON is the default; use format=csv.
	api.HandleFunc("/exports/cross-references", h.ExportCrossReferences).Methods("GET")
	api.HandleFunc("/exports/ean", h.ExportEAN).Methods("GET")
}
