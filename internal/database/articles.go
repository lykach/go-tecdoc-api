package database

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"go-tecdoc-api/internal/models"
)

// SearchArticlesByNumber searches every TecDoc lookup type for a number.
// This broad endpoint is useful for diagnostics, EAN lookups and legacy
// clients, but it must not be used to choose the source article for an analog
// chain because an OE/IAM number can point at another manufacturer's article.
func (q *Queries) SearchArticlesByNumber(ctx context.Context, searchNumber string, languageID int, countryID int, limit, offset int) ([]models.SearchResult, error) {
	return q.searchArticlesByNumber(ctx, searchNumber, languageID, countryID, limit, offset, false)
}

// SearchExactArticlesByNumber returns only articles whose own TecDoc
// ArticleNumber equals the requested number. This is the safe discovery step
// for resolving a concrete manufacturer+article source before cross-reference
// lookup.
func (q *Queries) SearchExactArticlesByNumber(ctx context.Context, searchNumber string, languageID int, countryID int, limit, offset int) ([]models.SearchResult, error) {
	return q.searchArticlesByNumber(ctx, searchNumber, languageID, countryID, limit, offset, true)
}

func (q *Queries) searchArticlesByNumber(ctx context.Context, searchNumber string, languageID int, countryID int, limit, offset int, exactArticleOnly bool) ([]models.SearchResult, error) {
	normalizedNumber := normalizeSearchNumber(searchNumber)
	lookupTypeFilter := ""
	if exactArticleOnly {
		lookupTypeFilter = " AND ART_LOOKUP.ARL_TYPE = 'ArticleNumber'"
	}

	query := `
		SELECT DISTINCT
			ARTICLES.ART_ID,
			ARTICLES.ART_ARTICLE_NR,
			ARTICLES.ART_SUP_BRAND,
			ARTICLES.ART_SUP_ID,
			CONCAT_WS(' ',
				COALESCE(
					(SELECT DES_TEXTS.TEX_TEXT
					 FROM TEXT_DESIGNATIONS
					 INNER JOIN DES_TEXTS ON DES_TEXTS.TEX_ID = TEXT_DESIGNATIONS.DES_TEX_ID
					 WHERE TEXT_DESIGNATIONS.DES_ID = ARTICLES.ART_COMPLETE_DES_ID
					 AND TEXT_DESIGNATIONS.DES_LNG_ID = ?
					 GROUP BY TEXT_DESIGNATIONS.DES_ID LIMIT 1),
					(SELECT DES_TEXTS.TEX_TEXT
					 FROM TEXT_DESIGNATIONS
					 INNER JOIN DES_TEXTS ON DES_TEXTS.TEX_ID = TEXT_DESIGNATIONS.DES_TEX_ID
					 WHERE TEXT_DESIGNATIONS.DES_ID = ARTICLES.ART_COMPLETE_DES_ID
					 AND TEXT_DESIGNATIONS.DES_LNG_ID = 4
					 GROUP BY TEXT_DESIGNATIONS.DES_ID LIMIT 1),
					''
				),
				COALESCE(
					(SELECT DES_TEXTS.TEX_TEXT
					 FROM TEXT_DESIGNATIONS
					 INNER JOIN DES_TEXTS ON DES_TEXTS.TEX_ID = TEXT_DESIGNATIONS.DES_TEX_ID
					 WHERE TEXT_DESIGNATIONS.DES_ID = ARTICLES.ART_DES_ID
					 AND TEXT_DESIGNATIONS.DES_LNG_ID = ?
					 GROUP BY TEXT_DESIGNATIONS.DES_ID LIMIT 1),
					(SELECT DES_TEXTS.TEX_TEXT
					 FROM TEXT_DESIGNATIONS
					 INNER JOIN DES_TEXTS ON DES_TEXTS.TEX_ID = TEXT_DESIGNATIONS.DES_TEX_ID
					 WHERE TEXT_DESIGNATIONS.DES_ID = ARTICLES.ART_DES_ID
					 AND TEXT_DESIGNATIONS.DES_LNG_ID = 4
					 GROUP BY TEXT_DESIGNATIONS.DES_ID LIMIT 1),
					''
				)
			) AS ART_PRODUCT_NAME,
			ART_LOOKUP.ARL_TYPE AS FOUND_VIA,
			'ArticleNumber' AS RELATION_TYPE
		FROM ART_LOOKUP
		INNER JOIN ARTICLES ON ARTICLES.ART_ID = ART_LOOKUP.ARL_ART_ID
		WHERE (ARTICLES.ART_CTM = 0 OR EXISTS (
			SELECT 1 FROM COUNTRY_RESTRICTIONS
			WHERE COUNTRY_RESTRICTIONS.CNTR_CTM_ID = ARTICLES.ART_CTM
				AND COUNTRY_RESTRICTIONS.CNTR_COU_ID = ?
		))
			AND ART_LOOKUP.ARL_SEARCH_NUMBER = ?` + lookupTypeFilter + `
		ORDER BY ARTICLES.ART_SUP_BRAND, ARTICLES.ART_ARTICLE_NR, ARTICLES.ART_ID
		LIMIT ? OFFSET ?
	`

	rows, err := q.db.QueryContext(ctx, query, languageID, languageID, countryID, normalizedNumber, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]models.SearchResult, 0)
	for rows.Next() {
		var sr models.SearchResult
		if err := rows.Scan(
			&sr.ArticleID,
			&sr.ArticleNr,
			&sr.Brand,
			&sr.SupplierID,
			&sr.Name,
			&sr.FoundVia,
			&sr.RelationType,
		); err != nil {
			return nil, err
		}
		results = append(results, sr)
	}
	return results, rows.Err()
}

// GetArticleByID - отримати детальну інформацію про запчастину
func (q *Queries) GetArticleByID(ctx context.Context, artID int, languageID int, countryID int, relatedLimit int) (*models.ArticleDetail, error) {
	if relatedLimit < 1 {
		relatedLimit = 50
	}
	if relatedLimit > 100 {
		relatedLimit = 100
	}
	query := `
		SELECT
			ARTICLES.ART_ID,
			ARTICLES.ART_ARTICLE_NR,
			ARTICLES.ART_SUP_BRAND,
			ARTICLES.ART_SUP_ID,
			CONCAT_WS(' ',
				COALESCE(
					(SELECT DES_TEXTS.TEX_TEXT
					 FROM TEXT_DESIGNATIONS
					 INNER JOIN DES_TEXTS ON DES_TEXTS.TEX_ID = TEXT_DESIGNATIONS.DES_TEX_ID
					 WHERE TEXT_DESIGNATIONS.DES_ID = ARTICLES.ART_COMPLETE_DES_ID
					 AND TEXT_DESIGNATIONS.DES_LNG_ID = ?
					 GROUP BY TEXT_DESIGNATIONS.DES_ID LIMIT 1),
					''
				),
				COALESCE(
					(SELECT DES_TEXTS.TEX_TEXT
					 FROM TEXT_DESIGNATIONS
					 INNER JOIN DES_TEXTS ON DES_TEXTS.TEX_ID = TEXT_DESIGNATIONS.DES_TEX_ID
					 WHERE TEXT_DESIGNATIONS.DES_ID = ARTICLES.ART_DES_ID
					 AND TEXT_DESIGNATIONS.DES_LNG_ID = ?
					 GROUP BY TEXT_DESIGNATIONS.DES_ID LIMIT 1),
					''
				)
			) AS ART_PRODUCT_NAME,
			ART_COUNTRY_SPECIFICS.ACS_PACK_UNIT,
			ART_COUNTRY_SPECIFICS.ACS_QUANTITY_PER_UNIT,
			ART_COUNTRY_SPECIFICS.ACS_STATUS_DATE,
			COALESCE(
				(SELECT DES_TEXTS.TEX_TEXT
				 FROM TEXT_DESIGNATIONS
				 INNER JOIN DES_TEXTS ON DES_TEXTS.TEX_ID = TEXT_DESIGNATIONS.DES_TEX_ID
				 WHERE TEXT_DESIGNATIONS.DES_ID = ART_COUNTRY_SPECIFICS.ACS_STATUS_DES_ID
				 AND TEXT_DESIGNATIONS.DES_LNG_ID = ?
				 GROUP BY TEXT_DESIGNATIONS.DES_ID LIMIT 1),
				''
			) AS ART_STATUS_TEXT,
			(SELECT GROUP_CONCAT(SUPERSEDED_ARTICLES.SUA_NUMBER ORDER BY SUPERSEDED_ARTICLES.SUA_NUMBER SEPARATOR ',')
			 FROM SUPERSEDED_ARTICLES
			 WHERE SUPERSEDED_ARTICLES.SUA_ART_ID = ARTICLES.ART_ID
				AND (SUPERSEDED_ARTICLES.SUA_CTM = 0 OR EXISTS (
					SELECT 1 FROM COUNTRY_RESTRICTIONS
					WHERE COUNTRY_RESTRICTIONS.CNTR_CTM_ID = SUPERSEDED_ARTICLES.SUA_CTM
						AND COUNTRY_RESTRICTIONS.CNTR_COU_ID = ?
				))) AS SUPERSEDED_BY,
			(SELECT GROUP_CONCAT(OLD_ARTICLES.ART_ARTICLE_NR ORDER BY OLD_ARTICLES.ART_ARTICLE_NR SEPARATOR ',')
			 FROM SUPERSEDED_ARTICLES
			 INNER JOIN ARTICLES AS OLD_ARTICLES ON OLD_ARTICLES.ART_ID = SUPERSEDED_ARTICLES.SUA_ART_ID
			 WHERE SUPERSEDED_ARTICLES.SUA_NEW_ART_ID = ARTICLES.ART_ID
				AND (SUPERSEDED_ARTICLES.SUA_CTM = 0 OR EXISTS (
					SELECT 1 FROM COUNTRY_RESTRICTIONS
					WHERE COUNTRY_RESTRICTIONS.CNTR_CTM_ID = SUPERSEDED_ARTICLES.SUA_CTM
						AND COUNTRY_RESTRICTIONS.CNTR_COU_ID = ?
				))) AS SUPERSEDED,
			(SELECT GROUP_CONCAT(CONCAT_WS(': ', ART_LOOKUP.ARL_BRA_BRAND, ART_LOOKUP.ARL_DISPLAY_NR) ORDER BY ART_LOOKUP.ARL_BRA_BRAND, ART_LOOKUP.ARL_DISPLAY_NR SEPARATOR '\n')
			 FROM ART_LOOKUP
			 WHERE ART_LOOKUP.ARL_ART_ID = ARTICLES.ART_ID AND ART_LOOKUP.ARL_TYPE = 'OENumber') AS OEM_NUMBERS,
			(SELECT GROUP_CONCAT(ART_LOOKUP.ARL_DISPLAY_NR ORDER BY ART_LOOKUP.ARL_DISPLAY_NR SEPARATOR '\n')
			 FROM ART_LOOKUP
			 WHERE ART_LOOKUP.ARL_ART_ID = ARTICLES.ART_ID AND ART_LOOKUP.ARL_TYPE = 'EAN') AS EAN_NUMBERS
		FROM
			ARTICLES
			LEFT OUTER JOIN ART_COUNTRY_SPECIFICS ON ART_COUNTRY_SPECIFICS.ACS_ART_ID = ARTICLES.ART_ID
				AND (ART_COUNTRY_SPECIFICS.ACS_COU_ID = ? OR ART_COUNTRY_SPECIFICS.ACS_COU_ID = 0)
		WHERE
			ARTICLES.ART_ID = ?
		ORDER BY (ART_COUNTRY_SPECIFICS.ACS_COU_ID = ?) DESC
		LIMIT 1
	`

	var ad models.ArticleDetail
	var packUnit, quantityPerUnit sql.NullInt64
	var statusDate sql.NullString
	var supersededBy, superseded, oemNumbers, eanNumbers sql.NullString

	err := q.db.QueryRowContext(ctx, query, languageID, languageID, languageID,
		countryID, countryID, countryID, artID, countryID).Scan(
		&ad.ID,
		&ad.ArticleNr,
		&ad.Brand,
		&ad.SupplierID,
		&ad.Name,
		&packUnit,
		&quantityPerUnit,
		&statusDate,
		&ad.Status,
		&supersededBy,
		&superseded,
		&oemNumbers,
		&eanNumbers,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Parse optional fields
	if packUnit.Valid {
		val := int(packUnit.Int64)
		ad.PackUnit = &val
	}
	if quantityPerUnit.Valid {
		val := int(quantityPerUnit.Int64)
		ad.Quantity = &val
	}
	if statusDate.Valid {
		ad.StatusDate = &statusDate.String
	}

	// Parse lists
	if supersededBy.Valid && supersededBy.String != "" {
		ad.SupersededBy = strings.Split(supersededBy.String, ",")
	} else {
		ad.SupersededBy = []string{}
	}

	if superseded.Valid && superseded.String != "" {
		ad.Superseded = strings.Split(superseded.String, ",")
	} else {
		ad.Superseded = []string{}
	}

	if oemNumbers.Valid && oemNumbers.String != "" {
		ad.OEMNumbers = strings.Split(oemNumbers.String, "\n")
	} else {
		ad.OEMNumbers = []string{}
	}

	if eanNumbers.Valid && eanNumbers.String != "" {
		ad.EANNumbers = strings.Split(eanNumbers.String, "\n")
	} else {
		ad.EANNumbers = []string{}
	}

	// The detail endpoint represents TecDoc's full second/article-detail page.
	// Keep every ARTICLE_CRITERIA row, including ACR_DISPLAY=0 detail-only
	// criteria, and expose a structured lossless view alongside the legacy map.
	ad.Criteria = make(map[string]string)
	ad.CriteriaDetails = []models.ArticleCriterion{}
	if criteriaByArticle, criteriaErr := q.GetArticleCriteriaBatch(ctx, []int{artID}, languageID, countryID, true); criteriaErr == nil {
		ad.CriteriaDetails = criteriaByArticle[artID]
		ad.Criteria = flattenArticleCriteria(ad.CriteriaDetails)
	}

	// TecDoc ARTICLE_INFO / TEXT_MODULE_TEXTS contains the descriptive text
	// that is not represented by normal article criteria. Keep the legacy
	// flat Info field populated while /articles/{id}/info exposes the
	// structured records.
	infoCtx, infoCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer infoCancel()
	if infoItems, infoErr := q.GetArticleInfo(infoCtx, artID, languageID, countryID, true); infoErr == nil {
		infoLines := make([]string, 0, len(infoItems))
		for _, item := range infoItems {
			if strings.TrimSpace(item.Text) == "" {
				continue
			}
			if strings.TrimSpace(item.Type) != "" {
				infoLines = append(infoLines, strings.TrimSpace(item.Type)+": "+strings.TrimSpace(item.Text))
			} else {
				infoLines = append(infoLines, strings.TrimSpace(item.Text))
			}
		}
		ad.Info = strings.Join(infoLines, "\n")
	}

	// Get media
	mediaList, err := q.GetArticleMedia(artID, languageID, countryID)
	if err == nil {
		for _, media := range mediaList {
			switch media.Type {
			case "JPEG", "PNG", "GIF", "BMP", "TIFF":
				ad.Images = append(ad.Images, media)
			case "PDF":
				ad.Documents = append(ad.Documents, media)
			}
		}
	} else {
		// Initialize empty slices if error
		ad.Images = []models.ArticleMedia{}
		ad.Documents = []models.ArticleMedia{}
	}

	// Get components
	components, err := q.GetArticleComponents(artID, languageID, countryID)
	if err == nil {
		ad.PartsList = components
	} else {
		ad.PartsList = []models.ArticlePart{}
	}

	// Keep the nested detail payload bounded. The dedicated /accessories
	// endpoint exposes the complete relation set with pagination.
	accessoryCtx, accessoryCancel := context.WithTimeout(ctx, 10*time.Second)
	defer accessoryCancel()
	accessories, accessoriesTotal, err := q.GetArticleAccessoriesPage(accessoryCtx, artID, languageID, countryID, relatedLimit, 0)
	if err == nil {
		ad.Accessories = accessories
		ad.AccessoriesTotal = accessoriesTotal
		ad.AccessoriesReturned = len(accessories)
		ad.AccessoriesTruncated = len(accessories) < accessoriesTotal
	} else {
		ad.Accessories = []models.ArticleAccessory{}
	}

	return &ad, nil
}

func flattenArticleCriteria(items []models.ArticleCriterion) map[string]string {
	out := make(map[string]string)
	seen := make(map[string]map[string]struct{})
	for _, item := range items {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			name = strings.TrimSpace(item.ShortName)
		}
		value := strings.TrimSpace(item.ValueDescription)
		if value == "" {
			value = strings.TrimSpace(item.RawValue)
		}
		if value == "" {
			value = strings.TrimSpace(item.Value)
		}
		if strings.EqualFold(strings.TrimSpace(item.Type), "NoValue") && value == "" {
			value = "true"
		}
		if name == "" || value == "" {
			continue
		}
		if unit := strings.TrimSpace(item.Unit); unit != "" && !strings.HasSuffix(value, " "+unit) {
			value += " " + unit
		}
		if seen[name] == nil {
			seen[name] = make(map[string]struct{})
		}
		if _, ok := seen[name][value]; ok {
			continue
		}
		seen[name][value] = struct{}{}
		if current := out[name]; current != "" {
			out[name] = current + "; " + value
		} else {
			out[name] = value
		}
	}
	return out
}

// getArticleCriteria - отримати критерії запчастини (внутрішня функція)
func (q *Queries) getArticleCriteria(artID int, languageID int) (map[string]string, error) {
	query := `
		SELECT
			COALESCE(
				(SELECT DES_TEXTS.TEX_TEXT
				 FROM TEXT_DESIGNATIONS
				 INNER JOIN DES_TEXTS ON DES_TEXTS.TEX_ID = TEXT_DESIGNATIONS.DES_TEX_ID
				 WHERE TEXT_DESIGNATIONS.DES_ID = CRITERIA.CRI_DES_ID
				 AND TEXT_DESIGNATIONS.DES_LNG_ID = ?
				 GROUP BY TEXT_DESIGNATIONS.DES_ID LIMIT 1),
				'Unknown'
			) AS CRITERIA_NAME,
			COALESCE(
				IF(ARTICLE_CRITERIA.ACR_DES_ID IS NULL,
					ARTICLE_CRITERIA.ACR_VALUE,
					(SELECT DES_TEXTS.TEX_TEXT
					 FROM TEXT_DESIGNATIONS
					 INNER JOIN DES_TEXTS ON DES_TEXTS.TEX_ID = TEXT_DESIGNATIONS.DES_TEX_ID
					 WHERE TEXT_DESIGNATIONS.DES_ID = ARTICLE_CRITERIA.ACR_DES_ID
					 AND TEXT_DESIGNATIONS.DES_LNG_ID = ?
					 GROUP BY TEXT_DESIGNATIONS.DES_ID LIMIT 1)
				),
				''
			) AS CRITERIA_VALUE
		FROM
			ARTICLE_CRITERIA
			INNER JOIN CRITERIA ON ARTICLE_CRITERIA.ACR_CRI_ID = CRITERIA.CRI_ID
		WHERE
			ARTICLE_CRITERIA.ACR_ART_ID = ?
		ORDER BY CRITERIA.CRI_ID, ARTICLE_CRITERIA.ACR_VALUE, COALESCE(ARTICLE_CRITERIA.ACR_DES_ID, 0)
	`

	rows, err := q.db.Query(query, languageID, languageID, artID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	criteria := make(map[string]string)
	for rows.Next() {
		var name, value string
		if err := rows.Scan(&name, &value); err != nil {
			continue
		}
		if name != "" && value != "" {
			criteria[name] = value
		}
	}

	return criteria, rows.Err()
}

// GetArticleCrossReferences returns only explicit TecDoc relationships of one
// concrete ART_ID. It never infers a cross merely because another supplier
// happens to use the same normalized article number.
func (q *Queries) GetArticleCrossReferences(ctx context.Context, artID int, languageID int, countryID int) ([]models.CrossReference, error) {
	articleName := `CONCAT_WS(' ', (` + BuildGetTextSubquery("ARTICLES.ART_COMPLETE_DES_ID", languageID) + `), (` + BuildGetTextSubquery("ARTICLES.ART_DES_ID", languageID) + `))`
	sourceArticleName := `CONCAT_WS(' ', (` + BuildGetTextSubquery("SOURCE_ARTICLE.ART_COMPLETE_DES_ID", languageID) + `), (` + BuildGetTextSubquery("SOURCE_ARTICLE.ART_DES_ID", languageID) + `))`

	query := `
		SELECT DISTINCT
			ART_ID,
			ART_ARTICLE_NR,
			LEGACY_BRAND,
			SUPPLIER_ID,
			SUPPLIER_NAME,
			MANUFACTURER_ID,
			MANUFACTURER_NAME,
			ART_PRODUCT_NAME,
			CROSS_TYPE
		FROM (
			SELECT
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
				AND SOURCE_LOOKUP.ARL_ART_ID = ?
				AND SOURCE_LOOKUP.ARL_TYPE = 'IAMNumber'
				AND ARTICLES.ART_ID <> ?

			UNION ALL

			SELECT
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
				AND SOURCE_LOOKUP.ARL_ART_ID = ?
				AND SOURCE_LOOKUP.ARL_TYPE = 'ArticleNumber'
				AND ARTICLES.ART_ID <> ?

			UNION ALL

			SELECT
				NULL AS ART_ID,
				SOURCE_LOOKUP.ARL_DISPLAY_NR AS ART_ARTICLE_NR,
				COALESCE(
					NULLIF(TRIM(SOURCE_LOOKUP.ARL_BRA_BRAND), ''),
					NULLIF(TRIM(OE_MANUFACTURER.MFA_BRAND), ''),
					''
				) AS LEGACY_BRAND,
				NULL AS SUPPLIER_ID,
				NULL AS SUPPLIER_NAME,
				SOURCE_LOOKUP.ARL_BRA_ID AS MANUFACTURER_ID,
				COALESCE(
					NULLIF(TRIM(SOURCE_LOOKUP.ARL_BRA_BRAND), ''),
					NULLIF(TRIM(OE_MANUFACTURER.MFA_BRAND), ''),
					''
				) AS MANUFACTURER_NAME,
				` + sourceArticleName + ` AS ART_PRODUCT_NAME,
				'OEM' AS CROSS_TYPE
			FROM ART_LOOKUP AS SOURCE_LOOKUP
			INNER JOIN ARTICLES AS SOURCE_ARTICLE ON SOURCE_ARTICLE.ART_ID = ?
			LEFT JOIN MANUFACTURERS AS OE_MANUFACTURER
				ON OE_MANUFACTURER.MFA_ID = SOURCE_LOOKUP.ARL_BRA_ID
			WHERE SOURCE_LOOKUP.ARL_ART_ID = ?
				AND SOURCE_LOOKUP.ARL_TYPE = 'OENumber'
		) AS CROSS_REF
		ORDER BY CROSS_TYPE, LEGACY_BRAND, ART_ARTICLE_NR
	`

	rows, err := q.db.QueryContext(ctx, query,
		countryID, artID, artID,
		countryID, artID, artID,
		artID, artID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	crosses := make([]models.CrossReference, 0)
	for rows.Next() {
		var cr models.CrossReference
		var linkedArticleID sql.NullInt64
		var legacyBrand sql.NullString
		var supplierID sql.NullInt64
		var supplierName sql.NullString
		var manufacturerID sql.NullInt64
		var manufacturerName sql.NullString
		if err := rows.Scan(
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
		crosses = append(crosses, cr)
	}
	return crosses, rows.Err()
}

// GetArticleApplicability returns the legacy detailed passenger-car list.
// New consumers should also use GetArticleApplicabilityLinkagesPage, which
// preserves LA_ID/PT_ID and supports every TecDoc target type.
func (q *Queries) GetArticleApplicability(ctx context.Context, articleID, langID, countryID, limit, offset int) ([]models.Applicability, bool, error) {
	if limit < 1 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	if offset < 0 {
		offset = 0
	}
	query := `
SELECT DISTINCT
PASSENGER_CARS.PC_ID,
MODELS_SERIES.MS_ID,
CONCAT_WS(' ', MANUFACTURERS.MFA_BRAND,
  (` + BuildGetTextSubquery("IFNULL(MS_COUNTRY_SPECIFICS.MSCS_NAME_DES, MODELS_SERIES.MS_NAME_DES)", langID) + `),
  (` + BuildGetTextSubquery("PASSENGER_CARS.PC_MODEL_DES", langID) + `) ) AS TYPE_NAME,
PC_COUNTRY_SPECIFICS.PCS_CI_FROM AS YEAR_FROM,
PC_COUNTRY_SPECIFICS.PCS_CI_TO AS YEAR_TO,
PC_COUNTRY_SPECIFICS.PCS_POWER_KW,
PC_COUNTRY_SPECIFICS.PCS_POWER_PS,
PC_COUNTRY_SPECIFICS.PCS_CAPACITY_TECH AS CAPACITY,
(` + BuildGetTextSubquery("PC_COUNTRY_SPECIFICS.PCS_BODY_TYPE", langID) + `) AS BODY_TYPE,
(SELECT GROUP_CONCAT(DISTINCT ENGINES.ENG_CODE ORDER BY ENGINES.ENG_CODE SEPARATOR ',')
 FROM ENGINES
 JOIN LINK_ENGINE_TYPE ON ENGINES.ENG_ID = LINK_ENGINE_TYPE.LET_ENG_ID
 WHERE LINK_ENGINE_TYPE.LET_TYPE_ID = PASSENGER_CARS.PC_ID
   AND LINK_ENGINE_TYPE.LET_TYPE = 'PC') AS ENGINE_CODES,
(SELECT GROUP_CONCAT(
 CONCAT_WS(': ', (` + BuildGetTextSubquery("CRITERIA.CRI_DES_ID", langID) + `),
IF(LA_CRITERIA.LAC_DES_ID IS NULL,
   LA_CRITERIA.LAC_VALUE,
   (` + BuildGetTextSubquery("LA_CRITERIA.LAC_DES_ID", langID) + `)))
 ORDER BY LA_CRITERIA.LAC_CRI_ID, LA_CRITERIA.LAC_VALUE, COALESCE(LA_CRITERIA.LAC_DES_ID, 0)
 SEPARATOR '; ')
 FROM LA_CRITERIA
 INNER JOIN CRITERIA ON LA_CRITERIA.LAC_CRI_ID = CRITERIA.CRI_ID
 WHERE LA_CRITERIA.LAC_LA_ID = LINK_LA_TYP.LAT_LA_ID
   AND (
     LA_CRITERIA.LAC_CTM = 0
     OR EXISTS (
       SELECT 1 FROM COUNTRY_RESTRICTIONS
       WHERE COUNTRY_RESTRICTIONS.CNTR_CTM_ID = LA_CRITERIA.LAC_CTM
         AND COUNTRY_RESTRICTIONS.CNTR_COU_ID = ?
     )
   )) AS TERMS_OF_USE
FROM
LINK_ART
INNER JOIN LINK_LA_TYP ON LINK_LA_TYP.LAT_LA_ID = LINK_ART.LA_ID
   AND LINK_LA_TYP.LAT_PT_ID = LINK_ART.LA_PT_ID
   AND LINK_LA_TYP.LAT_TYPE = 'PC'
INNER JOIN PASSENGER_CARS ON PASSENGER_CARS.PC_ID = LINK_LA_TYP.LAT_TYP_ID
INNER JOIN PC_COUNTRY_SPECIFICS ON PC_COUNTRY_SPECIFICS.PCS_PC_ID = PASSENGER_CARS.PC_ID
   AND PC_COUNTRY_SPECIFICS.PCS_COU_ID = IF(
   EXISTS (SELECT 1 FROM PC_COUNTRY_SPECIFICS AS PCS_EXACT WHERE PCS_EXACT.PCS_PC_ID = PASSENGER_CARS.PC_ID AND PCS_EXACT.PCS_COU_ID = ?),
   ?, 0
)
INNER JOIN MODELS_SERIES ON MODELS_SERIES.MS_ID = PASSENGER_CARS.PC_MS_ID
INNER JOIN MANUFACTURERS ON MANUFACTURERS.MFA_ID = PASSENGER_CARS.PC_MFA_ID
LEFT OUTER JOIN MS_COUNTRY_SPECIFICS
 ON MS_COUNTRY_SPECIFICS.MSCS_ID = MODELS_SERIES.MS_ID
AND MS_COUNTRY_SPECIFICS.MSCS_COU_ID = ?
WHERE
LINK_ART.LA_ART_ID = ?
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
ORDER BY TYPE_NAME, PASSENGER_CARS.PC_ID, LINK_LA_TYP.LAT_LA_ID
LIMIT ? OFFSET ?
`

	rows, err := q.db.QueryContext(ctx, query, countryID, countryID, countryID, countryID, articleID, countryID, countryID, limit+1, offset)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	applicabilities := make([]models.Applicability, 0)
	for rows.Next() {
		var a models.Applicability
		var msID int
		var typeName, bodyType, engineCodes, termsOfUse sql.NullString
		var yearFrom, yearTo sql.NullString
		var powerKW, powerPS, capacity sql.NullFloat64

		if err := rows.Scan(
			&a.PCID,
			&msID,
			&typeName,
			&yearFrom,
			&yearTo,
			&powerKW,
			&powerPS,
			&capacity,
			&bodyType,
			&engineCodes,
			&termsOfUse,
		); err != nil {
			return nil, false, err
		}

		a.TypeName = nullString(typeName)
		if yearFrom.Valid && len(yearFrom.String) >= 4 {
			if year, err := parseYear(yearFrom.String); err == nil {
				a.YearFrom = year
			}
		}
		if yearTo.Valid && len(yearTo.String) >= 4 {
			if year, err := parseYear(yearTo.String); err == nil {
				a.YearTo = year
			}
		}
		if powerKW.Valid {
			a.PowerKW = int(powerKW.Float64)
		}
		if powerPS.Valid {
			a.PowerHP = int(powerPS.Float64)
		}
		if capacity.Valid {
			a.Capacity = int(capacity.Float64)
		}
		a.BodyType = nullString(bodyType)
		if engineCodes.Valid && engineCodes.String != "" {
			a.EngineCodes = splitAndClean(engineCodes.String, ",")
		}
		a.TermsOfUse = nullString(termsOfUse)
		applicabilities = append(applicabilities, a)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	hasMore := len(applicabilities) > limit
	if hasMore {
		applicabilities = applicabilities[:limit]
	}
	return applicabilities, hasMore, nil
}

// GetArticleMedia повертає список медіа-файлів для запчастини
func (q *Queries) GetArticleMedia(articleID, langID, countryID int) ([]models.ArticleMedia, error) {
	query := `
SELECT
ART_MEDIA_INFO.ART_MEDIA_TYPE,
(CASE
WHEN ART_MEDIA_INFO.ART_MEDIA_TYPE = 'URL'
THEN ART_MEDIA_INFO.ART_MEDIA_HIPPERLINK
WHEN ART_MEDIA_INFO.ART_MEDIA_TYPE = 'PDF'
THEN CONCAT_WS('/', 'PDF', ART_MEDIA_INFO.ART_MEDIA_SUP_ID, ART_MEDIA_INFO.ART_MEDIA_FILE_NAME)
ELSE
CONCAT_WS('/', 'IMAGE', ART_MEDIA_INFO.ART_MEDIA_SUP_ID, ART_MEDIA_INFO.ART_MEDIA_FILE_NAME)
END) AS ART_MEDIA_SOURCE,
ART_MEDIA_INFO.ART_MEDIA_SUP_ID,
(` + BuildGetTextSubquery("ART_MEDIA_INFO.ART_MEDIA_NORM_DES_ID", langID) + `) AS DESCRIPTION
FROM
ART_MEDIA_INFO
WHERE
ART_MEDIA_INFO.ART_MEDIA_ART_ID = ?
AND (ART_MEDIA_INFO.ART_MEDIA_CTM=0 OR EXISTS(
    SELECT 1 FROM COUNTRY_RESTRICTIONS C
    WHERE C.CNTR_CTM_ID=ART_MEDIA_INFO.ART_MEDIA_CTM AND C.CNTR_COU_ID=?
))
ORDER BY ART_MEDIA_INFO.ART_MEDIA_SORT, ART_MEDIA_INFO.ART_MEDIA_ID, ART_MEDIA_INFO.ART_MEDIA_TYPE, ART_MEDIA_INFO.ART_MEDIA_LNG_ID
`

	rows, err := q.db.Query(query, articleID, countryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mediaList []models.ArticleMedia
	for rows.Next() {
		var media models.ArticleMedia
		var supID int
		var description sql.NullString

		err := rows.Scan(
			&media.Type,
			&media.URL,
			&supID,
			&description,
		)
		if err != nil {
			return nil, err
		}

		if description.Valid {
			media.Description = description.String
		}

		mediaList = append(mediaList, media)
	}

	return mediaList, rows.Err()
}

// GetArticleComponents повертає список компонентів запчастини
func (q *Queries) GetArticleComponents(articleID, langID, countryID int) ([]models.ArticlePart, error) {
	query := `
SELECT DISTINCT
ARTICLES.ART_ID,
ARTICLES.ART_ARTICLE_NR,
ARTICLES.ART_SUP_ID,
ARTICLES.ART_SUP_BRAND,
CONCAT_WS(' ',
(` + BuildGetTextSubquery("ARTICLES.ART_COMPLETE_DES_ID", langID) + `),
(` + BuildGetTextSubquery("ARTICLES.ART_DES_ID", langID) + `)
) AS ART_PRODUCT_NAME,
ARTICLES_PART_LIST.APL_QUANTITY AS QUANTITY,
ARTICLES_PART_LIST.APL_SEQ_NO AS ORDER_IN_LIST,
COALESCE(ARTICLES_PART_LIST.APL_PT_ID,0) AS PRODUCT_TYPE_ID
FROM
ARTICLES_PART_LIST
INNER JOIN ARTICLES ON ARTICLES.ART_ID = ARTICLES_PART_LIST.APL_ART_ID_COMPONENT
INNER JOIN ART_COUNTRY_SPECIFICS ON ART_COUNTRY_SPECIFICS.ACS_ART_ID = ARTICLES.ART_ID
AND (ART_COUNTRY_SPECIFICS.ACS_COU_ID = ? OR ART_COUNTRY_SPECIFICS.ACS_COU_ID = 0)
WHERE
ARTICLES_PART_LIST.APL_ART_ID = ?
AND (ARTICLES_PART_LIST.APL_CTM=0 OR EXISTS(
    SELECT 1 FROM COUNTRY_RESTRICTIONS C
    WHERE C.CNTR_CTM_ID=ARTICLES_PART_LIST.APL_CTM AND C.CNTR_COU_ID=?
))
ORDER BY ORDER_IN_LIST
`

	rows, err := q.db.Query(query, countryID, articleID, countryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var parts []models.ArticlePart
	for rows.Next() {
		var part models.ArticlePart

		err := rows.Scan(
			&part.ArticleID,
			&part.ArticleNr,
			&part.SupplierID,
			&part.Brand,
			&part.Name,
			&part.Quantity,
			&part.Order,
			&part.ProductTypeID,
		)
		if err != nil {
			return nil, err
		}

		parts = append(parts, part)
	}

	return parts, rows.Err()
}

// GetArticleAccessoriesPage returns a bounded, deterministic page of accessory
// relations. Criteria are loaded only for relation keys present on the page;
// this avoids the multi-megabyte response/query fan-out seen on articles with
// hundreds of TecDoc accessory relations.
func (q *Queries) GetArticleAccessoriesPage(ctx context.Context, articleID, langID, countryID, limit, offset int) ([]models.ArticleAccessory, int, error) {
	if limit < 1 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	if offset < 0 {
		offset = 0
	}

	countQuery := `
SELECT COUNT(*)
FROM ART_ACCS_LIST
INNER JOIN ARTICLES ON ARTICLES.ART_ID = ART_ACCS_LIST.ART_ACCS_ID
WHERE ART_ACCS_LIST.ART_ACCS_PARENT_ID = ?
	AND EXISTS (
		SELECT 1 FROM ART_COUNTRY_SPECIFICS
		WHERE ART_COUNTRY_SPECIFICS.ACS_ART_ID = ARTICLES.ART_ID
			AND ART_COUNTRY_SPECIFICS.ACS_COU_ID IN (?, 0)
	)`
	var total int
	if err := q.db.QueryRowContext(ctx, countQuery, articleID, countryID).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
SELECT
	ARTICLES.ART_ID,
	ARTICLES.ART_ARTICLE_NR,
	ARTICLES.ART_SUP_ID,
	ARTICLES.ART_SUP_BRAND,
	CONCAT_WS(' ',
		(` + BuildGetTextSubquery("ARTICLES.ART_COMPLETE_DES_ID", langID) + `),
		(` + BuildGetTextSubquery("ARTICLES.ART_DES_ID", langID) + `)
	) AS ART_PRODUCT_NAME,
	(` + BuildGetTextSubquery("ART_ACCS_LIST.ART_ACCS_GROUP_NAME", langID) + `) AS GROUP_NAME,
	ART_ACCS_LIST.ART_ACCS_TYPE,
	ART_ACCS_LIST.ART_ACCS_TYP_ID,
	COALESCE(ART_ACCS_LIST.ART_ACCS_QUANTITY, 1),
	ART_ACCS_LIST.ART_ACCS_PT_ID,
	ART_ACCS_LIST.ART_ACCS_SORT
FROM ART_ACCS_LIST
INNER JOIN ARTICLES ON ARTICLES.ART_ID = ART_ACCS_LIST.ART_ACCS_ID
WHERE ART_ACCS_LIST.ART_ACCS_PARENT_ID = ?
	AND EXISTS (
		SELECT 1 FROM ART_COUNTRY_SPECIFICS
		WHERE ART_COUNTRY_SPECIFICS.ACS_ART_ID = ARTICLES.ART_ID
			AND ART_COUNTRY_SPECIFICS.ACS_COU_ID IN (?, 0)
	)
ORDER BY ART_ACCS_LIST.ART_ACCS_SORT,
	ARTICLES.ART_SUP_BRAND,
	ARTICLES.ART_ARTICLE_NR,
	ARTICLES.ART_ID,
	ART_ACCS_LIST.ART_ACCS_TYPE,
	ART_ACCS_LIST.ART_ACCS_TYP_ID
LIMIT ? OFFSET ?`

	rows, err := q.db.QueryContext(ctx, query, articleID, countryID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	type accessoryKey struct {
		articleID int
		typ       string
		typID     int
	}
	accessories := make([]models.ArticleAccessory, 0, limit)
	indexes := make(map[accessoryKey][]int)
	keys := make([]accessoryKey, 0, limit)
	seenKeys := make(map[accessoryKey]struct{})
	for rows.Next() {
		var acc models.ArticleAccessory
		var groupName sql.NullString
		if err := rows.Scan(
			&acc.ArticleID,
			&acc.ArticleNr,
			&acc.SupplierID,
			&acc.Brand,
			&acc.Name,
			&groupName,
			&acc.RelationType,
			&acc.RelationTypeID,
			&acc.Quantity,
			&acc.ProductTypeID,
			&acc.Sort,
		); err != nil {
			return nil, 0, err
		}
		if groupName.Valid {
			acc.GroupName = groupName.String
		}
		acc.Criteria = make(map[string]string)
		acc.CriteriaDetails = []models.ArticleCriterion{}
		idx := len(accessories)
		accessories = append(accessories, acc)
		key := accessoryKey{articleID: acc.ArticleID, typ: acc.RelationType, typID: acc.RelationTypeID}
		indexes[key] = append(indexes[key], idx)
		if _, ok := seenKeys[key]; !ok {
			seenKeys[key] = struct{}{}
			keys = append(keys, key)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	if len(accessories) == 0 {
		return accessories, total, nil
	}

	placeholders := make([]string, 0, len(keys))
	args := make([]any, 0, len(keys)*3)
	for _, key := range keys {
		placeholders = append(placeholders, "(?,?,?)")
		args = append(args, key.articleID, key.typ, key.typID)
	}
	criteriaQuery := `
SELECT
	ART_ACCS_LIST_CRITERIA.ACCS_CRIT_ART_ID,
	ART_ACCS_LIST_CRITERIA.ACCS_CRIT_TYPE,
	ART_ACCS_LIST_CRITERIA.ACCS_CRIT_TYP_ID,
	CRITERIA.CRI_ID,
	COALESCE((` + BuildGetTextSubquery("CRITERIA.CRI_DES_ID", langID) + `), ''),
	COALESCE((` + BuildGetTextSubquery("CRITERIA.CRI_SHORT_DES_ID", langID) + `), ''),
	COALESCE(CRITERIA.CRI_TYPE, ''),
	COALESCE((` + BuildGetTextSubquery("CRITERIA.CRI_UNIT_DES_ID", langID) + `), ''),
	COALESCE(ART_ACCS_LIST_CRITERIA.ACCS_CRIT_VALUE, ''),
	COALESCE((` + BuildGetTextSubquery("ART_ACCS_LIST_CRITERIA.ACCS_CRIT_DES_ID", langID) + `), '')
FROM ART_ACCS_LIST_CRITERIA
INNER JOIN CRITERIA ON CRITERIA.CRI_ID = ART_ACCS_LIST_CRITERIA.ACCS_CRIT_ID
WHERE (ART_ACCS_LIST_CRITERIA.ACCS_CRIT_ART_ID,
	ART_ACCS_LIST_CRITERIA.ACCS_CRIT_TYPE,
	ART_ACCS_LIST_CRITERIA.ACCS_CRIT_TYP_ID) IN (` + strings.Join(placeholders, ",") + `)
ORDER BY ART_ACCS_LIST_CRITERIA.ACCS_CRIT_ART_ID,
	ART_ACCS_LIST_CRITERIA.ACCS_CRIT_TYPE,
	ART_ACCS_LIST_CRITERIA.ACCS_CRIT_TYP_ID,
	CRITERIA.CRI_ID,
	ART_ACCS_LIST_CRITERIA.ACCS_CRIT_VALUE,
	COALESCE(ART_ACCS_LIST_CRITERIA.ACCS_CRIT_DES_ID, 0)`
	criteriaRows, err := q.db.QueryContext(ctx, criteriaQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer criteriaRows.Close()
	for criteriaRows.Next() {
		var accessoryID, typID int
		var typ string
		var criterion models.ArticleCriterion
		if err := criteriaRows.Scan(
			&accessoryID,
			&typ,
			&typID,
			&criterion.CriteriaID,
			&criterion.Name,
			&criterion.ShortName,
			&criterion.Type,
			&criterion.Unit,
			&criterion.RawValue,
			&criterion.ValueDescription,
		); err != nil {
			return nil, 0, err
		}
		criterion.Value = criterion.RawValue
		criterion.Display = true
		if criterion.ShortName == "" {
			criterion.ShortName = criterion.Name
		}
		key := accessoryKey{articleID: accessoryID, typ: typ, typID: typID}
		for _, idx := range indexes[key] {
			displayValue := criterion.RawValue
			if criterion.ValueDescription != "" {
				displayValue = criterion.ValueDescription
			}
			if criterion.Unit != "" && displayValue != "" {
				displayValue += " " + criterion.Unit
			}
			appendAccessoryCriterionDisplayValue(&accessories[idx], criterion, displayValue)
		}
	}
	if err := criteriaRows.Err(); err != nil {
		return nil, 0, err
	}
	return accessories, total, nil
}

func appendAccessoryCriterionDisplayValue(accessory *models.ArticleAccessory, criterion models.ArticleCriterion, displayValue string) {
	if accessory == nil {
		return
	}
	accessory.CriteriaDetails = append(accessory.CriteriaDetails, criterion)
	name := strings.TrimSpace(criterion.Name)
	displayValue = strings.TrimSpace(displayValue)
	if name == "" || displayValue == "" {
		return
	}
	if accessory.Criteria == nil {
		accessory.Criteria = make(map[string]string)
	}
	current := strings.TrimSpace(accessory.Criteria[name])
	if current == "" {
		accessory.Criteria[name] = displayValue
		return
	}
	for _, existing := range strings.Split(current, ";") {
		if strings.EqualFold(strings.TrimSpace(existing), displayValue) {
			return
		}
	}
	accessory.Criteria[name] = current + "; " + displayValue
}

// GetArticleAccessories is kept for internal backwards compatibility. New
// HTTP callers use GetArticleAccessoriesPage with bounded pagination.
func (q *Queries) GetArticleAccessories(articleID, langID, countryID int) ([]models.ArticleAccessory, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	items, _, err := q.GetArticleAccessoriesPage(ctx, articleID, langID, countryID, 500, 0)
	return items, err
}

// GetArticleOEMNumbers повертає список OEM номерів для запчастини
func (q *Queries) GetArticleOEMNumbers(articleID int) ([]models.OEMNumber, error) {
	query := `
SELECT
	ART_LOOKUP.ARL_DISPLAY_NR AS OEM_NUMBER,
	COALESCE(
		NULLIF(TRIM(ART_LOOKUP.ARL_BRA_BRAND), ''),
		NULLIF(TRIM(MANUFACTURERS.MFA_BRAND), ''),
		''
	) AS BRAND,
	COALESCE(ART_LOOKUP.ARL_BRA_ID, 0) AS BRAND_ID,
	COALESCE(
		NULLIF(TRIM(ART_LOOKUP.ARL_BRA_BRAND), ''),
		NULLIF(TRIM(MANUFACTURERS.MFA_BRAND), ''),
		''
	) AS MANUFACTURER,
	COALESCE(ART_LOOKUP.ARL_BRA_ID, 0) AS MANUFACTURER_ID
FROM ART_LOOKUP
LEFT JOIN MANUFACTURERS
	ON MANUFACTURERS.MFA_ID = ART_LOOKUP.ARL_BRA_ID
WHERE ART_LOOKUP.ARL_ART_ID = ?
	AND ART_LOOKUP.ARL_TYPE = 'OENumber'
ORDER BY MANUFACTURER, ART_LOOKUP.ARL_DISPLAY_NR
`

	rows, err := q.db.Query(query, articleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	oemNumbers := make([]models.OEMNumber, 0)
	for rows.Next() {
		var oem models.OEMNumber
		if err := rows.Scan(
			&oem.Number,
			&oem.Brand,
			&oem.BrandID,
			&oem.Manufacturer,
			&oem.ManufacturerID,
		); err != nil {
			return nil, err
		}
		oemNumbers = append(oemNumbers, oem)
	}

	return oemNumbers, rows.Err()
}

// GetArticleCoordinates повертає координати запчастини на зображенні
func (q *Queries) GetArticleCoordinates(articleID int) ([]models.ArticleCoordinate, error) {
	query := `
SELECT
SENSITIVE_COORDINATES.SEN_COORD_ART_ID AS ART_ID,
ARTICLES.ART_ARTICLE_NR,
ARTICLES.ART_SUP_BRAND,
ART_MEDIA_INFO.ART_MEDIA_FILE_NAME AS MEDIA_SOURCE,
SENSITIVE_COORDINATES.SEN_COORD_ID,
SENSITIVE_COORDINATES.SEN_COORD_X,
SENSITIVE_COORDINATES.SEN_COORD_Y,
SENSITIVE_COORDINATES.SEN_COORD_WIDTH,
SENSITIVE_COORDINATES.SEN_COORD_HEIGHT,
SENSITIVE_COORDINATES.SEN_COORD_TYPE
FROM
SENSITIVE_COORDINATES
INNER JOIN ART_MEDIA_INFO ON ART_MEDIA_INFO.ART_MEDIA_ART_ID = SENSITIVE_COORDINATES.SEN_COORD_ART_ID
AND ART_MEDIA_INFO.ART_MEDIA_ID = SENSITIVE_COORDINATES.SEN_COORD_MEDIA_ID
AND ART_MEDIA_INFO.ART_MEDIA_TYPE = SENSITIVE_COORDINATES.SEN_COORD_MEDIA_TYPE
INNER JOIN ARTICLES ON ARTICLES.ART_ID = SENSITIVE_COORDINATES.SEN_COORD_ART_ID
WHERE
SENSITIVE_COORDINATES.SEN_COORD_ART_ID = ?
ORDER BY
SENSITIVE_COORDINATES.SEN_COORD_SEQ_NO
`

	rows, err := q.db.Query(query, articleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var coordinates []models.ArticleCoordinate
	for rows.Next() {
		var ac models.ArticleCoordinate

		err := rows.Scan(
			&ac.ArticleID,
			&ac.ArticleNumber,
			&ac.Brand,
			&ac.MediaSource,
			&ac.CoordinateID,
			&ac.X,
			&ac.Y,
			&ac.Width,
			&ac.Height,
			&ac.Type,
		)
		if err != nil {
			return nil, err
		}

		coordinates = append(coordinates, ac)
	}

	return coordinates, rows.Err()
}

// GetArticleCriteria returns article criteria. includeHidden=false preserves the
// legacy ACR_DISPLAY=1 behavior; enrichment can request the full criterion set.
func (q *Queries) GetArticleCriteria(ctx context.Context, articleID int, languageID, countryID int, includeHidden bool) ([]models.ArticleCriterion, error) {
	all, err := q.GetArticleCriteriaBatch(ctx, []int{articleID}, languageID, countryID, includeHidden)
	if err != nil {
		return nil, err
	}
	criteria := all[articleID]
	if criteria == nil {
		criteria = []models.ArticleCriterion{}
	}
	return criteria, nil
}
