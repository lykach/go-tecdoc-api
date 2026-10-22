package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"go-tecdoc-api/internal/database"
	"go-tecdoc-api/internal/models"
)

const (
	maxEnrichmentBatchItems            = 50
	maxEnrichmentBrandAliases          = 32
	maxTechnicalArticleIDsPerItem      = 200
	maxTechnicalArticleIDsPerBatch     = 1000
	defaultBatchApplicabilityTargets   = 1000
	maxBatchApplicabilityTargets       = 5000
	defaultBatchAccessories            = 100
	maxBatchAccessories                = 500
	maxEnrichmentBatchRequestBodyBytes = 1 << 20 // 1 MiB
	enrichmentQueryTimeout             = 25 * time.Second
)

func isSupportedApplicabilityTargetType(value string) bool {
	switch value {
	case "PC", "CV", "Motorcycle", "Engine", "Axle":
		return true
	default:
		return false
	}
}

func normalizeInclude(values []string) (map[string]bool, error) {
	allowed := map[string]bool{
		"classification":   false,
		"oem":              false,
		"cross_references": false,
		"criteria":         false,
		"applicability":    false,
		"barcodes":         false,
		"trade_numbers":    false,
		"media":            false,
		"components":       false,
		"accessories":      false,
		"article_info":     false,
		"supersessions":    false,
		"country_metadata": false,
	}
	if len(values) == 0 {
		// Preserve the pre-Stage-4 batch contract. Extended facts can be much
		// heavier (media/components/accessories) and must be explicitly requested.
		for _, key := range []string{"classification", "oem", "cross_references", "criteria", "applicability"} {
			allowed[key] = true
		}
		return allowed, nil
	}
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if _, ok := allowed[value]; !ok {
			return nil, fmt.Errorf("unsupported include value %q", value)
		}
		allowed[value] = true
	}
	return allowed, nil
}

func uniqueArticleIDs(values []int) []int {
	seen := make(map[int]struct{}, len(values))
	out := make([]int, 0, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Ints(out)
	return out
}

func compactCandidates(items []models.EnrichmentResolvedArticle) []models.EnrichmentResolvedArticle {
	seen := make(map[string]struct{}, len(items))
	out := make([]models.EnrichmentResolvedArticle, 0, len(items))
	for _, item := range items {
		key := fmt.Sprintf("%d|%s", item.SupplierID, database.NormalizeSearchNumber(item.ArticleNumber))
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, item)
	}
	return out
}

func hasEnrichmentSupplierIdentity(item models.EnrichmentBatchItem) bool {
	if item.SupplierID != nil || strings.TrimSpace(item.Brand) != "" {
		return true
	}
	for _, alias := range item.BrandAliases {
		if strings.TrimSpace(alias) != "" {
			return true
		}
	}
	return false
}

func enrichmentBrandNames(item models.EnrichmentBatchItem) []string {
	values := make([]string, 0, 1+len(item.BrandAliases))
	values = append(values, item.Brand)
	values = append(values, item.BrandAliases...)
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, raw := range values {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		key := strings.ToUpper(value)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	return out
}

func resolveCommercialIdentity(item models.EnrichmentBatchItem, candidates []models.EnrichmentResolvedArticle) (string, *models.EnrichmentResolvedArticle, []int, []models.EnrichmentResolvedArticle) {
	acceptedBrands := enrichmentBrandNames(item)
	filtered := make([]models.EnrichmentResolvedArticle, 0, len(candidates))
	for _, candidate := range candidates {
		if item.SupplierID != nil {
			if candidate.SupplierID != *item.SupplierID {
				continue
			}
		} else if len(acceptedBrands) > 0 {
			matched := false
			for _, brand := range acceptedBrands {
				if strings.EqualFold(strings.TrimSpace(brand), strings.TrimSpace(candidate.Brand)) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		filtered = append(filtered, candidate)
	}
	if len(filtered) == 0 {
		return "not_found", nil, nil, nil
	}

	bySupplier := make(map[int][]models.EnrichmentResolvedArticle)
	for _, candidate := range filtered {
		bySupplier[candidate.SupplierID] = append(bySupplier[candidate.SupplierID], candidate)
	}
	if len(bySupplier) != 1 {
		return "ambiguous", nil, nil, compactCandidates(filtered)
	}

	var group []models.EnrichmentResolvedArticle
	for _, values := range bySupplier {
		group = values
	}
	sort.Slice(group, func(i, j int) bool { return group[i].ArticleID < group[j].ArticleID })
	articleIDs := make([]int, 0, len(group))
	for _, candidate := range group {
		articleIDs = append(articleIDs, candidate.ArticleID)
	}
	resolved := group[0]
	return "exact", &resolved, articleIDs, nil
}

func resolveReviewedCommercialIdentity(item models.EnrichmentBatchItem, identities map[int]models.EnrichmentResolvedArticle) (string, *models.EnrichmentResolvedArticle, []int, string) {
	ids := uniqueArticleIDs(item.ArticleIDs)
	if len(ids) == 0 || item.SupplierID == nil || *item.SupplierID <= 0 {
		return "error", nil, nil, "reviewed article_ids requires supplier/article identity assertions"
	}
	expectedNumber := database.NormalizeSearchNumber(item.ArticleNumber)
	if expectedNumber == "" {
		return "error", nil, nil, "reviewed article_ids requires supplier/article identity assertions"
	}
	validated := make([]models.EnrichmentResolvedArticle, 0, len(ids))
	for _, articleID := range ids {
		identity, ok := identities[articleID]
		if !ok {
			return "not_found", nil, nil, ""
		}
		if identity.SupplierID != *item.SupplierID || database.NormalizeSearchNumber(identity.ArticleNumber) != expectedNumber {
			return "error", nil, nil, "reviewed article_ids no longer match the asserted supplier/article commercial identity"
		}
		validated = append(validated, identity)
	}
	sort.Slice(validated, func(i, j int) bool { return validated[i].ArticleID < validated[j].ArticleID })
	resolved := validated[0]
	return "exact", &resolved, ids, ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func appendUniquePositiveInt(values []int, value int) []int {
	if value <= 0 {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func mergeClassifications(articleIDs []int, data map[int][]models.ArticleProductTypeClassification) []models.ArticleProductTypeClassification {
	byPT := make(map[int]*models.ArticleProductTypeClassification)
	treeSeen := make(map[int]map[string]struct{})
	scopeSeen := make(map[int]map[string]struct{})
	for _, articleID := range articleIDs {
		for _, item := range data[articleID] {
			current := byPT[item.ProductTypeID]
			if current == nil {
				copyItem := item
				copyItem.SourceArticleIDs = []int{articleID}
				copyItem.Scopes = []string{}
				copyItem.TreeNodes = []models.ArticleClassificationTreeNode{}
				current = &copyItem
				byPT[item.ProductTypeID] = current
				treeSeen[item.ProductTypeID] = make(map[string]struct{})
				scopeSeen[item.ProductTypeID] = make(map[string]struct{})
			} else {
				current.SourceArticleIDs = appendUniquePositiveInt(current.SourceArticleIDs, articleID)
			}
			if item.LinkageCount > current.LinkageCount {
				current.LinkageCount = item.LinkageCount
			}
			if item.TargetCount > current.TargetCount {
				current.TargetCount = item.TargetCount
			}
			for _, scope := range item.Scopes {
				if _, ok := scopeSeen[item.ProductTypeID][scope]; ok {
					continue
				}
				scopeSeen[item.ProductTypeID][scope] = struct{}{}
				current.Scopes = append(current.Scopes, scope)
			}
			for _, node := range item.TreeNodes {
				key := fmt.Sprintf("%d|%s", node.ID, node.RawType)
				if _, ok := treeSeen[item.ProductTypeID][key]; ok {
					continue
				}
				treeSeen[item.ProductTypeID][key] = struct{}{}
				current.TreeNodes = append(current.TreeNodes, node)
			}
		}
	}
	ptIDs := make([]int, 0, len(byPT))
	for ptID := range byPT {
		ptIDs = append(ptIDs, ptID)
	}
	sort.Ints(ptIDs)
	out := make([]models.ArticleProductTypeClassification, 0, len(ptIDs))
	for _, ptID := range ptIDs {
		sort.Strings(byPT[ptID].Scopes)
		out = append(out, *byPT[ptID])
	}
	return out
}

func mergeOEMNumbers(articleIDs []int, data map[int][]models.OEMNumber) []models.OEMNumber {
	indexes := make(map[string]int)
	out := make([]models.OEMNumber, 0)
	for _, articleID := range articleIDs {
		for _, item := range data[articleID] {
			manufacturerKey := fmt.Sprintf("id:%d", item.ManufacturerID)
			if item.ManufacturerID <= 0 {
				manufacturerKey = "name:" + strings.ToLower(strings.TrimSpace(firstNonEmpty(item.Manufacturer, item.Brand)))
			}
			key := manufacturerKey + "|" + database.NormalizeSearchNumber(item.Number)
			if idx, ok := indexes[key]; ok {
				out[idx].SourceArticleIDs = appendUniquePositiveInt(out[idx].SourceArticleIDs, articleID)
				continue
			}
			item.SourceArticleIDs = []int{articleID}
			indexes[key] = len(out)
			out = append(out, item)
		}
	}
	return out
}

func mergeCrossReferences(articleIDs []int, data map[int][]models.CrossReference) []models.CrossReference {
	indexes := make(map[string]int)
	out := make([]models.CrossReference, 0)
	for _, articleID := range articleIDs {
		for _, item := range data[articleID] {
			linkedID := 0
			if item.ArticleID != nil {
				linkedID = *item.ArticleID
			}
			supplierID := 0
			if item.SupplierID != nil {
				supplierID = *item.SupplierID
			}
			manufacturerID := 0
			if item.ManufacturerID != nil {
				manufacturerID = *item.ManufacturerID
			}
			partyKey := ""
			if strings.EqualFold(strings.TrimSpace(item.Type), "OEM") {
				if manufacturerID > 0 {
					partyKey = fmt.Sprintf("manufacturer:%d", manufacturerID)
				} else {
					partyKey = "manufacturer-name:" + strings.ToLower(strings.TrimSpace(firstNonEmpty(item.Manufacturer, item.Brand)))
				}
			} else if supplierID > 0 {
				partyKey = fmt.Sprintf("supplier:%d", supplierID)
			} else {
				partyKey = "supplier-name:" + strings.ToLower(strings.TrimSpace(firstNonEmpty(item.Supplier, item.Brand)))
			}
			key := fmt.Sprintf("%s|%d|%s|%s", item.Type, linkedID, partyKey, database.NormalizeSearchNumber(item.ArticleNr))
			if idx, ok := indexes[key]; ok {
				out[idx].SourceArticleIDs = appendUniquePositiveInt(out[idx].SourceArticleIDs, articleID)
				continue
			}
			item.SourceArticleIDs = []int{articleID}
			indexes[key] = len(out)
			out = append(out, item)
		}
	}
	return out
}

func mergeCriteria(articleIDs []int, data map[int][]models.ArticleCriterion) []models.ArticleCriterion {
	indexes := make(map[string]int)
	out := make([]models.ArticleCriterion, 0)
	for _, articleID := range articleIDs {
		for _, item := range data[articleID] {
			descriptionIdentity := strings.TrimSpace(item.ValueDescription)
			if item.ValueDescriptionID > 0 {
				descriptionIdentity = fmt.Sprintf("des:%d", item.ValueDescriptionID)
			}
			key := fmt.Sprintf("%d|%s|%s|%s|%s|%t", item.CriteriaID, item.RawValue, descriptionIdentity, item.Type, item.Unit, item.Display)
			if idx, ok := indexes[key]; ok {
				out[idx].SourceArticleIDs = appendUniquePositiveInt(out[idx].SourceArticleIDs, articleID)
				continue
			}
			item.SourceArticleIDs = []int{articleID}
			indexes[key] = len(out)
			out = append(out, item)
		}
	}
	return out
}

func mergeApplicability(articleIDs []int, data map[int]models.ApplicabilityLinkagesPage) models.ApplicabilityLinkagesPage {
	type key struct {
		laID       int
		ptID       int
		targetType string
	}
	out := models.ApplicabilityLinkagesPage{
		Linkages:          []models.ApplicabilityLinkage{},
		TotalTargetsExact: true,
	}
	indexes := make(map[key]int)
	targetSeen := make(map[key]map[int]struct{})
	for _, articleID := range articleIDs {
		page := data[articleID]
		if page.Truncated {
			out.Truncated = true
		}
		for _, linkage := range page.Linkages {
			k := key{laID: linkage.LinkageID, ptID: linkage.ProductType, targetType: linkage.TargetType}
			idx, ok := indexes[k]
			if !ok {
				idx = len(out.Linkages)
				indexes[k] = idx
				targetSeen[k] = make(map[int]struct{})
				copyLinkage := linkage
				copyLinkage.SourceArticleIDs = []int{articleID}
				copyLinkage.Targets = []models.ApplicabilityTarget{}
				out.Linkages = append(out.Linkages, copyLinkage)
			} else {
				out.Linkages[idx].SourceArticleIDs = appendUniquePositiveInt(out.Linkages[idx].SourceArticleIDs, articleID)
			}
			for _, target := range linkage.Targets {
				if _, ok := targetSeen[k][target.ID]; ok {
					continue
				}
				targetSeen[k][target.ID] = struct{}{}
				out.Linkages[idx].Targets = append(out.Linkages[idx].Targets, target)
				out.Returned++
			}
		}
	}
	// TotalTargets is an exact per-ART_ID count before merging. When multiple
	// technical ART_IDs form one commercial article, summing could double-count
	// shared LA targets, so expose at least the unique returned count and mark
	// truncated whenever any source ART_ID was truncated.
	out.TotalTargets = out.Returned
	if out.Truncated {
		for _, articleID := range articleIDs {
			if data[articleID].TotalTargets > out.TotalTargets {
				out.TotalTargets = data[articleID].TotalTargets
			}
		}
		if len(articleIDs) > 1 {
			out.TotalTargetsExact = false
		}
	}
	return out
}

func mergeLookupNumbers(articleIDs []int, data map[int][]models.ArticleLookupNumber) []models.ArticleLookupNumber {
	indexes := map[string]int{}
	out := make([]models.ArticleLookupNumber, 0)
	for _, articleID := range articleIDs {
		for _, item := range data[articleID] {
			key := item.Type + "|" + database.NormalizeSearchNumber(item.Number)
			if idx, ok := indexes[key]; ok {
				out[idx].SourceArticleIDs = appendUniquePositiveInt(out[idx].SourceArticleIDs, articleID)
				continue
			}
			item.SourceArticleIDs = []int{articleID}
			indexes[key] = len(out)
			out = append(out, item)
		}
	}
	return out
}

func mergeMediaFacts(articleIDs []int, data map[int][]models.ArticleMediaFact) []models.ArticleMediaFact {
	seen := map[string]int{}
	out := make([]models.ArticleMediaFact, 0)
	for _, articleID := range articleIDs {
		for _, item := range data[articleID] {
			// Keep payload variants separate so downstream enrichment can detect a
			// provider disagreement, but never collapse distinct TecDoc 2024Q1
			// content-type rows that share media/type/sort/language/file payload.
			key := fmt.Sprintf("%d|%s|%d|%d|%d|%d|%s|%s|%s", item.MediaID, strings.ToUpper(strings.TrimSpace(item.MediaType)), item.Sort, item.SupplierID, item.CountryMarker, item.LanguageID, strings.ToUpper(strings.TrimSpace(item.ContentType)), strings.TrimSpace(item.FileName), strings.TrimSpace(item.URL))
			if idx, ok := seen[key]; ok {
				out[idx].SourceArticleIDs = appendUniquePositiveInt(out[idx].SourceArticleIDs, articleID)
				continue
			}
			item.SourceArticleIDs = []int{articleID}
			seen[key] = len(out)
			out = append(out, item)
		}
	}
	return out
}

func mergeComponentFacts(articleIDs []int, data map[int][]models.ArticleComponentFact) []models.ArticleComponentFact {
	seen := map[string]int{}
	out := make([]models.ArticleComponentFact, 0)
	for _, parentID := range articleIDs {
		for _, item := range data[parentID] {
			key := fmt.Sprintf("%d|%d|%d|%d", item.ArticleID, item.Sequence, item.Quantity, item.ProductTypeID)
			if idx, ok := seen[key]; ok {
				out[idx].SourceArticleIDs = appendUniquePositiveInt(out[idx].SourceArticleIDs, parentID)
				continue
			}
			item.SourceArticleIDs = []int{parentID}
			seen[key] = len(out)
			out = append(out, item)
		}
	}
	return out
}

func articleCriteriaSignature(items []models.ArticleCriterion) string {
	if len(items) == 0 {
		return ""
	}
	parts := make([]string, 0, len(items))
	for _, item := range items {
		parts = append(parts, fmt.Sprintf("%d|%s|%s|%s|%s", item.CriteriaID, strings.TrimSpace(item.Type), strings.TrimSpace(item.RawValue), strings.TrimSpace(item.ValueDescription), strings.TrimSpace(item.Unit)))
	}
	sort.Strings(parts)
	return strings.Join(parts, "||")
}

func mergeAccessoryPages(articleIDs []int, data map[int]models.ArticleAccessoriesPage) models.ArticleAccessoriesPage {
	out := models.ArticleAccessoriesPage{Items: []models.ArticleAccessory{}, CriteriaIncluded: true}
	seen := map[string]int{}
	totalOccurrences := 0
	for _, id := range articleIDs {
		p := data[id]
		if !p.CriteriaIncluded {
			out.CriteriaIncluded = false
		}
		totalOccurrences += p.Total
		if p.Truncated {
			out.Truncated = true
		}
		for _, item := range p.Items {
			key := fmt.Sprintf("%d|%s|%d|%d|%d|%d|%s|%s", item.ArticleID, item.RelationType, item.RelationTypeID, item.ProductTypeID, item.Sort, item.Quantity, strings.TrimSpace(item.GroupName), articleCriteriaSignature(item.CriteriaDetails))
			if idx, ok := seen[key]; ok {
				out.Items[idx].SourceArticleIDs = appendUniquePositiveInt(out.Items[idx].SourceArticleIDs, id)
				continue
			}
			item.SourceArticleIDs = []int{id}
			seen[key] = len(out.Items)
			out.Items = append(out.Items, item)
		}
	}
	out.Returned = len(out.Items)
	if out.Truncated {
		// When any technical ART_ID was bounded, retain the source occurrence
		// total as evidence that the merged unique relation set is incomplete.
		out.Total = totalOccurrences
	} else {
		// Duplicates across technical ART_IDs are provenance, not truncation.
		out.Total = out.Returned
	}
	return out
}

func mergeArticleInfoFacts(articleIDs []int, data map[int][]models.ArticleInfoFact) []models.ArticleInfoFact {
	seen := map[string]int{}
	out := make([]models.ArticleInfoFact, 0)
	for _, id := range articleIDs {
		for _, item := range data[id] {
			key := fmt.Sprintf("%d|%s|%s|%d|%t", item.TextID, item.Type, item.Text, item.Sort, item.Display)
			if idx, ok := seen[key]; ok {
				out[idx].SourceArticleIDs = appendUniquePositiveInt(out[idx].SourceArticleIDs, id)
				continue
			}
			item.SourceArticleIDs = []int{id}
			seen[key] = len(out)
			out = append(out, item)
		}
	}
	return out
}

func mergeSupersessionFacts(articleIDs []int, data map[int][]models.ArticleSupersessionFact) []models.ArticleSupersessionFact {
	seen := map[string]int{}
	out := make([]models.ArticleSupersessionFact, 0)
	for _, id := range articleIDs {
		for _, item := range data[id] {
			related := 0
			if item.RelatedArticleID != nil {
				related = *item.RelatedArticleID
			}
			sid := 0
			if item.SupplierID != nil {
				sid = *item.SupplierID
			}
			brandKey := ""
			if sid <= 0 {
				brandKey = strings.ToLower(strings.TrimSpace(item.Brand))
			}
			key := fmt.Sprintf("%s|%d|%d|%s|%s", item.Direction, related, sid, brandKey, database.NormalizeSearchNumber(item.ArticleNumber))
			if idx, ok := seen[key]; ok {
				out[idx].SourceArticleIDs = appendUniquePositiveInt(out[idx].SourceArticleIDs, id)
				continue
			}
			item.SourceArticleIDs = []int{id}
			seen[key] = len(out)
			out = append(out, item)
		}
	}
	return out
}

func mergeCountrySpecificFacts(articleIDs []int, data map[int]models.ArticleCountryMetadata) []models.ArticleCountryMetadata {
	out := make([]models.ArticleCountryMetadata, 0, len(articleIDs))
	seen := map[string]int{}
	for _, id := range articleIDs {
		item, ok := data[id]
		if !ok {
			continue
		}
		pack, qty := "", ""
		if item.PackUnit != nil {
			pack = fmt.Sprintf("%d", *item.PackUnit)
		}
		if item.QuantityPerUnit != nil {
			qty = fmt.Sprintf("%d", *item.QuantityPerUnit)
		}
		key := pack + "|" + qty + "|" + item.StatusDate + "|" + item.Status
		if idx, exists := seen[key]; exists {
			out[idx].SourceArticleIDs = appendUniquePositiveInt(out[idx].SourceArticleIDs, id)
			continue
		}
		item.SourceArticleIDs = []int{id}
		seen[key] = len(out)
		out = append(out, item)
	}
	return out
}

// EnrichArticlesBatch resolves and enriches up to 50 commercial article
// identities. Expensive data is fetched with set-based DB methods after one
// resolution pass instead of issuing 5-7 HTTP/SQL calls per input product.
func (h *Handler) EnrichArticlesBatch(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxEnrichmentBatchRequestBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var request models.EnrichmentBatchRequest
	if err := decoder.Decode(&request); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid JSON body: "+err.Error())
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		h.respondError(w, http.StatusBadRequest, "Request body must contain exactly one JSON object")
		return
	}
	if len(request.Items) == 0 || len(request.Items) > maxEnrichmentBatchItems {
		h.respondError(w, http.StatusBadRequest, fmt.Sprintf("items must contain 1..%d entries", maxEnrichmentBatchItems))
		return
	}
	include, err := normalizeInclude(request.Include)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	languageID := request.LanguageID
	if languageID <= 0 {
		languageID = h.defaultLanguageID
	}
	countryID := request.CountryID
	if countryID <= 0 {
		countryID = h.defaultCountryID
	}
	maxTargets := request.MaxApplicabilityTargets
	if maxTargets <= 0 {
		maxTargets = defaultBatchApplicabilityTargets
	}
	if maxTargets > maxBatchApplicabilityTargets {
		maxTargets = maxBatchApplicabilityTargets
	}
	maxAccessories := request.MaxAccessories
	if maxAccessories <= 0 {
		maxAccessories = defaultBatchAccessories
	}
	if maxAccessories > maxBatchAccessories {
		maxAccessories = maxBatchAccessories
	}

	seenRequestIDs := make(map[string]struct{}, len(request.Items))
	directIDs := make([]int, 0)
	exactLookups := make([]database.ExactArticleBatchLookup, 0)
	for i := range request.Items {
		item := &request.Items[i]
		item.RequestID = strings.TrimSpace(item.RequestID)
		item.ArticleNumber = strings.TrimSpace(item.ArticleNumber)
		item.Brand = strings.TrimSpace(item.Brand)
		item.BrandAliases = enrichmentBrandNames(*item)
		if len(item.BrandAliases) > maxEnrichmentBrandAliases {
			h.respondError(w, http.StatusBadRequest, fmt.Sprintf("items[%d].brand_aliases must contain at most %d unique values", i, maxEnrichmentBrandAliases))
			return
		}
		for _, alias := range item.BrandAliases {
			if len(alias) > 255 {
				h.respondError(w, http.StatusBadRequest, fmt.Sprintf("items[%d].brand_aliases values must be <= 255 chars", i))
				return
			}
		}
		if item.RequestID == "" || len(item.RequestID) > 128 {
			h.respondError(w, http.StatusBadRequest, fmt.Sprintf("items[%d].request_id is required and must be <= 128 chars", i))
			return
		}
		if _, exists := seenRequestIDs[item.RequestID]; exists {
			h.respondError(w, http.StatusBadRequest, "request_id values must be unique")
			return
		}
		seenRequestIDs[item.RequestID] = struct{}{}
		reviewedIDs := uniqueArticleIDs(item.ArticleIDs)
		if len(item.ArticleIDs) > 0 && len(reviewedIDs) != len(item.ArticleIDs) {
			h.respondError(w, http.StatusBadRequest, fmt.Sprintf("items[%d].article_ids must contain unique positive IDs", i))
			return
		}
		if len(reviewedIDs) > maxTechnicalArticleIDsPerItem {
			h.respondError(w, http.StatusBadRequest, fmt.Sprintf("items[%d].article_ids must contain at most %d IDs", i, maxTechnicalArticleIDsPerItem))
			return
		}
		if item.ArticleID != nil && len(reviewedIDs) > 0 {
			h.respondError(w, http.StatusBadRequest, fmt.Sprintf("items[%d] must use article_id or article_ids, not both", i))
			return
		}
		if item.ArticleID != nil && item.ArticleNumber != "" {
			h.respondError(w, http.StatusBadRequest, fmt.Sprintf("items[%d] must use article_id or article_number, not both", i))
			return
		}
		if item.ArticleID == nil && len(reviewedIDs) == 0 && item.ArticleNumber == "" {
			h.respondError(w, http.StatusBadRequest, fmt.Sprintf("items[%d] requires article_id, article_ids, or article_number", i))
			return
		}
		if item.ArticleID != nil {
			if *item.ArticleID <= 0 {
				h.respondError(w, http.StatusBadRequest, fmt.Sprintf("items[%d].article_id must be positive", i))
				return
			}
			directIDs = append(directIDs, *item.ArticleID)
		} else if len(reviewedIDs) > 0 {
			if database.NormalizeSearchNumber(item.ArticleNumber) == "" {
				h.respondError(w, http.StatusBadRequest, fmt.Sprintf("items[%d].article_ids requires article_number as an identity assertion", i))
				return
			}
			if item.SupplierID == nil || *item.SupplierID <= 0 {
				h.respondError(w, http.StatusBadRequest, fmt.Sprintf("items[%d].article_ids requires a positive supplier_id identity assertion", i))
				return
			}
			item.ArticleIDs = reviewedIDs
			directIDs = append(directIDs, reviewedIDs...)
		} else {
			if database.NormalizeSearchNumber(item.ArticleNumber) == "" {
				h.respondError(w, http.StatusBadRequest, fmt.Sprintf("items[%d].article_number must contain at least one letter or digit", i))
				return
			}
			if !hasEnrichmentSupplierIdentity(*item) {
				h.respondError(w, http.StatusBadRequest, fmt.Sprintf("items[%d].article_number requires supplier_id, brand, or brand_aliases", i))
				return
			}
			exactLookups = append(exactLookups, database.ExactArticleBatchLookup{
				SearchNumber: item.ArticleNumber,
				SupplierID:   item.SupplierID,
				Brand:        item.Brand,
				BrandAliases: item.BrandAliases,
			})
		}
		if item.SupplierID != nil && *item.SupplierID <= 0 {
			h.respondError(w, http.StatusBadRequest, fmt.Sprintf("items[%d].supplier_id must be positive", i))
			return
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), enrichmentQueryTimeout)
	defer cancel()

	directIdentities, err := h.queries.GetArticleIdentitiesByIDsBatch(ctx, directIDs, languageID, countryID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to validate article IDs: "+err.Error())
		return
	}
	exactCandidates, err := h.queries.FindExactArticlesBatch(ctx, exactLookups, languageID, countryID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to resolve exact articles: "+err.Error())
		return
	}

	results := make([]models.EnrichmentBatchResult, len(request.Items))
	allArticleIDs := make([]int, 0, len(request.Items))
	for i, item := range request.Items {
		result := models.EnrichmentBatchResult{RequestID: item.RequestID}
		if item.ArticleID != nil {
			identity, ok := directIdentities[*item.ArticleID]
			if !ok {
				result.Status = "not_found"
				results[i] = result
				continue
			}
			result.Status = "exact"
			result.Resolved = &identity
			result.ArticleIDs = []int{identity.ArticleID}
			allArticleIDs = append(allArticleIDs, identity.ArticleID)
			results[i] = result
			continue
		}
		if len(item.ArticleIDs) > 0 {
			status, resolved, articleIDs, reviewedErr := resolveReviewedCommercialIdentity(item, directIdentities)
			result.Status = status
			result.Resolved = resolved
			result.ArticleIDs = articleIDs
			result.Error = reviewedErr
			if status == "exact" {
				allArticleIDs = append(allArticleIDs, articleIDs...)
			}
			results[i] = result
			continue
		}
		candidates := exactCandidates[database.NormalizeSearchNumber(item.ArticleNumber)]
		status, resolved, articleIDs, ambiguous := resolveCommercialIdentity(item, candidates)
		result.Status = status
		result.Resolved = resolved
		result.ArticleIDs = articleIDs
		result.Candidates = ambiguous
		if status == "exact" && len(articleIDs) > maxTechnicalArticleIDsPerItem {
			result.Status = "error"
			result.Error = fmt.Sprintf("resolved commercial identity exceeds %d technical article IDs", maxTechnicalArticleIDsPerItem)
			result.ArticleIDs = nil
			result.Resolved = nil
		} else if status == "exact" {
			allArticleIDs = append(allArticleIDs, articleIDs...)
		}
		results[i] = result
	}

	allArticleIDs = uniqueArticleIDs(allArticleIDs)
	if len(allArticleIDs) > maxTechnicalArticleIDsPerBatch {
		h.respondError(w, http.StatusUnprocessableEntity, fmt.Sprintf("resolved batch exceeds %d unique technical article IDs; split the request", maxTechnicalArticleIDsPerBatch))
		return
	}

	var classifications map[int][]models.ArticleProductTypeClassification
	var oemNumbers map[int][]models.OEMNumber
	var crossReferences map[int][]models.CrossReference
	var criteria map[int][]models.ArticleCriterion
	var applicability map[int]models.ApplicabilityLinkagesPage
	var barcodes map[int][]models.ArticleLookupNumber
	var tradeNumbers map[int][]models.ArticleLookupNumber
	var media map[int][]models.ArticleMediaFact
	var components map[int][]models.ArticleComponentFact
	var accessories map[int]models.ArticleAccessoriesPage
	var articleInfo map[int][]models.ArticleInfoFact
	var supersessions map[int][]models.ArticleSupersessionFact
	var countryMetadata map[int]models.ArticleCountryMetadata
	if include["classification"] {
		classifications, err = h.queries.GetArticleClassificationsBatch(ctx, allArticleIDs, languageID, countryID)
		if err != nil {
			h.respondError(w, http.StatusInternalServerError, "Failed to fetch batch classification: "+err.Error())
			return
		}
	}
	if include["oem"] {
		oemNumbers, err = h.queries.GetArticleOEMNumbersBatch(ctx, allArticleIDs)
		if err != nil {
			h.respondError(w, http.StatusInternalServerError, "Failed to fetch batch OEM numbers: "+err.Error())
			return
		}
	}
	if include["cross_references"] {
		crossReferences, err = h.queries.GetArticleCrossReferencesBatch(ctx, allArticleIDs, languageID, countryID)
		if err != nil {
			h.respondError(w, http.StatusInternalServerError, "Failed to fetch batch cross references: "+err.Error())
			return
		}
	}
	if include["criteria"] {
		criteria, err = h.queries.GetArticleCriteriaBatch(ctx, allArticleIDs, languageID, countryID, true)
		if err != nil {
			h.respondError(w, http.StatusInternalServerError, "Failed to fetch batch criteria: "+err.Error())
			return
		}
	}
	if include["applicability"] {
		applicability, err = h.queries.GetArticleApplicabilityLinkagesBatch(ctx, allArticleIDs, languageID, countryID, maxTargets)
		if err != nil {
			h.respondError(w, http.StatusInternalServerError, "Failed to fetch batch applicability: "+err.Error())
			return
		}
	}
	if include["barcodes"] {
		barcodes, err = h.queries.GetArticleLookupNumbersBatch(ctx, allArticleIDs, "EAN", countryID)
		if err != nil {
			h.respondError(w, http.StatusInternalServerError, "Failed to fetch batch barcodes: "+err.Error())
			return
		}
	}
	if include["trade_numbers"] {
		tradeNumbers, err = h.queries.GetArticleLookupNumbersBatch(ctx, allArticleIDs, "TradeNumber", countryID)
		if err != nil {
			h.respondError(w, http.StatusInternalServerError, "Failed to fetch batch trade numbers: "+err.Error())
			return
		}
	}
	if include["media"] {
		media, err = h.queries.GetArticleMediaBatch(ctx, allArticleIDs, languageID, countryID)
		if err != nil {
			h.respondError(w, http.StatusInternalServerError, "Failed to fetch batch media: "+err.Error())
			return
		}
	}
	if include["components"] {
		components, err = h.queries.GetArticleComponentsBatch(ctx, allArticleIDs, languageID, countryID)
		if err != nil {
			h.respondError(w, http.StatusInternalServerError, "Failed to fetch batch components: "+err.Error())
			return
		}
	}
	if include["accessories"] {
		accessories, err = h.queries.GetArticleAccessoriesBatch(ctx, allArticleIDs, languageID, countryID, maxAccessories)
		if err != nil {
			h.respondError(w, http.StatusInternalServerError, "Failed to fetch batch accessories: "+err.Error())
			return
		}
	}
	if include["article_info"] {
		articleInfo, err = h.queries.GetArticleInfoBatch(ctx, allArticleIDs, languageID, countryID, true)
		if err != nil {
			h.respondError(w, http.StatusInternalServerError, "Failed to fetch batch article info: "+err.Error())
			return
		}
	}
	if include["supersessions"] {
		supersessions, err = h.queries.GetArticleSupersessionsBatch(ctx, allArticleIDs, countryID)
		if err != nil {
			h.respondError(w, http.StatusInternalServerError, "Failed to fetch batch supersessions: "+err.Error())
			return
		}
	}
	if include["country_metadata"] {
		countryMetadata, err = h.queries.GetArticleCountryMetadataBatch(ctx, allArticleIDs, languageID, countryID)
		if err != nil {
			h.respondError(w, http.StatusInternalServerError, "Failed to fetch batch country metadata: "+err.Error())
			return
		}
	}

	for i := range results {
		if results[i].Status != "exact" {
			continue
		}
		ids := results[i].ArticleIDs
		if include["classification"] {
			results[i].Classification = mergeClassifications(ids, classifications)
		}
		if include["oem"] {
			results[i].OEMNumbers = mergeOEMNumbers(ids, oemNumbers)
		}
		if include["cross_references"] {
			results[i].CrossReferences = mergeCrossReferences(ids, crossReferences)
		}
		if include["criteria"] {
			results[i].Criteria = mergeCriteria(ids, criteria)
		}
		if include["applicability"] {
			page := mergeApplicability(ids, applicability)
			results[i].Applicability = &page
		}
		if include["barcodes"] {
			results[i].Barcodes = mergeLookupNumbers(ids, barcodes)
		}
		if include["trade_numbers"] {
			results[i].TradeNumbers = mergeLookupNumbers(ids, tradeNumbers)
		}
		if include["media"] {
			results[i].Media = mergeMediaFacts(ids, media)
		}
		if include["components"] {
			results[i].Components = mergeComponentFacts(ids, components)
		}
		if include["accessories"] {
			page := mergeAccessoryPages(ids, accessories)
			results[i].Accessories = &page
		}
		if include["article_info"] {
			results[i].ArticleInfo = mergeArticleInfoFacts(ids, articleInfo)
		}
		if include["supersessions"] {
			results[i].Supersessions = mergeSupersessionFacts(ids, supersessions)
		}
		if include["country_metadata"] {
			results[i].CountrySpecific = mergeCountrySpecificFacts(ids, countryMetadata)
		}
	}

	h.respondJSON(w, http.StatusOK, models.EnrichmentBatchResponse{
		LanguageID: languageID,
		CountryID:  countryID,
		Results:    results,
		Total:      len(results),
	})
}
