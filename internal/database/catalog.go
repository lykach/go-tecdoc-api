package database

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"go-tecdoc-api/internal/models"
)

var datasetIdentifierRE = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

func normalizeVehicleType(value string) (string, bool) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "PC", "CAR", "CARS":
		return "PC", true
	case "CV", "COMMERCIAL", "COMMERCIAL_VEHICLE":
		return "CV", true
	case "MC", "MTB", "MOTORCYCLE", "MOTORBIKE":
		return "Motorcycle", true
	default:
		return "", false
	}
}

func normalizeCatalogType(value string) (string, bool) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "PC", "CAR", "CARS":
		return "PC", true
	case "CV", "COMMERCIAL", "COMMERCIAL_VEHICLE":
		return "CV", true
	case "MC", "MTB", "MOTORCYCLE", "MOTORBIKE":
		return "Motorcycle", true
	case "ENGINE", "ENGINES":
		return "Engine", true
	case "AXLE", "AXLES":
		return "Axle", true
	case "UNIVERSAL", "UNIVERSAL PARTS", "UNIVERSAL_PARTS":
		return "Universal Parts", true
	default:
		return "", false
	}
}

func (q *Queries) SearchTypeNumbers(ctx context.Context, number, numberType, vehicleType string, limit, offset int) ([]models.TypeNumberMatch, error) {
	number = normalizeSearchNumber(number)
	if number == "" {
		return []models.TypeNumberMatch{}, nil
	}
	if limit < 1 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	conditions := []string{"LTN_NUMBER = ?"}
	args := []any{number}
	if strings.TrimSpace(numberType) != "" {
		conditions = append(conditions, "LTN_NUMBER_TYPE = ?")
		args = append(args, strings.TrimSpace(numberType))
	}
	if strings.TrimSpace(vehicleType) != "" {
		vt, ok := normalizeVehicleType(vehicleType)
		if !ok {
			return nil, fmt.Errorf("unsupported vehicle type %q", vehicleType)
		}
		conditions = append(conditions, "LTN_VEHICLE_TYPE = ?")
		args = append(args, vt)
	}
	args = append(args, limit, offset)

	query := `
		SELECT LTN_VEHICLE_ID, LTN_VEHICLE_TYPE, LTN_NUMBER, LTN_NUMBER_TYPE, LTN_MFA_ID
		FROM TYPE_NUMBER_LOOKUP
		WHERE ` + strings.Join(conditions, " AND ") + `
		ORDER BY LTN_VEHICLE_TYPE, LTN_VEHICLE_ID, LTN_NUMBER_TYPE
		LIMIT ? OFFSET ?`

	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]models.TypeNumberMatch, 0)
	for rows.Next() {
		var item models.TypeNumberMatch
		if err := rows.Scan(&item.VehicleID, &item.VehicleType, &item.Number, &item.NumberType, &item.ManufacturerID); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (q *Queries) GetVehicleTypeNumbers(ctx context.Context, vehicleID int, vehicleType string) ([]models.TypeNumberMatch, error) {
	vt, ok := normalizeVehicleType(vehicleType)
	if !ok {
		return nil, fmt.Errorf("unsupported vehicle type %q", vehicleType)
	}
	rows, err := q.db.QueryContext(ctx, `
		SELECT LTN_VEHICLE_ID, LTN_VEHICLE_TYPE, LTN_NUMBER, LTN_NUMBER_TYPE, LTN_MFA_ID
		FROM TYPE_NUMBER_LOOKUP
		WHERE LTN_VEHICLE_ID = ? AND LTN_VEHICLE_TYPE = ?
		ORDER BY LTN_NUMBER_TYPE, LTN_NUMBER`, vehicleID, vt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]models.TypeNumberMatch, 0)
	for rows.Next() {
		var item models.TypeNumberMatch
		if err := rows.Scan(&item.VehicleID, &item.VehicleType, &item.Number, &item.NumberType, &item.ManufacturerID); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (q *Queries) GetProductTypeCriteria(ctx context.Context, productTypeID, languageID int, suggestedOnly bool) ([]models.ProductTypeCriterion, error) {
	query := `
		SELECT
			LINK_PT_CRI.LPC_PT_ID,
			CRITERIA.CRI_ID,
			COALESCE((` + BuildGetTextSubquery("CRITERIA.CRI_DES_ID", languageID) + `), ''),
			COALESCE((` + BuildGetTextSubquery("CRITERIA.CRI_SHORT_DES_ID", languageID) + `), ''),
			COALESCE((` + BuildGetTextSubquery("CRITERIA.CRI_UNIT_DES_ID", languageID) + `), ''),
			CRITERIA.CRI_TYPE,
			LINK_PT_CRI.LPC_SORT,
			COALESCE(LINK_PT_CRI.LPC_SUGGESTION, 0)
		FROM LINK_PT_CRI
		INNER JOIN CRITERIA ON CRITERIA.CRI_ID = LINK_PT_CRI.LPC_CRI_ID
		WHERE LINK_PT_CRI.LPC_PT_ID = ?`
	args := []any{productTypeID}
	if suggestedOnly {
		query += " AND COALESCE(LINK_PT_CRI.LPC_SUGGESTION, 0) = 1"
	}
	query += " ORDER BY COALESCE(LINK_PT_CRI.LPC_SORT, 32767), CRITERIA.CRI_ID"

	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]models.ProductTypeCriterion, 0)
	for rows.Next() {
		var item models.ProductTypeCriterion
		var sortNo sql.NullInt64
		var suggested int
		if err := rows.Scan(&item.ProductTypeID, &item.CriteriaID, &item.Name, &item.ShortName, &item.Unit, &item.Type, &sortNo, &suggested); err != nil {
			return nil, err
		}
		if sortNo.Valid {
			value := int(sortNo.Int64)
			item.Sort = &value
		}
		item.Suggested = suggested != 0
		items = append(items, item)
	}
	return items, rows.Err()
}

func vehicleCriterionSelectColumns(languageID int) string {
	return `
		CRITERIA.CRI_ID AS CRI_ID,
		COALESCE((` + BuildGetTextSubquery("CRITERIA.CRI_DES_ID", languageID) + `), '') AS CRI_NAME,
		COALESCE((` + BuildGetTextSubquery("CRITERIA.CRI_SHORT_DES_ID", languageID) + `), '') AS CRI_SHORT,
		COALESCE((` + BuildGetTextSubquery("CRITERIA.CRI_UNIT_DES_ID", languageID) + `), '') AS CRI_UNIT,
		CRITERIA.CRI_TYPE AS CRI_TYPE,
		COALESCE(LINK_PT_CRI.LPC_SUGGESTION, 0) AS SUGGESTED`
}

func (q *Queries) GetVehicleCriteria(ctx context.Context, vehicleID int, vehicleType string, languageID, productTypeID, supplierID int, suggestedOnly bool) ([]models.VehicleCriterion, error) {
	vt, ok := normalizeVehicleType(vehicleType)
	if !ok {
		return nil, fmt.Errorf("unsupported vehicle type %q", vehicleType)
	}

	where := []string{"LINK_LA_TYP.LAT_TYP_ID = ?", "LINK_LA_TYP.LAT_TYPE = ?"}
	argsBase := []any{vehicleID, vt}
	if productTypeID > 0 {
		where = append(where, "LINK_LA_TYP.LAT_PT_ID = ?")
		argsBase = append(argsBase, productTypeID)
	}
	if supplierID > 0 {
		where = append(where, "LINK_LA_TYP.LAT_SUP_ID = ?")
		argsBase = append(argsBase, supplierID)
	}
	if suggestedOnly {
		where = append(where, "COALESCE(LINK_PT_CRI.LPC_SUGGESTION, 0) = 1")
	}
	whereSQL := strings.Join(where, " AND ")

	criterionCols := vehicleCriterionSelectColumns(languageID)

	query := `
		SELECT DISTINCT CRI_ID, CRI_NAME, CRI_SHORT, CRI_UNIT, CRI_TYPE, SUGGESTED, CRI_VALUE
		FROM (
			SELECT ` + criterionCols + `,
				COALESCE(
					CASE WHEN ARTICLE_CRITERIA.ACR_DES_ID IS NULL THEN ARTICLE_CRITERIA.ACR_VALUE
					ELSE (` + BuildGetTextSubquery("ARTICLE_CRITERIA.ACR_DES_ID", languageID) + `) END,
					''
				) AS CRI_VALUE
			FROM LINK_LA_TYP
			INNER JOIN LINK_ART ON LINK_ART.LA_ID = LINK_LA_TYP.LAT_LA_ID
			INNER JOIN ARTICLE_CRITERIA ON ARTICLE_CRITERIA.ACR_ART_ID = LINK_ART.LA_ART_ID
			INNER JOIN CRITERIA ON CRITERIA.CRI_ID = ARTICLE_CRITERIA.ACR_CRI_ID
			LEFT JOIN LINK_PT_CRI ON LINK_PT_CRI.LPC_PT_ID = LINK_LA_TYP.LAT_PT_ID AND LINK_PT_CRI.LPC_CRI_ID = CRITERIA.CRI_ID
			WHERE ` + whereSQL + `

			UNION ALL

			SELECT ` + criterionCols + `,
				COALESCE(
					CASE WHEN LA_CRITERIA.LAC_DES_ID IS NULL THEN LA_CRITERIA.LAC_VALUE
					ELSE (` + BuildGetTextSubquery("LA_CRITERIA.LAC_DES_ID", languageID) + `) END,
					''
				) AS CRI_VALUE
			FROM LINK_LA_TYP
			INNER JOIN LA_CRITERIA ON LA_CRITERIA.LAC_LA_ID = LINK_LA_TYP.LAT_LA_ID
			INNER JOIN CRITERIA ON CRITERIA.CRI_ID = LA_CRITERIA.LAC_CRI_ID
			LEFT JOIN LINK_PT_CRI ON LINK_PT_CRI.LPC_PT_ID = LINK_LA_TYP.LAT_PT_ID AND LINK_PT_CRI.LPC_CRI_ID = CRITERIA.CRI_ID
			WHERE ` + whereSQL + `
		) AS ALL_CRITERIA
		ORDER BY CRI_NAME, CRI_ID, CRI_VALUE`

	args := make([]any, 0, len(argsBase)*2)
	args = append(args, argsBase...)
	args = append(args, argsBase...)
	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type acc struct {
		item   models.VehicleCriterion
		values map[string]struct{}
	}
	byID := make(map[int]*acc)
	order := make([]int, 0)
	for rows.Next() {
		var id int
		var name, shortName, unit, criterionType, value string
		var suggested int
		if err := rows.Scan(&id, &name, &shortName, &unit, &criterionType, &suggested, &value); err != nil {
			return nil, err
		}
		entry, exists := byID[id]
		if !exists {
			entry = &acc{item: models.VehicleCriterion{CriteriaID: id, Name: name, ShortName: shortName, Unit: unit, Type: criterionType, Suggested: suggested != 0}, values: make(map[string]struct{})}
			byID[id] = entry
			order = append(order, id)
		}
		if suggested != 0 {
			entry.item.Suggested = true
		}
		if value != "" {
			entry.values[value] = struct{}{}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	items := make([]models.VehicleCriterion, 0, len(order))
	for _, id := range order {
		entry := byID[id]
		entry.item.Values = make([]string, 0, len(entry.values))
		for value := range entry.values {
			entry.item.Values = append(entry.item.Values, value)
		}
		sort.Strings(entry.item.Values)
		items = append(items, entry.item)
	}
	return items, nil
}

func (q *Queries) SearchProductGroups(ctx context.Context, searchText, vehicleType string, languageID, limit, offset int) ([]models.ProductGroupSearchResult, error) {
	vt, ok := normalizeCatalogType(vehicleType)
	if !ok {
		return nil, fmt.Errorf("unsupported catalog type %q", vehicleType)
	}
	if limit < 1 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	searchText = strings.TrimSpace(searchText)
	if searchText == "" {
		return []models.ProductGroupSearchResult{}, nil
	}

	query := `
		WITH RECURSIVE category_path (STR_ID, STR_ID_PARENT, STR_LEVEL, PT_ID, STR_NODE_NAME, STR_TYPE, STR_PATH) AS (
			SELECT
				SEARCH_TREE.STR_ID,
				SEARCH_TREE.STR_ID_PARENT,
				SEARCH_TREE.STR_LEVEL,
				LINK_PT_STR.PT_ID,
				COALESCE((` + BuildGetTextSubquery("SEARCH_TREE.STR_DES_ID", languageID) + `), ''),
				SEARCH_TREE.STR_TYPE,
				COALESCE((` + BuildGetTextSubquery("SEARCH_TREE.STR_DES_ID", languageID) + `), '')
			FROM SEARCH_TREE
			INNER JOIN LINK_PT_STR ON LINK_PT_STR.STR_ID = SEARCH_TREE.STR_ID
				AND FIND_IN_SET(?, LINK_PT_STR.STR_TYPE) > 0
				AND LINK_PT_STR.PT_ID IN (
					SELECT DISTINCT STL_PT_ID FROM STR_LOOKUP
					WHERE STL_LNG_ID IN (?, 4, 1) AND STL_SEARCH_TEXT LIKE ?
				)
			WHERE FIND_IN_SET(?, SEARCH_TREE.STR_TYPE) > 0 AND SEARCH_TREE.STR_ID_PARENT IS NULL

			UNION ALL

			SELECT
				SEARCH_TREE.STR_ID,
				SEARCH_TREE.STR_ID_PARENT,
				SEARCH_TREE.STR_LEVEL,
				LINK_PT_STR.PT_ID,
				COALESCE((` + BuildGetTextSubquery("SEARCH_TREE.STR_DES_ID", languageID) + `), ''),
				SEARCH_TREE.STR_TYPE,
				CONCAT(category_path.STR_PATH, ' > ', COALESCE((` + BuildGetTextSubquery("SEARCH_TREE.STR_DES_ID", languageID) + `), ''))
			FROM category_path
			INNER JOIN SEARCH_TREE ON SEARCH_TREE.STR_ID_PARENT = category_path.STR_ID AND FIND_IN_SET(?, SEARCH_TREE.STR_TYPE) > 0
			INNER JOIN LINK_PT_STR ON LINK_PT_STR.STR_ID = SEARCH_TREE.STR_ID
				AND FIND_IN_SET(?, LINK_PT_STR.STR_TYPE) > 0
				AND LINK_PT_STR.PT_ID = category_path.PT_ID
		)
		SELECT DISTINCT
			category_path.STR_ID,
			category_path.STR_ID_PARENT,
			category_path.STR_LEVEL,
			category_path.STR_NODE_NAME,
			category_path.STR_PATH,
			category_path.STR_TYPE,
			category_path.PT_ID,
			COALESCE((` + BuildGetTextSubquery("PRODUCTS.PT_DES_ID", languageID) + `), '') AS PRODUCT_TYPE
		FROM category_path
		INNER JOIN PRODUCTS ON PRODUCTS.PT_ID = category_path.PT_ID
		ORDER BY category_path.STR_PATH, category_path.PT_ID
		LIMIT ? OFFSET ?`

	pattern := "%" + searchText + "%"
	rows, err := q.db.QueryContext(ctx, query, vt, languageID, pattern, vt, vt, vt, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]models.ProductGroupSearchResult, 0)
	for rows.Next() {
		var item models.ProductGroupSearchResult
		var parent sql.NullInt64
		if err := rows.Scan(&item.CategoryID, &parent, &item.Level, &item.Name, &item.Path, &item.VehicleType, &item.ProductTypeID, &item.ProductType); err != nil {
			return nil, err
		}
		if parent.Valid {
			v := int(parent.Int64)
			item.ParentID = &v
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (q *Queries) GetQuickStart(ctx context.Context, vehicleType string, languageID int) ([]models.QuickStartItem, error) {
	query := `
		SELECT
			QUICK_START.QSI_ID,
			QUICK_START.QSI_TYPE,
			COALESCE((` + BuildGetTextSubquery("QUICK_START.QSI_DES_ID", languageID) + `), ''),
			COALESCE(QUICK_START.QSI_IMG, ''),
			COALESCE((SELECT GROUP_CONCAT(DISTINCT LINK_QS_ST.STR_ID ORDER BY LINK_QS_ST.STR_ID SEPARATOR ',')
				FROM LINK_QS_ST WHERE LINK_QS_ST.QSI_ID = QUICK_START.QSI_ID AND LINK_QS_ST.QSI_TYPE = QUICK_START.QSI_TYPE), '')
		FROM QUICK_START`
	args := make([]any, 0)
	if strings.TrimSpace(vehicleType) != "" {
		vt, ok := normalizeCatalogType(vehicleType)
		if !ok {
			return nil, fmt.Errorf("unsupported catalog type %q", vehicleType)
		}
		query += " WHERE FIND_IN_SET(?, QUICK_START.QSI_TYPE) > 0"
		args = append(args, vt)
	}
	query += " ORDER BY QUICK_START.QSI_TYPE, QUICK_START.QSI_ID"

	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]models.QuickStartItem, 0)
	for rows.Next() {
		var item models.QuickStartItem
		var categoryIDs string
		if err := rows.Scan(&item.ID, &item.Type, &item.Name, &item.Image, &categoryIDs); err != nil {
			return nil, err
		}
		item.CategoryIDs = make([]int, 0)
		for _, raw := range strings.Split(categoryIDs, ",") {
			raw = strings.TrimSpace(raw)
			if raw == "" {
				continue
			}
			id, err := strconv.Atoi(raw)
			if err == nil {
				item.CategoryIDs = append(item.CategoryIDs, id)
			}
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (q *Queries) GetArticlePrices(ctx context.Context, articleID, countryID, languageID int, includeExpired bool) ([]models.ArticlePrice, error) {
	query := `
		SELECT
			PRICES.PRI_ART_ID,
			PRICES.PRI_COU_ID,
			COALESCE((` + BuildGetTextSubquery("PRICES.PRI_TYPE_DES", languageID) + `), ''),
			PRICES.PRI_PRICE,
			COALESCE(PRICES.PRI_CURRENCY_CODE, ''),
			DATE_FORMAT(PRICES.PRI_FROM, '%Y-%m-%d'),
			COALESCE(DATE_FORMAT(PRICES.PRI_TO, '%Y-%m-%d'), ''),
			COALESCE((` + BuildGetTextSubquery("PRICES.PRI_UNIT_DES", languageID) + `), ''),
			COALESCE((` + BuildGetTextSubquery("PRICES.PRI_QUANTITY_DES", languageID) + `), ''),
			COALESCE(PRICES.PRI_DISCOUNT_GROUP, '')
		FROM PRICES
		WHERE PRICES.PRI_ART_ID = ? AND (PRICES.PRI_COU_ID = ? OR PRICES.PRI_COU_ID = 0)`
	args := []any{articleID, countryID}
	if !includeExpired {
		query += " AND PRICES.PRI_FROM <= CURRENT_DATE() AND (PRICES.PRI_TO IS NULL OR PRICES.PRI_TO >= CURRENT_DATE())"
	}
	query += " ORDER BY PRICES.PRI_COU_ID DESC, PRICES.PRI_FROM DESC, PRICES.PRI_TYPE_DES"

	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]models.ArticlePrice, 0)
	for rows.Next() {
		var item models.ArticlePrice
		if err := rows.Scan(&item.ArticleID, &item.CountryID, &item.Type, &item.Price, &item.Currency, &item.ValidFrom, &item.ValidTo, &item.Unit, &item.Quantity, &item.DiscountGroup); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (q *Queries) GetSupplierAddresses(ctx context.Context, supplierID, languageID int) ([]models.SupplierAddress, error) {
	query := `
		SELECT
			SUPPLIER_ADDRESSES.SUP_ID,
			COALESCE((` + BuildGetTextSubquery("SUPPLIER_ADDRESSES.TYPE_OF_ADDRESS_DES_ID", languageID) + `), ''),
			SUPPLIER_ADDRESSES.COU_ID,
			COALESCE((` + BuildGetTextSubquery("COUNTRIES.COU_DES", languageID) + `), ''),
			COALESCE(SUPPLIER_ADDRESSES.NAME1, ''), COALESCE(SUPPLIER_ADDRESSES.NAME2, ''),
			COALESCE(SUPPLIER_ADDRESSES.CITY1, ''), COALESCE(SUPPLIER_ADDRESSES.CITY2, ''),
			COALESCE(SUPPLIER_ADDRESSES.STREET1, ''), COALESCE(SUPPLIER_ADDRESSES.STREET2, ''),
			COALESCE(SUPPLIER_ADDRESSES.POSTAL_CODE_CITY, ''), COALESCE(SUPPLIER_ADDRESSES.EMAIL, ''),
			COALESCE(SUPPLIER_ADDRESSES.HOMEPAGE, ''), COALESCE(SUPPLIER_ADDRESSES.TELEPHONE, ''), COALESCE(SUPPLIER_ADDRESSES.FAX, '')
		FROM SUPPLIER_ADDRESSES
		LEFT JOIN COUNTRIES ON COUNTRIES.COU_ID = SUPPLIER_ADDRESSES.COU_ID
		WHERE SUPPLIER_ADDRESSES.SUP_ID = ?
		ORDER BY SUPPLIER_ADDRESSES.COU_ID, SUPPLIER_ADDRESSES.TYPE_OF_ADDRESS_DES_ID`
	rows, err := q.db.QueryContext(ctx, query, supplierID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]models.SupplierAddress, 0)
	for rows.Next() {
		var item models.SupplierAddress
		if err := rows.Scan(&item.SupplierID, &item.AddressType, &item.CountryID, &item.Country, &item.Name1, &item.Name2, &item.City1, &item.City2, &item.Street1, &item.Street2, &item.PostalCode, &item.Email, &item.Homepage, &item.Telephone, &item.Fax); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (q *Queries) GetSupplierLogos(ctx context.Context, supplierID int) ([]models.SupplierLogo, error) {
	rows, err := q.db.QueryContext(ctx, `
		SELECT SUPLG_SUP_ID, SUPLG_COU_ID, SUPLG_LOGO_NAME
		FROM SUPPLIERS_LOGOS WHERE SUPLG_SUP_ID = ? ORDER BY SUPLG_COU_ID, SUPLG_LOGO_NAME`, supplierID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]models.SupplierLogo, 0)
	for rows.Next() {
		var item models.SupplierLogo
		if err := rows.Scan(&item.SupplierID, &item.CountryID, &item.FileName); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (q *Queries) GetArticleInfo(ctx context.Context, articleID, languageID, countryID int, includeHidden bool) ([]models.ArticleInfoItem, error) {
	query := `
		SELECT
			COALESCE((` + BuildGetTextSubquery("ARTICLE_INFO.AIN_TYPE_DES_ID", languageID) + `), ''),
			COALESCE(
				(SELECT TMT_TEXT FROM TEXT_MODULE_TEXTS WHERE TMT_ID = ARTICLE_INFO.AIN_TMT_ID AND TMT_LNG_ID = ? LIMIT 1),
				(SELECT TMT_TEXT FROM TEXT_MODULE_TEXTS WHERE TMT_ID = ARTICLE_INFO.AIN_TMT_ID AND TMT_LNG_ID = 255 LIMIT 1),
				(SELECT TMT_TEXT FROM TEXT_MODULE_TEXTS WHERE TMT_ID = ARTICLE_INFO.AIN_TMT_ID AND TMT_LNG_ID = 4 LIMIT 1),
				''
			),
			ARTICLE_INFO.AIN_SORT,
			COALESCE(ARTICLE_INFO.AIN_DISPLAY, 0),
			ARTICLE_INFO.AIN_TMT_ID
		FROM ARTICLE_INFO
		WHERE ARTICLE_INFO.AIN_ART_ID = ?
          AND (ARTICLE_INFO.AIN_CTM=0 OR EXISTS(
              SELECT 1 FROM COUNTRY_RESTRICTIONS C
              WHERE C.CNTR_CTM_ID=ARTICLE_INFO.AIN_CTM AND C.CNTR_COU_ID=?
          ))`
	args := []any{languageID, articleID, countryID}
	if !includeHidden {
		query += " AND COALESCE(ARTICLE_INFO.AIN_DISPLAY, 0) <> 0"
	}
	query += " ORDER BY ARTICLE_INFO.AIN_SORT, ARTICLE_INFO.AIN_TMT_ID"

	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]models.ArticleInfoItem, 0)
	for rows.Next() {
		var item models.ArticleInfoItem
		var display int
		if err := rows.Scan(&item.Type, &item.Text, &item.Sort, &display, &item.TextID); err != nil {
			return nil, err
		}
		item.Display = display != 0
		items = append(items, item)
	}
	return items, rows.Err()
}

func (q *Queries) GetCommercialVehicleChassis(ctx context.Context, vehicleID, languageID int) (*models.CommercialVehicleChassis, error) {
	result := &models.CommercialVehicleChassis{
		VehicleID: vehicleID, Wheelbases: []models.CVWheelbase{}, Suspensions: []models.CVSuspension{}, DriverCabs: []models.DriverCab{}, Axles: []models.CVAxle{},
	}

	rows, err := q.db.QueryContext(ctx, `SELECT AXLE_GROUP, WHELL_BASE FROM WHEELBASES WHERE CV_ID = ? ORDER BY AXLE_GROUP, WHELL_BASE`, vehicleID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var item models.CVWheelbase
		if err := rows.Scan(&item.AxleGroup, &item.Wheelbase); err != nil {
			rows.Close()
			return nil, err
		}
		result.Wheelbases = append(result.Wheelbases, item)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}

	rows, err = q.db.QueryContext(ctx, `
		SELECT COALESCE(CVS_AXLE_POS, ''), COALESCE((`+BuildGetTextSubquery("CV_SUSPENSIONS.CVS_DES", languageID)+`), '')
		FROM CV_SUSPENSIONS WHERE CVS_CV_ID = ? ORDER BY CVS_AXLE_POS`, vehicleID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var item models.CVSuspension
		if err := rows.Scan(&item.AxlePosition, &item.Description); err != nil {
			rows.Close()
			return nil, err
		}
		result.Suspensions = append(result.Suspensions, item)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}

	rows, err = q.db.QueryContext(ctx, `
		SELECT DRIVERS_CABS.DC_ID,
			COALESCE((`+BuildGetTextSubquery("DRIVERS_CABS.DC_MODEL_DES", languageID)+`), ''),
			COALESCE(DATE_FORMAT(LINK_DRIVERS_CABS.LDC_CI_FROM, '%Y-%m-%d'), ''),
			COALESCE(DATE_FORMAT(LINK_DRIVERS_CABS.LDC_CI_TO, '%Y-%m-%d'), ''),
			DRIVERS_CABS.DC_LENGTH, DRIVERS_CABS.DC_HEIGHT, DRIVERS_CABS.DC_WIDTH,
			COALESCE(DRIVERS_CABS.DC_DESIGN, ''), DRIVERS_CABS.DC_SIZE
		FROM LINK_DRIVERS_CABS
		INNER JOIN DRIVERS_CABS ON DRIVERS_CABS.DC_ID = LINK_DRIVERS_CABS.LDC_DC_ID
		WHERE LINK_DRIVERS_CABS.LDC_CV_ID = ? ORDER BY DRIVERS_CABS.DC_ID`, vehicleID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var item models.DriverCab
		var length, height, width, size sql.NullInt64
		if err := rows.Scan(&item.ID, &item.Name, &item.From, &item.To, &length, &height, &width, &item.Design, &size); err != nil {
			rows.Close()
			return nil, err
		}
		item.LengthMM = nullInt(length)
		item.HeightMM = nullInt(height)
		item.WidthMM = nullInt(width)
		item.Size = nullInt(size)
		result.DriverCabs = append(result.DriverCabs, item)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}

	rows, err = q.db.QueryContext(ctx, `
		SELECT AXLES.AXL_ID, AXLES.AXL_CODE, COALESCE(AXLES.AXL_DESCRIPTION, ''), COALESCE(LINK_AXLES_CV.LACV_AXLE_POS, ''),
			COALESCE(DATE_FORMAT(LINK_AXLES_CV.LACV_CI_FROM, '%Y-%m-%d'), ''), COALESCE(DATE_FORMAT(LINK_AXLES_CV.LACV_CI_TO, '%Y-%m-%d'), '')
		FROM LINK_AXLES_CV INNER JOIN AXLES ON AXLES.AXL_ID = LINK_AXLES_CV.LACV_AXL_ID
		WHERE LINK_AXLES_CV.LACV_CV_ID = ? ORDER BY LINK_AXLES_CV.LACV_SORT, LINK_AXLES_CV.LACV_SEQ_NO, AXLES.AXL_ID`, vehicleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var item models.CVAxle
		if err := rows.Scan(&item.ID, &item.Code, &item.Description, &item.Position, &item.From, &item.To); err != nil {
			return nil, err
		}
		result.Axles = append(result.Axles, item)
	}
	return result, rows.Err()
}

func isInternalDatasetName(name string) bool {
	lower := strings.ToLower(name)
	return lower == "api_keys" || lower == "api_key_usage_logs" || strings.HasPrefix(lower, "pma__")
}

func (q *Queries) ListDatasets(ctx context.Context) ([]models.DatasetInfo, error) {
	rows, err := q.db.QueryContext(ctx, `
		SELECT TABLE_NAME, COALESCE(TABLE_ROWS, 0)
		FROM INFORMATION_SCHEMA.TABLES
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_TYPE = 'BASE TABLE'
		ORDER BY TABLE_NAME`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]models.DatasetInfo, 0)
	for rows.Next() {
		var item models.DatasetInfo
		if err := rows.Scan(&item.Name, &item.EstimatedRows); err != nil {
			return nil, err
		}
		if isInternalDatasetName(item.Name) {
			continue
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (q *Queries) GetDatasetSchema(ctx context.Context, table string) ([]models.DatasetColumn, error) {
	if !datasetIdentifierRE.MatchString(table) || isInternalDatasetName(table) {
		return nil, sql.ErrNoRows
	}
	rows, err := q.db.QueryContext(ctx, `
		SELECT COLUMN_NAME, DATA_TYPE, COLUMN_TYPE, IS_NULLABLE, COLUMN_KEY, COLUMN_DEFAULT, EXTRA, ORDINAL_POSITION
		FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?
		ORDER BY ORDINAL_POSITION`, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]models.DatasetColumn, 0)
	for rows.Next() {
		var item models.DatasetColumn
		var nullable string
		var defaultValue sql.NullString
		if err := rows.Scan(&item.Name, &item.DataType, &item.ColumnType, &nullable, &item.Key, &defaultValue, &item.Extra, &item.Ordinal); err != nil {
			return nil, err
		}
		item.Nullable = strings.EqualFold(nullable, "YES")
		if defaultValue.Valid {
			v := defaultValue.String
			item.Default = &v
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, sql.ErrNoRows
	}
	return items, nil
}

func datasetOrderColumns(columns []models.DatasetColumn, orderBy string) []string {
	ordered := []string{orderBy}
	seen := map[string]struct{}{orderBy: {}}

	// Primary-key columns are the cheapest deterministic tie-breakers for the
	// large TecDoc source tables. INFORMATION_SCHEMA.COLUMNS does not expose
	// primary-key sequence, but TecDoc defines composite PK columns in table
	// column order, which is also the order returned by GetDatasetSchema.
	hasPrimary := false
	for _, column := range columns {
		if column.Key != "PRI" {
			continue
		}
		hasPrimary = true
		if _, ok := seen[column.Name]; ok {
			continue
		}
		ordered = append(ordered, column.Name)
		seen[column.Name] = struct{}{}
	}

	// Three TecDoc 2024Q1 domain tables have no PK. For those, append all
	// remaining columns so pagination is deterministic for non-identical rows.
	// Completely identical rows are observationally equivalent in the API.
	if !hasPrimary {
		for _, column := range columns {
			if _, ok := seen[column.Name]; ok {
				continue
			}
			ordered = append(ordered, column.Name)
			seen[column.Name] = struct{}{}
		}
	}

	return ordered
}

func (q *Queries) GetDatasetRows(ctx context.Context, table string, selectColumns []string, filters map[string][]string, orderBy, order string, limit, offset int, includeTotal bool) ([]map[string]any, *int64, error) {
	columns, err := q.GetDatasetSchema(ctx, table)
	if err != nil {
		return nil, nil, err
	}
	allowed := make(map[string]models.DatasetColumn, len(columns))
	for _, column := range columns {
		allowed[column.Name] = column
	}

	selected := make([]string, 0)
	if len(selectColumns) == 0 {
		for _, column := range columns {
			selected = append(selected, column.Name)
		}
	} else {
		seen := make(map[string]struct{})
		for _, name := range selectColumns {
			name = strings.TrimSpace(name)
			if _, ok := allowed[name]; !ok {
				return nil, nil, fmt.Errorf("unknown column %q", name)
			}
			if _, exists := seen[name]; exists {
				continue
			}
			seen[name] = struct{}{}
			selected = append(selected, name)
		}
	}
	if len(selected) == 0 {
		return nil, nil, fmt.Errorf("no columns selected")
	}

	where := make([]string, 0)
	args := make([]any, 0)
	filterNames := make([]string, 0, len(filters))
	for name := range filters {
		filterNames = append(filterNames, name)
	}
	sort.Strings(filterNames)
	for _, name := range filterNames {
		if _, ok := allowed[name]; !ok {
			return nil, nil, fmt.Errorf("unknown filter column %q", name)
		}
		values := filters[name]
		if len(values) == 0 {
			continue
		}
		placeholders := make([]string, 0, len(values))
		for _, value := range values {
			placeholders = append(placeholders, "?")
			args = append(args, value)
		}
		where = append(where, "`"+name+"` IN ("+strings.Join(placeholders, ",")+")")
	}

	if limit < 1 {
		limit = 100
	}
	if limit > 5000 {
		limit = 5000
	}
	if offset < 0 {
		offset = 0
	}
	if orderBy == "" {
		for _, column := range columns {
			if column.Key == "PRI" {
				orderBy = column.Name
				break
			}
		}
		if orderBy == "" {
			orderBy = columns[0].Name
		}
	}
	if _, ok := allowed[orderBy]; !ok {
		return nil, nil, fmt.Errorf("unknown order_by column %q", orderBy)
	}
	order = strings.ToUpper(strings.TrimSpace(order))
	if order == "" {
		order = "ASC"
	}
	if order != "ASC" && order != "DESC" {
		return nil, nil, fmt.Errorf("order must be ASC or DESC")
	}

	quotedSelect := make([]string, len(selected))
	for i, name := range selected {
		quotedSelect[i] = "`" + name + "`"
	}
	baseWhere := ""
	if len(where) > 0 {
		baseWhere = " WHERE " + strings.Join(where, " AND ")
	}
	orderColumns := datasetOrderColumns(columns, orderBy)
	orderParts := make([]string, 0, len(orderColumns))
	for _, name := range orderColumns {
		orderParts = append(orderParts, "`"+name+"` "+order)
	}
	query := "SELECT " + strings.Join(quotedSelect, ",") + " FROM `" + table + "`" + baseWhere + " ORDER BY " + strings.Join(orderParts, ", ") + " LIMIT ? OFFSET ?"
	queryArgs := append(append([]any{}, args...), limit, offset)

	rows, err := q.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	rowsOut := make([]map[string]any, 0)
	for rows.Next() {
		values := make([]any, len(selected))
		pointers := make([]any, len(selected))
		for i := range values {
			pointers[i] = &values[i]
		}
		if err := rows.Scan(pointers...); err != nil {
			return nil, nil, err
		}
		row := make(map[string]any, len(selected))
		for i, name := range selected {
			switch value := values[i].(type) {
			case []byte:
				row[name] = string(value)
			default:
				row[name] = value
			}
		}
		rowsOut = append(rowsOut, row)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	var total *int64
	if includeTotal {
		countQuery := "SELECT COUNT(*) FROM `" + table + "`" + baseWhere
		var count int64
		if err := q.db.QueryRowContext(ctx, countQuery, args...).Scan(&count); err != nil {
			return nil, nil, err
		}
		total = &count
	}
	return rowsOut, total, nil
}

// GetArticleSpecifications returns both article-level criteria and linkage
// restrictions. This mirrors the TecDoc 2024Q1 "all specifications" query
// without flattening LA_CRITERIA into ARTICLE_CRITERIA.
func (q *Queries) GetArticleSpecifications(ctx context.Context, articleID, languageID, countryID int, includeHidden bool, limit, offset int) ([]models.ArticleSpecification, int, error) {
	if limit < 1 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	if offset < 0 {
		offset = 0
	}
	articleWhere := `(
		ARTICLE_CRITERIA.ACR_CTM = 0 OR EXISTS (
			SELECT 1 FROM COUNTRY_RESTRICTIONS
			WHERE COUNTRY_RESTRICTIONS.CNTR_CTM_ID = ARTICLE_CRITERIA.ACR_CTM
				AND COUNTRY_RESTRICTIONS.CNTR_COU_ID = ?
		)
	)`
	linkWhere := `(
		LINK_ART.LA_CTM = 0 OR EXISTS (
			SELECT 1 FROM COUNTRY_RESTRICTIONS
			WHERE COUNTRY_RESTRICTIONS.CNTR_CTM_ID = LINK_ART.LA_CTM
				AND COUNTRY_RESTRICTIONS.CNTR_COU_ID = ?
		)
	) AND (
		LA_CRITERIA.LAC_CTM = 0 OR EXISTS (
			SELECT 1 FROM COUNTRY_RESTRICTIONS
			WHERE COUNTRY_RESTRICTIONS.CNTR_CTM_ID = LA_CRITERIA.LAC_CTM
				AND COUNTRY_RESTRICTIONS.CNTR_COU_ID = ?
		)
	)`

	articleCountWhere := articleWhere
	if !includeHidden {
		articleCountWhere += " AND ARTICLE_CRITERIA.ACR_DISPLAY = 1"
	}
	countQuery := `
		SELECT
			(SELECT COUNT(*) FROM ARTICLE_CRITERIA
			 WHERE ARTICLE_CRITERIA.ACR_ART_ID = ? AND ` + articleCountWhere + `)
			+
			(SELECT COUNT(*) FROM LINK_ART
			 INNER JOIN LA_CRITERIA ON LA_CRITERIA.LAC_LA_ID = LINK_ART.LA_ID
			 WHERE LINK_ART.LA_ART_ID = ? AND ` + linkWhere + `)`
	var total int
	if err := q.db.QueryRowContext(ctx, countQuery, articleID, countryID, articleID, countryID, countryID).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT SOURCE, LINKAGE_ID, PRODUCT_TYPE_ID, CRITERIA_ID, CRITERIA_NAME,
			CRITERIA_SHORT_NAME, CRITERIA_TYPE, CRITERIA_UNIT, RAW_VALUE,
			VALUE_DESCRIPTION, DISPLAY_VALUE
		FROM (
			SELECT
				'article' AS SOURCE,
				NULL AS LINKAGE_ID,
				NULL AS PRODUCT_TYPE_ID,
				ARTICLE_CRITERIA.ACR_CRI_ID AS CRITERIA_ID,
				COALESCE((` + BuildGetTextSubquery("CRITERIA.CRI_DES_ID", languageID) + `), '') AS CRITERIA_NAME,
				COALESCE((` + BuildGetTextSubquery("CRITERIA.CRI_SHORT_DES_ID", languageID) + `), '') AS CRITERIA_SHORT_NAME,
				COALESCE(CRITERIA.CRI_TYPE, '') AS CRITERIA_TYPE,
				COALESCE((` + BuildGetTextSubquery("CRITERIA.CRI_UNIT_DES_ID", languageID) + `), '') AS CRITERIA_UNIT,
				COALESCE(ARTICLE_CRITERIA.ACR_VALUE, '') AS RAW_VALUE,
				COALESCE((` + BuildGetTextSubquery("ARTICLE_CRITERIA.ACR_DES_ID", languageID) + `), '') AS VALUE_DESCRIPTION,
				ARTICLE_CRITERIA.ACR_DISPLAY AS DISPLAY_VALUE
			FROM ARTICLE_CRITERIA
			INNER JOIN CRITERIA ON CRITERIA.CRI_ID = ARTICLE_CRITERIA.ACR_CRI_ID
			WHERE ARTICLE_CRITERIA.ACR_ART_ID = ? AND ` + articleWhere
	if !includeHidden {
		query += " AND ARTICLE_CRITERIA.ACR_DISPLAY = 1\n"
	}
	query += `
			UNION ALL
			SELECT
				'linkage' AS SOURCE,
				LINK_ART.LA_ID AS LINKAGE_ID,
				LINK_ART.LA_PT_ID AS PRODUCT_TYPE_ID,
				LA_CRITERIA.LAC_CRI_ID AS CRITERIA_ID,
				COALESCE((` + BuildGetTextSubquery("CRITERIA.CRI_DES_ID", languageID) + `), '') AS CRITERIA_NAME,
				COALESCE((` + BuildGetTextSubquery("CRITERIA.CRI_SHORT_DES_ID", languageID) + `), '') AS CRITERIA_SHORT_NAME,
				COALESCE(CRITERIA.CRI_TYPE, '') AS CRITERIA_TYPE,
				COALESCE((` + BuildGetTextSubquery("CRITERIA.CRI_UNIT_DES_ID", languageID) + `), '') AS CRITERIA_UNIT,
				COALESCE(LA_CRITERIA.LAC_VALUE, '') AS RAW_VALUE,
				COALESCE((` + BuildGetTextSubquery("LA_CRITERIA.LAC_DES_ID", languageID) + `), '') AS VALUE_DESCRIPTION,
				NULL AS DISPLAY_VALUE
			FROM LINK_ART
			INNER JOIN LA_CRITERIA ON LA_CRITERIA.LAC_LA_ID = LINK_ART.LA_ID
			INNER JOIN CRITERIA ON CRITERIA.CRI_ID = LA_CRITERIA.LAC_CRI_ID
			WHERE LINK_ART.LA_ART_ID = ? AND ` + linkWhere + `
		) AS ALL_SPECIFICATIONS
		ORDER BY SOURCE, COALESCE(LINKAGE_ID, 0), COALESCE(PRODUCT_TYPE_ID, 0), CRITERIA_ID, RAW_VALUE, VALUE_DESCRIPTION
		LIMIT ? OFFSET ?`

	rows, err := q.db.QueryContext(ctx, query, articleID, countryID, articleID, countryID, countryID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]models.ArticleSpecification, 0)
	for rows.Next() {
		var item models.ArticleSpecification
		var linkageID, productTypeID, display sql.NullInt64
		if err := rows.Scan(
			&item.Source,
			&linkageID,
			&productTypeID,
			&item.CriteriaID,
			&item.Name,
			&item.ShortName,
			&item.Type,
			&item.Unit,
			&item.RawValue,
			&item.ValueDescription,
			&display,
		); err != nil {
			return nil, 0, err
		}
		item.ArticleID = articleID
		if linkageID.Valid {
			v := int(linkageID.Int64)
			item.LinkageID = &v
		}
		if productTypeID.Valid {
			v := int(productTypeID.Int64)
			item.ProductTypeID = &v
		}
		if display.Valid {
			v := display.Int64 != 0
			item.Display = &v
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

// GetArticleLinkageInfo exposes LA_INFO text modules associated with every
// LINK_ART row for an article, respecting TecDoc country restrictions.
func (q *Queries) GetArticleLinkageInfo(ctx context.Context, articleID, languageID, countryID, limit, offset int) ([]models.ArticleLinkageInfoItem, int, error) {
	if limit < 1 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	if offset < 0 {
		offset = 0
	}
	countQuery := `SELECT COUNT(*)
		FROM LINK_ART
		INNER JOIN LA_INFO ON LA_INFO.LIN_LA_ID = LINK_ART.LA_ID
		WHERE LINK_ART.LA_ART_ID = ?
			AND (LINK_ART.LA_CTM = 0 OR EXISTS (SELECT 1 FROM COUNTRY_RESTRICTIONS WHERE COUNTRY_RESTRICTIONS.CNTR_CTM_ID = LINK_ART.LA_CTM AND COUNTRY_RESTRICTIONS.CNTR_COU_ID = ?))
			AND (LA_INFO.LIN_CTM = 0 OR EXISTS (SELECT 1 FROM COUNTRY_RESTRICTIONS WHERE COUNTRY_RESTRICTIONS.CNTR_CTM_ID = LA_INFO.LIN_CTM AND COUNTRY_RESTRICTIONS.CNTR_COU_ID = ?))`
	var total int
	if err := q.db.QueryRowContext(ctx, countQuery, articleID, countryID, countryID).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT
			LINK_ART.LA_ID,
			LINK_ART.LA_PT_ID,
			COALESCE((` + BuildGetTextSubquery("LA_INFO.LIN_DES_ID", languageID) + `), ''),
			COALESCE(
				(SELECT TMT_TEXT FROM TEXT_MODULE_TEXTS WHERE TMT_ID = LA_INFO.LIN_TMT_ID AND TMT_LNG_ID = ? LIMIT 1),
				(SELECT TMT_TEXT FROM TEXT_MODULE_TEXTS WHERE TMT_ID = LA_INFO.LIN_TMT_ID AND TMT_LNG_ID = 255 LIMIT 1),
				(SELECT TMT_TEXT FROM TEXT_MODULE_TEXTS WHERE TMT_ID = LA_INFO.LIN_TMT_ID AND TMT_LNG_ID = 4 LIMIT 1),
				''
			),
			LA_INFO.LIN_TMT_ID
		FROM LINK_ART
		INNER JOIN LA_INFO ON LA_INFO.LIN_LA_ID = LINK_ART.LA_ID
		WHERE LINK_ART.LA_ART_ID = ?
			AND (LINK_ART.LA_CTM = 0 OR EXISTS (
				SELECT 1 FROM COUNTRY_RESTRICTIONS
				WHERE COUNTRY_RESTRICTIONS.CNTR_CTM_ID = LINK_ART.LA_CTM
					AND COUNTRY_RESTRICTIONS.CNTR_COU_ID = ?
			))
			AND (LA_INFO.LIN_CTM = 0 OR EXISTS (
				SELECT 1 FROM COUNTRY_RESTRICTIONS
				WHERE COUNTRY_RESTRICTIONS.CNTR_CTM_ID = LA_INFO.LIN_CTM
					AND COUNTRY_RESTRICTIONS.CNTR_COU_ID = ?
			))
		ORDER BY LINK_ART.LA_ID, LA_INFO.LIN_TMT_ID, LA_INFO.LIN_DES_ID
		LIMIT ? OFFSET ?`
	rows, err := q.db.QueryContext(ctx, query, languageID, articleID, countryID, countryID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]models.ArticleLinkageInfoItem, 0)
	for rows.Next() {
		var item models.ArticleLinkageInfoItem
		if err := rows.Scan(&item.LinkageID, &item.ProductTypeID, &item.Type, &item.Text, &item.TextID); err != nil {
			return nil, 0, err
		}
		item.ArticleID = articleID
		items = append(items, item)
	}
	return items, total, rows.Err()
}

// GetPartsDiagramCoordinates implements the TecDoc 2024Q1 LINK_PL_SC example:
// it maps child/component articles to sensitive coordinates on a parent image.
func (q *Queries) GetPartsDiagramCoordinates(ctx context.Context, articleID, limit, offset int) ([]models.PartsDiagramCoordinate, int, error) {
	if limit < 1 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	if offset < 0 {
		offset = 0
	}
	var total int
	if err := q.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM LINK_PL_SC WHERE LPC_ART_ID = ?`, articleID).Scan(&total); err != nil {
		return nil, 0, err
	}
	query := `
		SELECT
			LINK_PL_SC.LPC_ART_ID,
			ARTICLES.ART_ID,
			ARTICLES.ART_ARTICLE_NR,
			ARTICLES.ART_SUP_BRAND,
			ART_MEDIA_INFO.ART_MEDIA_ID,
			ART_MEDIA_INFO.ART_MEDIA_TYPE,
			CASE
				WHEN ART_MEDIA_INFO.ART_MEDIA_TYPE = 'URL' THEN COALESCE(ART_MEDIA_INFO.ART_MEDIA_HIPPERLINK, '')
				WHEN ART_MEDIA_INFO.ART_MEDIA_TYPE = 'PDF' THEN CONCAT_WS('/', 'PDF', ART_MEDIA_INFO.ART_MEDIA_SUP_ID, ART_MEDIA_INFO.ART_MEDIA_FILE_NAME)
				ELSE CONCAT_WS('/', 'IMAGE', ART_MEDIA_INFO.ART_MEDIA_SUP_ID, ART_MEDIA_INFO.ART_MEDIA_FILE_NAME)
			END,
			SENSITIVE_COORDINATES.SEN_COORD_ID,
			SENSITIVE_COORDINATES.SEN_COORD_X,
			SENSITIVE_COORDINATES.SEN_COORD_Y,
			SENSITIVE_COORDINATES.SEN_COORD_WIDTH,
			SENSITIVE_COORDINATES.SEN_COORD_HEIGHT,
			SENSITIVE_COORDINATES.SEN_COORD_TYPE
		FROM LINK_PL_SC
		INNER JOIN ARTICLES ON ARTICLES.ART_ID = LINK_PL_SC.LPC_ART_ID_COM
		INNER JOIN SENSITIVE_COORDINATES
			ON SENSITIVE_COORDINATES.SEN_COORD_ART_ID = LINK_PL_SC.LPC_ART_ID
			AND SENSITIVE_COORDINATES.SEN_COORD_ID = LINK_PL_SC.LPC_SEN_COORD_ID
			AND SENSITIVE_COORDINATES.SEN_COORD_MEDIA_ID = LINK_PL_SC.LPC_MEDIA_ID
			AND SENSITIVE_COORDINATES.SEN_COORD_MEDIA_TYPE = LINK_PL_SC.LPC_MEDIA_TYPE
		INNER JOIN ART_MEDIA_INFO
			ON ART_MEDIA_INFO.ART_MEDIA_ART_ID = LINK_PL_SC.LPC_ART_ID
			AND ART_MEDIA_INFO.ART_MEDIA_ID = LINK_PL_SC.LPC_MEDIA_ID
			AND ART_MEDIA_INFO.ART_MEDIA_TYPE = LINK_PL_SC.LPC_MEDIA_TYPE
		WHERE LINK_PL_SC.LPC_ART_ID = ?
		ORDER BY LINK_PL_SC.LPC_SEQ_NO, SENSITIVE_COORDINATES.SEN_COORD_SEQ_NO, ARTICLES.ART_ID, LINK_PL_SC.LPC_SEN_COORD_ID
		LIMIT ? OFFSET ?`
	rows, err := q.db.QueryContext(ctx, query, articleID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]models.PartsDiagramCoordinate, 0)
	for rows.Next() {
		var item models.PartsDiagramCoordinate
		if err := rows.Scan(
			&item.ParentArticleID,
			&item.ComponentArticleID,
			&item.ArticleNumber,
			&item.Brand,
			&item.MediaID,
			&item.MediaType,
			&item.MediaSource,
			&item.CoordinateID,
			&item.X,
			&item.Y,
			&item.Width,
			&item.Height,
			&item.Type,
		); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

const exportCrossReferencesSQL = `
	SELECT ARTICLES.ART_ARTICLE_NR, ARTICLES.ART_SUP_BRAND,
		export_rows.ARL_DISPLAY_NR, export_rows.ARL_BRA_BRAND, export_rows.ARL_TYPE
	FROM (
		SELECT ARL_ART_ID, ARL_DISPLAY_NR, COALESCE(ARL_BRA_BRAND, '') AS ARL_BRA_BRAND, ARL_TYPE, ARL_BRA_ID
		FROM ART_LOOKUP FORCE INDEX (PRIMARY)
		WHERE ARL_TYPE IN ('OENumber','IAMNumber')
		ORDER BY ARL_ART_ID, ARL_DISPLAY_NR, ARL_TYPE, ARL_BRA_ID
		LIMIT ? OFFSET ?
	) AS export_rows
	INNER JOIN ARTICLES ON ARTICLES.ART_ID = export_rows.ARL_ART_ID
	ORDER BY export_rows.ARL_ART_ID, export_rows.ARL_DISPLAY_NR, export_rows.ARL_TYPE, export_rows.ARL_BRA_ID`

const exportEANSQL = `
	SELECT ARTICLES.ART_ARTICLE_NR, ARTICLES.ART_SUP_BRAND, export_rows.ARL_DISPLAY_NR
	FROM (
		SELECT ARL_ART_ID, ARL_DISPLAY_NR, ARL_BRA_ID
		FROM ART_LOOKUP FORCE INDEX (ARL_TYPE_INDX)
		WHERE ARL_TYPE = 'EAN'
		ORDER BY ARL_TYPE, ARL_ART_ID, ARL_DISPLAY_NR, ARL_BRA_ID
		LIMIT ? OFFSET ?
	) AS export_rows
	INNER JOIN ARTICLES ON ARTICLES.ART_ID = export_rows.ARL_ART_ID
	ORDER BY export_rows.ARL_ART_ID, export_rows.ARL_DISPLAY_NR, export_rows.ARL_BRA_ID`

func (q *Queries) ExportCrossReferences(ctx context.Context, limit, offset int) ([]models.CrossReferenceExportRow, error) {
	if limit < 1 || limit > 5000 {
		limit = 1000
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := q.db.QueryContext(ctx, exportCrossReferencesSQL, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]models.CrossReferenceExportRow, 0, limit)
	for rows.Next() {
		var item models.CrossReferenceExportRow
		if err := rows.Scan(&item.ArticleNumber, &item.ArticleBrand, &item.CrossNumber, &item.CrossBrand, &item.CrossType); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (q *Queries) ExportEAN(ctx context.Context, limit, offset int) ([]models.EANExportRow, error) {
	if limit < 1 || limit > 5000 {
		limit = 1000
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := q.db.QueryContext(ctx, exportEANSQL, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]models.EANExportRow, 0, limit)
	for rows.Next() {
		var item models.EANExportRow
		if err := rows.Scan(&item.ArticleNumber, &item.ArticleBrand, &item.EAN); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (q *Queries) GetCriterionKeyValues(ctx context.Context, criteriaID, languageID, limit, offset int) ([]models.CriterionKeyValue, int, error) {
	if limit < 1 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	if offset < 0 {
		offset = 0
	}
	var keyTableID sql.NullInt64
	if err := q.db.QueryRowContext(ctx, `SELECT CRI_KT_ID FROM CRITERIA WHERE CRI_ID = ?`, criteriaID).Scan(&keyTableID); err != nil {
		return nil, 0, err
	}
	if !keyTableID.Valid || keyTableID.Int64 <= 0 {
		return []models.CriterionKeyValue{}, 0, nil
	}
	var total int
	if err := q.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM KEY_VALUES WHERE KV_KT_ID = ?`, keyTableID.Int64).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := q.db.QueryContext(ctx, `
		SELECT KEY_VALUES.KV_KT_ID, KEY_VALUES.KV_KV,
			COALESCE((`+BuildGetTextSubquery("KEY_VALUES.KV_NAME_DES", languageID)+`), ''),
			COALESCE((`+BuildGetTextSubquery("KEY_VALUES.KV_DES_ID", languageID)+`), '')
		FROM KEY_VALUES
		WHERE KEY_VALUES.KV_KT_ID = ?
		ORDER BY KEY_VALUES.KV_KV, KEY_VALUES.KV_DES_ID, KEY_VALUES.KV_NAME_DES
		LIMIT ? OFFSET ?`, keyTableID.Int64, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]models.CriterionKeyValue, 0)
	for rows.Next() {
		var item models.CriterionKeyValue
		item.CriteriaID = criteriaID
		if err := rows.Scan(&item.KeyTableID, &item.Value, &item.Name, &item.Description); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (q *Queries) GetAxleDetail(ctx context.Context, axleID, languageID int) (*models.AxleDetail, error) {
	query := `
		SELECT
			AXLES.AXL_ID,
			AXLES.AXL_MFA_ID,
			COALESCE(MANUFACTURERS.MFA_BRAND, ''),
			AXLES.AXL_MS_ID,
			COALESCE((` + BuildGetTextSubquery("MODELS_SERIES.MS_NAME_DES", languageID) + `), ''),
			AXLES.AXL_CODE,
			COALESCE(AXLES.AXL_DESCRIPTION, ''),
			AXLES.AXL_AXLE_STYLE,
			COALESCE((` + BuildGetTextSubquery("AXLES.AXL_AXLE_STYLE", languageID) + `), ''),
			AXLES.AXL_AXLE_BODY,
			COALESCE((` + BuildGetTextSubquery("AXLES.AXL_AXLE_BODY", languageID) + `), ''),
			AXLES.AXL_AXLE_TYPE,
			COALESCE((` + BuildGetTextSubquery("AXLES.AXL_AXLE_TYPE", languageID) + `), ''),
			COALESCE(DATE_FORMAT(AXLES.AXL_CI_FROM, '%Y-%m-%d'), ''),
			COALESCE(DATE_FORMAT(AXLES.AXL_CI_TO, '%Y-%m-%d'), ''),
			AXLES.AXL_ALLOWED_WEIGHT_KG_START,
			AXLES.AXL_ALLOWED_WEIGHT_KG_UPTO,
			AXLES.AXL_BRAKE_TYPE,
			COALESCE((` + BuildGetTextSubquery("AXLES.AXL_BRAKE_TYPE", languageID) + `), ''),
			COALESCE(AXLES.AXL_HUB_SYSTEM, ''),
			AXLES.AXL_DISTANCE_TO_VEHICLE_FRAME_MM_START,
			AXLES.AXL_DISTANCE_TO_VEHICLE_FRAME_MM_UPTO,
			AXLES.AXL_WHEEL_MOUNTING,
			COALESCE((` + BuildGetTextSubquery("AXLES.AXL_WHEEL_MOUNTING", languageID) + `), ''),
			AXLES.AXL_TRACK_WIDTH
		FROM AXLES
		LEFT JOIN MANUFACTURERS ON MANUFACTURERS.MFA_ID = AXLES.AXL_MFA_ID
		LEFT JOIN MODELS_SERIES ON MODELS_SERIES.MS_ID = AXLES.AXL_MS_ID
		WHERE AXLES.AXL_ID = ?`
	var item models.AxleDetail
	var axleStyle, axleBody, axleType, weightStart, weightUpto, brakeType, distanceStart, distanceUpto, wheelMounting, trackWidth sql.NullInt64
	if err := q.db.QueryRowContext(ctx, query, axleID).Scan(
		&item.ID,
		&item.ManufacturerID,
		&item.Manufacturer,
		&item.ModelSeriesID,
		&item.ModelSeries,
		&item.Code,
		&item.Description,
		&axleStyle,
		&item.AxleStyle,
		&axleBody,
		&item.AxleBody,
		&axleType,
		&item.AxleType,
		&item.ProductionFrom,
		&item.ProductionTo,
		&weightStart,
		&weightUpto,
		&brakeType,
		&item.BrakeType,
		&item.HubSystem,
		&distanceStart,
		&distanceUpto,
		&wheelMounting,
		&item.WheelMounting,
		&trackWidth,
	); err != nil {
		return nil, err
	}
	item.AxleStyleID = nullInt(axleStyle)
	item.AxleBodyID = nullInt(axleBody)
	item.AxleTypeID = nullInt(axleType)
	item.AllowedWeightKGStart = nullInt(weightStart)
	item.AllowedWeightKGUpTo = nullInt(weightUpto)
	item.BrakeTypeID = nullInt(brakeType)
	item.DistanceToVehicleFrameMMStart = nullInt(distanceStart)
	item.DistanceToVehicleFrameMMUpTo = nullInt(distanceUpto)
	item.WheelMountingID = nullInt(wheelMounting)
	item.TrackWidthMM = nullInt(trackWidth)
	item.Models = []models.AxleModelName{}
	item.BrakeSizes = []models.AxleBrakeSize{}

	rows, err := q.db.QueryContext(ctx, `
		SELECT AXL_COU_ID, AXL_SORTNO, AXL_MODEL
		FROM AXLE_MODELS WHERE AXL_ID = ? ORDER BY AXL_COU_ID, AXL_SORTNO, AXL_MODEL`, axleID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var model models.AxleModelName
		if err := rows.Scan(&model.CountryID, &model.Sort, &model.Model); err != nil {
			rows.Close()
			return nil, err
		}
		item.Models = append(item.Models, model)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}

	rows, err = q.db.QueryContext(ctx, `
		SELECT ABS_SEQ_NO, COALESCE(ABS_SIZE, ''), COALESCE(ABS_DESCRIPTION, '')
		FROM AXL_BRAKE_SIZES WHERE ABS_AXL_ID = ? ORDER BY ABS_SEQ_NO`, axleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var brake models.AxleBrakeSize
		if err := rows.Scan(&brake.Sequence, &brake.Size, &brake.Description); err != nil {
			return nil, err
		}
		item.BrakeSizes = append(item.BrakeSizes, brake)
	}
	return &item, rows.Err()
}
