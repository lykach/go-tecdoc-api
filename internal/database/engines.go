package database

import (
	"database/sql"
	"fmt"

	"go-tecdoc-api/internal/models"
)

type engineScanner interface {
	Scan(dest ...any) error
}

func engineSelectSQL(languageID int) string {
	return `
		SELECT
			ENGINES.ENG_ID,
			ENGINES.ENG_CODE,
			ENGINES.ENG_MFA_ID,
			ENGINES.ENG_POWER_KW_START,
			ENGINES.ENG_POWER_KW_UPTO,
			ENGINES.ENG_POWER_PS_START,
			ENGINES.ENG_POWER_PS_UPTO,
			ENGINES.ENG_CAPACITY_CCM_START,
			ENGINES.ENG_CAPACITY_CCM_UPTO,
			ENGINES.ENG_CI_FROM,
			ENGINES.ENG_CI_TO,
			ENGINES.ENG_COMPRESSION_START,
			ENGINES.ENG_COMPRESSION_UPTO,
			ENGINES.ENG_TORQUE_NM_START,
			ENGINES.ENG_TORQUE_NM_UPTO,
			ENGINES.ENG_BORE,
			ENGINES.ENG_STROKE,
			ENGINES.ENG_NUMBER_OF_CYLINDERS,
			ENGINES.ENG_NUMBER_OF_MAIN_BEARINGS,
			ENGINES.ENG_NUMBER_OF_VALVES,
			(` + BuildGetTextSubquery("ENGINES.ENG_ENGINE_CONSTRUCTION", languageID) + `) AS ENGINE_CONSTRUCTION,
			(` + BuildGetTextSubquery("ENGINES.ENG_FUEL_TYPE", languageID) + `) AS FUEL_TYPE,
			(` + BuildGetTextSubquery("ENGINES.ENG_FUEL_MIXTURE", languageID) + `) AS FUEL_MIXTURE,
			(` + BuildGetTextSubquery("ENGINES.ENG_CHARGE_TYPE", languageID) + `) AS CHARGE_TYPE,
			(` + BuildGetTextSubquery("ENGINES.ENG_EMISSION_NORM", languageID) + `) AS EMISSION_NORM,
			(` + BuildGetTextSubquery("ENGINES.ENG_CYLINDER_CONSTRUCTION", languageID) + `) AS CYLINDER_CONSTRUCTION,
			(` + BuildGetTextSubquery("ENGINES.ENG_ENGINE_MANAGEMENT", languageID) + `) AS ENGINE_MANAGEMENT,
			(` + BuildGetTextSubquery("ENGINES.ENG_VALVE_MANAGEMENT", languageID) + `) AS VALVE_MANAGEMENT,
			(` + BuildGetTextSubquery("ENGINES.ENG_COOLING_TYPE", languageID) + `) AS COOLING_TYPE,
			(` + BuildGetTextSubquery("ENGINES.ENG_ENGINE_TYPE", languageID) + `) AS ENGINE_TYPE
		FROM ENGINES`
}

func scanEngineDetail(scanner engineScanner) (*models.EngineDetail, error) {
	var ed models.EngineDetail
	var dateFrom, dateTo sql.NullString
	var powerKwStart, powerKwUpto, powerPsStart, powerPsUpto sql.NullFloat64
	var capacityStart, capacityUpto sql.NullFloat64
	var compressionStart, compressionUpto, torqueStart, torqueUpto sql.NullFloat64
	var bore, stroke sql.NullFloat64
	var cylinders, mainBearings, valves sql.NullInt64
	var engineConstruction, fuelType, fuelMixture, chargeType sql.NullString
	var emissionNorm, cylinderConstruction, engineManagement sql.NullString
	var valveManagement, coolingType, engineType sql.NullString

	if err := scanner.Scan(
		&ed.EngineID, &ed.EngineCode, &ed.ManufacturerID,
		&powerKwStart, &powerKwUpto, &powerPsStart, &powerPsUpto,
		&capacityStart, &capacityUpto, &dateFrom, &dateTo,
		&compressionStart, &compressionUpto, &torqueStart, &torqueUpto,
		&bore, &stroke, &cylinders, &mainBearings, &valves,
		&engineConstruction, &fuelType, &fuelMixture, &chargeType,
		&emissionNorm, &cylinderConstruction, &engineManagement,
		&valveManagement, &coolingType, &engineType,
	); err != nil {
		return nil, err
	}

	ed.Specs = make(map[string]interface{})
	putFloat := func(key string, v sql.NullFloat64) {
		if v.Valid {
			ed.Specs[key] = v.Float64
		}
	}
	putInt := func(key string, v sql.NullInt64) {
		if v.Valid {
			ed.Specs[key] = int(v.Int64)
		}
	}
	putString := func(key string, v sql.NullString) {
		if v.Valid && v.String != "" {
			ed.Specs[key] = v.String
		}
	}
	putFloat("power_kw_from", powerKwStart)
	putFloat("power_kw_to", powerKwUpto)
	putFloat("power_ps_from", powerPsStart)
	putFloat("power_ps_to", powerPsUpto)
	putFloat("capacity_from", capacityStart)
	putFloat("capacity_to", capacityUpto)
	putFloat("compression_from", compressionStart)
	putFloat("compression_to", compressionUpto)
	putFloat("torque_nm_from", torqueStart)
	putFloat("torque_nm_to", torqueUpto)
	putFloat("bore", bore)
	putFloat("stroke", stroke)
	putInt("cylinders", cylinders)
	putInt("main_bearings", mainBearings)
	putInt("valves", valves)
	putString("engine_construction", engineConstruction)
	putString("fuel_type", fuelType)
	putString("fuel_mixture", fuelMixture)
	putString("charge_type", chargeType)
	putString("emission_norm", emissionNorm)
	putString("cylinder_construction", cylinderConstruction)
	putString("engine_management", engineManagement)
	putString("valve_management", valveManagement)
	putString("cooling_type", coolingType)
	putString("engine_type", engineType)
	if dateFrom.Valid {
		ed.DateFrom = dateFrom.String
	}
	if dateTo.Valid {
		ed.DateTo = dateTo.String
	}
	return &ed, nil
}

// GetEngineByID returns a single TecDoc engine master row.
func (q *Queries) GetEngineByID(engID int, languageID int) (*models.EngineDetail, error) {
	row := q.db.QueryRow(engineSelectSQL(languageID)+` WHERE ENGINES.ENG_ID = ?`, engID)
	ed, err := scanEngineDetail(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return ed, nil
}

// GetEngines returns a stable, paged view of the TecDoc ENGINES master table.
// manufacturerID=nil means all manufacturers.
func (q *Queries) GetEngines(manufacturerID *int, languageID, limit, offset int) ([]models.EngineDetail, error) {
	query := engineSelectSQL(languageID)
	args := make([]any, 0, 3)
	if manufacturerID != nil {
		query += ` WHERE ENGINES.ENG_MFA_ID = ?`
		args = append(args, *manufacturerID)
	}
	query += ` ORDER BY ENGINES.ENG_ID LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := q.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.EngineDetail, 0, limit)
	for rows.Next() {
		ed, err := scanEngineDetail(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *ed)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (q *Queries) CountEngines(manufacturerID *int) (int, error) {
	query := `SELECT COUNT(*) FROM ENGINES`
	args := []any{}
	if manufacturerID != nil {
		query += ` WHERE ENGINES.ENG_MFA_ID = ?`
		args = append(args, *manufacturerID)
	}
	var total int
	if err := q.db.QueryRow(query, args...).Scan(&total); err != nil {
		return 0, fmt.Errorf("count engines: %w", err)
	}
	return total, nil
}
