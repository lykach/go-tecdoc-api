package database

import (
	"database/sql"
	"sort"

	"go-tecdoc-api/internal/models"
)

// GetProductGroups - отримати дерево категорій запчастин (тільки корені)
func (q *Queries) GetProductGroups(vehicleType string, languageID int, limit, offset int) ([]models.Category, error) {
	query := `
		SELECT
			SEARCH_TREE.STR_ID,
			SEARCH_TREE.STR_ID_PARENT,
			SEARCH_TREE.STR_LEVEL,
			SEARCH_TREE.STR_TYPE,
			(` + BuildGetTextSubquery("SEARCH_TREE.STR_DES_ID", languageID) + `) AS STR_NODE_NAME
		FROM
			SEARCH_TREE
		WHERE
			FIND_IN_SET(?, SEARCH_TREE.STR_TYPE) > 0
			AND SEARCH_TREE.STR_ID_PARENT IS NULL
		ORDER BY STR_NODE_NAME, SEARCH_TREE.STR_ID
		LIMIT ? OFFSET ?
	`

	rows, err := q.db.Query(query, vehicleType, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []models.Category
	for rows.Next() {
		var cat models.Category
		var parentID sql.NullInt64

		err := rows.Scan(
			&cat.ID,
			&parentID,
			&cat.Level,
			&cat.Type,
			&cat.Name,
		)
		if err != nil {
			return nil, err
		}

		if parentID.Valid {
			val := int(parentID.Int64)
			cat.ParentID = &val
		}

		cat.Path = cat.Name
		categories = append(categories, cat)
	}

	return categories, rows.Err()
}

// GetCategoryChildren - отримати дочірні категорії
func (q *Queries) GetCategoryChildren(parentID int, vehicleType string, languageID int) ([]models.Category, error) {
	query := `
		SELECT
			SEARCH_TREE.STR_ID,
			SEARCH_TREE.STR_ID_PARENT,
			SEARCH_TREE.STR_LEVEL,
			SEARCH_TREE.STR_TYPE,
			(` + BuildGetTextSubquery("SEARCH_TREE.STR_DES_ID", languageID) + `) AS STR_NODE_NAME
		FROM
			SEARCH_TREE
		WHERE
			FIND_IN_SET(?, SEARCH_TREE.STR_TYPE) > 0
			AND SEARCH_TREE.STR_ID_PARENT = ?
		ORDER BY STR_NODE_NAME, SEARCH_TREE.STR_ID
	`

	rows, err := q.db.Query(query, vehicleType, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []models.Category
	for rows.Next() {
		var cat models.Category
		var parentIDVal sql.NullInt64

		err := rows.Scan(
			&cat.ID,
			&parentIDVal,
			&cat.Level,
			&cat.Type,
			&cat.Name,
		)
		if err != nil {
			return nil, err
		}

		if parentIDVal.Valid {
			val := int(parentIDVal.Int64)
			cat.ParentID = &val
		}

		cat.Path = cat.Name
		categories = append(categories, cat)
	}

	return categories, rows.Err()
}

// GetCarProductGroups - отримати категорії запчастин для конкретного автомобіля
func (q *Queries) GetCarProductGroups(pcID int, vehicleType string, languageID int, limit, offset int) ([]models.Category, error) {
	query := `
		SELECT DISTINCT
			SEARCH_TREE.STR_ID,
			SEARCH_TREE.STR_ID_PARENT,
			SEARCH_TREE.STR_LEVEL,
			SEARCH_TREE.STR_TYPE,
			(` + BuildGetTextSubquery("SEARCH_TREE.STR_DES_ID", languageID) + `) AS STR_NODE_NAME
		FROM
			SEARCH_TREE
			INNER JOIN LINK_PT_STR ON FIND_IN_SET(?, LINK_PT_STR.STR_TYPE) > 0
				AND LINK_PT_STR.STR_ID = SEARCH_TREE.STR_ID
				AND LINK_PT_STR.PT_ID IN (
					SELECT DISTINCT LINK_LA_TYP.LAT_PT_ID
					FROM LINK_LA_TYP
					WHERE LINK_LA_TYP.LAT_TYP_ID = ?
					AND LINK_LA_TYP.LAT_TYPE = ?
				)
		WHERE
			FIND_IN_SET(?, SEARCH_TREE.STR_TYPE) > 0
			AND SEARCH_TREE.STR_ID_PARENT IS NULL
		ORDER BY STR_NODE_NAME, SEARCH_TREE.STR_ID
		LIMIT ? OFFSET ?
	`

	rows, err := q.db.Query(query, vehicleType, pcID, vehicleType, vehicleType, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []models.Category
	for rows.Next() {
		var cat models.Category
		var parentID sql.NullInt64

		err := rows.Scan(
			&cat.ID,
			&parentID,
			&cat.Level,
			&cat.Type,
			&cat.Name,
		)
		if err != nil {
			return nil, err
		}

		if parentID.Valid {
			val := int(parentID.Int64)
			cat.ParentID = &val
		}

		cat.Path = cat.Name
		categories = append(categories, cat)
	}

	return categories, rows.Err()
}

// productGroupArticleScope keeps list and count on the same vehicle, country
// and criterion boundary. CTM values identify restriction sets, not bit masks.
func productGroupArticleScope(strID, carID int, vehicleType string, countryID int, criteria map[int]string) (string, []interface{}) {
	scope := `
        FROM SEARCH_TREE
        INNER JOIN LINK_PT_STR ON LINK_PT_STR.STR_ID = SEARCH_TREE.STR_ID
            AND FIND_IN_SET(?, LINK_PT_STR.STR_TYPE) > 0
        INNER JOIN LINK_LA_TYP ON LINK_LA_TYP.LAT_TYP_ID = ?
            AND LINK_LA_TYP.LAT_TYPE = ?
            AND LINK_LA_TYP.LAT_PT_ID = LINK_PT_STR.PT_ID
        INNER JOIN LINK_ART ON LINK_ART.LA_ID = LINK_LA_TYP.LAT_LA_ID
            AND LINK_ART.LA_PT_ID = LINK_LA_TYP.LAT_PT_ID
        INNER JOIN ARTICLES ON ARTICLES.ART_ID = LINK_ART.LA_ART_ID
        LEFT JOIN ART_COUNTRY_SPECIFICS ON ART_COUNTRY_SPECIFICS.ACS_ART_ID = ARTICLES.ART_ID
            AND ART_COUNTRY_SPECIFICS.ACS_COU_ID = IF(
                EXISTS (SELECT 1 FROM ART_COUNTRY_SPECIFICS AS ACS_EXACT
                    WHERE ACS_EXACT.ACS_ART_ID = ARTICLES.ART_ID AND ACS_EXACT.ACS_COU_ID = ?), ?, 0)
        WHERE SEARCH_TREE.STR_ID = ?
            AND FIND_IN_SET(?, SEARCH_TREE.STR_TYPE) > 0
            AND (ARTICLES.ART_CTM = 0 OR EXISTS (SELECT 1 FROM COUNTRY_RESTRICTIONS
                WHERE CNTR_CTM_ID = ARTICLES.ART_CTM AND CNTR_COU_ID = ?))
            AND (LINK_ART.LA_CTM = 0 OR EXISTS (SELECT 1 FROM COUNTRY_RESTRICTIONS
                WHERE CNTR_CTM_ID = LINK_ART.LA_CTM AND CNTR_COU_ID = ?))
            AND (LINK_LA_TYP.LAT_CTM = 0 OR EXISTS (SELECT 1 FROM COUNTRY_RESTRICTIONS
                WHERE CNTR_CTM_ID = LINK_LA_TYP.LAT_CTM AND CNTR_COU_ID = ?))
    `
	args := []interface{}{vehicleType, carID, vehicleType, countryID, countryID, strID, vehicleType, countryID, countryID, countryID}
	criterionIDs := make([]int, 0, len(criteria))
	for id := range criteria {
		criterionIDs = append(criterionIDs, id)
	}
	sort.Ints(criterionIDs)
	for _, id := range criterionIDs {
		scope += ` AND EXISTS (SELECT 1 FROM ARTICLE_CRITERIA
            WHERE ARTICLE_CRITERIA.ACR_ART_ID = ARTICLES.ART_ID
                AND ARTICLE_CRITERIA.ACR_CRI_ID = ? AND ARTICLE_CRITERIA.ACR_VALUE = ?
                AND (ARTICLE_CRITERIA.ACR_CTM = 0 OR EXISTS (SELECT 1 FROM COUNTRY_RESTRICTIONS
                    WHERE CNTR_CTM_ID = ARTICLE_CRITERIA.ACR_CTM AND CNTR_COU_ID = ?)))`
		args = append(args, id, criteria[id], countryID)
	}
	return scope, args
}

func (q *Queries) GetProductGroupArticles(strID, carID int, vehicleType string, languageID, countryID, limit, offset int) ([]models.Article, error) {
	return q.GetProductGroupArticlesWithCriteria(strID, carID, vehicleType, languageID, countryID, nil, limit, offset)
}

func (q *Queries) CountProductGroupArticles(strID, carID int, vehicleType string, countryID int, criteria map[int]string) (int, error) {
	scope, args := productGroupArticleScope(strID, carID, vehicleType, countryID, criteria)
	var count int
	err := q.db.QueryRow("SELECT COUNT(DISTINCT ARTICLES.ART_ID) "+scope, args...).Scan(&count)
	return count, err
}

// GetProductGroupArticlesWithCriteria preserves the full Article contract and
// binds every criterion value. Linkage-only criteria are not global attributes.
func (q *Queries) GetProductGroupArticlesWithCriteria(strID, carID int, vehicleType string, languageID, countryID int, criteria map[int]string, limit, offset int) ([]models.Article, error) {
	scope, args := productGroupArticleScope(strID, carID, vehicleType, countryID, criteria)
	query := `SELECT DISTINCT ARTICLES.ART_ID, ARTICLES.ART_ARTICLE_NR,
        ARTICLES.ART_SUP_BRAND, ARTICLES.ART_SUP_ID,
        CONCAT_WS(' ', (` + BuildGetTextSubquery("ARTICLES.ART_COMPLETE_DES_ID", languageID) + `),
            (` + BuildGetTextSubquery("ARTICLES.ART_DES_ID", languageID) + `)) AS ART_PRODUCT_NAME,
        (` + BuildGetTextSubquery("ART_COUNTRY_SPECIFICS.ACS_STATUS_DES_ID", languageID) + `) AS ART_STATUS
        ` + scope + ` ORDER BY ARTICLES.ART_SUP_BRAND, ARTICLES.ART_ARTICLE_NR, ARTICLES.ART_ID LIMIT ? OFFSET ?`
	args = append(args, limit, offset)
	rows, err := q.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	articles := make([]models.Article, 0)
	for rows.Next() {
		var article models.Article
		if err := rows.Scan(&article.ID, &article.ArticleNr, &article.Brand, &article.SupplierID, &article.Name, &article.Status); err != nil {
			return nil, err
		}
		articles = append(articles, article)
	}
	return articles, rows.Err()
}
