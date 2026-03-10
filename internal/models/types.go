package models

// ============== Manufacturers ==============

type VehicleType struct {
ID          int    `json:"id"`
Type        string `json:"type"` // PC, CV, MC
Description string `json:"description"`
}

type Manufacturer struct {
ID           int    `json:"id" db:"MFA_ID"`
Brand        string `json:"brand" db:"MFA_BRAND"`
Type         string `json:"type" db:"MFA_TYPE"`
ModelsCount  int    `json:"models_count" db:"MFA_MODELS_COUNT"`
SupplierID   *int   `json:"supplier_id,omitempty" db:"MFA_SUP_ID"`
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
ID          int    `json:"id" db:"MS_ID"`
MfaID       int    `json:"mfa_id" db:"MS_MFA_ID"`
Name        string `json:"name"`
YearFrom    int    `json:"year_from" db:"MS_CI_FROM"`
YearTo      int    `json:"year_to" db:"MS_CI_TO"`
Type        string `json:"type" db:"MS_TYPE"`
}

type ModelDetail struct {
ID          int    `json:"id"`
MfaID       int    `json:"mfa_id"`
Manufacturer string `json:"manufacturer"`
Name        string `json:"name"`
YearFrom    int    `json:"year_from"`
YearTo      int    `json:"year_to"`
Type        string `json:"type"`
}

// ============== Vehicles ==============

type PassengerCar struct {
ID          int      `json:"id" db:"PC_ID"`
MfaID       int      `json:"mfa_id" db:"PC_MFA_ID"`
MsID        int      `json:"ms_id" db:"PC_MS_ID"`
TypeName    string   `json:"type_name"`
YearFrom    int      `json:"year_from"`
YearTo      int      `json:"year_to"`
PowerKW     int      `json:"power_kw"`
PowerHP     int      `json:"power_hp"`
Capacity    int      `json:"capacity"`
FuelType    string   `json:"fuel_type"`
BodyType    string   `json:"body_type"`
EngineType  string   `json:"engine_type"`
EngineCodes []string `json:"engine_codes"`
}

type VehicleDetail struct {
ID            int                    `json:"id"`
TypeName      string                 `json:"type_name"`
Manufacturer  string                 `json:"manufacturer"`
ModelSeries   string                 `json:"model_series"`
YearFrom      int                    `json:"year_from"`
YearTo        int                    `json:"year_to"`
Specs         map[string]interface{} `json:"specs"`
EngineCodes   []string               `json:"engine_codes"`
}

type Engine struct {
ID       int    `json:"id" db:"ENG_ID"`
Code     string `json:"code" db:"ENG_CODE"`
Name     string `json:"name" db:"ENG_NAME"`
}

// ============== Categories ==============

type Category struct {
ID       int         `json:"id" db:"STR_ID"`
ParentID *int        `json:"parent_id" db:"STR_ID_PARENT"`
Level    int         `json:"level" db:"STR_LEVEL"`
Name     string      `json:"name"`
Path     string      `json:"path"`
Type     string      `json:"type" db:"STR_TYPE"`
Children []Category  `json:"children,omitempty"`
}

type Product struct {
ID          int    `json:"id" db:"PT_ID"`
Name        string `json:"name"`
Description string `json:"description"`
}

// ============== Articles ==============

type Article struct {
ID          int                    `json:"id" db:"ART_ID"`
ArticleNr   string                 `json:"article_nr" db:"ART_ARTICLE_NR"`
Brand       string                 `json:"brand" db:"ART_SUP_BRAND"`
SupplierID  int                    `json:"supplier_id" db:"ART_SUP_ID"`
Name        string                 `json:"name"`
Description string                 `json:"description"`
PackUnit    *int                   `json:"pack_unit,omitempty"`
Status      string                 `json:"status"`
Criteria    map[string]string      `json:"criteria,omitempty"`
}

type ArticleDetail struct {
ID           int                 `json:"id"`
ArticleNr    string              `json:"article_nr"`
Brand        string              `json:"brand"`
SupplierID   int                 `json:"supplier_id"`
Name         string              `json:"name"`
Description  string              `json:"description"`
Info         string              `json:"info,omitempty"`
PackUnit     *int                `json:"pack_unit,omitempty"`
Quantity     *int                `json:"quantity_per_unit,omitempty"`
Status       string              `json:"status"`
StatusDate   *string             `json:"status_date,omitempty"`
Criteria     map[string]string   `json:"criteria"`
OEMNumbers   []string            `json:"oem_numbers"`
EANNumbers   []string            `json:"ean_numbers"`
Superseded   []string            `json:"superseded"`
SupersededBy []string            `json:"superseded_by"`
Images       []ArticleMedia      `json:"images"`
Documents    []ArticleMedia      `json:"documents"`
PartsList    []ArticlePart       `json:"parts_list,omitempty"`
Accessories  []ArticleAccessory  `json:"accessories,omitempty"`
}

type ArticleMedia struct {
Type        string `json:"type"`
URL         string `json:"url"`
Description string `json:"description,omitempty"`
}

type ArticlePart struct {
ArticleNr string `json:"article_nr"`
Brand     string `json:"brand"`
Name      string `json:"name"`
Quantity  int    `json:"quantity"`
Order     int    `json:"order"`
}

type ArticleAccessory struct {
ArticleID   int               `json:"article_id"`
ArticleNr   string            `json:"article_nr"`
Brand       string            `json:"brand"`
Name        string            `json:"name"`
GroupName   string            `json:"group_name"`
Criteria    map[string]string `json:"criteria,omitempty"`
}


// ============== Cross References ==============

type CrossReference struct {
ArticleID *int   `json:"article_id"`
ArticleNr string `json:"article_nr"`
Brand     string `json:"brand"`
Name      string `json:"name"`
Type      string `json:"type"` // OEM, IAM, ArticleNumber
}

// ============== Search ==============

type SearchResult struct {
ArticleID int    `json:"article_id"`
ArticleNr string `json:"article_nr"`
Brand     string `json:"brand"`
Name      string `json:"name"`
FoundVia  string `json:"found_via"`
}

// ============== Applicability ==============

type Applicability struct {
PCID       int      `json:"pc_id"`
TypeName   string   `json:"type_name"`
YearFrom   int      `json:"year_from"`
YearTo     int      `json:"year_to"`
PowerKW    int      `json:"power_kw"`
PowerHP    int      `json:"power_hp"`
Capacity   int      `json:"capacity"`
BodyType   string   `json:"body_type"`
EngineCodes []string `json:"engine_codes"`
TermsOfUse string   `json:"terms_of_use,omitempty"`
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
Number  string `json:"number"`
Brand   string `json:"brand"`
BrandID int    `json:"brand_id"`
}

type OEMNumbersResponse struct {
OEMNumbers []OEMNumber `json:"oem_numbers"`
Total      int         `json:"total"`
}

// ============== Commercial Vehicles ==============

type CommercialVehicle struct {
ID            int      `json:"id"`
MfaID         int      `json:"mfa_id"`
MsID          int      `json:"ms_id"`
TypeName      string   `json:"type_name"`
YearFrom      int      `json:"year_from"`
YearTo        int      `json:"year_to"`
PowerKW       int      `json:"power_kw"`
PowerHP       int      `json:"power_hp"`
Capacity      int      `json:"capacity"`
Tonnage       float64  `json:"tonnage"`
EngineType    string   `json:"engine_type"`
PlatformType  string   `json:"platform_type"`
EngineCodes   []string `json:"engine_codes"`
}

type CommercialVehiclesResponse struct {
Vehicles []CommercialVehicle `json:"vehicles"`
Total    int                  `json:"total"`
}

// ============== Motorcycles ==============

type Motorcycle struct {
    ID          int      `json:"id"`
    MfaID       int      `json:"mfa_id"`
    MsID        int      `json:"ms_id"`
    TypeName    string   `json:"type_name"`
    YearFrom    int      `json:"year_from"`
    YearTo      int      `json:"year_to"`
    PowerKW     int      `json:"power_kw"`
    PowerHP     int      `json:"power_hp"`
    Capacity    int      `json:"capacity"`
    EngineType  string   `json:"engine_type"`
    FuelType    string   `json:"fuel_type"`
    Type        string   `json:"type"`
    EngineCodes []string `json:"engine_codes"`
}

// Add this missing struct:
type MotorcyclesResponse struct {
    Motorcycles []Motorcycle `json:"motorcycles"`
    Total       int        `json:"total"`
}

// ============== OEM Cross References ==============

type OEMCrossReference struct {
ArticleID   int    `json:"article_id"`
CrossBrand  string `json:"cross_brand"`
CrossNumber string `json:"cross_number"`
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
CriteriaID       int    `json:"criteria_id"`
Name             string `json:"name"`
ShortName        string `json:"short_name"`
Value            string `json:"value"`
ValueDescription string `json:"value_description,omitempty"`
Type             string `json:"type"` // Numerical, Alphanumerical, KeyValue
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
}
