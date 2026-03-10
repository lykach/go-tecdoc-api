package database

import (
	"database/sql"
	"strings"
	"go-tecdoc-api/internal/models"
)

// SearchArticlesByNumber - пошук запчастин за номером
func (q *Queries) SearchArticlesByNumber(searchNumber string, languageID int, countryID int, limit, offset int) ([]models.SearchResult, error) {
	// Normalize search number (remove spaces, dashes, dots)
	normalizedNumber := strings.ReplaceAll(searchNumber, " ", "")
	normalizedNumber = strings.ReplaceAll(normalizedNumber, "-", "")
	normalizedNumber = strings.ReplaceAll(normalizedNumber, ".", "")
	normalizedNumber = strings.ToUpper(normalizedNumber)

	query := `
		SELECT  
			ARTICLES.ART_ID, 
			ARTICLES.ART_ARTICLE_NR,
			ARTICLES.ART_SUP_BRAND,
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
			ART_LOOKUP.ARL_TYPE AS FOUND_VIA
		FROM 
			ART_LOOKUP
			INNER JOIN ARTICLES ON ARTICLES.ART_ID = ART_LOOKUP.ARL_ART_ID
			INNER JOIN COUNTRY_RESTRICTIONS ON COUNTRY_RESTRICTIONS.CNTR_CTM_ID = ARTICLES.ART_CTM
				AND COUNTRY_RESTRICTIONS.CNTR_COU_ID = ?
		WHERE 
			ART_LOOKUP.ARL_SEARCH_NUMBER = ?
		ORDER BY 
			ARTICLES.ART_SUP_BRAND
		LIMIT ? OFFSET ?
	`

	rows, err := q.db.Query(query, languageID, languageID, countryID, normalizedNumber, limit, offset)
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

// GetArticleByID - отримати детальну інформацію про запчастину
func (q *Queries) GetArticleByID(artID int, languageID int, countryID int) (*models.ArticleDetail, error) {
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
			(SELECT GROUP_CONCAT(SUPERSEDED_ARTICLES.SUA_NUMBER)
			 FROM SUPERSEDED_ARTICLES
			 INNER JOIN COUNTRY_RESTRICTIONS ON COUNTRY_RESTRICTIONS.CNTR_CTM_ID = SUPERSEDED_ARTICLES.SUA_CTM
				AND COUNTRY_RESTRICTIONS.CNTR_COU_ID = ?
			 WHERE SUPERSEDED_ARTICLES.SUA_ART_ID = ARTICLES.ART_ID) AS SUPERSEDED_BY,
			(SELECT GROUP_CONCAT(ARTICLES.ART_ARTICLE_NR)
			 FROM SUPERSEDED_ARTICLES
			 INNER JOIN ARTICLES ON ARTICLES.ART_ID = SUPERSEDED_ARTICLES.SUA_ART_ID
			 INNER JOIN COUNTRY_RESTRICTIONS ON COUNTRY_RESTRICTIONS.CNTR_CTM_ID = SUPERSEDED_ARTICLES.SUA_CTM
				AND COUNTRY_RESTRICTIONS.CNTR_COU_ID = ?
			 WHERE SUPERSEDED_ARTICLES.SUA_NEW_ART_ID = ARTICLES.ART_ID) AS SUPERSEDED,
			(SELECT GROUP_CONCAT(CONCAT_WS(': ', ART_LOOKUP.ARL_BRA_BRAND, ART_LOOKUP.ARL_DISPLAY_NR) SEPARATOR '\n')
			 FROM ART_LOOKUP
			 WHERE ART_LOOKUP.ARL_ART_ID = ARTICLES.ART_ID AND ART_LOOKUP.ARL_TYPE = 'OENumber') AS OEM_NUMBERS,
			(SELECT GROUP_CONCAT(ART_LOOKUP.ARL_DISPLAY_NR SEPARATOR '\n')
			 FROM ART_LOOKUP
			 WHERE ART_LOOKUP.ARL_ART_ID = ARTICLES.ART_ID AND ART_LOOKUP.ARL_TYPE = 'EAN') AS EAN_NUMBERS
		FROM
			ARTICLES
			LEFT OUTER JOIN ART_COUNTRY_SPECIFICS ON ART_COUNTRY_SPECIFICS.ACS_ART_ID = ARTICLES.ART_ID
				AND (ART_COUNTRY_SPECIFICS.ACS_COU_ID = ? OR ART_COUNTRY_SPECIFICS.ACS_COU_ID = 0)
		WHERE
			ARTICLES.ART_ID = ?
	`

	var ad models.ArticleDetail
	var packUnit, quantityPerUnit sql.NullInt64
	var statusDate sql.NullString
	var supersededBy, superseded, oemNumbers, eanNumbers sql.NullString

	err := q.db.QueryRow(query, languageID, languageID, languageID, 
		countryID, countryID, countryID, artID).Scan(
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

	// Get criteria
	ad.Criteria = make(map[string]string)
	criteria, err := q.getArticleCriteria(artID, languageID)
	if err == nil {
		ad.Criteria = criteria
	}
	
	// Get media
    mediaList, err := q.GetArticleMedia(artID, languageID)
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

	// Get accessories
    accessories, err := q.GetArticleAccessories(artID, languageID, countryID)
    if err == nil {
	  ad.Accessories = accessories
    } else {
	  ad.Accessories = []models.ArticleAccessory{}
    }

	return &ad, nil
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

// GetArticleCrossReferences - отримати крос-референси запчастини
func (q *Queries) GetArticleCrossReferences(artID int, languageID int) ([]models.CrossReference, error) {
	query := `
		SELECT DISTINCT * FROM (
			(SELECT
				ARTICLES.ART_ID, 
				ARTICLES.ART_ARTICLE_NR,
				ARTICLES.ART_SUP_BRAND,
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
				'IAM' AS CROSS_TYPE
			FROM 
				ART_LOOKUP
				INNER JOIN MANUFACTURERS ON MANUFACTURERS.MFA_ID = ART_LOOKUP.ARL_BRA_ID
				INNER JOIN ART_LOOKUP AS ART_LOOKUP_CROSS ON ART_LOOKUP_CROSS.ARL_SEARCH_NUMBER = ART_LOOKUP.ARL_SEARCH_NUMBER
					AND ART_LOOKUP_CROSS.ARL_TYPE = 'ArticleNumber'
					AND ART_LOOKUP_CROSS.ARL_BRA_ID = MANUFACTURERS.MFA_SUP_ID
				INNER JOIN ARTICLES ON ARTICLES.ART_ID = ART_LOOKUP_CROSS.ARL_ART_ID
			WHERE 
				ART_LOOKUP.ARL_ART_ID = ? AND ART_LOOKUP.ARL_TYPE = 'IAMNumber')
			
			UNION ALL 
			
			(SELECT
				ARTICLES.ART_ID, 
				ARTICLES.ART_ARTICLE_NR,
				ARTICLES.ART_SUP_BRAND,
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
				'ArticleNumber' AS CROSS_TYPE
			FROM 
				ART_LOOKUP 
				INNER JOIN MANUFACTURERS ON MANUFACTURERS.MFA_SUP_ID = ART_LOOKUP.ARL_BRA_ID
				INNER JOIN ART_LOOKUP AS ART_LOOKUP_CROSS ON ART_LOOKUP_CROSS.ARL_SEARCH_NUMBER = ART_LOOKUP.ARL_SEARCH_NUMBER
					AND ART_LOOKUP_CROSS.ARL_TYPE = 'IAMNumber'
					AND ART_LOOKUP_CROSS.ARL_BRA_ID =  MANUFACTURERS.MFA_ID
				INNER JOIN ARTICLES ON ARTICLES.ART_ID = ART_LOOKUP_CROSS.ARL_ART_ID
			WHERE 
				ART_LOOKUP.ARL_ART_ID = ? AND ART_LOOKUP.ARL_TYPE = 'ArticleNumber')
			
			UNION ALL 
			
			(SELECT
				NULL, 
				ART_LOOKUP.ARL_DISPLAY_NR,
				ART_LOOKUP.ARL_BRA_BRAND,
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
				'OEM' AS CROSS_TYPE
			FROM 
				ART_LOOKUP 
				INNER JOIN ARTICLES ON ARTICLES.ART_ID = ?
			WHERE 
				ART_LOOKUP.ARL_ART_ID = ? AND ART_LOOKUP.ARL_TYPE = 'OENumber')
		) as CROSS_REFF
	`

	rows, err := q.db.Query(query, languageID, languageID, artID, 
		languageID, languageID, artID, languageID, languageID, artID, artID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var crosses []models.CrossReference
	for rows.Next() {
		var cr models.CrossReference
		var artID sql.NullInt64

		err := rows.Scan(
			&artID,
			&cr.ArticleNr,
			&cr.Brand,
			&cr.Name,
			&cr.Type,
		)
		if err != nil {
			continue
		}

		if artID.Valid {
			val := int(artID.Int64)
			cr.ArticleID = &val
		}

		crosses = append(crosses, cr)
	}

	return crosses, rows.Err()
}
// GetArticleApplicability повертає список автомобілів для яких підходить запчастина
func (q *Queries) GetArticleApplicability(articleID, langID, countryID int) ([]models.Applicability, error) {
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
(SELECT GROUP_CONCAT(ENGINES.ENG_CODE)
 FROM ENGINES
 JOIN LINK_ENGINE_TYPE ON ENGINES.ENG_ID = LINK_ENGINE_TYPE.LET_ENG_ID
 WHERE LINK_ENGINE_TYPE.LET_TYPE_ID = PASSENGER_CARS.PC_ID 
   AND LINK_ENGINE_TYPE.LET_TYPE = 'PC') AS ENGINE_CODES,
(SELECT GROUP_CONCAT(
 CONCAT_WS(': ', (` + BuildGetTextSubquery("CRITERIA.CRI_DES_ID", langID) + `),
IF(LA_CRITERIA.LAC_DES_ID IS NULL,
   LA_CRITERIA.LAC_VALUE,
   (` + BuildGetTextSubquery("LA_CRITERIA.LAC_DES_ID", langID) + `)))
 SEPARATOR '; ')
 FROM LA_CRITERIA
 INNER JOIN CRITERIA ON LA_CRITERIA.LAC_CRI_ID = CRITERIA.CRI_ID
 WHERE LA_CRITERIA.LAC_LA_ID = LINK_LA_TYP.LAT_LA_ID) AS TERMS_OF_USE
FROM
LINK_ART 
INNER JOIN LINK_LA_TYP ON LINK_LA_TYP.LAT_LA_ID = LINK_ART.LA_ID
   AND LINK_LA_TYP.LAT_TYPE = 'PC'
INNER JOIN PASSENGER_CARS ON PASSENGER_CARS.PC_ID = LINK_LA_TYP.LAT_TYP_ID
INNER JOIN PC_COUNTRY_SPECIFICS ON PC_COUNTRY_SPECIFICS.PCS_PC_ID = PASSENGER_CARS.PC_ID
   AND (PC_COUNTRY_SPECIFICS.PCS_COU_ID = ? OR PC_COUNTRY_SPECIFICS.PCS_COU_ID = 0)
INNER JOIN MODELS_SERIES ON MODELS_SERIES.MS_ID = PASSENGER_CARS.PC_MS_ID     
INNER JOIN MANUFACTURERS ON MANUFACTURERS.MFA_ID = PASSENGER_CARS.PC_MFA_ID
LEFT OUTER JOIN MS_COUNTRY_SPECIFICS
 ON MS_COUNTRY_SPECIFICS.MSCS_ID = MODELS_SERIES.MS_ID
AND MS_COUNTRY_SPECIFICS.MSCS_COU_ID = ?
WHERE
LINK_ART.LA_ART_ID = ?
ORDER BY TYPE_NAME
`

rows, err := q.db.Query(query, countryID, countryID, articleID)
if err != nil {
return nil, err
}
defer rows.Close()

var applicabilities []models.Applicability
for rows.Next() {
var a models.Applicability
var msID int
var typeName, bodyType, engineCodes, termsOfUse sql.NullString
var yearFrom, yearTo sql.NullString
var powerKW, powerPS, capacity sql.NullFloat64

err := rows.Scan(
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
)
if err != nil {
return nil, err
}

a.TypeName = nullString(typeName)

// Конвертуємо дати
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

// Парсимо коди двигунів
if engineCodes.Valid && engineCodes.String != "" {
a.EngineCodes = splitAndClean(engineCodes.String, ",")
}

a.TermsOfUse = nullString(termsOfUse)

applicabilities = append(applicabilities, a)
}

return applicabilities, rows.Err()
}


// GetArticleMedia повертає список медіа-файлів для запчастини
func (q *Queries) GetArticleMedia(articleID, langID int) ([]models.ArticleMedia, error) {
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
`

rows, err := q.db.Query(query, articleID)
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
ARTICLES.ART_SUP_BRAND,
CONCAT_WS(' ', 
(` + BuildGetTextSubquery("ARTICLES.ART_COMPLETE_DES_ID", langID) + `),
(` + BuildGetTextSubquery("ARTICLES.ART_DES_ID", langID) + `)
) AS ART_PRODUCT_NAME,
ARTICLES_PART_LIST.APL_QUANTITY AS QUANTITY,
ARTICLES_PART_LIST.APL_SEQ_NO AS ORDER_IN_LIST
FROM
ARTICLES_PART_LIST
INNER JOIN ARTICLES ON ARTICLES.ART_ID = ARTICLES_PART_LIST.APL_ART_ID_COMPONENT
INNER JOIN ART_COUNTRY_SPECIFICS ON ART_COUNTRY_SPECIFICS.ACS_ART_ID = ARTICLES.ART_ID
AND (ART_COUNTRY_SPECIFICS.ACS_COU_ID = ? OR ART_COUNTRY_SPECIFICS.ACS_COU_ID = 0)
WHERE
ARTICLES_PART_LIST.APL_ART_ID = ?
ORDER BY ORDER_IN_LIST
`

rows, err := q.db.Query(query, countryID, articleID)
if err != nil {
return nil, err
}
defer rows.Close()

var parts []models.ArticlePart
for rows.Next() {
var part models.ArticlePart
var artID int

err := rows.Scan(
&artID,
&part.ArticleNr,
&part.Brand,
&part.Name,
&part.Quantity,
&part.Order,
)
if err != nil {
return nil, err
}

parts = append(parts, part)
}

return parts, rows.Err()
}

// GetArticleAccessories повертає список аксесуарів для запчастини
func (q *Queries) GetArticleAccessories(articleID, langID, countryID int) ([]models.ArticleAccessory, error) {
query := `
SELECT DISTINCT
ARTICLES.ART_ID,
ARTICLES.ART_ARTICLE_NR,
ARTICLES.ART_SUP_BRAND,
CONCAT_WS(' ', 
(` + BuildGetTextSubquery("ARTICLES.ART_COMPLETE_DES_ID", langID) + `),
(` + BuildGetTextSubquery("ARTICLES.ART_DES_ID", langID) + `)
) AS ART_PRODUCT_NAME,
(` + BuildGetTextSubquery("ART_ACCS_LIST.ART_ACCS_GROUP_NAME", langID) + `) AS GROUP_NAME
FROM
ART_ACCS_LIST
INNER JOIN ARTICLES ON ARTICLES.ART_ID = ART_ACCS_LIST.ART_ACCS_ID
INNER JOIN ART_COUNTRY_SPECIFICS ON ART_COUNTRY_SPECIFICS.ACS_ART_ID = ARTICLES.ART_ID
AND (ART_COUNTRY_SPECIFICS.ACS_COU_ID = ? OR ART_COUNTRY_SPECIFICS.ACS_COU_ID = 0)
WHERE
ART_ACCS_LIST.ART_ACCS_PARENT_ID = ?
`

rows, err := q.db.Query(query, countryID, articleID)
if err != nil {
return nil, err
}
defer rows.Close()

var accessories []models.ArticleAccessory
for rows.Next() {
var acc models.ArticleAccessory
var groupName sql.NullString

err := rows.Scan(
&acc.ArticleID,
&acc.ArticleNr,
&acc.Brand,
&acc.Name,
&groupName,
)
if err != nil {
return nil, err
}

if groupName.Valid {
acc.GroupName = groupName.String
}

// TODO: Get criteria if needed
acc.Criteria = make(map[string]string)

accessories = append(accessories, acc)
}

return accessories, rows.Err()
}

// GetArticleOEMNumbers повертає список OEM номерів для запчастини
func (q *Queries) GetArticleOEMNumbers(articleID int) ([]models.OEMNumber, error) {
query := `
SELECT 
ART_LOOKUP.ARL_DISPLAY_NR AS OEM_NUMBER,
ART_LOOKUP.ARL_BRA_BRAND AS BRAND,
ART_LOOKUP.ARL_BRA_ID AS BRAND_ID
FROM 
ART_LOOKUP
WHERE 
ART_LOOKUP.ARL_ART_ID = ?
AND ART_LOOKUP.ARL_TYPE = 'OENumber'
ORDER BY 
ART_LOOKUP.ARL_BRA_BRAND, ART_LOOKUP.ARL_DISPLAY_NR
`

rows, err := q.db.Query(query, articleID)
if err != nil {
return nil, err
}
defer rows.Close()

var oemNumbers []models.OEMNumber
for rows.Next() {
var oem models.OEMNumber

err := rows.Scan(
&oem.Number,
&oem.Brand,
&oem.BrandID,
)
if err != nil {
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
INNER JOIN ART_MEDIA_INFO ON ART_MEDIA_INFO.ART_MEDIA_ID = SENSITIVE_COORDINATES.SEN_COORD_MEDIA_ID
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

// GetArticleCriteria повертає критерії (характеристики) запчастини
func (q *Queries) GetArticleCriteria(articleID int, languageID int) ([]models.ArticleCriterion, error) {
query := `
SELECT 
ARTICLE_CRITERIA.ACR_CRI_ID,
(` + BuildGetTextSubquery("CRITERIA.CRI_DES_ID", languageID) + `) AS CRI_NAME,
(` + BuildGetTextSubquery("CRITERIA.CRI_SHORT_DES_ID", languageID) + `) AS CRI_SHORT_NAME,
ARTICLE_CRITERIA.ACR_VALUE,
(` + BuildGetTextSubquery("ARTICLE_CRITERIA.ACR_DES_ID", languageID) + `) AS CRI_VALUE_DES,
CRITERIA.CRI_TYPE
FROM 
ARTICLE_CRITERIA
INNER JOIN CRITERIA ON CRITERIA.CRI_ID = ARTICLE_CRITERIA.ACR_CRI_ID
WHERE 
ARTICLE_CRITERIA.ACR_ART_ID = ?
AND ARTICLE_CRITERIA.ACR_DISPLAY = 1
ORDER BY 
ARTICLE_CRITERIA.ACR_CRI_ID
`

rows, err := q.db.Query(query, articleID)
if err != nil {
return nil, err
}
defer rows.Close()

var criteria []models.ArticleCriterion
for rows.Next() {
var ac models.ArticleCriterion
var valueDes sql.NullString
var shortName sql.NullString

err := rows.Scan(
&ac.CriteriaID,
&ac.Name,
&shortName,
&ac.Value,
&valueDes,
&ac.Type,
)
if err != nil {
return nil, err
}

if shortName.Valid && shortName.String != "" {
ac.ShortName = shortName.String
} else {
ac.ShortName = ac.Name
}

if valueDes.Valid && valueDes.String != "" {
ac.ValueDescription = valueDes.String
}

criteria = append(criteria, ac)
}

return criteria, rows.Err()
}
