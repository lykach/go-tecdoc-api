package database

import (
	"database/sql"
	"fmt"
	"strings"

	"go-tecdoc-api/internal/models"
)

// VehicleImportStats is a cheap aggregate view used by bulk import previews.
// It deliberately counts directly in TecDoc instead of forcing a client to
// enumerate every manufacturer/model/vehicle page just to compute totals.
type VehicleImportStats struct {
	ManufacturersTotal int            `json:"manufacturers_total"`
	ModelsTotal        int            `json:"models_total"`
	VehiclesTotal      int            `json:"vehicles_total"`
	EnginesTotal       int            `json:"engines_total"`
	ByType             map[string]int `json:"by_type"`
}

// GetVehicleImportStats returns exact source-table counts for the selected
// catalog target types. Supported values are PC, LCV, E-PC, E-LCV, CV,
// Motorcycle and Engine. Passenger subtypes share the PC manufacturer/model
// family but keep independent PASSENGER_CARS.PC_TYPE counts.
func (q *Queries) GetVehicleImportStats(vehicleTypes []string) (*VehicleImportStats, error) {
	selected := make(map[string]struct{}, len(vehicleTypes))
	for _, raw := range vehicleTypes {
		v := strings.TrimSpace(raw)
		if v != "" {
			selected[v] = struct{}{}
		}
	}

	manufacturerFamilies := make([]string, 0, 4)
	modelFamilies := make([]string, 0, 3)
	passengerTypes := make([]string, 0, 4)
	for _, v := range []string{"PC", "LCV", "E-PC", "E-LCV"} {
		if _, ok := selected[v]; ok {
			passengerTypes = append(passengerTypes, v)
		}
	}
	if len(passengerTypes) > 0 {
		manufacturerFamilies = append(manufacturerFamilies, "PC")
		modelFamilies = append(modelFamilies, "PC")
	}
	if _, ok := selected["CV"]; ok {
		manufacturerFamilies = append(manufacturerFamilies, "CV")
		modelFamilies = append(modelFamilies, "CV")
	}
	if _, ok := selected["Motorcycle"]; ok {
		manufacturerFamilies = append(manufacturerFamilies, "Motorcycle")
		modelFamilies = append(modelFamilies, "Motorcycle")
	}
	if _, ok := selected["Engine"]; ok {
		manufacturerFamilies = append(manufacturerFamilies, "Engine")
	}

	result := &VehicleImportStats{ByType: make(map[string]int, len(selected))}

	if len(manufacturerFamilies) > 0 {
		clauses := make([]string, 0, len(manufacturerFamilies))
		args := make([]any, 0, len(manufacturerFamilies))
		for _, family := range manufacturerFamilies {
			clauses = append(clauses, "FIND_IN_SET(?, MFA_TYPE) > 0")
			args = append(args, family)
		}
		query := `SELECT COUNT(DISTINCT MFA_ID) FROM MANUFACTURERS WHERE MFA_MODELS_COUNT > 0 AND (` + strings.Join(clauses, " OR ") + `)`
		if err := q.db.QueryRow(query, args...).Scan(&result.ManufacturersTotal); err != nil {
			return nil, fmt.Errorf("count import manufacturers: %w", err)
		}
	}

	if len(modelFamilies) > 0 {
		clauses := make([]string, 0, len(modelFamilies))
		args := make([]any, 0, len(modelFamilies))
		for _, family := range modelFamilies {
			clauses = append(clauses, "FIND_IN_SET(?, MS_TYPE) > 0")
			args = append(args, family)
		}
		query := `SELECT COUNT(DISTINCT MS_ID) FROM MODELS_SERIES WHERE ` + strings.Join(clauses, " OR ")
		if err := q.db.QueryRow(query, args...).Scan(&result.ModelsTotal); err != nil {
			return nil, fmt.Errorf("count import model series: %w", err)
		}
	}

	if len(passengerTypes) > 0 {
		// PC_TYPE is a MySQL SET, not a scalar enum. A row can therefore carry
		// multiple memberships (for example "PC,LCV"). Count each requested
		// subtype by SET membership and count the selected passenger population
		// only once for VehiclesTotal.
		typeCounts := map[string]int{}
		var pcCount, lcvCount, epcCount, elcvCount int
		if err := q.db.QueryRow(`
SELECT
    COUNT(CASE WHEN FIND_IN_SET('PC', PC_TYPE) > 0 THEN 1 END),
    COUNT(CASE WHEN FIND_IN_SET('LCV', PC_TYPE) > 0 THEN 1 END),
    COUNT(CASE WHEN FIND_IN_SET('E-PC', PC_TYPE) > 0 THEN 1 END),
    COUNT(CASE WHEN FIND_IN_SET('E-LCV', PC_TYPE) > 0 THEN 1 END)
FROM PASSENGER_CARS`).Scan(&pcCount, &lcvCount, &epcCount, &elcvCount); err != nil {
			return nil, fmt.Errorf("count passenger import subtype memberships: %w", err)
		}
		typeCounts["PC"] = pcCount
		typeCounts["LCV"] = lcvCount
		typeCounts["E-PC"] = epcCount
		typeCounts["E-LCV"] = elcvCount

		whereParts := make([]string, 0, len(passengerTypes))
		whereArgs := make([]any, 0, len(passengerTypes))
		for _, v := range passengerTypes {
			result.ByType[v] = typeCounts[v]
			whereParts = append(whereParts, "FIND_IN_SET(?, PC_TYPE) > 0")
			whereArgs = append(whereArgs, v)
		}
		var selectedPassengerCount int
		if err := q.db.QueryRow(`SELECT COUNT(*) FROM PASSENGER_CARS WHERE `+strings.Join(whereParts, " OR "), whereArgs...).Scan(&selectedPassengerCount); err != nil {
			return nil, fmt.Errorf("count selected passenger import vehicles: %w", err)
		}
		result.VehiclesTotal += selectedPassengerCount
	}

	if _, ok := selected["CV"]; ok {
		var count int
		if err := q.db.QueryRow(`SELECT COUNT(*) FROM COMMERCIAL_VEHICLES`).Scan(&count); err != nil {
			return nil, fmt.Errorf("count commercial import vehicles: %w", err)
		}
		result.ByType["CV"] = count
		result.VehiclesTotal += count
	}
	if _, ok := selected["Motorcycle"]; ok {
		var count int
		if err := q.db.QueryRow(`SELECT COUNT(*) FROM MOTORBIKES`).Scan(&count); err != nil {
			return nil, fmt.Errorf("count motorcycle import vehicles: %w", err)
		}
		result.ByType["Motorcycle"] = count
		result.VehiclesTotal += count
	}
	if _, ok := selected["Engine"]; ok {
		if err := q.db.QueryRow(`SELECT COUNT(*) FROM ENGINES`).Scan(&result.EnginesTotal); err != nil {
			return nil, fmt.Errorf("count import engines: %w", err)
		}
		result.ByType["Engine"] = result.EnginesTotal
	}

	return result, nil
}

// GetImportProductGroups returns a stable, flat, paged view of SEARCH_TREE for
// bulk importers. A single row is returned per SEARCH_TREE node even when the
// node belongs to multiple selected target families. This avoids recursively
// issuing one HTTP request per tree node.
func (q *Queries) GetImportProductGroups(vehicleTypes []string, languageID, limit, offset int) ([]models.Category, int, error) {
	if limit <= 0 || limit > 500 {
		limit = 500
	}
	if offset < 0 {
		offset = 0
	}

	clauses := make([]string, 0, len(vehicleTypes))
	filterArgs := make([]any, 0, len(vehicleTypes))
	for _, raw := range vehicleTypes {
		vehicleType := strings.TrimSpace(raw)
		if vehicleType == "" {
			continue
		}
		clauses = append(clauses, "FIND_IN_SET(?, SEARCH_TREE.STR_TYPE) > 0")
		filterArgs = append(filterArgs, vehicleType)
	}
	if len(clauses) == 0 {
		return []models.Category{}, 0, nil
	}
	where := "(" + strings.Join(clauses, " OR ") + ")"

	var total int
	if err := q.db.QueryRow(`SELECT COUNT(*) FROM SEARCH_TREE WHERE `+where, filterArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count import product groups: %w", err)
	}

	query := `
SELECT
    SEARCH_TREE.STR_ID,
    SEARCH_TREE.STR_ID_PARENT,
    SEARCH_TREE.STR_LEVEL,
    SEARCH_TREE.STR_TYPE,
    (` + BuildGetTextSubquery("SEARCH_TREE.STR_DES_ID", languageID) + `) AS STR_NODE_NAME
FROM SEARCH_TREE
WHERE ` + where + `
ORDER BY SEARCH_TREE.STR_ID
LIMIT ? OFFSET ?`
	args := append(append([]any(nil), filterArgs...), limit, offset)
	rows, err := q.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("fetch import product groups: %w", err)
	}
	defer rows.Close()

	categories := make([]models.Category, 0, min(limit, total))
	for rows.Next() {
		var item models.Category
		var parentID sql.NullInt64
		if err := rows.Scan(&item.ID, &parentID, &item.Level, &item.Type, &item.Name); err != nil {
			return nil, 0, fmt.Errorf("scan import product group: %w", err)
		}
		if parentID.Valid {
			value := int(parentID.Int64)
			item.ParentID = &value
		}
		item.Path = item.Name
		categories = append(categories, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate import product groups: %w", err)
	}
	return categories, total, nil
}
