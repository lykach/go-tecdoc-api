package database

import (
	"database/sql"
	"go-tecdoc-api/internal/models"
)

// GetEngineByID повертає детальну інформацію про двигун
func (q *Queries) GetEngineByID(engID int, languageID int) (*models.EngineDetail, error) {
	query := `
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
		FROM 
			ENGINES
		WHERE 
			ENGINES.ENG_ID = ?
	`

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

	err := q.db.QueryRow(query, engID).Scan(
		&ed.EngineID,
		&ed.EngineCode,
		&ed.ManufacturerID,
		&powerKwStart,
		&powerKwUpto,
		&powerPsStart,
		&powerPsUpto,
		&capacityStart,
		&capacityUpto,
		&dateFrom,
		&dateTo,
		&compressionStart,
		&compressionUpto,
		&torqueStart,
		&torqueUpto,
		&bore,
		&stroke,
		&cylinders,
		&mainBearings,
		&valves,
		&engineConstruction,
		&fuelType,
		&fuelMixture,
		&chargeType,
		&emissionNorm,
		&cylinderConstruction,
		&engineManagement,
		&valveManagement,
		&coolingType,
		&engineType,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Build specs map
	ed.Specs = make(map[string]interface{})

	if powerKwStart.Valid {
		ed.Specs["power_kw_from"] = powerKwStart.Float64
	}
	if powerKwUpto.Valid {
		ed.Specs["power_kw_to"] = powerKwUpto.Float64
	}
	if powerPsStart.Valid {
		ed.Specs["power_ps_from"] = powerPsStart.Float64
	}
	if powerPsUpto.Valid {
		ed.Specs["power_ps_to"] = powerPsUpto.Float64
	}
	if capacityStart.Valid {
		ed.Specs["capacity_from"] = capacityStart.Float64
	}
	if capacityUpto.Valid {
		ed.Specs["capacity_to"] = capacityUpto.Float64
	}
	if compressionStart.Valid {
		ed.Specs["compression_from"] = compressionStart.Float64
	}
	if compressionUpto.Valid {
		ed.Specs["compression_to"] = compressionUpto.Float64
	}
	if torqueStart.Valid {
		ed.Specs["torque_nm_from"] = torqueStart.Float64
	}
	if torqueUpto.Valid {
		ed.Specs["torque_nm_to"] = torqueUpto.Float64
	}
	if bore.Valid {
		ed.Specs["bore"] = bore.Float64
	}
	if stroke.Valid {
		ed.Specs["stroke"] = stroke.Float64
	}
	if cylinders.Valid {
		ed.Specs["cylinders"] = int(cylinders.Int64)
	}
	if mainBearings.Valid {
		ed.Specs["main_bearings"] = int(mainBearings.Int64)
	}
	if valves.Valid {
		ed.Specs["valves"] = int(valves.Int64)
	}
	if engineConstruction.Valid && engineConstruction.String != "" {
		ed.Specs["engine_construction"] = engineConstruction.String
	}
	if fuelType.Valid && fuelType.String != "" {
		ed.Specs["fuel_type"] = fuelType.String
	}
	if fuelMixture.Valid && fuelMixture.String != "" {
		ed.Specs["fuel_mixture"] = fuelMixture.String
	}
	if chargeType.Valid && chargeType.String != "" {
		ed.Specs["charge_type"] = chargeType.String
	}
	if emissionNorm.Valid && emissionNorm.String != "" {
		ed.Specs["emission_norm"] = emissionNorm.String
	}
	if cylinderConstruction.Valid && cylinderConstruction.String != "" {
		ed.Specs["cylinder_construction"] = cylinderConstruction.String
	}
	if engineManagement.Valid && engineManagement.String != "" {
		ed.Specs["engine_management"] = engineManagement.String
	}
	if valveManagement.Valid && valveManagement.String != "" {
		ed.Specs["valve_management"] = valveManagement.String
	}
	if coolingType.Valid && coolingType.String != "" {
		ed.Specs["cooling_type"] = coolingType.String
	}
	if engineType.Valid && engineType.String != "" {
		ed.Specs["engine_type"] = engineType.String
	}

	if dateFrom.Valid {
		ed.DateFrom = dateFrom.String
	}
	if dateTo.Valid {
		ed.DateTo = dateTo.String
	}

	return &ed, nil
}