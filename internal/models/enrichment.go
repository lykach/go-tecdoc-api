package models

// ArticleClassificationTreeNode is one TecDoc SEARCH_TREE placement for a
// product type. STR_ID is globally unique in the currently deployed 2024Q1
// dataset, while RawType and Scopes are still returned so consumers do not
// need to infer vehicle scope from the numeric ID.
type ArticleClassificationTreeNode struct {
	ID       int      `json:"id"`
	ParentID *int     `json:"parent_id,omitempty"`
	Level    int      `json:"level"`
	IsLeaf   bool     `json:"is_leaf"`
	RawType  string   `json:"raw_type"`
	Scopes   []string `json:"scopes"`
	Name     string   `json:"name"`
}

// ArticleProductTypeClassification describes one TecDoc product type (PT_ID)
// attached to an article. Linkage/target counts are evidence, not an automatic
// instruction to pick this product type as the local shop category.
type ArticleProductTypeClassification struct {
	SourceArticleIDs []int                           `json:"source_article_ids,omitempty"`
	ProductTypeID    int                             `json:"product_type_id"`
	Name             string                          `json:"name"`
	Scopes           []string                        `json:"scopes"`
	TreeNodes        []ArticleClassificationTreeNode `json:"tree_nodes"`
	LinkageCount     int                             `json:"linkage_count"`
	TargetCount      int                             `json:"target_count"`
}

type ArticleClassificationResponse struct {
	ArticleID    int                                `json:"article_id"`
	ProductTypes []ArticleProductTypeClassification `json:"product_types"`
	Total        int                                `json:"total"`
}

// ApplicabilityTarget is intentionally normalized to TecDoc target type + ID.
// autoparts-platform already owns vehicle details locally and can resolve this
// stable identity without duplicating large vehicle payloads for every link.
type ApplicabilityTarget struct {
	Type string `json:"type"`
	ID   int    `json:"id"`
}

type ApplicabilityRestriction struct {
	CriteriaID       int    `json:"criteria_id"`
	Name             string `json:"name"`
	ShortName        string `json:"short_name"`
	Type             string `json:"type"`
	Unit             string `json:"unit,omitempty"`
	RawValue         string `json:"raw_value"`
	ValueDescription string `json:"value_description,omitempty"`
}

type ApplicabilityLinkage struct {
	SourceArticleIDs []int                      `json:"source_article_ids,omitempty"`
	LinkageID        int                        `json:"la_id"`
	ProductType      int                        `json:"pt_id"`
	TargetType       string                     `json:"target_type"`
	Targets          []ApplicabilityTarget      `json:"targets"`
	Restrictions     []ApplicabilityRestriction `json:"restrictions"`
}

type ApplicabilityLinkagesPage struct {
	Linkages          []ApplicabilityLinkage `json:"linkages"`
	TotalTargets      int                    `json:"total_targets"`
	Returned          int                    `json:"returned_targets"`
	Truncated         bool                   `json:"truncated"`
	TotalTargetsExact bool                   `json:"total_targets_exact"`
}

// EnrichmentBatchRequest is the server-to-server bulk contract used by the
// catalog enrichment worker. ArticleID is accepted for a single already-resolved
// technical item. ArticleIDs reuses an immutable reviewed commercial identity and
// is always revalidated against ArticleNumber + SupplierID before facts are read.
// Otherwise ArticleNumber is resolved in one set-based lookup. SupplierID (or an
// explicit Brand/BrandAliases manufacturer identity as a fallback) prevents
// short/common article numbers from silently binding to the wrong manufacturer.
type EnrichmentBatchRequest struct {
	LanguageID              int                   `json:"language_id,omitempty"`
	CountryID               int                   `json:"country_id,omitempty"`
	Include                 []string              `json:"include,omitempty"`
	MaxApplicabilityTargets int                   `json:"max_applicability_targets,omitempty"`
	MaxAccessories          int                   `json:"max_accessories,omitempty"`
	Items                   []EnrichmentBatchItem `json:"items"`
}

type EnrichmentBatchItem struct {
	RequestID     string   `json:"request_id"`
	ArticleID     *int     `json:"article_id,omitempty"`
	ArticleIDs    []int    `json:"article_ids,omitempty"`
	ArticleNumber string   `json:"article_number,omitempty"`
	SupplierID    *int     `json:"supplier_id,omitempty"`
	Brand         string   `json:"brand,omitempty"`
	BrandAliases  []string `json:"brand_aliases,omitempty"`
}

type EnrichmentResolvedArticle struct {
	ArticleID     int    `json:"article_id"`
	ArticleNumber string `json:"article_number"`
	SupplierID    int    `json:"supplier_id"`
	Brand         string `json:"brand"`
	Name          string `json:"name"`
}

type EnrichmentBatchResult struct {
	RequestID       string                             `json:"request_id"`
	Status          string                             `json:"status"` // exact, ambiguous, not_found, error
	Error           string                             `json:"error,omitempty"`
	Resolved        *EnrichmentResolvedArticle         `json:"resolved,omitempty"`
	ArticleIDs      []int                              `json:"article_ids,omitempty"`
	Candidates      []EnrichmentResolvedArticle        `json:"candidates,omitempty"`
	Classification  []ArticleProductTypeClassification `json:"classification,omitempty"`
	OEMNumbers      []OEMNumber                        `json:"oem_numbers,omitempty"`
	CrossReferences []CrossReference                   `json:"cross_references,omitempty"`
	Criteria        []ArticleCriterion                 `json:"criteria,omitempty"`
	Applicability   *ApplicabilityLinkagesPage         `json:"applicability,omitempty"`
	Barcodes        []ArticleLookupNumber              `json:"barcodes,omitempty"`
	TradeNumbers    []ArticleLookupNumber              `json:"trade_numbers,omitempty"`
	Media           []ArticleMediaFact                 `json:"media,omitempty"`
	Components      []ArticleComponentFact             `json:"components,omitempty"`
	Accessories     *ArticleAccessoriesPage            `json:"accessories,omitempty"`
	ArticleInfo     []ArticleInfoFact                  `json:"article_info,omitempty"`
	Supersessions   []ArticleSupersessionFact          `json:"supersessions,omitempty"`
	CountrySpecific []ArticleCountryMetadata           `json:"country_specifics,omitempty"`
}

// ArticleLookupNumber is a normalized number from ART_LOOKUP. SourceArticleIDs
// retains every technical ART_ID that contributed to the commercial identity.
type ArticleLookupNumber struct {
	SourceArticleIDs []int  `json:"source_article_ids,omitempty"`
	Number           string `json:"number"`
	SearchNumber     string `json:"search_number,omitempty"`
	Type             string `json:"type"`
}

// ArticleMediaFact exposes TecDoc media identity separately from the derived
// path/URL. ART_MEDIA_ID alone is not unique enough for article media. TecDoc
// 2024Q1 distinguishes linkage rows by article + media id + media type + sort
// and also carries country/language/content-type dimensions. CountryMarker is
// an applicability scope filtered for the requested country; filename/hyperlink
// are mutable payload and must not be treated as durable downstream identity.
type ArticleMediaFact struct {
	SourceArticleIDs []int  `json:"source_article_ids,omitempty"`
	ArticleID        int    `json:"article_id"`
	MediaID          int    `json:"media_id"`
	MediaType        string `json:"media_type"`
	SupplierID       int    `json:"supplier_id"`
	FileName         string `json:"file_name,omitempty"`
	URL              string `json:"url"`
	Sort             int    `json:"sort"`
	CountryMarker    int    `json:"country_marker,omitempty"`
	LanguageID       int    `json:"language_id,omitempty"`
	ContentType      string `json:"content_type,omitempty"`
	Width            int    `json:"width,omitempty"`
	Height           int    `json:"height,omitempty"`
	ColorDepth       int    `json:"color_depth,omitempty"`
	NormalizedDesc   string `json:"normalized_description,omitempty"`
	SupplierDesc     string `json:"supplier_description,omitempty"`
}

type ArticleComponentFact struct {
	SourceArticleIDs []int  `json:"source_article_ids,omitempty"`
	ParentArticleID  int    `json:"parent_article_id"`
	ArticleID        int    `json:"article_id"`
	ArticleNr        string `json:"article_nr"`
	SupplierID       int    `json:"supplier_id"`
	Brand            string `json:"brand"`
	Name             string `json:"name"`
	Quantity         int    `json:"quantity"`
	Sequence         int    `json:"sequence"`
	ProductTypeID    int    `json:"product_type_id,omitempty"`
}

type ArticleAccessoriesPage struct {
	Items            []ArticleAccessory `json:"items"`
	Total            int                `json:"total"`
	Returned         int                `json:"returned"`
	Truncated        bool               `json:"truncated"`
	CriteriaIncluded bool               `json:"criteria_included"`
}

type ArticleInfoFact struct {
	SourceArticleIDs []int  `json:"source_article_ids,omitempty"`
	ArticleID        int    `json:"article_id"`
	Type             string `json:"type,omitempty"`
	Text             string `json:"text"`
	Sort             int    `json:"sort"`
	Display          bool   `json:"display"`
	TextID           int    `json:"text_id"`
}

type ArticleSupersessionFact struct {
	SourceArticleIDs []int  `json:"source_article_ids,omitempty"`
	ArticleID        int    `json:"article_id"`
	Direction        string `json:"direction"` // superseded_by or supersedes
	RelatedArticleID *int   `json:"related_article_id,omitempty"`
	ArticleNumber    string `json:"article_number"`
	SupplierID       *int   `json:"supplier_id,omitempty"`
	Brand            string `json:"brand,omitempty"`
}

type ArticleCountryMetadata struct {
	SourceArticleIDs []int  `json:"source_article_ids,omitempty"`
	PackUnit         *int   `json:"pack_unit,omitempty"`
	QuantityPerUnit  *int   `json:"quantity_per_unit,omitempty"`
	StatusDate       string `json:"status_date,omitempty"`
	Status           string `json:"status,omitempty"`
}

type EnrichmentBatchResponse struct {
	LanguageID int                     `json:"language_id"`
	CountryID  int                     `json:"country_id"`
	Results    []EnrichmentBatchResult `json:"results"`
	Total      int                     `json:"total"`
}
