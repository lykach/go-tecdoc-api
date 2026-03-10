package database

import (
	"database/sql"
	"go-tecdoc-api/internal/models"
)

// GetSuppliers повертає список постачальників з пагінацією
func (q *Queries) GetSuppliers(page, limit int) ([]models.Supplier, int, error) {
	offset := (page - 1) * limit

	// Підрахунок загальної кількості
	var total int
	countQuery := `SELECT COUNT(*) FROM SUPPLIERS`
	if err := q.db.QueryRow(countQuery).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Запит списку постачальників
	query := `
		SELECT 
			SUP_ID,
			SUP_BRAND,
			SUP_FULL_NAME,
			SUP_LOGO_NAME,
			(SELECT SUPLG_LOGO_NAME 
			 FROM SUPPLIERS_LOGOS 
			 WHERE SUPLG_SUP_ID = SUPPLIERS.SUP_ID 
			   AND SUPLG_COU_ID = 0 
			 LIMIT 1) AS WEBP_LOGO
		FROM 
			SUPPLIERS
		ORDER BY 
			SUP_BRAND
		LIMIT ? OFFSET ?
	`

	rows, err := q.db.Query(query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var suppliers []models.Supplier
	for rows.Next() {
		var s models.Supplier
		var fullName, logoPNG, logoWEBP sql.NullString

		err := rows.Scan(&s.ID, &s.Brand, &fullName, &logoPNG, &logoWEBP)
		if err != nil {
			return nil, 0, err
		}

		if fullName.Valid {
			s.FullName = &fullName.String
		}
		if logoPNG.Valid {
			s.LogoPNG = &logoPNG.String
		}
		if logoWEBP.Valid {
			s.LogoWEBP = &logoWEBP.String
		}

		suppliers = append(suppliers, s)
	}

	return suppliers, total, rows.Err()
}

// GetSupplierByID повертає деталі постачальника за ID
func (q *Queries) GetSupplierByID(id int) (*models.Supplier, error) {
	query := `
		SELECT 
			SUP_ID,
			SUP_BRAND,
			SUP_FULL_NAME,
			SUP_LOGO_NAME,
			(SELECT SUPLG_LOGO_NAME 
			 FROM SUPPLIERS_LOGOS 
			 WHERE SUPLG_SUP_ID = SUPPLIERS.SUP_ID 
			   AND SUPLG_COU_ID = 0 
			 LIMIT 1) AS WEBP_LOGO
		FROM 
			SUPPLIERS
		WHERE 
			SUP_ID = ?
	`

	var s models.Supplier
	var fullName, logoPNG, logoWEBP sql.NullString

	err := q.db.QueryRow(query, id).Scan(&s.ID, &s.Brand, &fullName, &logoPNG, &logoWEBP)
	if err != nil {
		return nil, err
	}

	if fullName.Valid {
		s.FullName = &fullName.String
	}
	if logoPNG.Valid {
		s.LogoPNG = &logoPNG.String
	}
	if logoWEBP.Valid {
		s.LogoWEBP = &logoWEBP.String
	}

	return &s, nil
}

// SearchSuppliersByBrand пошук постачальників за назвою бренду
func (q *Queries) SearchSuppliersByBrand(brand string, limit int) ([]models.Supplier, error) {
	query := `
		SELECT 
			SUP_ID,
			SUP_BRAND,
			SUP_FULL_NAME,
			SUP_LOGO_NAME,
			(SELECT SUPLG_LOGO_NAME 
			 FROM SUPPLIERS_LOGOS 
			 WHERE SUPLG_SUP_ID = SUPPLIERS.SUP_ID 
			   AND SUPLG_COU_ID = 0 
			 LIMIT 1) AS WEBP_LOGO
		FROM 
			SUPPLIERS
		WHERE 
			SUP_BRAND LIKE ?
		ORDER BY 
			SUP_BRAND
		LIMIT ?
	`

	rows, err := q.db.Query(query, "%"+brand+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var suppliers []models.Supplier
	for rows.Next() {
		var s models.Supplier
		var fullName, logoPNG, logoWEBP sql.NullString

		err := rows.Scan(&s.ID, &s.Brand, &fullName, &logoPNG, &logoWEBP)
		if err != nil {
			return nil, err
		}

		if fullName.Valid {
			s.FullName = &fullName.String
		}
		if logoPNG.Valid {
			s.LogoPNG = &logoPNG.String
		}
		if logoWEBP.Valid {
			s.LogoWEBP = &logoWEBP.String
		}

		suppliers = append(suppliers, s)
	}

	return suppliers, rows.Err()
}
// GetSupplierProducts повертає товарні групи для постачальника
func (q *Queries) GetSupplierProducts(supplierID int, languageID int, limit, offset int) ([]models.SupplierProduct, error) {
query := `
SELECT DISTINCT
PRODUCTS.PT_ID,
(` + BuildGetTextSubquery("PRODUCTS.PT_DES_ID", languageID) + `) AS PRODUCT_NAME,
COUNT(DISTINCT LINK_ART_PT.LAP_ART_ID) AS ARTICLES_COUNT
FROM 
LINK_ART_PT
INNER JOIN PRODUCTS ON PRODUCTS.PT_ID = LINK_ART_PT.LAP_PT_ID
WHERE 
LINK_ART_PT.LAP_SUP_ID = ?
GROUP BY 
PRODUCTS.PT_ID, PRODUCT_NAME
ORDER BY 
ARTICLES_COUNT DESC
LIMIT ? OFFSET ?
`

rows, err := q.db.Query(query, supplierID, limit, offset)
if err != nil {
return nil, err
}
defer rows.Close()

var products []models.SupplierProduct
for rows.Next() {
var sp models.SupplierProduct

err := rows.Scan(
&sp.ProductID,
&sp.ProductName,
&sp.ArticlesCount,
)
if err != nil {
return nil, err
}

products = append(products, sp)
}

return products, rows.Err()
}

// CountSupplierProducts підраховує кількість товарних груп для постачальника
func (q *Queries) CountSupplierProducts(supplierID int) (int, error) {
query := `
SELECT COUNT(DISTINCT LINK_ART_PT.LAP_PT_ID)
FROM LINK_ART_PT
WHERE LINK_ART_PT.LAP_SUP_ID = ?
`

var count int
err := q.db.QueryRow(query, supplierID).Scan(&count)
return count, err
}
