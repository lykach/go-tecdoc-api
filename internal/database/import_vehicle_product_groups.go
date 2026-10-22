package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"go-tecdoc-api/internal/models"
)

// GetImportVehicleProductGroupIDs fetches applicable root SEARCH_TREE ids for
// many vehicle IDs. The query intentionally performs no text translation and
// de-duplicates vehicle/product-type and product-type/root mappings before the
// final join. This keeps the importer path bounded for models with many
// applicability rows.
func (q *Queries) GetImportVehicleProductGroupIDs(ctx context.Context, vehicleIDs []int, vehicleType string) ([]models.VehicleProductGroupIDs, error) {
	if len(vehicleIDs) == 0 {
		return []models.VehicleProductGroupIDs{}, nil
	}
	if len(vehicleIDs) > 500 {
		return nil, fmt.Errorf("vehicle_ids exceeds hard limit of 500")
	}

	vehicleType = strings.TrimSpace(vehicleType)
	if vehicleType != "PC" && vehicleType != "CV" && vehicleType != "Motorcycle" {
		return nil, fmt.Errorf("unsupported vehicle_type %q", vehicleType)
	}

	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(vehicleIDs)), ",")
	query := `
SELECT
    vehicle_pt.LAT_TYP_ID,
    root_map.STR_ID
FROM (
    SELECT
        LINK_LA_TYP.LAT_TYP_ID,
        LINK_LA_TYP.LAT_PT_ID
    FROM LINK_LA_TYP
    WHERE LINK_LA_TYP.LAT_TYPE = ?
      AND LINK_LA_TYP.LAT_TYP_ID IN (` + placeholders + `)
    GROUP BY LINK_LA_TYP.LAT_TYP_ID, LINK_LA_TYP.LAT_PT_ID
) AS vehicle_pt
INNER JOIN (
    SELECT
        LINK_PT_STR.PT_ID,
        LINK_PT_STR.STR_ID
    FROM LINK_PT_STR
    INNER JOIN SEARCH_TREE
        ON SEARCH_TREE.STR_ID = LINK_PT_STR.STR_ID
       AND SEARCH_TREE.STR_ID_PARENT IS NULL
       AND FIND_IN_SET(?, SEARCH_TREE.STR_TYPE) > 0
    WHERE FIND_IN_SET(?, LINK_PT_STR.STR_TYPE) > 0
    GROUP BY LINK_PT_STR.PT_ID, LINK_PT_STR.STR_ID
) AS root_map
    ON root_map.PT_ID = vehicle_pt.LAT_PT_ID
GROUP BY vehicle_pt.LAT_TYP_ID, root_map.STR_ID
ORDER BY vehicle_pt.LAT_TYP_ID, root_map.STR_ID`

	args := make([]any, 0, len(vehicleIDs)+3)
	args = append(args, vehicleType)
	for _, id := range vehicleIDs {
		args = append(args, id)
	}
	args = append(args, vehicleType, vehicleType)

	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("fetch import vehicle product group ids: %w", err)
	}
	defer rows.Close()

	byVehicle := make(map[int][]int, len(vehicleIDs))
	for rows.Next() {
		var vehicleID, categoryID int
		if err := rows.Scan(&vehicleID, &categoryID); err != nil {
			return nil, fmt.Errorf("scan import vehicle product group id: %w", err)
		}
		byVehicle[vehicleID] = append(byVehicle[vehicleID], categoryID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate import vehicle product group ids: %w", err)
	}

	result := make([]models.VehicleProductGroupIDs, 0, len(vehicleIDs))
	for _, vehicleID := range vehicleIDs {
		ids := byVehicle[vehicleID]
		if ids == nil {
			ids = []int{}
		}
		result = append(result, models.VehicleProductGroupIDs{VehicleID: vehicleID, CategoryIDs: ids})
	}
	return result, nil
}

// GetImportVehicleProductGroups fetches the same root identities as the compact
// importer query and enriches only the unique root nodes with metadata. Keeping
// translation out of the large applicability join prevents correlated text
// subqueries from being evaluated against an exploded intermediate result.
func (q *Queries) GetImportVehicleProductGroups(ctx context.Context, vehicleIDs []int, vehicleType string, languageID int) ([]models.VehicleProductGroups, error) {
	idItems, err := q.GetImportVehicleProductGroupIDs(ctx, vehicleIDs, vehicleType)
	if err != nil {
		return nil, err
	}

	uniqueIDs := make([]int, 0)
	seen := make(map[int]struct{})
	for _, item := range idItems {
		for _, id := range item.CategoryIDs {
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			uniqueIDs = append(uniqueIDs, id)
		}
	}

	categoriesByID := make(map[int]models.Category, len(uniqueIDs))
	if len(uniqueIDs) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(uniqueIDs)), ",")
		query := `
SELECT
    SEARCH_TREE.STR_ID,
    SEARCH_TREE.STR_ID_PARENT,
    SEARCH_TREE.STR_LEVEL,
    SEARCH_TREE.STR_TYPE,
    (` + BuildGetTextSubquery("SEARCH_TREE.STR_DES_ID", languageID) + `) AS STR_NODE_NAME
FROM SEARCH_TREE
WHERE SEARCH_TREE.STR_ID IN (` + placeholders + `)
ORDER BY SEARCH_TREE.STR_ID`

		args := make([]any, 0, len(uniqueIDs))
		for _, id := range uniqueIDs {
			args = append(args, id)
		}
		rows, err := q.db.QueryContext(ctx, query, args...)
		if err != nil {
			return nil, fmt.Errorf("fetch import vehicle product group metadata: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var category models.Category
			var parentID sql.NullInt64
			if err := rows.Scan(&category.ID, &parentID, &category.Level, &category.Type, &category.Name); err != nil {
				return nil, fmt.Errorf("scan import vehicle product group metadata: %w", err)
			}
			if parentID.Valid {
				value := int(parentID.Int64)
				category.ParentID = &value
			}
			category.Path = category.Name
			categoriesByID[category.ID] = category
		}
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("iterate import vehicle product group metadata: %w", err)
		}
	}

	result := make([]models.VehicleProductGroups, 0, len(idItems))
	for _, item := range idItems {
		categories := make([]models.Category, 0, len(item.CategoryIDs))
		for _, id := range item.CategoryIDs {
			if category, ok := categoriesByID[id]; ok {
				categories = append(categories, category)
			}
		}
		result = append(result, models.VehicleProductGroups{VehicleID: item.VehicleID, Categories: categories})
	}
	return result, nil
}
