package models

// TypeNumberMatch exposes one row from TYPE_NUMBER_LOOKUP. Use the specific
// vehicle endpoint afterwards when the full vehicle payload is required.
type TypeNumberMatch struct {
	VehicleID      int    `json:"vehicle_id"`
	VehicleType    string `json:"vehicle_type"`
	Number         string `json:"number"`
	NumberType     string `json:"number_type"`
	ManufacturerID int    `json:"manufacturer_id"`
}

type TypeNumberMatchesResponse struct {
	Matches []TypeNumberMatch `json:"matches"`
	Total   int               `json:"total"`
}

type VehicleCriterion struct {
	CriteriaID int      `json:"criteria_id"`
	Name       string   `json:"name"`
	ShortName  string   `json:"short_name,omitempty"`
	Unit       string   `json:"unit,omitempty"`
	Type       string   `json:"type"`
	Suggested  bool     `json:"suggested"`
	Values     []string `json:"values"`
}

type VehicleCriteriaResponse struct {
	VehicleID   int                `json:"vehicle_id"`
	VehicleType string             `json:"vehicle_type"`
	Criteria    []VehicleCriterion `json:"criteria"`
	Total       int                `json:"total"`
}

type ProductTypeCriterion struct {
	ProductTypeID int    `json:"product_type_id"`
	CriteriaID    int    `json:"criteria_id"`
	Name          string `json:"name"`
	ShortName     string `json:"short_name,omitempty"`
	Unit          string `json:"unit,omitempty"`
	Type          string `json:"type"`
	Sort          *int   `json:"sort,omitempty"`
	Suggested     bool   `json:"suggested"`
}

type ProductTypeCriteriaResponse struct {
	Criteria []ProductTypeCriterion `json:"criteria"`
	Total    int                    `json:"total"`
}

type ProductGroupSearchResult struct {
	CategoryID    int    `json:"category_id"`
	ParentID      *int   `json:"parent_id,omitempty"`
	Level         int    `json:"level"`
	Name          string `json:"name"`
	Path          string `json:"path"`
	VehicleType   string `json:"vehicle_type"`
	ProductTypeID int    `json:"product_type_id"`
	ProductType   string `json:"product_type"`
}

type ProductGroupSearchResponse struct {
	Results []ProductGroupSearchResult `json:"results"`
	Total   int                        `json:"total"`
}

type QuickStartItem struct {
	ID          int    `json:"id"`
	Type        string `json:"type"`
	Name        string `json:"name"`
	Image       string `json:"image,omitempty"`
	CategoryIDs []int  `json:"category_ids,omitempty"`
}

type QuickStartResponse struct {
	Items []QuickStartItem `json:"items"`
	Total int              `json:"total"`
}

type ArticlePrice struct {
	ArticleID     int     `json:"article_id"`
	CountryID     int     `json:"country_id"`
	Type          string  `json:"type"`
	Price         float64 `json:"price"`
	Currency      string  `json:"currency,omitempty"`
	ValidFrom     string  `json:"valid_from"`
	ValidTo       string  `json:"valid_to,omitempty"`
	Unit          string  `json:"unit,omitempty"`
	Quantity      string  `json:"quantity,omitempty"`
	DiscountGroup string  `json:"discount_group,omitempty"`
}

type ArticlePricesResponse struct {
	Prices []ArticlePrice `json:"prices"`
	Total  int            `json:"total"`
}

type SupplierAddress struct {
	SupplierID  int    `json:"supplier_id"`
	AddressType string `json:"address_type,omitempty"`
	CountryID   int    `json:"country_id"`
	Country     string `json:"country,omitempty"`
	Name1       string `json:"name1,omitempty"`
	Name2       string `json:"name2,omitempty"`
	City1       string `json:"city1,omitempty"`
	City2       string `json:"city2,omitempty"`
	Street1     string `json:"street1,omitempty"`
	Street2     string `json:"street2,omitempty"`
	PostalCode  string `json:"postal_code,omitempty"`
	Email       string `json:"email,omitempty"`
	Homepage    string `json:"homepage,omitempty"`
	Telephone   string `json:"telephone,omitempty"`
	Fax         string `json:"fax,omitempty"`
}

type SupplierAddressesResponse struct {
	Addresses []SupplierAddress `json:"addresses"`
	Total     int               `json:"total"`
}

type SupplierLogo struct {
	SupplierID int    `json:"supplier_id"`
	CountryID  int    `json:"country_id"`
	FileName   string `json:"file_name"`
}

type SupplierLogosResponse struct {
	Logos []SupplierLogo `json:"logos"`
	Total int            `json:"total"`
}

type ArticleInfoItem struct {
	Type    string `json:"type,omitempty"`
	Text    string `json:"text"`
	Sort    int    `json:"sort"`
	Display bool   `json:"display"`
	TextID  int    `json:"text_id"`
}

type ArticleInfoResponse struct {
	Items []ArticleInfoItem `json:"items"`
	Total int               `json:"total"`
}

type CVWheelbase struct {
	AxleGroup string `json:"axle_group"`
	Wheelbase int    `json:"wheelbase_mm"`
}

type CVSuspension struct {
	AxlePosition string `json:"axle_position,omitempty"`
	Description  string `json:"description,omitempty"`
}

type DriverCab struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	From     string `json:"from,omitempty"`
	To       string `json:"to,omitempty"`
	LengthMM *int   `json:"length_mm,omitempty"`
	HeightMM *int   `json:"height_mm,omitempty"`
	WidthMM  *int   `json:"width_mm,omitempty"`
	Design   string `json:"design,omitempty"`
	Size     *int   `json:"size,omitempty"`
}

type CVAxle struct {
	ID          int    `json:"id"`
	Code        string `json:"code"`
	Description string `json:"description,omitempty"`
	Position    string `json:"position,omitempty"`
	From        string `json:"from,omitempty"`
	To          string `json:"to,omitempty"`
}

type CommercialVehicleChassis struct {
	VehicleID   int            `json:"vehicle_id"`
	Wheelbases  []CVWheelbase  `json:"wheelbases"`
	Suspensions []CVSuspension `json:"suspensions"`
	DriverCabs  []DriverCab    `json:"driver_cabs"`
	Axles       []CVAxle       `json:"axles"`
}

// Dataset metadata and rows provide complete read-only access to the TecDoc
// source tables without exposing API credentials or phpMyAdmin system tables.
type DatasetColumn struct {
	Name       string  `json:"name"`
	DataType   string  `json:"data_type"`
	ColumnType string  `json:"column_type"`
	Nullable   bool    `json:"nullable"`
	Key        string  `json:"key,omitempty"`
	Default    *string `json:"default,omitempty"`
	Extra      string  `json:"extra,omitempty"`
	Ordinal    int     `json:"ordinal"`
}

type DatasetInfo struct {
	Name          string `json:"name"`
	EstimatedRows int64  `json:"estimated_rows"`
}

type DatasetsResponse struct {
	Datasets []DatasetInfo `json:"datasets"`
	Total    int           `json:"total"`
}

type DatasetSchemaResponse struct {
	Table   string          `json:"table"`
	Columns []DatasetColumn `json:"columns"`
}

type DatasetRowsResponse struct {
	Table  string           `json:"table"`
	Rows   []map[string]any `json:"rows"`
	Limit  int              `json:"limit"`
	Offset int              `json:"offset"`
	Total  *int64           `json:"total,omitempty"`
}

// ArticleSpecification is the complete TecDoc specification layer for an
// article. source=article originates from ARTICLE_CRITERIA; source=linkage
// originates from LA_CRITERIA and therefore applies only to that LA_ID.
type ArticleSpecification struct {
	ArticleID        int    `json:"article_id"`
	Source           string `json:"source"`
	LinkageID        *int   `json:"linkage_id,omitempty"`
	ProductTypeID    *int   `json:"product_type_id,omitempty"`
	CriteriaID       int    `json:"criteria_id"`
	Name             string `json:"name"`
	ShortName        string `json:"short_name,omitempty"`
	Type             string `json:"type"`
	Unit             string `json:"unit,omitempty"`
	RawValue         string `json:"raw_value"`
	ValueDescription string `json:"value_description,omitempty"`
	Display          *bool  `json:"display,omitempty"`
}

type ArticleSpecificationsResponse struct {
	ArticleID      int                    `json:"article_id"`
	Specifications []ArticleSpecification `json:"specifications"`
	Total          int                    `json:"total"`
	Limit          int                    `json:"limit"`
	Offset         int                    `json:"offset"`
	Returned       int                    `json:"returned"`
	HasMore        bool                   `json:"has_more"`
}

type ArticleLinkageInfoItem struct {
	ArticleID     int    `json:"article_id"`
	LinkageID     int    `json:"linkage_id"`
	ProductTypeID int    `json:"product_type_id"`
	Type          string `json:"type,omitempty"`
	Text          string `json:"text"`
	TextID        int    `json:"text_id"`
}

type ArticleLinkageInfoResponse struct {
	ArticleID int                      `json:"article_id"`
	Items     []ArticleLinkageInfoItem `json:"items"`
	Total     int                      `json:"total"`
	Limit     int                      `json:"limit"`
	Offset    int                      `json:"offset"`
	Returned  int                      `json:"returned"`
	HasMore   bool                     `json:"has_more"`
}

type PartsDiagramCoordinate struct {
	ParentArticleID    int    `json:"parent_article_id"`
	ComponentArticleID int    `json:"component_article_id"`
	ArticleNumber      string `json:"article_number"`
	Brand              string `json:"brand"`
	MediaID            int    `json:"media_id"`
	MediaType          string `json:"media_type"`
	MediaSource        string `json:"media_source"`
	CoordinateID       int    `json:"coordinate_id"`
	X                  int    `json:"x"`
	Y                  int    `json:"y"`
	Width              int    `json:"width"`
	Height             int    `json:"height"`
	Type               string `json:"type"`
}

type PartsDiagramResponse struct {
	ArticleID  int                      `json:"article_id"`
	Components []PartsDiagramCoordinate `json:"components"`
	Total      int                      `json:"total"`
	Limit      int                      `json:"limit"`
	Offset     int                      `json:"offset"`
	Returned   int                      `json:"returned"`
	HasMore    bool                     `json:"has_more"`
}

type CrossReferenceExportRow struct {
	ArticleNumber string `json:"article_number"`
	ArticleBrand  string `json:"article_brand"`
	CrossNumber   string `json:"cross_number"`
	CrossBrand    string `json:"cross_brand"`
	CrossType     string `json:"cross_type"`
}

type EANExportRow struct {
	ArticleNumber string `json:"article_number"`
	ArticleBrand  string `json:"article_brand"`
	EAN           string `json:"ean"`
}

type CrossReferenceExportResponse struct {
	Rows   []CrossReferenceExportRow `json:"rows"`
	Limit  int                       `json:"limit"`
	Offset int                       `json:"offset"`
}

type EANExportResponse struct {
	Rows   []EANExportRow `json:"rows"`
	Limit  int            `json:"limit"`
	Offset int            `json:"offset"`
}

type CriterionKeyValue struct {
	CriteriaID  int    `json:"criteria_id"`
	KeyTableID  int    `json:"key_table_id"`
	Value       string `json:"value"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

type CriterionKeyValuesResponse struct {
	CriteriaID int                 `json:"criteria_id"`
	Values     []CriterionKeyValue `json:"values"`
	Total      int                 `json:"total"`
	Limit      int                 `json:"limit"`
	Offset     int                 `json:"offset"`
	Returned   int                 `json:"returned"`
	HasMore    bool                `json:"has_more"`
}

type AxleModelName struct {
	CountryID int    `json:"country_id"`
	Sort      int    `json:"sort"`
	Model     string `json:"model"`
}

type AxleBrakeSize struct {
	Sequence    int    `json:"sequence"`
	Size        string `json:"size,omitempty"`
	Description string `json:"description,omitempty"`
}

type AxleDetail struct {
	ID                            int             `json:"id"`
	ManufacturerID                int             `json:"manufacturer_id"`
	Manufacturer                  string          `json:"manufacturer,omitempty"`
	ModelSeriesID                 int             `json:"model_series_id"`
	ModelSeries                   string          `json:"model_series,omitempty"`
	Code                          string          `json:"code"`
	Description                   string          `json:"description,omitempty"`
	AxleStyleID                   *int            `json:"axle_style_id,omitempty"`
	AxleStyle                     string          `json:"axle_style,omitempty"`
	AxleBodyID                    *int            `json:"axle_body_id,omitempty"`
	AxleBody                      string          `json:"axle_body,omitempty"`
	AxleTypeID                    *int            `json:"axle_type_id,omitempty"`
	AxleType                      string          `json:"axle_type,omitempty"`
	ProductionFrom                string          `json:"production_from,omitempty"`
	ProductionTo                  string          `json:"production_to,omitempty"`
	AllowedWeightKGStart          *int            `json:"allowed_weight_kg_start,omitempty"`
	AllowedWeightKGUpTo           *int            `json:"allowed_weight_kg_upto,omitempty"`
	BrakeTypeID                   *int            `json:"brake_type_id,omitempty"`
	BrakeType                     string          `json:"brake_type,omitempty"`
	HubSystem                     string          `json:"hub_system,omitempty"`
	DistanceToVehicleFrameMMStart *int            `json:"distance_to_vehicle_frame_mm_start,omitempty"`
	DistanceToVehicleFrameMMUpTo  *int            `json:"distance_to_vehicle_frame_mm_upto,omitempty"`
	WheelMountingID               *int            `json:"wheel_mounting_id,omitempty"`
	WheelMounting                 string          `json:"wheel_mounting,omitempty"`
	TrackWidthMM                  *int            `json:"track_width_mm,omitempty"`
	Models                        []AxleModelName `json:"models"`
	BrakeSizes                    []AxleBrakeSize `json:"brake_sizes"`
}
