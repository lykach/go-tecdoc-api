package database

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"go-tecdoc-api/internal/models"
)

const maxSQLInClauseItems = 1000

func sqlPlaceholders(count int) string {
	if count <= 0 {
		return ""
	}
	return strings.TrimSuffix(strings.Repeat("?,", count), ",")
}

func uniqueSortedPositiveInts(values []int) []int {
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

func uniqueNonEmptyStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func splitTecDocScopes(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{}
	}
	parts := strings.Split(raw, ",")
	return uniqueNonEmptyStrings(parts)
}

// ExactArticleBatchLookup is a safe exact-identity lookup for one own article
// number. At least SupplierID or Brand must be present; a bare number is not a
// production-safe identity because TecDoc article numbers are not globally
// unique.
type ExactArticleBatchLookup struct {
	SearchNumber string
	SupplierID   *int
	Brand        string
	BrandAliases []string
}

// FindExactArticlesBatch resolves many supplier-qualified own-article numbers
// with one indexed ART_LOOKUP query. The dynamic OR list is bounded by the
// HTTP batch limit (50) and every branch includes a supplier discriminator, so
// short/common article numbers cannot fan out across the whole TecDoc corpus.
func exactLookupBrandNames(lookup ExactArticleBatchLookup) []string {
	values := make([]string, 0, 1+len(lookup.BrandAliases))
	values = append(values, lookup.Brand)
	values = append(values, lookup.BrandAliases...)
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

// FindExactArticlesBatch resolves many supplier-qualified own-article numbers
// with one indexed ART_LOOKUP query. The dynamic OR list is bounded by the
// HTTP batch limit and every branch includes a supplier discriminator or one
// explicit canonical/synonym brand set, so short/common article numbers cannot
// fan out across the whole TecDoc corpus.
func (q *Queries) FindExactArticlesBatch(ctx context.Context, lookups []ExactArticleBatchLookup, languageID, countryID int) (map[string][]models.EnrichmentResolvedArticle, error) {
	result := make(map[string][]models.EnrichmentResolvedArticle)
	conditions := make([]string, 0, len(lookups))
	args := make([]interface{}, 0, 1+len(lookups)*4)
	args = append(args, countryID)
	seen := make(map[string]struct{}, len(lookups))

	for _, lookup := range lookups {
		normalized := normalizeSearchNumber(lookup.SearchNumber)
		brands := exactLookupBrandNames(lookup)
		if normalized == "" || (lookup.SupplierID == nil && len(brands) == 0) {
			continue
		}
		var key string
		switch {
		case lookup.SupplierID != nil:
			// SupplierID is the strongest TecDoc manufacturer identity. Once it is
			// known, brand spellings are evidence/display aliases and must not make a
			// valid supplier-qualified article fail because of naming differences.
			key = fmt.Sprintf("%s|sid:%d", normalized, *lookup.SupplierID)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			conditions = append(conditions, "(ART_LOOKUP.ARL_SEARCH_NUMBER = ? AND ARTICLES.ART_SUP_ID = ?)")
			args = append(args, normalized, *lookup.SupplierID)
		default:
			normalizedBrands := make([]string, 0, len(brands))
			for _, brand := range brands {
				normalizedBrands = append(normalizedBrands, strings.ToUpper(strings.TrimSpace(brand)))
			}
			sort.Strings(normalizedBrands)
			key = fmt.Sprintf("%s|brands:%s", normalized, strings.Join(normalizedBrands, "\x1f"))
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			conditions = append(conditions, "(ART_LOOKUP.ARL_SEARCH_NUMBER = ? AND UPPER(TRIM(ARTICLES.ART_SUP_BRAND)) IN ("+sqlPlaceholders(len(brands))+"))")
			args = append(args, normalized)
			for _, brand := range brands {
				args = append(args, brand)
			}
		}
	}
	if len(conditions) == 0 {
		return result, nil
	}

	query := `
SELECT DISTINCT
	ART_LOOKUP.ARL_SEARCH_NUMBER,
	ARTICLES.ART_ID,
	ARTICLES.ART_ARTICLE_NR,
	ARTICLES.ART_SUP_ID,
	ARTICLES.ART_SUP_BRAND,
	CONCAT_WS(' ',
		(` + BuildGetTextSubquery("ARTICLES.ART_COMPLETE_DES_ID", languageID) + `),
		(` + BuildGetTextSubquery("ARTICLES.ART_DES_ID", languageID) + `)
	) AS ART_PRODUCT_NAME
FROM ART_LOOKUP
INNER JOIN ARTICLES ON ARTICLES.ART_ID = ART_LOOKUP.ARL_ART_ID
WHERE (ARTICLES.ART_CTM = 0 OR EXISTS (
	SELECT 1 FROM COUNTRY_RESTRICTIONS
	WHERE COUNTRY_RESTRICTIONS.CNTR_CTM_ID = ARTICLES.ART_CTM
		AND COUNTRY_RESTRICTIONS.CNTR_COU_ID = ?
))
	AND ART_LOOKUP.ARL_TYPE = 'ArticleNumber'
	AND (` + strings.Join(conditions, " OR ") + `)
ORDER BY ART_LOOKUP.ARL_SEARCH_NUMBER, ARTICLES.ART_SUP_ID, ARTICLES.ART_ID
`

	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var searchNumber string
		var item models.EnrichmentResolvedArticle
		if err := rows.Scan(
			&searchNumber,
			&item.ArticleID,
			&item.ArticleNumber,
			&item.SupplierID,
			&item.Brand,
			&item.Name,
		); err != nil {
			return nil, err
		}
		result[searchNumber] = append(result[searchNumber], item)
	}
	return result, rows.Err()
}

// GetArticleIdentitiesByIDsBatch validates already-resolved ART_IDs against the
// requested country context using one query.
func (q *Queries) GetArticleIdentitiesByIDsBatch(ctx context.Context, articleIDs []int, languageID, countryID int) (map[int]models.EnrichmentResolvedArticle, error) {
	ids := uniqueSortedPositiveInts(articleIDs)
	result := make(map[int]models.EnrichmentResolvedArticle, len(ids))
	if len(ids) == 0 {
		return result, nil
	}

	query := `
SELECT DISTINCT
	ARTICLES.ART_ID,
	ARTICLES.ART_ARTICLE_NR,
	ARTICLES.ART_SUP_ID,
	ARTICLES.ART_SUP_BRAND,
	CONCAT_WS(' ',
		(` + BuildGetTextSubquery("ARTICLES.ART_COMPLETE_DES_ID", languageID) + `),
		(` + BuildGetTextSubquery("ARTICLES.ART_DES_ID", languageID) + `)
	) AS ART_PRODUCT_NAME
FROM ARTICLES
WHERE (ARTICLES.ART_CTM = 0 OR EXISTS (
	SELECT 1 FROM COUNTRY_RESTRICTIONS
	WHERE COUNTRY_RESTRICTIONS.CNTR_CTM_ID = ARTICLES.ART_CTM
		AND COUNTRY_RESTRICTIONS.CNTR_COU_ID = ?
))
	AND ARTICLES.ART_ID IN (` + sqlPlaceholders(len(ids)) + `)
ORDER BY ARTICLES.ART_ID
`
	args := make([]interface{}, 0, len(ids)+1)
	args = append(args, countryID)
	for _, id := range ids {
		args = append(args, id)
	}

	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item models.EnrichmentResolvedArticle
		if err := rows.Scan(&item.ArticleID, &item.ArticleNumber, &item.SupplierID, &item.Brand, &item.Name); err != nil {
			return nil, err
		}
		result[item.ArticleID] = item
	}
	return result, rows.Err()
}

func (q *Queries) GetArticleClassification(ctx context.Context, articleID, languageID, countryID int) ([]models.ArticleProductTypeClassification, error) {
	all, err := q.GetArticleClassificationsBatch(ctx, []int{articleID}, languageID, countryID)
	if err != nil {
		return nil, err
	}
	return all[articleID], nil
}

// GetArticleClassificationsBatch returns raw TecDoc Product Type/Search Tree
// classification plus actual LINK_ART/LINK_LA_TYP usage evidence. It never
// guesses one "best" category because TecDoc legitimately assigns multiple
// product types to universal/accessory articles.
func (q *Queries) GetArticleClassificationsBatch(ctx context.Context, articleIDs []int, languageID, countryID int) (map[int][]models.ArticleProductTypeClassification, error) {
	ids := uniqueSortedPositiveInts(articleIDs)
	result := make(map[int][]models.ArticleProductTypeClassification, len(ids))
	if len(ids) == 0 {
		return result, nil
	}

	query := `
SELECT
	LAP.LAP_ART_ID,
	LAP.LAP_PT_ID,
	(` + BuildGetTextSubquery("PRODUCTS.PT_DES_ID", languageID) + `) AS PT_NAME,
	LPS.STR_ID,
	LPS.STR_TYPE,
	SEARCH_TREE.STR_LEVEL,
	SEARCH_TREE.STR_IS_LEAF,
	SEARCH_TREE.STR_ID_PARENT,
	(` + BuildGetTextSubquery("SEARCH_TREE.STR_DES_ID", languageID) + `) AS STR_NAME
FROM LINK_ART_PT LAP
INNER JOIN PRODUCTS ON PRODUCTS.PT_ID = LAP.LAP_PT_ID
LEFT JOIN LINK_PT_STR LPS ON LPS.PT_ID = LAP.LAP_PT_ID
LEFT JOIN SEARCH_TREE
	ON SEARCH_TREE.STR_ID = LPS.STR_ID
	AND SEARCH_TREE.STR_TYPE = LPS.STR_TYPE
WHERE LAP.LAP_ART_ID IN (` + sqlPlaceholders(len(ids)) + `)
ORDER BY LAP.LAP_ART_ID, LAP.LAP_PT_ID, LPS.STR_TYPE, SEARCH_TREE.STR_LEVEL, LPS.STR_ID
`
	args := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}

	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type aggregate struct {
		item      models.ArticleProductTypeClassification
		scopeSeen map[string]struct{}
		treeSeen  map[string]struct{}
	}
	byArticle := make(map[int]map[int]*aggregate, len(ids))
	for rows.Next() {
		var articleID, productTypeID int
		var ptName, rawType, treeName sql.NullString
		var strID, level, isLeaf, parentID sql.NullInt64
		if err := rows.Scan(&articleID, &productTypeID, &ptName, &strID, &rawType, &level, &isLeaf, &parentID, &treeName); err != nil {
			return nil, err
		}
		if byArticle[articleID] == nil {
			byArticle[articleID] = make(map[int]*aggregate)
		}
		agg := byArticle[articleID][productTypeID]
		if agg == nil {
			agg = &aggregate{
				item: models.ArticleProductTypeClassification{
					ProductTypeID: productTypeID,
					Name:          nullString(ptName),
					Scopes:        []string{},
					TreeNodes:     []models.ArticleClassificationTreeNode{},
				},
				scopeSeen: make(map[string]struct{}),
				treeSeen:  make(map[string]struct{}),
			}
			byArticle[articleID][productTypeID] = agg
		}
		if !strID.Valid {
			continue
		}
		scopes := splitTecDocScopes(nullString(rawType))
		for _, scope := range scopes {
			if _, ok := agg.scopeSeen[scope]; !ok {
				agg.scopeSeen[scope] = struct{}{}
				agg.item.Scopes = append(agg.item.Scopes, scope)
			}
		}
		key := fmt.Sprintf("%d|%s", strID.Int64, nullString(rawType))
		if _, ok := agg.treeSeen[key]; ok {
			continue
		}
		agg.treeSeen[key] = struct{}{}
		node := models.ArticleClassificationTreeNode{
			ID:      int(strID.Int64),
			RawType: nullString(rawType),
			Scopes:  scopes,
			Name:    nullString(treeName),
		}
		if level.Valid {
			node.Level = int(level.Int64)
		}
		if isLeaf.Valid {
			node.IsLeaf = isLeaf.Int64 != 0
		}
		if parentID.Valid {
			parent := int(parentID.Int64)
			node.ParentID = &parent
		}
		agg.item.TreeNodes = append(agg.item.TreeNodes, node)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}

	usageQuery := `
SELECT
	LINK_ART.LA_ART_ID,
	LINK_ART.LA_PT_ID,
	COUNT(DISTINCT LINK_ART.LA_ID) AS LINKAGE_COUNT,
	COUNT(DISTINCT CONCAT(LINK_LA_TYP.LAT_TYPE, '#', LINK_LA_TYP.LAT_TYP_ID)) AS TARGET_COUNT
FROM LINK_ART
LEFT JOIN LINK_LA_TYP
	ON LINK_LA_TYP.LAT_LA_ID = LINK_ART.LA_ID
	AND LINK_LA_TYP.LAT_PT_ID = LINK_ART.LA_PT_ID
WHERE LINK_ART.LA_ART_ID IN (` + sqlPlaceholders(len(ids)) + `)
	AND (
		LINK_ART.LA_CTM = 0
		OR EXISTS (
			SELECT 1 FROM COUNTRY_RESTRICTIONS
			WHERE COUNTRY_RESTRICTIONS.CNTR_CTM_ID = LINK_ART.LA_CTM
				AND COUNTRY_RESTRICTIONS.CNTR_COU_ID = ?
		)
	)
	AND (
		LINK_LA_TYP.LAT_LA_ID IS NULL
		OR LINK_LA_TYP.LAT_CTM = 0
		OR EXISTS (
			SELECT 1 FROM COUNTRY_RESTRICTIONS
			WHERE COUNTRY_RESTRICTIONS.CNTR_CTM_ID = LINK_LA_TYP.LAT_CTM
				AND COUNTRY_RESTRICTIONS.CNTR_COU_ID = ?
		)
	)
GROUP BY LINK_ART.LA_ART_ID, LINK_ART.LA_PT_ID
`
	usageArgs := make([]interface{}, 0, len(ids)+2)
	for _, id := range ids {
		usageArgs = append(usageArgs, id)
	}
	usageArgs = append(usageArgs, countryID, countryID)
	usageRows, err := q.db.QueryContext(ctx, usageQuery, usageArgs...)
	if err != nil {
		return nil, err
	}
	defer usageRows.Close()
	for usageRows.Next() {
		var articleID, productTypeID, linkageCount, targetCount int
		if err := usageRows.Scan(&articleID, &productTypeID, &linkageCount, &targetCount); err != nil {
			return nil, err
		}
		if agg := byArticle[articleID][productTypeID]; agg != nil {
			agg.item.LinkageCount = linkageCount
			agg.item.TargetCount = targetCount
		}
	}
	if err := usageRows.Err(); err != nil {
		return nil, err
	}

	for _, articleID := range ids {
		productMap := byArticle[articleID]
		if len(productMap) == 0 {
			result[articleID] = []models.ArticleProductTypeClassification{}
			continue
		}
		ptIDs := make([]int, 0, len(productMap))
		for ptID := range productMap {
			ptIDs = append(ptIDs, ptID)
		}
		sort.Ints(ptIDs)
		items := make([]models.ArticleProductTypeClassification, 0, len(ptIDs))
		for _, ptID := range ptIDs {
			agg := productMap[ptID]
			sort.Strings(agg.item.Scopes)
			items = append(items, agg.item)
		}
		result[articleID] = items
	}
	return result, nil
}

func (q *Queries) GetArticleCriteriaBatch(ctx context.Context, articleIDs []int, languageID, countryID int, includeHidden bool) (map[int][]models.ArticleCriterion, error) {
	ids := uniqueSortedPositiveInts(articleIDs)
	result := make(map[int][]models.ArticleCriterion, len(ids))
	if len(ids) == 0 {
		return result, nil
	}

	query := `
SELECT
	ARTICLE_CRITERIA.ACR_ART_ID,
	ARTICLE_CRITERIA.ACR_CRI_ID,
	(` + BuildGetTextSubquery("CRITERIA.CRI_DES_ID", languageID) + `) AS CRI_NAME,
	(` + BuildGetTextSubquery("CRITERIA.CRI_SHORT_DES_ID", languageID) + `) AS CRI_SHORT_NAME,
	ARTICLE_CRITERIA.ACR_VALUE,
	COALESCE(ARTICLE_CRITERIA.ACR_DES_ID, 0) AS CRI_VALUE_DES_ID,
	(` + BuildGetTextSubquery("ARTICLE_CRITERIA.ACR_DES_ID", languageID) + `) AS CRI_VALUE_DES,
	(` + BuildGetTextSubquery("CRITERIA.CRI_UNIT_DES_ID", languageID) + `) AS CRI_UNIT,
	CRITERIA.CRI_TYPE,
	ARTICLE_CRITERIA.ACR_DISPLAY
FROM ARTICLE_CRITERIA
INNER JOIN CRITERIA ON CRITERIA.CRI_ID = ARTICLE_CRITERIA.ACR_CRI_ID
WHERE ARTICLE_CRITERIA.ACR_ART_ID IN (` + sqlPlaceholders(len(ids)) + `)
	AND (
		ARTICLE_CRITERIA.ACR_CTM = 0
		OR EXISTS (
			SELECT 1 FROM COUNTRY_RESTRICTIONS
			WHERE COUNTRY_RESTRICTIONS.CNTR_CTM_ID = ARTICLE_CRITERIA.ACR_CTM
				AND COUNTRY_RESTRICTIONS.CNTR_COU_ID = ?
		)
	)
`
	if !includeHidden {
		query += " AND ARTICLE_CRITERIA.ACR_DISPLAY = 1\n"
	}
	query += "ORDER BY ARTICLE_CRITERIA.ACR_ART_ID, ARTICLE_CRITERIA.ACR_DISPLAY DESC, ARTICLE_CRITERIA.ACR_CRI_ID\n"

	args := make([]interface{}, 0, len(ids)+1)
	for _, id := range ids {
		args = append(args, id)
	}
	args = append(args, countryID)
	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var articleID int
		var criterion models.ArticleCriterion
		var name, shortName, valueDescription, unit, criterionType sql.NullString
		var display sql.NullInt64
		if err := rows.Scan(
			&articleID,
			&criterion.CriteriaID,
			&name,
			&shortName,
			&criterion.RawValue,
			&criterion.ValueDescriptionID,
			&valueDescription,
			&unit,
			&criterionType,
			&display,
		); err != nil {
			return nil, err
		}
		criterion.Name = nullString(name)
		criterion.Type = nullString(criterionType)
		criterion.Value = criterion.RawValue
		criterion.ShortName = nullString(shortName)
		if criterion.ShortName == "" {
			criterion.ShortName = criterion.Name
		}
		criterion.ValueDescription = nullString(valueDescription)
		criterion.Unit = nullString(unit)
		criterion.Display = display.Valid && display.Int64 != 0
		result[articleID] = append(result[articleID], criterion)
	}
	return result, rows.Err()
}

func (q *Queries) GetArticleOEMNumbersBatch(ctx context.Context, articleIDs []int) (map[int][]models.OEMNumber, error) {
	ids := uniqueSortedPositiveInts(articleIDs)
	result := make(map[int][]models.OEMNumber, len(ids))
	if len(ids) == 0 {
		return result, nil
	}

	query := `
SELECT
	ART_LOOKUP.ARL_ART_ID,
	ART_LOOKUP.ARL_DISPLAY_NR AS OEM_NUMBER,
	COALESCE(NULLIF(TRIM(ART_LOOKUP.ARL_BRA_BRAND), ''), NULLIF(TRIM(MANUFACTURERS.MFA_BRAND), ''), '') AS BRAND,
	COALESCE(ART_LOOKUP.ARL_BRA_ID, 0) AS BRAND_ID,
	COALESCE(NULLIF(TRIM(ART_LOOKUP.ARL_BRA_BRAND), ''), NULLIF(TRIM(MANUFACTURERS.MFA_BRAND), ''), '') AS MANUFACTURER,
	COALESCE(ART_LOOKUP.ARL_BRA_ID, 0) AS MANUFACTURER_ID
FROM ART_LOOKUP
LEFT JOIN MANUFACTURERS ON MANUFACTURERS.MFA_ID = ART_LOOKUP.ARL_BRA_ID
WHERE ART_LOOKUP.ARL_ART_ID IN (` + sqlPlaceholders(len(ids)) + `)
	AND ART_LOOKUP.ARL_TYPE = 'OENumber'
ORDER BY ART_LOOKUP.ARL_ART_ID, MANUFACTURER, ART_LOOKUP.ARL_DISPLAY_NR
`
	args := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}
	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var articleID int
		var item models.OEMNumber
		if err := rows.Scan(&articleID, &item.Number, &item.Brand, &item.BrandID, &item.Manufacturer, &item.ManufacturerID); err != nil {
			return nil, err
		}
		result[articleID] = append(result[articleID], item)
	}
	return result, rows.Err()
}

// GetArticleCrossReferencesBatch returns explicit IAM/OEM references for many
// concrete ART_IDs without issuing one query per article. The source ART_ID is
// preserved in the result map so commercial identities represented by several
// technical article rows can be merged safely by the enrichment handler.
func (q *Queries) GetArticleCrossReferencesBatch(ctx context.Context, articleIDs []int, languageID, countryID int) (map[int][]models.CrossReference, error) {
	ids := uniqueSortedPositiveInts(articleIDs)
	result := make(map[int][]models.CrossReference, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	if len(ids) > maxSQLInClauseItems {
		return nil, fmt.Errorf("cross-reference batch exceeds %d article IDs", maxSQLInClauseItems)
	}

	articleName := `CONCAT_WS(' ', (` + BuildGetTextSubquery("ARTICLES.ART_COMPLETE_DES_ID", languageID) + `), (` + BuildGetTextSubquery("ARTICLES.ART_DES_ID", languageID) + `))`
	sourceArticleName := `CONCAT_WS(' ', (` + BuildGetTextSubquery("SOURCE_ARTICLE.ART_COMPLETE_DES_ID", languageID) + `), (` + BuildGetTextSubquery("SOURCE_ARTICLE.ART_DES_ID", languageID) + `))`
	placeholders := sqlPlaceholders(len(ids))

	queries := []struct {
		sql            string
		includeCountry bool
	}{
		{
			sql: `
SELECT DISTINCT
	SOURCE_LOOKUP.ARL_ART_ID AS SOURCE_ART_ID,
	ARTICLES.ART_ID,
	ARTICLES.ART_ARTICLE_NR,
	ARTICLES.ART_SUP_BRAND AS LEGACY_BRAND,
	ARTICLES.ART_SUP_ID AS SUPPLIER_ID,
	ARTICLES.ART_SUP_BRAND AS SUPPLIER_NAME,
	NULL AS MANUFACTURER_ID,
	NULL AS MANUFACTURER_NAME,
	` + articleName + ` AS ART_PRODUCT_NAME,
	'IAM' AS CROSS_TYPE
FROM ART_LOOKUP AS SOURCE_LOOKUP
INNER JOIN MANUFACTURERS
	ON MANUFACTURERS.MFA_ID = SOURCE_LOOKUP.ARL_BRA_ID
INNER JOIN ART_LOOKUP AS TARGET_LOOKUP
	ON TARGET_LOOKUP.ARL_SEARCH_NUMBER = SOURCE_LOOKUP.ARL_SEARCH_NUMBER
	AND TARGET_LOOKUP.ARL_TYPE = 'ArticleNumber'
	AND TARGET_LOOKUP.ARL_BRA_ID = MANUFACTURERS.MFA_SUP_ID
INNER JOIN ARTICLES ON ARTICLES.ART_ID = TARGET_LOOKUP.ARL_ART_ID
WHERE (ARTICLES.ART_CTM = 0 OR EXISTS (
	SELECT 1 FROM COUNTRY_RESTRICTIONS
	WHERE COUNTRY_RESTRICTIONS.CNTR_CTM_ID = ARTICLES.ART_CTM
		AND COUNTRY_RESTRICTIONS.CNTR_COU_ID = ?
))
	AND SOURCE_LOOKUP.ARL_ART_ID IN (` + placeholders + `)
	AND SOURCE_LOOKUP.ARL_TYPE = 'IAMNumber'
	AND ARTICLES.ART_ID <> SOURCE_LOOKUP.ARL_ART_ID
ORDER BY SOURCE_ART_ID, LEGACY_BRAND, ART_ARTICLE_NR
`,
			includeCountry: true,
		},
		{
			sql: `
SELECT DISTINCT
	SOURCE_LOOKUP.ARL_ART_ID AS SOURCE_ART_ID,
	ARTICLES.ART_ID,
	ARTICLES.ART_ARTICLE_NR,
	ARTICLES.ART_SUP_BRAND AS LEGACY_BRAND,
	ARTICLES.ART_SUP_ID AS SUPPLIER_ID,
	ARTICLES.ART_SUP_BRAND AS SUPPLIER_NAME,
	NULL AS MANUFACTURER_ID,
	NULL AS MANUFACTURER_NAME,
	` + articleName + ` AS ART_PRODUCT_NAME,
	'IAM' AS CROSS_TYPE
FROM ART_LOOKUP AS SOURCE_LOOKUP
INNER JOIN MANUFACTURERS
	ON MANUFACTURERS.MFA_SUP_ID = SOURCE_LOOKUP.ARL_BRA_ID
INNER JOIN ART_LOOKUP AS TARGET_LOOKUP
	ON TARGET_LOOKUP.ARL_SEARCH_NUMBER = SOURCE_LOOKUP.ARL_SEARCH_NUMBER
	AND TARGET_LOOKUP.ARL_TYPE = 'IAMNumber'
	AND TARGET_LOOKUP.ARL_BRA_ID = MANUFACTURERS.MFA_ID
INNER JOIN ARTICLES ON ARTICLES.ART_ID = TARGET_LOOKUP.ARL_ART_ID
WHERE (ARTICLES.ART_CTM = 0 OR EXISTS (
	SELECT 1 FROM COUNTRY_RESTRICTIONS
	WHERE COUNTRY_RESTRICTIONS.CNTR_CTM_ID = ARTICLES.ART_CTM
		AND COUNTRY_RESTRICTIONS.CNTR_COU_ID = ?
))
	AND SOURCE_LOOKUP.ARL_ART_ID IN (` + placeholders + `)
	AND SOURCE_LOOKUP.ARL_TYPE = 'ArticleNumber'
	AND ARTICLES.ART_ID <> SOURCE_LOOKUP.ARL_ART_ID
ORDER BY SOURCE_ART_ID, LEGACY_BRAND, ART_ARTICLE_NR
`,
			includeCountry: true,
		},
		{
			sql: `
SELECT DISTINCT
	SOURCE_LOOKUP.ARL_ART_ID AS SOURCE_ART_ID,
	NULL AS ART_ID,
	SOURCE_LOOKUP.ARL_DISPLAY_NR AS ART_ARTICLE_NR,
	COALESCE(NULLIF(TRIM(SOURCE_LOOKUP.ARL_BRA_BRAND), ''), NULLIF(TRIM(OE_MANUFACTURER.MFA_BRAND), ''), '') AS LEGACY_BRAND,
	NULL AS SUPPLIER_ID,
	NULL AS SUPPLIER_NAME,
	SOURCE_LOOKUP.ARL_BRA_ID AS MANUFACTURER_ID,
	COALESCE(NULLIF(TRIM(SOURCE_LOOKUP.ARL_BRA_BRAND), ''), NULLIF(TRIM(OE_MANUFACTURER.MFA_BRAND), ''), '') AS MANUFACTURER_NAME,
	` + sourceArticleName + ` AS ART_PRODUCT_NAME,
	'OEM' AS CROSS_TYPE
FROM ART_LOOKUP AS SOURCE_LOOKUP
INNER JOIN ARTICLES AS SOURCE_ARTICLE ON SOURCE_ARTICLE.ART_ID = SOURCE_LOOKUP.ARL_ART_ID
LEFT JOIN MANUFACTURERS AS OE_MANUFACTURER
	ON OE_MANUFACTURER.MFA_ID = SOURCE_LOOKUP.ARL_BRA_ID
WHERE SOURCE_LOOKUP.ARL_ART_ID IN (` + placeholders + `)
	AND SOURCE_LOOKUP.ARL_TYPE = 'OENumber'
ORDER BY SOURCE_ART_ID, LEGACY_BRAND, ART_ARTICLE_NR
`,
		},
	}

	for _, batchQuery := range queries {
		args := make([]interface{}, 0, len(ids)+1)
		if batchQuery.includeCountry {
			args = append(args, countryID)
		}
		for _, id := range ids {
			args = append(args, id)
		}
		rows, err := q.db.QueryContext(ctx, batchQuery.sql, args...)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var sourceArticleID int
			var cr models.CrossReference
			var linkedArticleID, supplierID, manufacturerID sql.NullInt64
			var legacyBrand, supplierName, manufacturerName sql.NullString
			if err := rows.Scan(
				&sourceArticleID,
				&linkedArticleID,
				&cr.ArticleNr,
				&legacyBrand,
				&supplierID,
				&supplierName,
				&manufacturerID,
				&manufacturerName,
				&cr.Name,
				&cr.Type,
			); err != nil {
				_ = rows.Close()
				return nil, err
			}
			if linkedArticleID.Valid {
				value := int(linkedArticleID.Int64)
				cr.ArticleID = &value
			}
			if legacyBrand.Valid {
				cr.Brand = strings.TrimSpace(legacyBrand.String)
			}
			if supplierID.Valid {
				value := int(supplierID.Int64)
				cr.SupplierID = &value
			}
			if supplierName.Valid {
				cr.Supplier = strings.TrimSpace(supplierName.String)
			}
			if manufacturerID.Valid {
				value := int(manufacturerID.Int64)
				cr.ManufacturerID = &value
			}
			if manufacturerName.Valid {
				cr.Manufacturer = strings.TrimSpace(manufacturerName.String)
			}
			if cr.Type == "IAM" && cr.Supplier != "" {
				cr.Brand = cr.Supplier
			}
			if cr.Type == "OEM" && cr.Manufacturer != "" {
				cr.Brand = cr.Manufacturer
			}
			result[sourceArticleID] = append(result[sourceArticleID], cr)
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return nil, err
		}
		if err := rows.Close(); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (q *Queries) loadApplicabilityRestrictions(ctx context.Context, linkageIDs []int, languageID, countryID int) (map[int][]models.ApplicabilityRestriction, error) {
	ids := uniqueSortedPositiveInts(linkageIDs)
	result := make(map[int][]models.ApplicabilityRestriction, len(ids))
	if len(ids) == 0 {
		return result, nil
	}

	// Keep IN lists bounded. A 50-item enrichment batch can legitimately touch
	// thousands of LA_IDs, and one unbounded placeholder list risks exceeding
	// MariaDB parameter/packet limits even though each individual lookup is
	// index-friendly on LAC_LA_ID.
	for start := 0; start < len(ids); start += maxSQLInClauseItems {
		end := start + maxSQLInClauseItems
		if end > len(ids) {
			end = len(ids)
		}
		chunk := ids[start:end]
		query := `
SELECT
	LA_CRITERIA.LAC_LA_ID,
	LA_CRITERIA.LAC_CRI_ID,
	(` + BuildGetTextSubquery("CRITERIA.CRI_DES_ID", languageID) + `) AS CRI_NAME,
	(` + BuildGetTextSubquery("CRITERIA.CRI_SHORT_DES_ID", languageID) + `) AS CRI_SHORT_NAME,
	CRITERIA.CRI_TYPE,
	(` + BuildGetTextSubquery("CRITERIA.CRI_UNIT_DES_ID", languageID) + `) AS CRI_UNIT,
	LA_CRITERIA.LAC_VALUE,
	(` + BuildGetTextSubquery("LA_CRITERIA.LAC_DES_ID", languageID) + `) AS VALUE_DESCRIPTION
FROM LA_CRITERIA
INNER JOIN CRITERIA ON CRITERIA.CRI_ID = LA_CRITERIA.LAC_CRI_ID
WHERE LA_CRITERIA.LAC_LA_ID IN (` + sqlPlaceholders(len(chunk)) + `)
	AND (
		LA_CRITERIA.LAC_CTM = 0
		OR EXISTS (
			SELECT 1 FROM COUNTRY_RESTRICTIONS
			WHERE COUNTRY_RESTRICTIONS.CNTR_CTM_ID = LA_CRITERIA.LAC_CTM
				AND COUNTRY_RESTRICTIONS.CNTR_COU_ID = ?
		)
	)
ORDER BY LA_CRITERIA.LAC_LA_ID, LA_CRITERIA.LAC_CRI_ID, LA_CRITERIA.LAC_VALUE
`
		args := make([]interface{}, 0, len(chunk)+1)
		for _, id := range chunk {
			args = append(args, id)
		}
		args = append(args, countryID)
		rows, err := q.db.QueryContext(ctx, query, args...)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var linkageID int
			var restriction models.ApplicabilityRestriction
			var name, shortName, criterionType, unit, valueDescription sql.NullString
			if err := rows.Scan(
				&linkageID,
				&restriction.CriteriaID,
				&name,
				&shortName,
				&criterionType,
				&unit,
				&restriction.RawValue,
				&valueDescription,
			); err != nil {
				_ = rows.Close()
				return nil, err
			}
			restriction.Name = nullString(name)
			restriction.Type = nullString(criterionType)
			restriction.ShortName = nullString(shortName)
			if restriction.ShortName == "" {
				restriction.ShortName = restriction.Name
			}
			restriction.Unit = nullString(unit)
			restriction.ValueDescription = nullString(valueDescription)
			result[linkageID] = append(result[linkageID], restriction)
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return nil, err
		}
		if err := rows.Close(); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func aggregateApplicabilityRows(rows *sql.Rows) (map[int]models.ApplicabilityLinkagesPage, []int, error) {
	type linkageKey struct {
		articleID   int
		linkageID   int
		productType int
		targetType  string
	}
	pages := make(map[int]models.ApplicabilityLinkagesPage)
	linkageIndexes := make(map[linkageKey]int)
	linkageIDs := make([]int, 0)
	linkageSeen := make(map[int]struct{})

	for rows.Next() {
		var articleID, linkageID, productType, targetID, totalTargets int
		var targetType string
		if err := rows.Scan(&articleID, &linkageID, &productType, &targetType, &targetID, &totalTargets); err != nil {
			return nil, nil, err
		}
		page := pages[articleID]
		page.TotalTargets = totalTargets
		page.TotalTargetsExact = true
		page.Returned++
		key := linkageKey{articleID: articleID, linkageID: linkageID, productType: productType, targetType: targetType}
		idx, ok := linkageIndexes[key]
		if !ok {
			idx = len(page.Linkages)
			linkageIndexes[key] = idx
			page.Linkages = append(page.Linkages, models.ApplicabilityLinkage{
				LinkageID:    linkageID,
				ProductType:  productType,
				TargetType:   targetType,
				Targets:      []models.ApplicabilityTarget{},
				Restrictions: []models.ApplicabilityRestriction{},
			})
		}
		page.Linkages[idx].Targets = append(page.Linkages[idx].Targets, models.ApplicabilityTarget{Type: targetType, ID: targetID})
		pages[articleID] = page
		if _, ok := linkageSeen[linkageID]; !ok {
			linkageSeen[linkageID] = struct{}{}
			linkageIDs = append(linkageIDs, linkageID)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	return pages, linkageIDs, nil
}

func attachApplicabilityRestrictions(pages map[int]models.ApplicabilityLinkagesPage, restrictions map[int][]models.ApplicabilityRestriction) {
	for articleID, page := range pages {
		for i := range page.Linkages {
			page.Linkages[i].Restrictions = restrictions[page.Linkages[i].LinkageID]
			if page.Linkages[i].Restrictions == nil {
				page.Linkages[i].Restrictions = []models.ApplicabilityRestriction{}
			}
		}
		pages[articleID] = page
	}
}

// GetArticleApplicabilityLinkagesPage returns one target-row page. A linkage
// can span pages; consumers should merge by (la_id, pt_id, target_type).
func (q *Queries) countArticleApplicabilityTargets(ctx context.Context, articleID, countryID int, targetType string) (int, error) {
	whereType := ""
	args := []interface{}{countryID, countryID, articleID}
	if targetType != "" {
		whereType = " AND LINK_LA_TYP.LAT_TYPE = ?\n"
		args = append(args, targetType)
	}
	query := `
SELECT COUNT(*)
FROM LINK_ART
INNER JOIN LINK_LA_TYP
	ON LINK_LA_TYP.LAT_LA_ID = LINK_ART.LA_ID
	AND LINK_LA_TYP.LAT_PT_ID = LINK_ART.LA_PT_ID
WHERE (
		LINK_ART.LA_CTM = 0
		OR EXISTS (
			SELECT 1 FROM COUNTRY_RESTRICTIONS
			WHERE COUNTRY_RESTRICTIONS.CNTR_CTM_ID = LINK_ART.LA_CTM
				AND COUNTRY_RESTRICTIONS.CNTR_COU_ID = ?
		)
	)
	AND (
		LINK_LA_TYP.LAT_CTM = 0
		OR EXISTS (
			SELECT 1 FROM COUNTRY_RESTRICTIONS
			WHERE COUNTRY_RESTRICTIONS.CNTR_CTM_ID = LINK_LA_TYP.LAT_CTM
				AND COUNTRY_RESTRICTIONS.CNTR_COU_ID = ?
		)
	)
	AND LINK_ART.LA_ART_ID = ?
` + whereType
	var total int
	if err := q.db.QueryRowContext(ctx, query, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

func (q *Queries) GetArticleApplicabilityLinkagesPage(ctx context.Context, articleID, languageID, countryID int, targetType string, limit, offset int) (models.ApplicabilityLinkagesPage, error) {
	whereType := ""
	args := make([]interface{}, 0, 6)
	args = append(args, countryID, countryID, articleID)
	if targetType != "" {
		whereType = " AND LINK_LA_TYP.LAT_TYPE = ?\n"
		args = append(args, targetType)
	}
	args = append(args, limit, offset)
	query := `
SELECT
	LINK_ART.LA_ART_ID,
	LINK_ART.LA_ID,
	LINK_ART.LA_PT_ID,
	LINK_LA_TYP.LAT_TYPE,
	LINK_LA_TYP.LAT_TYP_ID,
	COUNT(*) OVER () AS TOTAL_TARGETS
FROM LINK_ART
INNER JOIN LINK_LA_TYP
	ON LINK_LA_TYP.LAT_LA_ID = LINK_ART.LA_ID
	AND LINK_LA_TYP.LAT_PT_ID = LINK_ART.LA_PT_ID
WHERE (
		LINK_ART.LA_CTM = 0
		OR EXISTS (
			SELECT 1 FROM COUNTRY_RESTRICTIONS
			WHERE COUNTRY_RESTRICTIONS.CNTR_CTM_ID = LINK_ART.LA_CTM
				AND COUNTRY_RESTRICTIONS.CNTR_COU_ID = ?
		)
	)
	AND (
		LINK_LA_TYP.LAT_CTM = 0
		OR EXISTS (
			SELECT 1 FROM COUNTRY_RESTRICTIONS
			WHERE COUNTRY_RESTRICTIONS.CNTR_CTM_ID = LINK_LA_TYP.LAT_CTM
				AND COUNTRY_RESTRICTIONS.CNTR_COU_ID = ?
		)
	)
	AND LINK_ART.LA_ART_ID = ?
` + whereType + `
ORDER BY LINK_ART.LA_ID, LINK_ART.LA_PT_ID, LINK_LA_TYP.LAT_TYPE, LINK_LA_TYP.LAT_TYP_ID
LIMIT ? OFFSET ?
`
	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return models.ApplicabilityLinkagesPage{}, err
	}
	defer rows.Close()
	pages, linkageIDs, err := aggregateApplicabilityRows(rows)
	if err != nil {
		_ = rows.Close()
		return models.ApplicabilityLinkagesPage{}, err
	}
	if err := rows.Close(); err != nil {
		return models.ApplicabilityLinkagesPage{}, err
	}
	page := pages[articleID]
	if page.Linkages == nil {
		page.Linkages = []models.ApplicabilityLinkage{}
	}
	if page.TotalTargets == 0 && offset > 0 {
		total, err := q.countArticleApplicabilityTargets(ctx, articleID, countryID, targetType)
		if err != nil {
			return models.ApplicabilityLinkagesPage{}, err
		}
		page.TotalTargets = total
		page.TotalTargetsExact = true
		page.Truncated = offset < total
	}
	restrictions, err := q.loadApplicabilityRestrictions(ctx, linkageIDs, languageID, countryID)
	if err != nil {
		return models.ApplicabilityLinkagesPage{}, err
	}
	pages[articleID] = page
	attachApplicabilityRestrictions(pages, restrictions)
	page = pages[articleID]
	page.Truncated = offset+page.Returned < page.TotalTargets
	pages[articleID] = page
	return page, nil
}

// GetArticleApplicabilityLinkagesBatch uses window functions to cap returned
// target rows per article while keeping one set-based query for the whole
// batch. TotalTargets/Truncated tell the caller which articles need paged
// follow-up through /articles/{id}/applicability.
func (q *Queries) GetArticleApplicabilityLinkagesBatch(ctx context.Context, articleIDs []int, languageID, countryID, maxTargetsPerArticle int) (map[int]models.ApplicabilityLinkagesPage, error) {
	ids := uniqueSortedPositiveInts(articleIDs)
	result := make(map[int]models.ApplicabilityLinkagesPage, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	if maxTargetsPerArticle < 1 {
		maxTargetsPerArticle = 1
	}

	query := `
WITH ranked_applicability AS (
	SELECT
		LINK_ART.LA_ART_ID,
		LINK_ART.LA_ID,
		LINK_ART.LA_PT_ID,
		LINK_LA_TYP.LAT_TYPE,
		LINK_LA_TYP.LAT_TYP_ID,
		COUNT(*) OVER (PARTITION BY LINK_ART.LA_ART_ID) AS TOTAL_TARGETS,
		ROW_NUMBER() OVER (
			PARTITION BY LINK_ART.LA_ART_ID
			ORDER BY LINK_ART.LA_ID, LINK_ART.LA_PT_ID, LINK_LA_TYP.LAT_TYPE, LINK_LA_TYP.LAT_TYP_ID
		) AS RN
	FROM LINK_ART
	INNER JOIN LINK_LA_TYP
		ON LINK_LA_TYP.LAT_LA_ID = LINK_ART.LA_ID
		AND LINK_LA_TYP.LAT_PT_ID = LINK_ART.LA_PT_ID
	WHERE LINK_ART.LA_ART_ID IN (` + sqlPlaceholders(len(ids)) + `)
		AND (
			LINK_ART.LA_CTM = 0
			OR EXISTS (
				SELECT 1 FROM COUNTRY_RESTRICTIONS
				WHERE COUNTRY_RESTRICTIONS.CNTR_CTM_ID = LINK_ART.LA_CTM
					AND COUNTRY_RESTRICTIONS.CNTR_COU_ID = ?
			)
		)
		AND (
			LINK_LA_TYP.LAT_CTM = 0
			OR EXISTS (
				SELECT 1 FROM COUNTRY_RESTRICTIONS
				WHERE COUNTRY_RESTRICTIONS.CNTR_CTM_ID = LINK_LA_TYP.LAT_CTM
					AND COUNTRY_RESTRICTIONS.CNTR_COU_ID = ?
			)
		)
)
SELECT LA_ART_ID, LA_ID, LA_PT_ID, LAT_TYPE, LAT_TYP_ID, TOTAL_TARGETS
FROM ranked_applicability
WHERE RN <= ?
ORDER BY LA_ART_ID, RN
`
	args := make([]interface{}, 0, len(ids)+3)
	for _, id := range ids {
		args = append(args, id)
	}
	args = append(args, countryID, countryID, maxTargetsPerArticle)
	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	pages, linkageIDs, err := aggregateApplicabilityRows(rows)
	if err != nil {
		_ = rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	restrictions, err := q.loadApplicabilityRestrictions(ctx, linkageIDs, languageID, countryID)
	if err != nil {
		return nil, err
	}
	attachApplicabilityRestrictions(pages, restrictions)
	for _, id := range ids {
		page := pages[id]
		if page.Linkages == nil {
			page.Linkages = []models.ApplicabilityLinkage{}
			page.TotalTargetsExact = true
		}
		page.Truncated = page.Returned < page.TotalTargets
		result[id] = page
	}
	return result, nil
}

// GetArticleLookupNumbersBatch returns EAN or TradeNumber rows without losing
// the technical ART_ID that supplied each number.
func (q *Queries) GetArticleLookupNumbersBatch(ctx context.Context, articleIDs []int, lookupType string, countryID int) (map[int][]models.ArticleLookupNumber, error) {
	ids := uniqueSortedPositiveInts(articleIDs)
	result := make(map[int][]models.ArticleLookupNumber, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	if lookupType != "EAN" && lookupType != "TradeNumber" {
		return nil, fmt.Errorf("unsupported ART_LOOKUP type %q", lookupType)
	}
	query := `
SELECT ART_LOOKUP.ARL_ART_ID,
       COALESCE(ART_LOOKUP.ARL_DISPLAY_NR,''),
       COALESCE(ART_LOOKUP.ARL_SEARCH_NUMBER,''),
       ART_LOOKUP.ARL_TYPE
FROM ART_LOOKUP
WHERE ART_LOOKUP.ARL_ART_ID IN (` + sqlPlaceholders(len(ids)) + `)
  AND ART_LOOKUP.ARL_TYPE = ?
  AND (ART_LOOKUP.ARL_CTM=0 OR EXISTS(
      SELECT 1 FROM COUNTRY_RESTRICTIONS C
      WHERE C.CNTR_CTM_ID=ART_LOOKUP.ARL_CTM AND C.CNTR_COU_ID=?
  ))
ORDER BY ART_LOOKUP.ARL_ART_ID, ART_LOOKUP.ARL_DISPLAY_NR, ART_LOOKUP.ARL_SEARCH_NUMBER`
	args := make([]any, 0, len(ids)+2)
	for _, id := range ids {
		args = append(args, id)
	}
	args = append(args, lookupType, countryID)
	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var articleID int
		var item models.ArticleLookupNumber
		if err := rows.Scan(&articleID, &item.Number, &item.SearchNumber, &item.Type); err != nil {
			return nil, err
		}
		result[articleID] = append(result[articleID], item)
	}
	return result, rows.Err()
}

func (q *Queries) GetArticleMediaBatch(ctx context.Context, articleIDs []int, languageID, countryID int) (map[int][]models.ArticleMediaFact, error) {
	ids := uniqueSortedPositiveInts(articleIDs)
	result := make(map[int][]models.ArticleMediaFact, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	query := `
SELECT
  ART_MEDIA_INFO.ART_MEDIA_ART_ID,
  ART_MEDIA_INFO.ART_MEDIA_ID,
  ART_MEDIA_INFO.ART_MEDIA_TYPE,
  ART_MEDIA_INFO.ART_MEDIA_SUP_ID,
  COALESCE(ART_MEDIA_INFO.ART_MEDIA_FILE_NAME,''),
  CASE
    WHEN ART_MEDIA_INFO.ART_MEDIA_TYPE='URL' THEN COALESCE(ART_MEDIA_INFO.ART_MEDIA_HIPPERLINK,'')
    WHEN ART_MEDIA_INFO.ART_MEDIA_TYPE='PDF' THEN CONCAT_WS('/','PDF',ART_MEDIA_INFO.ART_MEDIA_SUP_ID,ART_MEDIA_INFO.ART_MEDIA_FILE_NAME)
    ELSE CONCAT_WS('/','IMAGE',ART_MEDIA_INFO.ART_MEDIA_SUP_ID,ART_MEDIA_INFO.ART_MEDIA_FILE_NAME)
  END,
  COALESCE(ART_MEDIA_INFO.ART_MEDIA_SORT,0),
  COALESCE(ART_MEDIA_INFO.ART_MEDIA_CTM,0),
  COALESCE(ART_MEDIA_INFO.ART_MEDIA_LNG_ID,0),
  COALESCE(ART_MEDIA_INFO.ART_MEDIA_CONTENT_TYPE,''),
  COALESCE(ART_MEDIA_INFO.ART_MEDIA_WIDTH,0),
  COALESCE(ART_MEDIA_INFO.ART_MEDIA_HEIGHT,0),
  COALESCE(ART_MEDIA_INFO.ART_MEDIA_COLOR_DEPTH,0),
  COALESCE((` + BuildGetTextSubquery("ART_MEDIA_INFO.ART_MEDIA_NORM_DES_ID", languageID) + `),''),
  COALESCE((` + BuildGetTextSubquery("ART_MEDIA_INFO.ART_MEDIA_DES_ID", languageID) + `),'')
FROM ART_MEDIA_INFO
WHERE ART_MEDIA_INFO.ART_MEDIA_ART_ID IN (` + sqlPlaceholders(len(ids)) + `)
  AND (ART_MEDIA_INFO.ART_MEDIA_CTM=0 OR EXISTS(
      SELECT 1 FROM COUNTRY_RESTRICTIONS C
      WHERE C.CNTR_CTM_ID=ART_MEDIA_INFO.ART_MEDIA_CTM AND C.CNTR_COU_ID=?
  ))
ORDER BY ART_MEDIA_INFO.ART_MEDIA_ART_ID, ART_MEDIA_INFO.ART_MEDIA_SORT,
         ART_MEDIA_INFO.ART_MEDIA_ID, ART_MEDIA_INFO.ART_MEDIA_TYPE, ART_MEDIA_INFO.ART_MEDIA_CTM, ART_MEDIA_INFO.ART_MEDIA_LNG_ID,
         ART_MEDIA_INFO.ART_MEDIA_CONTENT_TYPE, ART_MEDIA_INFO.ART_MEDIA_FILE_NAME, ART_MEDIA_INFO.ART_MEDIA_HIPPERLINK`
	args := make([]any, 0, len(ids)+1)
	for _, id := range ids {
		args = append(args, id)
	}
	args = append(args, countryID)
	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var item models.ArticleMediaFact
		if err := rows.Scan(&item.ArticleID, &item.MediaID, &item.MediaType, &item.SupplierID, &item.FileName, &item.URL, &item.Sort, &item.CountryMarker, &item.LanguageID, &item.ContentType, &item.Width, &item.Height, &item.ColorDepth, &item.NormalizedDesc, &item.SupplierDesc); err != nil {
			return nil, err
		}
		result[item.ArticleID] = append(result[item.ArticleID], item)
	}
	return result, rows.Err()
}

func (q *Queries) GetArticleComponentsBatch(ctx context.Context, articleIDs []int, languageID, countryID int) (map[int][]models.ArticleComponentFact, error) {
	ids := uniqueSortedPositiveInts(articleIDs)
	result := make(map[int][]models.ArticleComponentFact, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	query := `
SELECT DISTINCT
  ARTICLES_PART_LIST.APL_ART_ID,
  ARTICLES.ART_ID,
  ARTICLES.ART_ARTICLE_NR,
  ARTICLES.ART_SUP_ID,
  ARTICLES.ART_SUP_BRAND,
  CONCAT_WS(' ', (` + BuildGetTextSubquery("ARTICLES.ART_COMPLETE_DES_ID", languageID) + `), (` + BuildGetTextSubquery("ARTICLES.ART_DES_ID", languageID) + `)),
  COALESCE(ARTICLES_PART_LIST.APL_QUANTITY,1),
  COALESCE(ARTICLES_PART_LIST.APL_SEQ_NO,0),
  COALESCE(ARTICLES_PART_LIST.APL_PT_ID,0)
FROM ARTICLES_PART_LIST
INNER JOIN ARTICLES ON ARTICLES.ART_ID=ARTICLES_PART_LIST.APL_ART_ID_COMPONENT
WHERE ARTICLES_PART_LIST.APL_ART_ID IN (` + sqlPlaceholders(len(ids)) + `)
  AND EXISTS (
    SELECT 1 FROM ART_COUNTRY_SPECIFICS
    WHERE ART_COUNTRY_SPECIFICS.ACS_ART_ID=ARTICLES.ART_ID
      AND ART_COUNTRY_SPECIFICS.ACS_COU_ID IN (?,0)
  )
  AND (ARTICLES_PART_LIST.APL_CTM=0 OR EXISTS(
      SELECT 1 FROM COUNTRY_RESTRICTIONS C
      WHERE C.CNTR_CTM_ID=ARTICLES_PART_LIST.APL_CTM AND C.CNTR_COU_ID=?
  ))
ORDER BY ARTICLES_PART_LIST.APL_ART_ID, ARTICLES_PART_LIST.APL_SEQ_NO, ARTICLES.ART_ID`
	args := make([]any, 0, len(ids)+2)
	for _, id := range ids {
		args = append(args, id)
	}
	args = append(args, countryID, countryID)
	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var item models.ArticleComponentFact
		if err := rows.Scan(&item.ParentArticleID, &item.ArticleID, &item.ArticleNr, &item.SupplierID, &item.Brand, &item.Name, &item.Quantity, &item.Sequence, &item.ProductTypeID); err != nil {
			return nil, err
		}
		result[item.ParentArticleID] = append(result[item.ParentArticleID], item)
	}
	return result, rows.Err()
}

func (q *Queries) GetArticleInfoBatch(ctx context.Context, articleIDs []int, languageID, countryID int, includeHidden bool) (map[int][]models.ArticleInfoFact, error) {
	ids := uniqueSortedPositiveInts(articleIDs)
	result := make(map[int][]models.ArticleInfoFact, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	query := `
SELECT ARTICLE_INFO.AIN_ART_ID,
       COALESCE((` + BuildGetTextSubquery("ARTICLE_INFO.AIN_TYPE_DES_ID", languageID) + `),''),
       COALESCE((SELECT TMT_TEXT FROM TEXT_MODULE_TEXTS WHERE TMT_ID=ARTICLE_INFO.AIN_TMT_ID AND TMT_LNG_ID=? LIMIT 1),(SELECT TMT_TEXT FROM TEXT_MODULE_TEXTS WHERE TMT_ID=ARTICLE_INFO.AIN_TMT_ID AND TMT_LNG_ID=255 LIMIT 1),(SELECT TMT_TEXT FROM TEXT_MODULE_TEXTS WHERE TMT_ID=ARTICLE_INFO.AIN_TMT_ID AND TMT_LNG_ID=4 LIMIT 1),''),
       COALESCE(ARTICLE_INFO.AIN_SORT,0), COALESCE(ARTICLE_INFO.AIN_DISPLAY,0), ARTICLE_INFO.AIN_TMT_ID
FROM ARTICLE_INFO
WHERE ARTICLE_INFO.AIN_ART_ID IN (` + sqlPlaceholders(len(ids)) + `)
  AND (ARTICLE_INFO.AIN_CTM=0 OR EXISTS(
      SELECT 1 FROM COUNTRY_RESTRICTIONS C
      WHERE C.CNTR_CTM_ID=ARTICLE_INFO.AIN_CTM AND C.CNTR_COU_ID=?
  ))`
	args := make([]any, 0, len(ids)+2)
	args = append(args, languageID)
	for _, id := range ids {
		args = append(args, id)
	}
	args = append(args, countryID)
	if !includeHidden {
		query += " AND COALESCE(ARTICLE_INFO.AIN_DISPLAY,0)<>0"
	}
	query += " ORDER BY ARTICLE_INFO.AIN_ART_ID, ARTICLE_INFO.AIN_SORT, ARTICLE_INFO.AIN_TMT_ID"
	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var item models.ArticleInfoFact
		var display int
		if err := rows.Scan(&item.ArticleID, &item.Type, &item.Text, &item.Sort, &display, &item.TextID); err != nil {
			return nil, err
		}
		item.Display = display != 0
		result[item.ArticleID] = append(result[item.ArticleID], item)
	}
	return result, rows.Err()
}

func (q *Queries) GetArticleSupersessionsBatch(ctx context.Context, articleIDs []int, countryID int) (map[int][]models.ArticleSupersessionFact, error) {
	ids := uniqueSortedPositiveInts(articleIDs)
	result := make(map[int][]models.ArticleSupersessionFact, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	// Old/current article -> replacement.
	query := `
SELECT S.SUA_ART_ID, S.SUA_NEW_ART_ID, COALESCE(S.SUA_NUMBER,''),
       N.ART_SUP_ID, N.ART_SUP_BRAND
FROM SUPERSEDED_ARTICLES S
LEFT JOIN ARTICLES N ON N.ART_ID=S.SUA_NEW_ART_ID
WHERE S.SUA_ART_ID IN (` + sqlPlaceholders(len(ids)) + `)
  AND (S.SUA_CTM=0 OR EXISTS(SELECT 1 FROM COUNTRY_RESTRICTIONS C WHERE C.CNTR_CTM_ID=S.SUA_CTM AND C.CNTR_COU_ID=?))
ORDER BY S.SUA_ART_ID, S.SUA_NEW_ART_ID, S.SUA_NUMBER`
	args := make([]any, 0, len(ids)+1)
	for _, id := range ids {
		args = append(args, id)
	}
	args = append(args, countryID)
	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var sourceID int
		var related sql.NullInt64
		var number string
		var supplier sql.NullInt64
		var brand sql.NullString
		if err := rows.Scan(&sourceID, &related, &number, &supplier, &brand); err != nil {
			rows.Close()
			return nil, err
		}
		item := models.ArticleSupersessionFact{ArticleID: sourceID, Direction: "superseded_by", ArticleNumber: number, Brand: nullString(brand)}
		if related.Valid {
			v := int(related.Int64)
			item.RelatedArticleID = &v
		}
		if supplier.Valid {
			v := int(supplier.Int64)
			item.SupplierID = &v
		}
		result[sourceID] = append(result[sourceID], item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	// Replacement/current article -> old articles that it supersedes.
	query2 := `
SELECT S.SUA_NEW_ART_ID, S.SUA_ART_ID, O.ART_ARTICLE_NR, O.ART_SUP_ID, O.ART_SUP_BRAND
FROM SUPERSEDED_ARTICLES S
INNER JOIN ARTICLES O ON O.ART_ID=S.SUA_ART_ID
WHERE S.SUA_NEW_ART_ID IN (` + sqlPlaceholders(len(ids)) + `)
  AND (S.SUA_CTM=0 OR EXISTS(SELECT 1 FROM COUNTRY_RESTRICTIONS C WHERE C.CNTR_CTM_ID=S.SUA_CTM AND C.CNTR_COU_ID=?))
ORDER BY S.SUA_NEW_ART_ID, S.SUA_ART_ID`
	args2 := make([]any, 0, len(ids)+1)
	for _, id := range ids {
		args2 = append(args2, id)
	}
	args2 = append(args2, countryID)
	rows, err = q.db.QueryContext(ctx, query2, args2...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var sourceID, relatedID, supplierID int
		var number, brand string
		if err := rows.Scan(&sourceID, &relatedID, &number, &supplierID, &brand); err != nil {
			return nil, err
		}
		rid, sid := relatedID, supplierID
		result[sourceID] = append(result[sourceID], models.ArticleSupersessionFact{ArticleID: sourceID, Direction: "supersedes", RelatedArticleID: &rid, ArticleNumber: number, SupplierID: &sid, Brand: brand})
	}
	return result, rows.Err()
}

func (q *Queries) GetArticleCountryMetadataBatch(ctx context.Context, articleIDs []int, languageID, countryID int) (map[int]models.ArticleCountryMetadata, error) {
	ids := uniqueSortedPositiveInts(articleIDs)
	result := make(map[int]models.ArticleCountryMetadata, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	query := `
SELECT X.ACS_ART_ID,X.ACS_PACK_UNIT,X.ACS_QUANTITY_PER_UNIT,X.ACS_STATUS_DATE,
       COALESCE((` + BuildGetTextSubquery("X.ACS_STATUS_DES_ID", languageID) + `),'')
FROM ART_COUNTRY_SPECIFICS X
WHERE X.ACS_ART_ID IN (` + sqlPlaceholders(len(ids)) + `) AND X.ACS_COU_ID IN (?,0)
ORDER BY X.ACS_ART_ID,(X.ACS_COU_ID=?) DESC`
	args := make([]any, 0, len(ids)+2)
	for _, id := range ids {
		args = append(args, id)
	}
	args = append(args, countryID, countryID)
	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	seen := map[int]struct{}{}
	for rows.Next() {
		var articleID int
		var pack, qty sql.NullInt64
		var date sql.NullString
		var status string
		if err := rows.Scan(&articleID, &pack, &qty, &date, &status); err != nil {
			return nil, err
		}
		if _, ok := seen[articleID]; ok {
			continue
		}
		seen[articleID] = struct{}{}
		item := models.ArticleCountryMetadata{Status: status}
		if pack.Valid {
			v := int(pack.Int64)
			item.PackUnit = &v
		}
		if qty.Valid {
			v := int(qty.Int64)
			item.QuantityPerUnit = &v
		}
		if date.Valid {
			item.StatusDate = date.String
		}
		result[articleID] = item
	}
	return result, rows.Err()
}

// GetArticleAccessoriesBatch returns a bounded page per parent ART_ID using one
// set-based relation query plus one set-based ART_ACCS_LIST_CRITERIA hydration
// query for the returned relations. Truncation is explicit so callers can page
// /articles/{id}/accessories to complete the relation set without losing criteria.
func (q *Queries) GetArticleAccessoriesBatch(ctx context.Context, articleIDs []int, languageID, countryID, maxPerArticle int) (map[int]models.ArticleAccessoriesPage, error) {
	ids := uniqueSortedPositiveInts(articleIDs)
	result := make(map[int]models.ArticleAccessoriesPage, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	for _, id := range ids {
		result[id] = models.ArticleAccessoriesPage{
			Items:            []models.ArticleAccessory{},
			CriteriaIncluded: true,
		}
	}
	if maxPerArticle < 1 {
		maxPerArticle = 100
	}
	if maxPerArticle > 500 {
		maxPerArticle = 500
	}
	query := `
WITH ranked_accessories AS (
  SELECT
    A.ART_ACCS_PARENT_ID AS PARENT_ID,
    ART.ART_ID,
    ART.ART_ARTICLE_NR,
    ART.ART_SUP_ID,
    ART.ART_SUP_BRAND,
    CONCAT_WS(' ', (` + BuildGetTextSubquery("ART.ART_COMPLETE_DES_ID", languageID) + `), (` + BuildGetTextSubquery("ART.ART_DES_ID", languageID) + `)) AS ART_PRODUCT_NAME,
    (` + BuildGetTextSubquery("A.ART_ACCS_GROUP_NAME", languageID) + `) AS GROUP_NAME,
    A.ART_ACCS_TYPE,
    A.ART_ACCS_TYP_ID,
    COALESCE(A.ART_ACCS_QUANTITY,1) AS QUANTITY,
    COALESCE(A.ART_ACCS_PT_ID,0) AS PT_ID,
    COALESCE(A.ART_ACCS_SORT,0) AS SORT_NO,
    ROW_NUMBER() OVER (PARTITION BY A.ART_ACCS_PARENT_ID ORDER BY A.ART_ACCS_SORT,ART.ART_SUP_BRAND,ART.ART_ARTICLE_NR,ART.ART_ID,A.ART_ACCS_TYPE,A.ART_ACCS_TYP_ID) AS RN,
    COUNT(*) OVER (PARTITION BY A.ART_ACCS_PARENT_ID) AS TOTAL_COUNT
  FROM ART_ACCS_LIST A
  INNER JOIN ARTICLES ART ON ART.ART_ID=A.ART_ACCS_ID
  WHERE A.ART_ACCS_PARENT_ID IN (` + sqlPlaceholders(len(ids)) + `)
    AND EXISTS (SELECT 1 FROM ART_COUNTRY_SPECIFICS ACS WHERE ACS.ACS_ART_ID=ART.ART_ID AND ACS.ACS_COU_ID IN (?,0))
)
SELECT PARENT_ID,ART_ID,ART_ARTICLE_NR,ART_SUP_ID,ART_SUP_BRAND,ART_PRODUCT_NAME,GROUP_NAME,
       ART_ACCS_TYPE,ART_ACCS_TYP_ID,QUANTITY,PT_ID,SORT_NO,TOTAL_COUNT
FROM ranked_accessories
WHERE RN<=?
ORDER BY PARENT_ID,RN`
	args := make([]any, 0, len(ids)+2)
	for _, id := range ids {
		args = append(args, id)
	}
	args = append(args, countryID, maxPerArticle)
	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var parentID, total int
		var acc models.ArticleAccessory
		var group sql.NullString
		if err := rows.Scan(&parentID, &acc.ArticleID, &acc.ArticleNr, &acc.SupplierID, &acc.Brand, &acc.Name, &group, &acc.RelationType, &acc.RelationTypeID, &acc.Quantity, &acc.ProductTypeID, &acc.Sort, &total); err != nil {
			return nil, err
		}
		if group.Valid {
			acc.GroupName = group.String
		}
		acc.Criteria = make(map[string]string)
		acc.CriteriaDetails = []models.ArticleCriterion{}
		page := result[parentID]
		page.Items = append(page.Items, acc)
		page.Total = total
		page.Returned = len(page.Items)
		page.Truncated = page.Returned < page.Total
		result[parentID] = page
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := q.hydrateArticleAccessoriesBatchCriteria(ctx, result, languageID); err != nil {
		return nil, err
	}
	for parentID, page := range result {
		page.CriteriaIncluded = true
		result[parentID] = page
	}
	return result, nil
}

func (q *Queries) hydrateArticleAccessoriesBatchCriteria(ctx context.Context, pages map[int]models.ArticleAccessoriesPage, languageID int) error {
	type accessoryKey struct {
		articleID int
		typ       string
		typID     int
	}
	type locator struct {
		parentID int
		index    int
	}
	locators := make(map[accessoryKey][]locator)
	keys := make([]accessoryKey, 0)
	for parentID, page := range pages {
		for idx, item := range page.Items {
			key := accessoryKey{articleID: item.ArticleID, typ: item.RelationType, typID: item.RelationTypeID}
			if _, exists := locators[key]; !exists {
				keys = append(keys, key)
			}
			locators[key] = append(locators[key], locator{parentID: parentID, index: idx})
		}
	}
	if len(keys) == 0 {
		return nil
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].articleID != keys[j].articleID {
			return keys[i].articleID < keys[j].articleID
		}
		if keys[i].typ != keys[j].typ {
			return keys[i].typ < keys[j].typ
		}
		return keys[i].typID < keys[j].typID
	})
	const maxAccessoryCriteriaTupleItems = 500
	for start := 0; start < len(keys); start += maxAccessoryCriteriaTupleItems {
		end := start + maxAccessoryCriteriaTupleItems
		if end > len(keys) {
			end = len(keys)
		}
		chunk := keys[start:end]
		placeholders := make([]string, 0, len(chunk))
		args := make([]any, 0, len(chunk)*3)
		for _, key := range chunk {
			placeholders = append(placeholders, "(?,?,?)")
			args = append(args, key.articleID, key.typ, key.typID)
		}
		query := `
SELECT
  AAC.ACCS_CRIT_ART_ID,
  AAC.ACCS_CRIT_TYPE,
  AAC.ACCS_CRIT_TYP_ID,
  C.CRI_ID,
  COALESCE((` + BuildGetTextSubquery("C.CRI_DES_ID", languageID) + `), ''),
  COALESCE((` + BuildGetTextSubquery("C.CRI_SHORT_DES_ID", languageID) + `), ''),
  COALESCE(C.CRI_TYPE, ''),
  COALESCE((` + BuildGetTextSubquery("C.CRI_UNIT_DES_ID", languageID) + `), ''),
  COALESCE(AAC.ACCS_CRIT_VALUE, ''),
  COALESCE((` + BuildGetTextSubquery("AAC.ACCS_CRIT_DES_ID", languageID) + `), '')
FROM ART_ACCS_LIST_CRITERIA AAC
INNER JOIN CRITERIA C ON C.CRI_ID=AAC.ACCS_CRIT_ID
WHERE (AAC.ACCS_CRIT_ART_ID,AAC.ACCS_CRIT_TYPE,AAC.ACCS_CRIT_TYP_ID) IN (` + strings.Join(placeholders, ",") + `)
ORDER BY AAC.ACCS_CRIT_ART_ID,AAC.ACCS_CRIT_TYPE,AAC.ACCS_CRIT_TYP_ID,C.CRI_ID,AAC.ACCS_CRIT_VALUE,COALESCE(AAC.ACCS_CRIT_DES_ID,0)`
		rows, err := q.db.QueryContext(ctx, query, args...)
		if err != nil {
			return err
		}
		for rows.Next() {
			var articleID, typID int
			var typ string
			var criterion models.ArticleCriterion
			if err := rows.Scan(&articleID, &typ, &typID, &criterion.CriteriaID, &criterion.Name, &criterion.ShortName, &criterion.Type, &criterion.Unit, &criterion.RawValue, &criterion.ValueDescription); err != nil {
				_ = rows.Close()
				return err
			}
			criterion.Value = criterion.RawValue
			criterion.Display = true
			if criterion.ShortName == "" {
				criterion.ShortName = criterion.Name
			}
			displayValue := criterion.RawValue
			if criterion.ValueDescription != "" {
				displayValue = criterion.ValueDescription
			}
			if criterion.Unit != "" && displayValue != "" {
				displayValue += " " + criterion.Unit
			}
			key := accessoryKey{articleID: articleID, typ: typ, typID: typID}
			for _, loc := range locators[key] {
				page := pages[loc.parentID]
				if loc.index < 0 || loc.index >= len(page.Items) {
					continue
				}
				item := page.Items[loc.index]
				appendAccessoryCriterionDisplayValue(&item, criterion, displayValue)
				page.Items[loc.index] = item
				pages[loc.parentID] = page
			}
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return err
		}
		if err := rows.Close(); err != nil {
			return err
		}
	}
	return nil
}
