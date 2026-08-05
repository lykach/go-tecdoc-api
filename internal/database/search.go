package database

import (
	"context"
	"sort"
	"strings"

	"go-tecdoc-api/internal/models"
)

// SearchByOEMNumber searches articles explicitly linked to one OE/OEM number.
func (q *Queries) SearchByOEMNumber(ctx context.Context, oemNumber string, languageID int, countryID int, limit, offset int) ([]models.SearchResult, error) {
	normalizedNumber := normalizeSearchNumber(oemNumber)

	query := `
		SELECT DISTINCT
			ARTICLES.ART_ID,
			ARTICLES.ART_ARTICLE_NR,
			ARTICLES.ART_SUP_BRAND,
			ARTICLES.ART_SUP_ID,
			CONCAT_WS(' ',
				(` + BuildGetTextSubquery("ARTICLES.ART_COMPLETE_DES_ID", languageID) + `),
				(` + BuildGetTextSubquery("ARTICLES.ART_DES_ID", languageID) + `)
			) AS ART_PRODUCT_NAME,
			'OENumber' AS FOUND_VIA
		FROM ART_LOOKUP
		INNER JOIN ARTICLES ON ARTICLES.ART_ID = ART_LOOKUP.ARL_ART_ID
		INNER JOIN COUNTRY_RESTRICTIONS
			ON COUNTRY_RESTRICTIONS.CNTR_CTM_ID = ARTICLES.ART_CTM
			AND COUNTRY_RESTRICTIONS.CNTR_COU_ID = ?
		WHERE ART_LOOKUP.ARL_SEARCH_NUMBER = ?
			AND ART_LOOKUP.ARL_TYPE = 'OENumber'
		ORDER BY ARTICLES.ART_SUP_BRAND, ARTICLES.ART_ARTICLE_NR, ARTICLES.ART_ID
		LIMIT ? OFFSET ?
	`

	rows, err := q.db.QueryContext(ctx, query, countryID, normalizedNumber, limit, offset)
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
		); err != nil {
			return nil, err
		}
		results = append(results, sr)
	}
	return results, rows.Err()
}

// ArticleExistsForCountry verifies that the source article is available in the
// requested TecDoc country context before any cross-reference traversal.
func (q *Queries) ArticleExistsForCountry(ctx context.Context, articleID int, countryID int) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1
			FROM ARTICLES
			INNER JOIN COUNTRY_RESTRICTIONS
				ON COUNTRY_RESTRICTIONS.CNTR_CTM_ID = ARTICLES.ART_CTM
				AND COUNTRY_RESTRICTIONS.CNTR_COU_ID = ?
			WHERE ARTICLES.ART_ID = ?
		)
	`
	var exists bool
	if err := q.db.QueryRowContext(ctx, query, countryID, articleID).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

// SearchAnalogsByArticleID returns IAM articles connected through TecDoc's
// explicit cross-reference graph for one concrete source ART_ID. It does not
// join unrelated ArticleNumber rows solely by equal normalized number.
func (q *Queries) SearchAnalogsByArticleID(ctx context.Context, articleID int, languageID int, countryID int, limit, offset int) ([]models.SearchResult, error) {
	crosses, err := q.GetArticleCrossReferences(ctx, articleID, languageID, countryID)
	if err != nil {
		return nil, err
	}

	seen := make(map[int]struct{}, len(crosses))
	results := make([]models.SearchResult, 0, len(crosses))
	for _, cross := range crosses {
		if cross.ArticleID == nil || *cross.ArticleID <= 0 || strings.EqualFold(strings.TrimSpace(cross.Type), "OEM") {
			continue
		}
		if _, exists := seen[*cross.ArticleID]; exists {
			continue
		}
		seen[*cross.ArticleID] = struct{}{}
		results = append(results, models.SearchResult{
			ArticleID: *cross.ArticleID,
			ArticleNr: cross.ArticleNr,
			Brand:     cross.Brand,
			Name:      cross.Name,
			FoundVia:  "IAM",
		})
	}

	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Brand != results[j].Brand {
			return results[i].Brand < results[j].Brand
		}
		if results[i].ArticleNr != results[j].ArticleNr {
			return results[i].ArticleNr < results[j].ArticleNr
		}
		return results[i].ArticleID < results[j].ArticleID
	})
	return paginateSearchResults(results, limit, offset), nil
}

// FindExactArticleIDsBySupplier resolves a number only when both parts of the
// identity are supplied: TecDoc supplier ID + the supplier's own ArticleNumber.
func (q *Queries) FindExactArticleIDsBySupplier(ctx context.Context, searchNumber string, supplierID int, countryID int, limit int) ([]int, error) {
	normalizedNumber := normalizeSearchNumber(searchNumber)
	query := `
		SELECT DISTINCT ARTICLES.ART_ID
		FROM ART_LOOKUP
		INNER JOIN ARTICLES ON ARTICLES.ART_ID = ART_LOOKUP.ARL_ART_ID
		INNER JOIN COUNTRY_RESTRICTIONS
			ON COUNTRY_RESTRICTIONS.CNTR_CTM_ID = ARTICLES.ART_CTM
			AND COUNTRY_RESTRICTIONS.CNTR_COU_ID = ?
		WHERE ART_LOOKUP.ARL_SEARCH_NUMBER = ?
			AND ART_LOOKUP.ARL_TYPE = 'ArticleNumber'
			AND ARTICLES.ART_SUP_ID = ?
		ORDER BY ARTICLES.ART_ID
		LIMIT ?
	`
	rows, err := q.db.QueryContext(ctx, query, countryID, normalizedNumber, supplierID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]int, 0)
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// SearchOEMByOEMNumber returns other OE numbers explicitly attached to the
// same TecDoc articles as the supplied OE number.
func (q *Queries) SearchOEMByOEMNumber(ctx context.Context, oemNumber string, limit, offset int) ([]models.OEMCrossReference, error) {
	normalizedNumber := normalizeSearchNumber(oemNumber)

	query := `
		SELECT DISTINCT
			SOURCE.ARL_ART_ID AS ART_ID,
			ART_LOOKUP.ARL_BRA_BRAND AS CROSS_BRAND,
			ART_LOOKUP.ARL_DISPLAY_NR AS CROSS_NUMBER
		FROM ART_LOOKUP AS SOURCE
		INNER JOIN ART_LOOKUP
			ON ART_LOOKUP.ARL_ART_ID = SOURCE.ARL_ART_ID
			AND ART_LOOKUP.ARL_TYPE = 'OENumber'
			AND ART_LOOKUP.ARL_SEARCH_NUMBER != SOURCE.ARL_SEARCH_NUMBER
		WHERE SOURCE.ARL_SEARCH_NUMBER = ?
			AND SOURCE.ARL_TYPE = 'OENumber'
		ORDER BY ART_LOOKUP.ARL_BRA_BRAND, ART_LOOKUP.ARL_DISPLAY_NR
		LIMIT ? OFFSET ?
	`

	rows, err := q.db.QueryContext(ctx, query, normalizedNumber, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]models.OEMCrossReference, 0)
	for rows.Next() {
		var item models.OEMCrossReference
		if err := rows.Scan(&item.ArticleID, &item.CrossBrand, &item.CrossNumber); err != nil {
			return nil, err
		}
		results = append(results, item)
	}
	return results, rows.Err()
}

func paginateSearchResults(items []models.SearchResult, limit, offset int) []models.SearchResult {
	if offset < 0 {
		offset = 0
	}
	if offset >= len(items) {
		return []models.SearchResult{}
	}
	end := len(items)
	if limit > 0 && offset+limit < end {
		end = offset + limit
	}
	return items[offset:end]
}
