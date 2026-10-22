package database

import (
	"database/sql"
	"strings"

	"go-tecdoc-api/internal/models"
)

// GetManufacturers повертає список виробників з фільтрацією
func (q *Queries) GetManufacturers(vehicleType string, limit, offset int) ([]models.Manufacturer, error) {
	vehicleType = strings.TrimSpace(vehicleType)
	availabilityPredicate := "MFA_MODELS_COUNT > 0"
	if vehicleType == "Engine" {
		// Engine is a global TecDoc master target. Some manufacturers have
		// ENGINES rows without ordinary model-series rows, so MFA_MODELS_COUNT
		// is not a safe availability predicate for Engine imports.
		availabilityPredicate = "EXISTS (SELECT 1 FROM ENGINES WHERE ENGINES.ENG_MFA_ID = MANUFACTURERS.MFA_ID)"
	}
	query := `
		SELECT
			MFA_ID,
			MFA_BRAND,
			MFA_TYPE,
			MFA_MODELS_COUNT,
			MFA_SUP_ID
		FROM
			MANUFACTURERS
		WHERE
			FIND_IN_SET(?, MFA_TYPE) > 0
			AND ` + availabilityPredicate + `
		ORDER BY
			MFA_BRAND, MFA_ID
		LIMIT ? OFFSET ?
	`

	rows, err := q.db.Query(query, vehicleType, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var manufacturers []models.Manufacturer
	for rows.Next() {
		var m models.Manufacturer
		var supID sql.NullInt64

		err := rows.Scan(
			&m.ID,
			&m.Brand,
			&m.Type,
			&m.ModelsCount,
			&supID,
		)
		if err != nil {
			return nil, err
		}

		if supID.Valid {
			val := int(supID.Int64)
			m.SupplierID = &val
		}

		manufacturers = append(manufacturers, m)
	}

	return manufacturers, rows.Err()
}

// GetManufacturerByID повертає детальну інформацію про виробника
func (q *Queries) GetManufacturerByID(id int) (*models.ManufacturerDetail, error) {
	query := `
		SELECT
			MFA_ID,
			MFA_BRAND,
			MFA_TYPE,
			MFA_MODELS_COUNT,
			MFA_SUP_ID
		FROM
			MANUFACTURERS
		WHERE
			MFA_ID = ?
	`

	var m models.ManufacturerDetail
	var supID sql.NullInt64

	err := q.db.QueryRow(query, id).Scan(
		&m.ID,
		&m.Brand,
		&m.Type,
		&m.ModelsCount,
		&supID,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if supID.Valid {
		val := int(supID.Int64)
		m.SupplierID = &val
	}

	return &m, nil
}

// CountManufacturers повертає загальну кількість виробників
func (q *Queries) CountManufacturers(vehicleType string) (int, error) {
	vehicleType = strings.TrimSpace(vehicleType)
	availabilityPredicate := "MFA_MODELS_COUNT > 0"
	if vehicleType == "Engine" {
		availabilityPredicate = "EXISTS (SELECT 1 FROM ENGINES WHERE ENGINES.ENG_MFA_ID = MANUFACTURERS.MFA_ID)"
	}
	query := `
		SELECT
			COUNT(*)
		FROM
			MANUFACTURERS
		WHERE
			FIND_IN_SET(?, MFA_TYPE) > 0
			AND ` + availabilityPredicate + `
	`

	var count int
	err := q.db.QueryRow(query, vehicleType).Scan(&count)
	return count, err
}
