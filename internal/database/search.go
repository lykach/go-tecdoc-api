package database

import (
	"go-tecdoc-api/internal/models"
	"strings"
)

// SearchByOEMNumber пошук аналогів за OEM номером
func (q *Queries) SearchByOEMNumber(oemNumber string, languageID int, countryID int, limit, offset int) ([]models.SearchResult, error) {
	// Normalize OEM number (remove spaces, dashes, dots)
	normalizedNumber := strings.ReplaceAll(oemNumber, " ", "")
	normalizedNumber = strings.ReplaceAll(normalizedNumber, "-", "")
	normalizedNumber = strings.ReplaceAll(normalizedNumber, ".", "")
	normalizedNumber = strings.ToUpper(normalizedNumber)

	query := `
		SELECT DISTINCT
			ARTICLES.ART_ID,
			ARTICLES.ART_ARTICLE_NR,
			ARTICLES.ART_SUP_BRAND,
			CONCAT_WS(' ', 
				(` + BuildGetTextSubquery("ARTICLES.ART_COMPLETE_DES_ID", languageID) + `),
				(` + BuildGetTextSubquery("ARTICLES.ART_DES_ID", languageID) + `)
			) AS ART_PRODUCT_NAME,
			'OENumber' AS FOUND_VIA
		FROM 
			ART_LOOKUP
			INNER JOIN ARTICLES ON ARTICLES.ART_ID = ART_LOOKUP.ARL_ART_ID
			INNER JOIN COUNTRY_RESTRICTIONS ON COUNTRY_RESTRICTIONS.CNTR_CTM_ID = ARTICLES.ART_CTM
				AND COUNTRY_RESTRICTIONS.CNTR_COU_ID = ?
		WHERE 
			ART_LOOKUP.ARL_SEARCH_NUMBER = ?
			AND ART_LOOKUP.ARL_TYPE = 'OENumber'
		ORDER BY 
			ARTICLES.ART_SUP_BRAND, ARTICLES.ART_ARTICLE_NR
		LIMIT ? OFFSET ?
	`

	rows, err := q.db.Query(query, countryID, normalizedNumber, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.SearchResult
	for rows.Next() {
		var sr models.SearchResult

		err := rows.Scan(
			&sr.ArticleID,
			&sr.ArticleNr,
			&sr.Brand,
			&sr.Name,
			&sr.FoundVia,
		)
		if err != nil {
			return nil, err
		}

		results = append(results, sr)
	}

	return results, rows.Err()
}
// SearchAnalogsByArticleID пошук IAM аналогів за ID запчастини
func (q *Queries) SearchAnalogsByArticleID(articleID int, languageID int, countryID int, limit, offset int) ([]models.SearchResult, error) {
query := `
SELECT DISTINCT
ARTICLES.ART_ID,
ARTICLES.ART_ARTICLE_NR,
ARTICLES.ART_SUP_BRAND,
CONCAT_WS(' ', 
(` + BuildGetTextSubquery("ARTICLES.ART_COMPLETE_DES_ID", languageID) + `),
(` + BuildGetTextSubquery("ARTICLES.ART_DES_ID", languageID) + `)
) AS ART_PRODUCT_NAME,
ART_LOOKUP.ARL_TYPE AS FOUND_VIA
FROM 
ART_LOOKUP AS SOURCE
INNER JOIN ART_LOOKUP ON ART_LOOKUP.ARL_SEARCH_NUMBER = SOURCE.ARL_SEARCH_NUMBER
AND ART_LOOKUP.ARL_TYPE IN ('IAMNumber', 'ArticleNumber')
AND ART_LOOKUP.ARL_ART_ID != SOURCE.ARL_ART_ID
INNER JOIN ARTICLES ON ARTICLES.ART_ID = ART_LOOKUP.ARL_ART_ID
INNER JOIN COUNTRY_RESTRICTIONS ON COUNTRY_RESTRICTIONS.CNTR_CTM_ID = ARTICLES.ART_CTM
AND COUNTRY_RESTRICTIONS.CNTR_COU_ID = ?
WHERE 
SOURCE.ARL_ART_ID = ?
AND SOURCE.ARL_TYPE = 'ArticleNumber'
ORDER BY 
ARTICLES.ART_SUP_BRAND, ARTICLES.ART_ARTICLE_NR
LIMIT ? OFFSET ?
`

rows, err := q.db.Query(query, countryID, articleID, limit, offset)
if err != nil {
return nil, err
}
defer rows.Close()

var results []models.SearchResult
for rows.Next() {
var sr models.SearchResult

err := rows.Scan(
&sr.ArticleID,
&sr.ArticleNr,
&sr.Brand,
&sr.Name,
&sr.FoundVia,
)
if err != nil {
return nil, err
}

results = append(results, sr)
}

return results, rows.Err()
}

// SearchAnalogsByNumber пошук IAM аналогів за номером запчастини
func (q *Queries) SearchAnalogsByNumber(searchNumber string, languageID int, countryID int, limit, offset int) ([]models.SearchResult, error) {
// Normalize search number
normalizedNumber := strings.ReplaceAll(searchNumber, " ", "")
normalizedNumber = strings.ReplaceAll(normalizedNumber, "-", "")
normalizedNumber = strings.ReplaceAll(normalizedNumber, ".", "")
normalizedNumber = strings.ToUpper(normalizedNumber)

query := `
SELECT DISTINCT
ARTICLES.ART_ID,
ARTICLES.ART_ARTICLE_NR,
ARTICLES.ART_SUP_BRAND,
CONCAT_WS(' ', 
(` + BuildGetTextSubquery("ARTICLES.ART_COMPLETE_DES_ID", languageID) + `),
(` + BuildGetTextSubquery("ARTICLES.ART_DES_ID", languageID) + `)
) AS ART_PRODUCT_NAME,
ART_LOOKUP.ARL_TYPE AS FOUND_VIA
FROM 
ART_LOOKUP AS SOURCE
INNER JOIN ART_LOOKUP ON ART_LOOKUP.ARL_SEARCH_NUMBER = SOURCE.ARL_SEARCH_NUMBER
AND ART_LOOKUP.ARL_TYPE IN ('IAMNumber', 'ArticleNumber')
INNER JOIN ARTICLES ON ARTICLES.ART_ID = ART_LOOKUP.ARL_ART_ID
INNER JOIN COUNTRY_RESTRICTIONS ON COUNTRY_RESTRICTIONS.CNTR_CTM_ID = ARTICLES.ART_CTM
AND COUNTRY_RESTRICTIONS.CNTR_COU_ID = ?
WHERE 
SOURCE.ARL_SEARCH_NUMBER = ?
AND SOURCE.ARL_TYPE IN ('IAMNumber', 'ArticleNumber')
ORDER BY 
ARTICLES.ART_SUP_BRAND, ARTICLES.ART_ARTICLE_NR
LIMIT ? OFFSET ?
`

rows, err := q.db.Query(query, countryID, normalizedNumber, limit, offset)
if err != nil {
return nil, err
}
defer rows.Close()

var results []models.SearchResult
for rows.Next() {
var sr models.SearchResult

err := rows.Scan(
&sr.ArticleID,
&sr.ArticleNr,
&sr.Brand,
&sr.Name,
&sr.FoundVia,
)
if err != nil {
return nil, err
}

results = append(results, sr)
}

return results, rows.Err()
}

// SearchOEMByOEMNumber пошук OEM аналогів для OEM номера
func (q *Queries) SearchOEMByOEMNumber(oemNumber string, limit, offset int) ([]models.OEMCrossReference, error) {
// Normalize OEM number
normalizedNumber := strings.ReplaceAll(oemNumber, " ", "")
normalizedNumber = strings.ReplaceAll(normalizedNumber, "-", "")
normalizedNumber = strings.ReplaceAll(normalizedNumber, ".", "")
normalizedNumber = strings.ToUpper(normalizedNumber)

query := `
SELECT DISTINCT
SOURCE.ARL_ART_ID AS ART_ID,
ART_LOOKUP.ARL_BRA_BRAND AS CROSS_BRAND,
ART_LOOKUP.ARL_DISPLAY_NR AS CROSS_NUMBER
FROM 
ART_LOOKUP AS SOURCE
INNER JOIN ART_LOOKUP ON ART_LOOKUP.ARL_ART_ID = SOURCE.ARL_ART_ID
AND ART_LOOKUP.ARL_TYPE = 'OENumber'
AND ART_LOOKUP.ARL_SEARCH_NUMBER != SOURCE.ARL_SEARCH_NUMBER
WHERE 
SOURCE.ARL_SEARCH_NUMBER = ?
AND SOURCE.ARL_TYPE = 'OENumber'
ORDER BY 
ART_LOOKUP.ARL_BRA_BRAND, ART_LOOKUP.ARL_DISPLAY_NR
LIMIT ? OFFSET ?
`

rows, err := q.db.Query(query, normalizedNumber, limit, offset)
if err != nil {
return nil, err
}
defer rows.Close()

var results []models.OEMCrossReference
for rows.Next() {
var ocr models.OEMCrossReference

err := rows.Scan(
&ocr.ArticleID,
&ocr.CrossBrand,
&ocr.CrossNumber,
)
if err != nil {
return nil, err
}

results = append(results, ocr)
}

return results, rows.Err()
}
