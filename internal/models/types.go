package models

// ============== Manufacturers ==============

type VehicleType struct {
	ID          int    `json:"id"`
	Type        string `json:"type"` // PC, CV, MC
	Description string `json:"description"`
}

type Manufacturer struct {
	ID          int    `json:"id" db:"MFA_ID"`
	Brand       string `json:"brand" db:"MFA_BRAND"`
	Type        string `json:"type" db:"MFA_TYPE"`
	ModelsCount int    `json:"models_count" db:"MFA_MODELS_COUNT"`
	SupplierID  *int   `json:"supplier_id,omitempty" db:"MFA_SUP_ID"`
}

type ManufacturerDetail struct {
	ID          int    `json:"id"`
	Brand       string `json:"brand"`
	Type        string `json:"type"`
	ModelsCount int    `json:"models_count"`
	SupplierID  *int   `json:"supplier_id,omitempty"`
}

// ============== Models ==============

type ModelSeries struct {
	ID       int    `json:"id" db:"MS_ID"`
	MfaID    int    `json:"mfa_id" db:"MS_MFA_ID"`
	Name     string `json:"name"`
	YearFrom int    `json:"year_from" db:"MS_CI_FROM"`
	YearTo   int    `json:"year_to" db:"MS_CI_TO"`
	Type     string `json:"type" db:"MS_TYPE"`
}

type ModelDetail struct {
	ID           int    `json:"id"`
	MfaID        int    `json:"mfa_id"`
	Manufacturer string `json:"manufacturer"`
	Name         string `json:"name"`
	YearFrom     int    `json:"year_from"`
	YearTo       int    `json:"year_to"`
	Type         string `json:"type"`
}

// ============== Vehicles ==============

type PassengerCar struct {
	ID             int      `json:"id" db:"PC_ID"`
	MfaID          int      `json:"mfa_id" db:"PC_MFA_ID"`
	MsID           int      `json:"ms_id" db:"PC_MS_ID"`
	VehicleType    string   `json:"vehicle_type" db:"PC_TYPE"`
	TypeName       string   `json:"type_name"`
	YearFrom       int      `json:"year_from"`
	YearTo         int      `json:"year_to"`
	PowerKW        int      `json:"power_kw"`
	PowerHP        int      `json:"power_hp"`
	Capacity       int      `json:"capacity"`
	CapacityLiters *float64 `json:"capacity_liters,omitempty"`
	Cylinders      *int     `json:"cylinders,omitempty"`
	Valves         *int     `json:"valves,omitempty"`
	FuelType       string   `json:"fuel_type"`
	BodyType       string   `json:"body_type"`
	EngineType     string   `json:"engine_type"`
	DriveType      string   `json:"drive_type"`
	EngineCodes    []string `json:"engine_codes"`
	Engines        []Engine `json:"engines"`
}

type VehicleDetail struct {
	ID           int                    `json:"id"`
	VehicleType  string                 `json:"vehicle_type,omitempty"`
	TypeName     string                 `json:"type_name"`
	Manufacturer string                 `json:"manufacturer"`
	ModelSeries  string                 `json:"model_series"`
	YearFrom     int                    `json:"year_from"`
	YearTo       int                    `json:"year_to"`
	Specs        map[string]interface{} `json:"specs"`
	EngineCodes  []string               `json:"engine_codes"`
}

type Engine struct {
	ID   int    `json:"id" db:"ENG_ID"`
	Code string `json:"code" db:"ENG_CODE"`
	Name string `json:"name" db:"ENG_NAME"`
}

// ============== Categories ==============

type Category struct {
	ID       int        `json:"id" db:"STR_ID"`
	ParentID *int       `json:"parent_id" db:"STR_ID_PARENT"`
	Level    int        `json:"level" db:"STR_LEVEL"`
	Name     string     `json:"name"`
	Path     string     `json:"path"`
	Type     string     `json:"type" db:"STR_TYPE"`
	Children []Category `json:"children,omitempty"`
}

// VehicleProductGroups groups applicable root SEARCH_TREE nodes by TecDoc vehicle ID.
// It is used by the server-to-server importer to avoid one HTTP request per vehicle.
type VehicleProductGroups struct {
	VehicleID  int        `json:"vehicle_id"`
	Categories []Category `json:"categories"`
}

// VehicleProductGroupIDs is the compact importer representation used when
// callers only need canonical SEARCH_TREE identities. It avoids translation
// and metadata work during large vehicle synchronizations.
type VehicleProductGroupIDs struct {
	VehicleID   int   `json:"vehicle_id"`
	CategoryIDs []int `json:"category_ids"`
}

type Product struct {
	ID          int    `json:"id" db:"PT_ID"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ============== Articles ==============

type Article struct {
	ID          int               `json:"id" db:"ART_ID"`
	ArticleNr   string            `json:"article_nr" db:"ART_ARTICLE_NR"`
	Brand       string            `json:"brand" db:"ART_SUP_BRAND"`
	SupplierID  int               `json:"supplier_id" db:"ART_SUP_ID"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	PackUnit    *int              `json:"pack_unit,omitempty"`
	Status      string            `json:"status"`
	Criteria    map[string]string `json:"criteria,omitempty"`
}

type ArticleDetail struct {
	ID                   int                `json:"id"`
	ArticleNr            string             `json:"article_nr"`
	Brand                string             `json:"brand"`
	SupplierID           int                `json:"supplier_id"`
	Name                 string             `json:"name"`
	Description          string             `json:"description"`
	Info                 string             `json:"info,omitempty"`
	PackUnit             *int               `json:"pack_unit,omitempty"`
	Quantity             *int               `json:"quantity_per_unit,omitempty"`
	Status               string             `json:"status"`
	StatusDate           *string            `json:"status_date,omitempty"`
	Criteria             map[string]string  `json:"criteria"`
	CriteriaDetails      []ArticleCriterion `json:"criteria_details,omitempty"`
	OEMNumbers           []string           `json:"oem_numbers"`
	EANNumbers           []string           `json:"ean_numbers"`
	Superseded           []string           `json:"superseded"`
	SupersededBy         []string           `json:"superseded_by"`
	Images               []ArticleMedia     `json:"images"`
	Documents            []ArticleMedia     `json:"documents"`
	PartsList            []ArticlePart      `json:"parts_list,omitempty"`
	Accessories          []ArticleAccessory `json:"accessories,omitempty"`
	AccessoriesTotal     int                `json:"accessories_total"`
	AccessoriesReturned  int                `json:"accessories_returned"`
	AccessoriesTruncated bool               `json:"accessories_truncated"`
}

type ArticleMedia struct {
	Type        string `json:"type"`
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

type ArticlePart struct {
	ArticleID     int    `json:"article_id"`
	ArticleNr     string `json:"article_nr"`
	SupplierID    int    `json:"supplier_id"`
	Brand         string `json:"brand"`
	Name          string `json:"name"`
	Quantity      int    `json:"quantity"`
	Order         int    `json:"order"`
	ProductTypeID int    `json:"product_type_id"`
}

type ArticleAccessory struct {
	SourceArticleIDs []int              `json:"source_article_ids,omitempty"`
	ArticleID        int                `json:"article_id"`
	ArticleNr        string             `json:"article_nr"`
	SupplierID       int                `json:"supplier_id"`
	Brand            string             `json:"brand"`
	Name             string             `json:"name"`
	GroupName        string             `json:"group_name"`
	RelationType     string             `json:"relation_type,omitempty"`
	RelationTypeID   int                `json:"relation_type_id,omitempty"`
	Quantity         int                `json:"quantity,omitempty"`
	ProductTypeID    int                `json:"product_type_id,omitempty"`
	Sort             int                `json:"sort,omitempty"`
	Criteria         map[string]string  `json:"criteria,omitempty"` // Backward-compatible flattened view.
	CriteriaDetails  []ArticleCriterion `json:"criteria_details,omitempty"`
}

// ============== Cross References ==============

type CrossReference struct {
	SourceArticleIDs []int  `json:"source_article_ids,omitempty"`
	ArticleID        *int   `json:"article_id"`
	ArticleNr        string `json:"article_nr"`
	Brand            string `json:"brand"` // Backward-compatible resolved supplier/manufacturer name.
	SupplierID       *int   `json:"supplier_id,omitempty"`
	Supplier         string `json:"supplier,omitempty"`
	ManufacturerID   *int   `json:"manufacturer_id,omitempty"`
	Manufacturer     string `json:"manufacturer,omitempty"`
	Name             string `json:"name"`
	Type             string `json:"type"` // OEM, IAM, ArticleNumber
}

// ============== Search ==============

type SearchResult struct {
	ArticleID    int    `json:"article_id"`
	ArticleNr    string `json:"article_nr"`
	Brand        string `json:"brand"`
	SupplierID   int    `json:"supplier_id"`
	Name         string `json:"name"`
	FoundVia     string `json:"found_via"`               // Lookup path: ArticleNumber, IAMNumber, OENumber, EAN.
	RelationType string `json:"relation_type,omitempty"` // Entity returned by this row; search results are supplier articles.
}

// ============== Applicability ==============

type Applicability struct {
	PCID        int      `json:"pc_id"`
	TypeName    string   `json:"type_name"`
	YearFrom    int      `json:"year_from"`
	YearTo      int      `json:"year_to"`
	PowerKW     int      `json:"power_kw"`
	PowerHP     int      `json:"power_hp"`
	Capacity    int      `json:"capacity"`
	BodyType    string   `json:"body_type"`
	EngineCodes []string `json:"engine_codes"`
	TermsOfUse  string   `json:"terms_of_use,omitempty"`
}

// ============== Response Wrappers ==============

type VehicleTypesResponse struct {
	Types []VehicleType `json:"types"`
}

type ManufacturersResponse struct {
	Manufacturers []Manufacturer `json:"manufacturers"`
}

type ManufacturerDetailResponse struct {
	Manufacturer ManufacturerDetail `json:"manufacturer"`
}

type ModelSeriesResponse struct {
	Models []ModelSeries `json:"models"`
}

type ModelDetailResponse struct {
	Model ModelDetail `json:"model"`
}

type PassengerCarsResponse struct {
	Vehicles []PassengerCar `json:"vehicles"`
}

type VehicleDetailResponse struct {
	Vehicle VehicleDetail `json:"vehicle"`
}

type EnginesResponse struct {
	Engines []Engine `json:"engines"`
}

type CategoriesResponse struct {
	Categories []Category `json:"categories"`
}

type ArticlesResponse struct {
	Articles []Article `json:"articles"`
}

type ArticleDetailResponse struct {
	Article ArticleDetail `json:"article"`
}

type CrossReferencesResponse struct {
	Crosses []CrossReference `json:"crosses"`
}

type SearchResponse struct {
	Results []SearchResult `json:"results"`
}

type ApplicabilityResponse struct {
	Vehicles []Applicability `json:"vehicles"`
}

// ============== OEM Numbers ==============

type OEMNumber struct {
	SourceArticleIDs []int  `json:"source_article_ids,omitempty"`
	Number           string `json:"number"`
	Brand            string `json:"brand"`    // Backward-compatible alias for manufacturer.
	BrandID          int    `json:"brand_id"` // Backward-compatible alias for manufacturer_id.
	Manufacturer     string `json:"manufacturer"`
	ManufacturerID   int    `json:"manufacturer_id"`
}

type OEMNumbersResponse struct {
	OEMNumbers []OEMNumber `json:"oem_numbers"`
	Total      int         `json:"total"`
}

// ============== Commercial Vehicles ==============

type CommercialVehicle struct {
	ID           int      `json:"id"`
	MfaID        int      `json:"mfa_id"`
	MsID         int      `json:"ms_id"`
	TypeName     string   `json:"type_name"`
	YearFrom     int      `json:"year_from"`
	YearTo       int      `json:"year_to"`
	PowerKW      int      `json:"power_kw"`
	PowerHP      int      `json:"power_hp"`
	Capacity     int      `json:"capacity"`
	Tonnage      float64  `json:"tonnage"`
	EngineType   string   `json:"engine_type"`
	PlatformType string   `json:"platform_type"`
	EngineCodes  []string `json:"engine_codes"`
	Engines      []Engine `json:"engines"`
}

type CommercialVehiclesResponse struct {
	Vehicles []CommercialVehicle `json:"vehicles"`
	Total    int                 `json:"total"`
	Limit    int                 `json:"limit"`
	Offset   int                 `json:"offset"`
}

// ============== Motorcycles ==============

type Motorcycle struct {
	ID             int      `json:"id"`
	MfaID          int      `json:"mfa_id"`
	MsID           int      `json:"ms_id"`
	TypeName       string   `json:"type_name"`
	YearFrom       int      `json:"year_from"`
	YearTo         int      `json:"year_to"`
	PowerKW        int      `json:"power_kw"`
	PowerHP        int      `json:"power_hp"`
	Capacity       int      `json:"capacity"`
	CapacityLiters *float64 `json:"capacity_liters,omitempty"`
	Cylinders      *int     `json:"cylinders,omitempty"`
	Valves         *int     `json:"valves,omitempty"`
	EngineType     string   `json:"engine_type"`
	FuelType       string   `json:"fuel_type"`
	DriveType      string   `json:"drive_type"`
	Type           string   `json:"type"`
	EngineCodes    []string `json:"engine_codes"`
	Engines        []Engine `json:"engines"`
}

// Add this missing struct:
type MotorcyclesResponse struct {
	Motorcycles []Motorcycle `json:"motorcycles"`
	Total       int          `json:"total"`
	Limit       int          `json:"limit"`
	Offset      int          `json:"offset"`
}

// ============== OEM Cross References ==============

type OEMCrossReference struct {
	ArticleID      int    `json:"article_id"`
	CrossBrand     string `json:"cross_brand"` // Backward-compatible alias for manufacturer.
	CrossNumber    string `json:"cross_number"`
	ManufacturerID int    `json:"manufacturer_id"`
	Manufacturer   string `json:"manufacturer"`
}

type OEMCrossReferenceResponse struct {
	References []OEMCrossReference `json:"references"`
	Total      int                 `json:"total"`
}

// ============== Engine Details ==============

type EngineDetail struct {
	EngineID       int                    `json:"engine_id"`
	EngineCode     string                 `json:"engine_code"`
	ManufacturerID int                    `json:"manufacturer_id"`
	DateFrom       string                 `json:"date_from,omitempty"`
	DateTo         string                 `json:"date_to,omitempty"`
	Specs          map[string]interface{} `json:"specs"`
}

type EngineDetailResponse struct {
	Engine EngineDetail `json:"engine"`
}

type EngineListResponse struct {
	Engines []EngineDetail `json:"engines"`
	Total   int            `json:"total"`
	Limit   int            `json:"limit"`
	Offset  int            `json:"offset"`
}

// ============== Article Coordinates ==============

type ArticleCoordinate struct {
	ArticleID     int    `json:"article_id"`
	ArticleNumber string `json:"article_number"`
	Brand         string `json:"brand"`
	MediaSource   string `json:"media_source"`
	CoordinateID  int    `json:"coordinate_id"`
	X             int    `json:"x"`
	Y             int    `json:"y"`
	Width         int    `json:"width"`
	Height        int    `json:"height"`
	Type          string `json:"type"` // Circle, Square
}

type ArticleCoordinatesResponse struct {
	Coordinates []ArticleCoordinate `json:"coordinates"`
	Total       int                 `json:"total"`
}

// ============== Article Criteria ==============

type ArticleCriterion struct {
	SourceArticleIDs   []int  `json:"source_article_ids,omitempty"`
	CriteriaID         int    `json:"criteria_id"`
	Name               string `json:"name"`
	ShortName          string `json:"short_name"`
	Value              string `json:"value"` // Backward-compatible alias of RawValue.
	RawValue           string `json:"raw_value"`
	ValueDescriptionID int    `json:"value_description_id,omitempty"`
	ValueDescription   string `json:"value_description,omitempty"`
	Unit               string `json:"unit,omitempty"`
	Type               string `json:"type"` // Numerical, Alphanumerical, KeyValue, Date, NoValue, Interval, Boolean.
	Display            bool   `json:"display"`
}

type ArticleCriteriaResponse struct {
	Criteria []ArticleCriterion `json:"criteria"`
	Total    int                `json:"total"`
}

// ============== Supplier Products ==============

type SupplierProduct struct {
	ProductID     int    `json:"product_id"`
	ProductName   string `json:"product_name"`
	ArticlesCount int    `json:"articles_count"`
}

type SupplierProductsResponse struct {
	Products []SupplierProduct `json:"products"`
	Total    int               `json:"total"`
	Limit    int               `json:"limit"`
	Offset   int               `json:"offset"`
}
