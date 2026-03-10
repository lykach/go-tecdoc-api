package database

import (
	"database/sql"
	"fmt"

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
		ORDER BY STR_NODE_NAME
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
		ORDER BY STR_NODE_NAME
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
		ORDER BY STR_NODE_NAME
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

// GetProductGroupArticles - отримати запчастини для категорії та автомобіля
func (q *Queries) GetProductGroupArticles(strID int, pcID int, vehicleType string, languageID int, countryID int, limit, offset int) ([]models.Article, error) {
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
			(` + BuildGetTextSubquery("ART_COUNTRY_SPECIFICS.ACS_STATUS_DES_ID", languageID) + `) AS ART_STATUS
		FROM 
			SEARCH_TREE
			INNER JOIN LINK_PT_STR ON LINK_PT_STR.STR_ID = ?
				AND FIND_IN_SET(?, LINK_PT_STR.STR_TYPE) > 0
			INNER JOIN LINK_LA_TYP ON LINK_LA_TYP.LAT_TYP_ID = ?
				AND LINK_LA_TYP.LAT_TYPE = ?
				AND LINK_LA_TYP.LAT_PT_ID = LINK_PT_STR.PT_ID
			INNER JOIN LINK_ART ON LINK_ART.LA_ID = LINK_LA_TYP.LAT_LA_ID
			INNER JOIN ARTICLES ON ARTICLES.ART_ID = LINK_ART.LA_ART_ID
			LEFT OUTER JOIN ART_COUNTRY_SPECIFICS ON ART_COUNTRY_SPECIFICS.ACS_ART_ID = ARTICLES.ART_ID
				AND (ART_COUNTRY_SPECIFICS.ACS_COU_ID = ? OR ART_COUNTRY_SPECIFICS.ACS_COU_ID = 0)
		WHERE
			SEARCH_TREE.STR_ID = ?
			AND FIND_IN_SET(?, SEARCH_TREE.STR_TYPE) > 0
		ORDER BY ARTICLES.ART_SUP_BRAND, ARTICLES.ART_ARTICLE_NR
		LIMIT ? OFFSET ?
	`

	rows, err := q.db.Query(query, strID, vehicleType, pcID, vehicleType, countryID, strID, vehicleType, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var articles []models.Article
	for rows.Next() {
		var art models.Article
		var packUnit sql.NullInt64

		err := rows.Scan(
			&art.ID,
			&art.ArticleNr,
			&art.Brand,
			&art.SupplierID,
			&art.Name,
			&art.Status,
		)
		if err != nil {
			return nil, err
		}

		if packUnit.Valid {
			val := int(packUnit.Int64)
			art.PackUnit = &val
		}

		articles = append(articles, art)
	}

	return articles, rows.Err()
}

// CountProductGroupArticles - підрахунок запчастин для категорії
func (q *Queries) CountProductGroupArticles(strID int, pcID int, vehicleType string) (int, error) {
	query := `
		SELECT COUNT(DISTINCT ARTICLES.ART_ID)
		FROM 
			SEARCH_TREE
			INNER JOIN LINK_PT_STR ON LINK_PT_STR.STR_ID = ?
				AND FIND_IN_SET(?, LINK_PT_STR.STR_TYPE) > 0
			INNER JOIN LINK_LA_TYP ON LINK_LA_TYP.LAT_TYP_ID = ?
				AND LINK_LA_TYP.LAT_TYPE = ?
				AND LINK_LA_TYP.LAT_PT_ID = LINK_PT_STR.PT_ID
			INNER JOIN LINK_ART ON LINK_ART.LA_ID = LINK_LA_TYP.LAT_LA_ID
			INNER JOIN ARTICLES ON ARTICLES.ART_ID = LINK_ART.LA_ART_ID
		WHERE
			SEARCH_TREE.STR_ID = ?
			AND FIND_IN_SET(?, SEARCH_TREE.STR_TYPE) > 0
	`

	var count int
	err := q.db.QueryRow(query, strID, vehicleType, pcID, vehicleType, strID, vehicleType).Scan(&count)
	return count, err
}

// GetProductGroupArticlesWithCriteria повертає запчастини з фільтрацією за критеріями
func (q *Queries) GetProductGroupArticlesWithCriteria(productGroupID int, carID int, languageID int, countryID int, criteriaFilters map[int]string, limit, offset int) ([]models.Article, error) {
	// Base query
	baseQuery := `
		SELECT DISTINCT
			ARTICLES.ART_ID,
			ARTICLES.ART_ARTICLE_NR,
			ARTICLES.ART_SUP_BRAND,
			CONCAT_WS(' ', 
				(` + BuildGetTextSubquery("ARTICLES.ART_COMPLETE_DES_ID", languageID) + `),
				(` + BuildGetTextSubquery("ARTICLES.ART_DES_ID", languageID) + `)
			) AS ART_PRODUCT_NAME
		FROM 
			ARTICLES
			INNER JOIN LINK_ART ON LINK_ART.LA_ART_ID = ARTICLES.ART_ID
	`

	// Add link to vehicle if carID provided
	if carID > 0 {
		baseQuery += `
			INNER JOIN LINK_LA_TYP ON LINK_LA_TYP.LAT_LA_ID = LINK_ART.LA_ID
				AND LINK_LA_TYP.LAT_TYP_ID = ?
		`
	}

	// Add product group links
	baseQuery += `
		INNER JOIN LINK_ART_PT ON LINK_ART_PT.LAP_ART_ID = ARTICLES.ART_ID
		INNER JOIN LINK_PT_STR ON LINK_PT_STR.PT_ID = LINK_ART_PT.LAP_PT_ID
			AND LINK_PT_STR.STR_ID = ?
	`

	// Add criteria joins if filters provided
	criteriaJoins := ""
	criteriaIndex := 0
	for criID := range criteriaFilters {
		criteriaJoins += fmt.Sprintf(`
			INNER JOIN ARTICLE_CRITERIA AS AC%d ON AC%d.ACR_ART_ID = ARTICLES.ART_ID
				AND AC%d.ACR_CRI_ID = %d
		`, criteriaIndex, criteriaIndex, criteriaIndex, criID)
		criteriaIndex++
	}

	// WHERE conditions with criteria values
	whereConditions := `
		WHERE 
			(ARTICLES.ART_CTM = 0 OR ARTICLES.ART_CTM & (SELECT COU_CTM FROM COUNTRIES WHERE COU_ID = ?) > 0)
	`

	// Add criteria value conditions
	criteriaConditions := ""
	criteriaIndex = 0
	for _, value := range criteriaFilters {
		criteriaConditions += fmt.Sprintf(" AND AC%d.ACR_VALUE = '%s'", criteriaIndex, value)
		criteriaIndex++
	}

	query := baseQuery + criteriaJoins + whereConditions + criteriaConditions + `
		ORDER BY 
			ARTICLES.ART_SUP_BRAND, ARTICLES.ART_ARTICLE_NR
		LIMIT ? OFFSET ?
	`

	// Prepare query parameters
	var args []interface{}
	if carID > 0 {
		args = append(args, carID)
	}
	args = append(args, productGroupID, countryID, limit, offset)

	rows, err := q.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var articles []models.Article
	for rows.Next() {
		var article models.Article

		err := rows.Scan(
			&article.ID,
			&article.ArticleNr,
			&article.Brand,
			&article.Name,
		)
		if err != nil {
			return nil, err
		}

		articles = append(articles, article)
	}

	return articles, rows.Err()
}