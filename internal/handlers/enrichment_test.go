package handlers

import (
	"testing"

	"go-tecdoc-api/internal/models"
)

func TestResolveCommercialIdentityAcceptsMultipleTechnicalArticleIDsForOneSupplier(t *testing.T) {
	supplierID := 72
	item := models.EnrichmentBatchItem{RequestID: "p1", ArticleNumber: "50390266", SupplierID: &supplierID}
	status, resolved, ids, ambiguous := resolveCommercialIdentity(item, []models.EnrichmentResolvedArticle{
		{ArticleID: 3575879, ArticleNumber: "50390266", SupplierID: 72, Brand: "HERTH+BUSS ELPARTS"},
		{ArticleID: 3575880, ArticleNumber: "50390266", SupplierID: 72, Brand: "HERTH+BUSS ELPARTS"},
	})
	if status != "exact" || resolved == nil || resolved.ArticleID != 3575879 {
		t.Fatalf("status=%q resolved=%#v", status, resolved)
	}
	if len(ids) != 2 || ids[0] != 3575879 || ids[1] != 3575880 {
		t.Fatalf("ids=%#v", ids)
	}
	if len(ambiguous) != 0 {
		t.Fatalf("ambiguous=%#v", ambiguous)
	}
}

func TestResolveCommercialIdentityDoesNotGuessAcrossSuppliers(t *testing.T) {
	item := models.EnrichmentBatchItem{RequestID: "p1", ArticleNumber: "ABC123"}
	status, resolved, ids, ambiguous := resolveCommercialIdentity(item, []models.EnrichmentResolvedArticle{
		{ArticleID: 10, ArticleNumber: "ABC123", SupplierID: 1, Brand: "A"},
		{ArticleID: 20, ArticleNumber: "ABC123", SupplierID: 2, Brand: "B"},
	})
	if status != "ambiguous" || resolved != nil || len(ids) != 0 || len(ambiguous) != 2 {
		t.Fatalf("status=%q resolved=%#v ids=%#v ambiguous=%#v", status, resolved, ids, ambiguous)
	}
}

func TestNormalizeIncludeDefaultsToLegacyEnrichmentSections(t *testing.T) {
	include, err := normalizeInclude(nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"classification", "oem", "cross_references", "criteria", "applicability"} {
		if !include[key] {
			t.Fatalf("%s was not enabled: %#v", key, include)
		}
	}
	for _, key := range []string{"barcodes", "trade_numbers", "media", "components", "accessories", "article_info", "supersessions", "country_metadata"} {
		if include[key] {
			t.Fatalf("extended section %s must require explicit include: %#v", key, include)
		}
	}
}

func TestUniqueArticleIDsDeduplicatesAndSorts(t *testing.T) {
	got := uniqueArticleIDs([]int{5, 2, 5, 0, -1, 3})
	want := []int{2, 3, 5}
	if len(got) != len(want) {
		t.Fatalf("got=%#v want=%#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got=%#v want=%#v", got, want)
		}
	}
}

func TestResolveCommercialIdentityAcceptsExplicitBrandAlias(t *testing.T) {
	item := models.EnrichmentBatchItem{
		RequestID:     "p1",
		ArticleNumber: "WK8422",
		Brand:         "MANN",
		BrandAliases:  []string{"MANN-FILTER"},
	}
	status, resolved, ids, ambiguous := resolveCommercialIdentity(item, []models.EnrichmentResolvedArticle{
		{ArticleID: 8422, ArticleNumber: "WK 842/2", SupplierID: 4, Brand: "MANN-FILTER"},
	})
	if status != "exact" || resolved == nil || resolved.SupplierID != 4 || len(ids) != 1 || len(ambiguous) != 0 {
		t.Fatalf("status=%q resolved=%#v ids=%#v ambiguous=%#v", status, resolved, ids, ambiguous)
	}
}

func TestResolveCommercialIdentityKeepsAliasCollisionAmbiguous(t *testing.T) {
	item := models.EnrichmentBatchItem{
		RequestID:     "p1",
		ArticleNumber: "ABC123",
		Brand:         "LOCAL BRAND",
		BrandAliases:  []string{"BRAND-A", "BRAND-B"},
	}
	status, resolved, ids, ambiguous := resolveCommercialIdentity(item, []models.EnrichmentResolvedArticle{
		{ArticleID: 10, ArticleNumber: "ABC123", SupplierID: 1, Brand: "BRAND-A"},
		{ArticleID: 20, ArticleNumber: "ABC123", SupplierID: 2, Brand: "BRAND-B"},
	})
	if status != "ambiguous" || resolved != nil || len(ids) != 0 || len(ambiguous) != 2 {
		t.Fatalf("status=%q resolved=%#v ids=%#v ambiguous=%#v", status, resolved, ids, ambiguous)
	}
}

func TestMergeCriteriaKeepsSourceArticleProvenanceAndRepeatedValues(t *testing.T) {
	data := map[int][]models.ArticleCriterion{
		101: {
			{CriteriaID: 209, RawValue: "79", ValueDescription: "79", Display: true},
			{CriteriaID: 209, RawValue: "80", ValueDescription: "80", Display: true},
		},
		202: {
			{CriteriaID: 209, RawValue: "79", ValueDescription: "79", Display: true},
		},
	}
	got := mergeCriteria([]int{101, 202}, data)
	if len(got) != 2 {
		t.Fatalf("criteria=%#v, want two distinct repeated CRI_ID values", got)
	}
	var shared *models.ArticleCriterion
	for i := range got {
		if got[i].RawValue == "79" {
			shared = &got[i]
		}
	}
	if shared == nil || len(shared.SourceArticleIDs) != 2 || shared.SourceArticleIDs[0] != 101 || shared.SourceArticleIDs[1] != 202 {
		t.Fatalf("shared criterion provenance=%#v, want ART_IDs 101 and 202", shared)
	}
}

func TestMergeCriteriaUsesStableValueDescriptionIDAcrossLocalization(t *testing.T) {
	data := map[int][]models.ArticleCriterion{
		101: {{CriteriaID: 500, ValueDescriptionID: 7001, ValueDescription: "Left", Type: "KeyValue", Display: true}},
		202: {{CriteriaID: 500, ValueDescriptionID: 7001, ValueDescription: "Links", Type: "KeyValue", Display: true}},
	}
	got := mergeCriteria([]int{101, 202}, data)
	if len(got) != 1 {
		t.Fatalf("criteria=%#v, want one stable ACR_DES_ID identity despite localized text", got)
	}
	if got[0].ValueDescriptionID != 7001 || len(got[0].SourceArticleIDs) != 2 {
		t.Fatalf("criterion=%#v, want description id 7001 with both source articles", got[0])
	}
}

func TestMergeCriteriaDoesNotCollapseDisplayOrUnitSemantics(t *testing.T) {
	data := map[int][]models.ArticleCriterion{
		101: {{CriteriaID: 209, RawValue: "79", ValueDescription: "79", Type: "Numerical", Unit: "mm", Display: true}},
		202: {{CriteriaID: 209, RawValue: "79", ValueDescription: "79", Type: "Numerical", Unit: "mm", Display: false}},
		303: {{CriteriaID: 209, RawValue: "79", ValueDescription: "79", Type: "Numerical", Unit: "cm", Display: true}},
	}
	got := mergeCriteria([]int{101, 202, 303}, data)
	if len(got) != 3 {
		t.Fatalf("criteria=%#v, want distinct rows for display/unit semantics", got)
	}
}

func TestMergeOEMNumbersKeepsAllSourceArticleIDs(t *testing.T) {
	data := map[int][]models.OEMNumber{
		101: {{ManufacturerID: 5, Number: "03L 115 562", Manufacturer: "AUDI", Brand: "AUDI"}},
		202: {{ManufacturerID: 5, Number: "03L115562", Manufacturer: "AUDI", Brand: "AUDI"}},
	}
	got := mergeOEMNumbers([]int{101, 202}, data)
	if len(got) != 1 {
		t.Fatalf("OEM rows=%#v, want one normalized row", got)
	}
	if len(got[0].SourceArticleIDs) != 2 || got[0].SourceArticleIDs[0] != 101 || got[0].SourceArticleIDs[1] != 202 {
		t.Fatalf("OEM provenance=%#v, want ART_IDs 101 and 202", got[0].SourceArticleIDs)
	}
}

func TestMergeApplicabilityMarksMergedLinkageContributors(t *testing.T) {
	data := map[int]models.ApplicabilityLinkagesPage{
		101: {
			Linkages: []models.ApplicabilityLinkage{{
				LinkageID: 10, ProductType: 20, TargetType: "PC",
				Targets: []models.ApplicabilityTarget{{Type: "PC", ID: 1}},
			}},
			TotalTargetsExact: true,
		},
		202: {
			Linkages: []models.ApplicabilityLinkage{{
				LinkageID: 10, ProductType: 20, TargetType: "PC",
				Targets: []models.ApplicabilityTarget{{Type: "PC", ID: 2}},
			}},
			TotalTargetsExact: true,
		},
	}
	got := mergeApplicability([]int{101, 202}, data)
	if len(got.Linkages) != 1 {
		t.Fatalf("applicability=%#v, want one merged batch linkage", got)
	}
	if len(got.Linkages[0].SourceArticleIDs) != 2 || got.Linkages[0].SourceArticleIDs[0] != 101 || got.Linkages[0].SourceArticleIDs[1] != 202 {
		t.Fatalf("linkage contributors=%#v, want ART_IDs 101 and 202", got.Linkages[0].SourceArticleIDs)
	}
	if len(got.Linkages[0].Targets) != 2 {
		t.Fatalf("merged targets=%#v, want both batch targets", got.Linkages[0].Targets)
	}
}

func TestEnrichmentBrandNamesAcceptCanonicalBrandWithoutAliases(t *testing.T) {
	item := models.EnrichmentBatchItem{ArticleNumber: "1900", Brand: "BOSCH"}
	got := enrichmentBrandNames(item)
	if len(got) != 1 || got[0] != "BOSCH" {
		t.Fatalf("brand identity=%#v, want canonical brand", got)
	}
}

func TestHasEnrichmentSupplierIdentity(t *testing.T) {
	brandOnly := models.EnrichmentBatchItem{ArticleNumber: "1900", Brand: "BOSCH"}
	if !hasEnrichmentSupplierIdentity(brandOnly) {
		t.Fatal("canonical brand must satisfy exact enrichment supplier identity")
	}
	numberOnly := models.EnrichmentBatchItem{ArticleNumber: "1900"}
	if hasEnrichmentSupplierIdentity(numberOnly) {
		t.Fatal("number-only enrichment lookup must remain unsafe")
	}
	emptyAliases := models.EnrichmentBatchItem{ArticleNumber: "1900", BrandAliases: []string{"", "   "}}
	if hasEnrichmentSupplierIdentity(emptyAliases) {
		t.Fatal("blank brand aliases must not make a number-only lookup safe")
	}
}

func TestNormalizeIncludeSupportsExtendedFacts(t *testing.T) {
	include, err := normalizeInclude([]string{"barcodes", "trade_numbers", "media", "components", "accessories", "article_info", "supersessions", "country_metadata"})
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"barcodes", "trade_numbers", "media", "components", "accessories", "article_info", "supersessions", "country_metadata"} {
		if !include[key] {
			t.Fatalf("%s was not enabled: %#v", key, include)
		}
	}
}

func TestMergeLookupNumbersPreservesTechnicalArticleProvenance(t *testing.T) {
	data := map[int][]models.ArticleLookupNumber{
		101: {{Number: "4006381333931", Type: "EAN"}},
		202: {{Number: "4006381333931", Type: "EAN"}},
	}
	got := mergeLookupNumbers([]int{101, 202}, data)
	if len(got) != 1 || len(got[0].SourceArticleIDs) != 2 || got[0].SourceArticleIDs[0] != 101 || got[0].SourceArticleIDs[1] != 202 {
		t.Fatalf("got=%#v", got)
	}
}

func TestMergeMediaIdentityUsesMediaTypeAndSort(t *testing.T) {
	data := map[int][]models.ArticleMediaFact{101: {
		{MediaID: 7, MediaType: "JPG", Sort: 1, SupplierID: 4, URL: "IMAGE/4/a.jpg"},
		{MediaID: 7, MediaType: "PDF", Sort: 1, SupplierID: 4, URL: "PDF/4/a.pdf"},
	}}
	got := mergeMediaFacts([]int{101}, data)
	if len(got) != 2 {
		t.Fatalf("got=%#v", got)
	}
}

func TestMergeAccessoryPagesPreservesParentArticleProvenance(t *testing.T) {
	data := map[int]models.ArticleAccessoriesPage{
		101: {Items: []models.ArticleAccessory{{ArticleID: 404, ArticleNr: "ACC1", SupplierID: 10, RelationType: "PC", RelationTypeID: 1, ProductTypeID: 55, Sort: 2, Quantity: 1}}, Total: 1, Returned: 1},
		202: {Items: []models.ArticleAccessory{{ArticleID: 404, ArticleNr: "ACC1", SupplierID: 10, RelationType: "PC", RelationTypeID: 1, ProductTypeID: 55, Sort: 2, Quantity: 1}}, Total: 1, Returned: 1},
	}
	got := mergeAccessoryPages([]int{101, 202}, data)
	if len(got.Items) != 1 || len(got.Items[0].SourceArticleIDs) != 2 || got.Truncated || got.Total != 1 || got.Returned != 1 {
		t.Fatalf("got=%#v", got)
	}
}

func TestMergeCountrySpecificFactsDoesNotHideConflicts(t *testing.T) {
	one, two, three := 1, 2, 3
	data := map[int]models.ArticleCountryMetadata{
		101: {PackUnit: &one, QuantityPerUnit: &two, Status: "Normal", StatusDate: "2024-01-01"},
		202: {PackUnit: &one, QuantityPerUnit: &two, Status: "Normal", StatusDate: "2024-01-01"},
		303: {PackUnit: &one, QuantityPerUnit: &three, Status: "Normal", StatusDate: "2024-01-01"},
	}
	got := mergeCountrySpecificFacts([]int{101, 202, 303}, data)
	if len(got) != 2 {
		t.Fatalf("got=%#v, want two distinct variants", got)
	}
	if len(got[0].SourceArticleIDs) != 2 {
		t.Fatalf("first provenance=%#v", got[0].SourceArticleIDs)
	}
}

func TestMergeAccessoryPagesDoesNotCollapseDifferentGroups(t *testing.T) {
	data := map[int]models.ArticleAccessoriesPage{101: {Items: []models.ArticleAccessory{
		{ArticleID: 200, ArticleNr: "A", RelationType: "Accessory", RelationTypeID: 9, ProductTypeID: 10, Sort: 1, Quantity: 1, GroupName: "Mounting"},
		{ArticleID: 200, ArticleNr: "A", RelationType: "Accessory", RelationTypeID: 9, ProductTypeID: 10, Sort: 1, Quantity: 1, GroupName: "Service"},
	}, Total: 2, Returned: 2, CriteriaIncluded: true}}
	got := mergeAccessoryPages([]int{101}, data)
	if len(got.Items) != 2 {
		t.Fatalf("accessories=%#v, want both group variants", got.Items)
	}
}

func TestMergeAccessoryPagesDoesNotCollapseDifferentCriteriaVariants(t *testing.T) {
	data := map[int]models.ArticleAccessoriesPage{
		101: {Items: []models.ArticleAccessory{{ArticleID: 404, ArticleNr: "ACC1", SupplierID: 10, Brand: "ATE", RelationType: "PC", RelationTypeID: 1, ProductTypeID: 55, Sort: 2, Quantity: 1, CriteriaDetails: []models.ArticleCriterion{{CriteriaID: 100, RawValue: "Rear axle"}}}}, Total: 1, Returned: 1},
		202: {Items: []models.ArticleAccessory{{ArticleID: 404, ArticleNr: "ACC1", SupplierID: 10, Brand: "ATE", RelationType: "PC", RelationTypeID: 1, ProductTypeID: 55, Sort: 2, Quantity: 1, CriteriaDetails: []models.ArticleCriterion{{CriteriaID: 100, RawValue: "Front axle"}}}}, Total: 1, Returned: 1},
	}
	got := mergeAccessoryPages([]int{101, 202}, data)
	if len(got.Items) != 2 {
		t.Fatalf("accessories=%#v, want both criteria variants preserved", got.Items)
	}
}

func TestMergeOEMNumbersDoesNotCollapseDifferentFallbackManufacturers(t *testing.T) {
	data := map[int][]models.OEMNumber{
		101: {{ManufacturerID: 0, Number: "31210 05020", Manufacturer: "TOYOTA", Brand: "TOYOTA"}},
		202: {{ManufacturerID: 0, Number: "3121005020", Manufacturer: "LEXUS", Brand: "LEXUS"}},
	}
	got := mergeOEMNumbers([]int{101, 202}, data)
	if len(got) != 2 {
		t.Fatalf("OEM rows=%#v, want different fallback manufacturers preserved", got)
	}
}

func TestMergeCrossReferencesDoesNotCollapseDifferentFallbackBrands(t *testing.T) {
	data := map[int][]models.CrossReference{
		101: {{ArticleNr: "ABC 123", Brand: "TOYOTA", Manufacturer: "TOYOTA", Type: "OEM"}},
		202: {{ArticleNr: "ABC123", Brand: "LEXUS", Manufacturer: "LEXUS", Type: "OEM"}},
	}
	got := mergeCrossReferences([]int{101, 202}, data)
	if len(got) != 2 {
		t.Fatalf("cross rows=%#v, want different fallback brands preserved", got)
	}
}

func TestMergeCrossReferencesPreservesTechnicalArticleProvenance(t *testing.T) {
	data := map[int][]models.CrossReference{
		101: {{ArticleNr: "ABC 123", Brand: "TOYOTA", Manufacturer: "TOYOTA", Type: "OEM"}},
		202: {{ArticleNr: "ABC123", Brand: "TOYOTA", Manufacturer: "TOYOTA", Type: "OEM"}},
	}
	got := mergeCrossReferences([]int{101, 202}, data)
	if len(got) != 1 || len(got[0].SourceArticleIDs) != 2 || got[0].SourceArticleIDs[0] != 101 || got[0].SourceArticleIDs[1] != 202 {
		t.Fatalf("cross rows=%#v, want one factual row with both ART_ID contributors", got)
	}
}

func TestMergeMediaIdentityKeepsLanguageVariants(t *testing.T) {
	data := map[int][]models.ArticleMediaFact{101: {
		{MediaID: 7, MediaType: "PDF", Sort: 1, SupplierID: 4, LanguageID: 4, FileName: "manual-en.pdf", URL: "PDF/4/manual-en.pdf"},
		{MediaID: 7, MediaType: "PDF", Sort: 1, SupplierID: 4, LanguageID: 48, FileName: "manual-uk.pdf", URL: "PDF/4/manual-uk.pdf"},
	}}
	got := mergeMediaFacts([]int{101}, data)
	if len(got) != 2 {
		t.Fatalf("media=%#v, want both language variants", got)
	}
}

func TestMergeMediaIdentityKeepsContentTypeVariants(t *testing.T) {
	data := map[int][]models.ArticleMediaFact{101: {
		{MediaID: 7, MediaType: "JPG", Sort: 1, SupplierID: 4, LanguageID: 4, ContentType: "ArticlePicture", FileName: "same.jpg", URL: "IMAGE/4/same.jpg"},
		{MediaID: 7, MediaType: "JPG", Sort: 1, SupplierID: 4, LanguageID: 4, ContentType: "TechnicalDrawing", FileName: "same.jpg", URL: "IMAGE/4/same.jpg"},
	}}
	got := mergeMediaFacts([]int{101}, data)
	if len(got) != 2 {
		t.Fatalf("media=%#v, want both content-type variants", got)
	}
}

func TestMergeArticleInfoDoesNotCollapseDisplayOrSortVariants(t *testing.T) {
	data := map[int][]models.ArticleInfoFact{
		101: {{TextID: 9001, Type: "Info", Text: "Same text", Sort: 1, Display: true}},
		202: {{TextID: 9001, Type: "Info", Text: "Same text", Sort: 2, Display: false}},
	}
	got := mergeArticleInfoFacts([]int{101, 202}, data)
	if len(got) != 2 {
		t.Fatalf("article info=%#v, want display/sort variants preserved", got)
	}
}

func TestResolveReviewedCommercialIdentityReusesMultipleTechnicalArticleIDs(t *testing.T) {
	supplierID := 72
	item := models.EnrichmentBatchItem{
		RequestID:     "p1",
		ArticleIDs:    []int{3575880, 3575879},
		ArticleNumber: "50 390 266",
		SupplierID:    &supplierID,
	}
	status, resolved, ids, message := resolveReviewedCommercialIdentity(item, map[int]models.EnrichmentResolvedArticle{
		3575879: {ArticleID: 3575879, ArticleNumber: "50390266", SupplierID: 72, Brand: "HERTH+BUSS ELPARTS"},
		3575880: {ArticleID: 3575880, ArticleNumber: "50-390-266", SupplierID: 72, Brand: "HERTH+BUSS ELPARTS"},
	})
	if status != "exact" || resolved == nil || resolved.ArticleID != 3575879 || message != "" {
		t.Fatalf("status=%q resolved=%#v message=%q", status, resolved, message)
	}
	if len(ids) != 2 || ids[0] != 3575879 || ids[1] != 3575880 {
		t.Fatalf("ids=%#v", ids)
	}
}

func TestResolveReviewedCommercialIdentityRejectsChangedCommercialIdentity(t *testing.T) {
	supplierID := 72
	item := models.EnrichmentBatchItem{
		ArticleIDs:    []int{10, 20},
		ArticleNumber: "ABC123",
		SupplierID:    &supplierID,
	}
	status, resolved, ids, message := resolveReviewedCommercialIdentity(item, map[int]models.EnrichmentResolvedArticle{
		10: {ArticleID: 10, ArticleNumber: "ABC123", SupplierID: 72},
		20: {ArticleID: 20, ArticleNumber: "OTHER", SupplierID: 72},
	})
	if status != "error" || resolved != nil || len(ids) != 0 || message == "" {
		t.Fatalf("status=%q resolved=%#v ids=%#v message=%q", status, resolved, ids, message)
	}
}

func TestResolveReviewedCommercialIdentityRejectsMissingArticleID(t *testing.T) {
	supplierID := 72
	item := models.EnrichmentBatchItem{ArticleIDs: []int{10, 20}, ArticleNumber: "ABC123", SupplierID: &supplierID}
	status, resolved, ids, message := resolveReviewedCommercialIdentity(item, map[int]models.EnrichmentResolvedArticle{
		10: {ArticleID: 10, ArticleNumber: "ABC123", SupplierID: 72},
	})
	if status != "not_found" || resolved != nil || len(ids) != 0 || message != "" {
		t.Fatalf("status=%q resolved=%#v ids=%#v message=%q", status, resolved, ids, message)
	}
}

func TestMergeMediaKeepsCountryScopeVariantsForDownstreamReconciliation(t *testing.T) {
	data := map[int][]models.ArticleMediaFact{101: {
		{MediaID: 7, MediaType: "JPG", Sort: 1, SupplierID: 4, CountryMarker: 0, LanguageID: 4, ContentType: "ArticlePicture", FileName: "same.jpg", URL: "IMAGE/4/same.jpg"},
		{MediaID: 7, MediaType: "JPG", Sort: 1, SupplierID: 4, CountryMarker: 123, LanguageID: 4, ContentType: "ArticlePicture", FileName: "same.jpg", URL: "IMAGE/4/same.jpg"},
	}}
	got := mergeMediaFacts([]int{101}, data)
	if len(got) != 2 {
		t.Fatalf("media=%#v, want both applicable country-scope rows preserved", got)
	}
}
