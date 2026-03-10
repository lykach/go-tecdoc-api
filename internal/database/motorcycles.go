package database

import (
	"database/sql"
	"go-tecdoc-api/internal/models"
	"strings"
)

// GetMotorcycles повертає список мотоциклів для серії моделі
func (q *Queries) GetMotorcycles(msID int, languageID int, countryID int, limit, offset int) ([]models.Motorcycle, error) {
	query := `
		SELECT
			MOTORBIKES.MTB_ID,
			MOTORBIKES.MTB_MFA_ID,
			MOTORBIKES.MTB_MS_ID,
			MTB_COUNTRY_SPECIFICS.MCS_CI_FROM,
			MTB_COUNTRY_SPECIFICS.MCS_CI_TO,
			MTB_COUNTRY_SPECIFICS.MCS_POWER_KW,
			MTB_COUNTRY_SPECIFICS.MCS_POWER_PS,
			MTB_COUNTRY_SPECIFICS.MCS_CAPACITY_TECH,
			CONCAT_WS(' ', 
				MANUFACTURERS.MFA_BRAND,
				(` + BuildGetTextSubquery("IFNULL(MS_COUNTRY_SPECIFICS.MSCS_NAME_DES, MODELS_SERIES.MS_NAME_DES)", languageID) + `),
				(` + BuildGetTextSubquery("COALESCE(MTB_COUNTRY_SPECIFICS.MCS_MODEL_DES, MOTORBIKES.MTB_MODEL_DES)", languageID) + `)
			) AS TYPE_NAME,
			(` + BuildGetTextSubquery("MTB_COUNTRY_SPECIFICS.MCS_ENGINE_TYPE", languageID) + `) AS ENGINE_TYPE,
			(` + BuildGetTextSubquery("MTB_COUNTRY_SPECIFICS.MCS_FUEL_TYPE", languageID) + `) AS FUEL_TYPE,
			MOTORBIKES.MTB_TYPE,
			(SELECT GROUP_CONCAT(ENGINES.ENG_CODE)
			 FROM ENGINES
			 JOIN LINK_ENGINE_TYPE ON ENGINES.ENG_ID = LINK_ENGINE_TYPE.LET_ENG_ID
			 WHERE LINK_ENGINE_TYPE.LET_TYPE_ID = MOTORBIKES.MTB_ID 
			 AND LINK_ENGINE_TYPE.LET_TYPE = 'MC') AS ENG_CODES
		FROM 
			MOTORBIKES
			INNER JOIN MTB_COUNTRY_SPECIFICS ON MTB_COUNTRY_SPECIFICS.MCS_MTB_ID = MOTORBIKES.MTB_ID
				AND (MTB_COUNTRY_SPECIFICS.MCS_COU_ID = ? OR MTB_COUNTRY_SPECIFICS.MCS_COU_ID = 0)
			INNER JOIN MODELS_SERIES ON MODELS_SERIES.MS_ID = ?
			INNER JOIN MANUFACTURERS ON MANUFACTURERS.MFA_ID = MOTORBIKES.MTB_MFA_ID
			LEFT OUTER JOIN MS_COUNTRY_SPECIFICS
				ON MS_COUNTRY_SPECIFICS.MSCS_ID = MODELS_SERIES.MS_ID
				AND MS_COUNTRY_SPECIFICS.MSCS_COU_ID = ?
		WHERE 
			MOTORBIKES.MTB_MS_ID = ?
			AND FIND_IN_SET('Motorcycle', MODELS_SERIES.MS_TYPE) > 0
		ORDER BY TYPE_NAME
		LIMIT ? OFFSET ?
	`

	rows, err := q.db.Query(query, countryID, msID, countryID, msID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var motorcycles []models.Motorcycle
	for rows.Next() {
		var mc models.Motorcycle
		var yearFrom, yearTo sql.NullString
		var powerKW, powerPS sql.NullInt64
		var capacity sql.NullInt64
		var engineType, fuelType sql.NullString
		var engineCodes sql.NullString

		err := rows.Scan(
			&mc.ID,
			&mc.MfaID,
			&mc.MsID,
			&yearFrom,
			&yearTo,
			&powerKW,
			&powerPS,
			&capacity,
			&mc.TypeName,
			&engineType,
			&fuelType,
			&mc.Type,
			&engineCodes,
		)
		if err != nil {
			return nil, err
		}

		// Parse years
		if yearFrom.Valid && len(yearFrom.String) >= 4 {
			if year, err := parseYear(yearFrom.String); err == nil {
				mc.YearFrom = year
			}
		}
		if yearTo.Valid && len(yearTo.String) >= 4 {
			if year, err := parseYear(yearTo.String); err == nil {
				mc.YearTo = year
			}
		}

		// Parse power
		if powerKW.Valid {
			mc.PowerKW = int(powerKW.Int64)
		}
		if powerPS.Valid {
			mc.PowerHP = int(powerPS.Int64)
		}

		// Parse capacity
		if capacity.Valid {
			mc.Capacity = int(capacity.Int64)
		}

		mc.EngineType = nullString(engineType)
		mc.FuelType = nullString(fuelType)

		// Parse engine codes
		if engineCodes.Valid && engineCodes.String != "" {
			mc.EngineCodes = strings.Split(engineCodes.String, ",")
		}

		motorcycles = append(motorcycles, mc)
	}

	return motorcycles, rows.Err()
}

// CountMotorcycles повертає кількість мотоциклів для серії моделі
func (q *Queries) CountMotorcycles(msID int) (int, error) {
	query := `
		SELECT 
			COUNT(*)
		FROM 
			MOTORBIKES
			INNER JOIN MODELS_SERIES ON MODELS_SERIES.MS_ID = ?
		WHERE 
			MOTORBIKES.MTB_MS_ID = ?
			AND FIND_IN_SET('Motorcycle', MODELS_SERIES.MS_TYPE) > 0
	`

	var count int
	err := q.db.QueryRow(query, msID, msID).Scan(&count)
	return count, err
}
// GetMotorcycleByID повертає детальну інформацію про мотоцикл
func (q *Queries) GetMotorcycleByID(mtbID int, languageID int, countryID int) (*models.VehicleDetail, error) {
query := `
SELECT
MOTORBIKES.MTB_ID,
CONCAT_WS(' ', 
MANUFACTURERS.MFA_BRAND,
(` + BuildGetTextSubquery("IFNULL(MS_COUNTRY_SPECIFICS.MSCS_NAME_DES, MODELS_SERIES.MS_NAME_DES)", languageID) + `),
(` + BuildGetTextSubquery("COALESCE(MTB_COUNTRY_SPECIFICS.MCS_MODEL_DES, MOTORBIKES.MTB_MODEL_DES)", languageID) + `)
) AS TYPE_NAME,
MANUFACTURERS.MFA_BRAND AS MANUFACTURER,
(` + BuildGetTextSubquery("IFNULL(MS_COUNTRY_SPECIFICS.MSCS_NAME_DES, MODELS_SERIES.MS_NAME_DES)", languageID) + `) AS MODEL_SERIES,
MTB_COUNTRY_SPECIFICS.MCS_CI_FROM,
MTB_COUNTRY_SPECIFICS.MCS_CI_TO,
MTB_COUNTRY_SPECIFICS.MCS_POWER_KW,
MTB_COUNTRY_SPECIFICS.MCS_POWER_PS,
MTB_COUNTRY_SPECIFICS.MCS_CAPACITY_TECH,
MTB_COUNTRY_SPECIFICS.MCS_CAPACITY_LT,
MTB_COUNTRY_SPECIFICS.MCS_NUMBER_OF_CYLINDERS,
MTB_COUNTRY_SPECIFICS.MCS_NUMBER_OF_VALVES,
MTB_COUNTRY_SPECIFICS.MCS_VOLTAGE,
MTB_COUNTRY_SPECIFICS.MCS_FUEL_TANK,
(` + BuildGetTextSubquery("MTB_COUNTRY_SPECIFICS.MCS_ENGINE_TYPE", languageID) + `) AS ENGINE_TYPE,
(` + BuildGetTextSubquery("MTB_COUNTRY_SPECIFICS.MCS_FUEL_TYPE", languageID) + `) AS FUEL_TYPE,
(` + BuildGetTextSubquery("MTB_COUNTRY_SPECIFICS.MCS_FUEL_MIXTURE", languageID) + `) AS FUEL_MIXTURE,
(` + BuildGetTextSubquery("MTB_COUNTRY_SPECIFICS.MCS_DRIVE_TYPE", languageID) + `) AS DRIVE_TYPE,
MOTORBIKES.MTB_TYPE,
(SELECT GROUP_CONCAT(ENGINES.ENG_CODE)
 FROM ENGINES
 JOIN LINK_ENGINE_TYPE ON ENGINES.ENG_ID = LINK_ENGINE_TYPE.LET_ENG_ID
 WHERE LINK_ENGINE_TYPE.LET_TYPE_ID = MOTORBIKES.MTB_ID 
 AND LINK_ENGINE_TYPE.LET_TYPE = 'MC') AS ENG_CODES
FROM 
MOTORBIKES
INNER JOIN MTB_COUNTRY_SPECIFICS ON MTB_COUNTRY_SPECIFICS.MCS_MTB_ID = MOTORBIKES.MTB_ID
AND (MTB_COUNTRY_SPECIFICS.MCS_COU_ID = ? OR MTB_COUNTRY_SPECIFICS.MCS_COU_ID = 0)
INNER JOIN MODELS_SERIES ON MODELS_SERIES.MS_ID = MOTORBIKES.MTB_MS_ID
INNER JOIN MANUFACTURERS ON MANUFACTURERS.MFA_ID = MOTORBIKES.MTB_MFA_ID
LEFT OUTER JOIN MS_COUNTRY_SPECIFICS
ON MS_COUNTRY_SPECIFICS.MSCS_ID = MODELS_SERIES.MS_ID
AND MS_COUNTRY_SPECIFICS.MSCS_COU_ID = ? 
WHERE 
MOTORBIKES.MTB_ID = ?
`

var vd models.VehicleDetail
var yearFrom, yearTo sql.NullString
var powerKW, powerPS sql.NullInt64
var capacityTech sql.NullInt64
var capacityLt sql.NullFloat64
var cylinders, valves sql.NullInt64
var voltage, fuelTank sql.NullFloat64
var engineType, fuelType, fuelMixture, driveType, mtbType sql.NullString
var engineCodes sql.NullString

err := q.db.QueryRow(query, countryID, countryID, mtbID).Scan(
&vd.ID,
&vd.TypeName,
&vd.Manufacturer,
&vd.ModelSeries,
&yearFrom,
&yearTo,
&powerKW,
&powerPS,
&capacityTech,
&capacityLt,
&cylinders,
&valves,
&voltage,
&fuelTank,
&engineType,
&fuelType,
&fuelMixture,
&driveType,
&mtbType,
&engineCodes,
)

if err != nil {
if err == sql.ErrNoRows {
return nil, nil
}
return nil, err
}

// Parse years
if yearFrom.Valid && len(yearFrom.String) >= 4 {
if year, err := parseYear(yearFrom.String); err == nil {
vd.YearFrom = year
}
}
if yearTo.Valid && len(yearTo.String) >= 4 {
if year, err := parseYear(yearTo.String); err == nil {
vd.YearTo = year
}
}

// Build specs map
vd.Specs = make(map[string]interface{})

if powerKW.Valid {
vd.Specs["power_kw"] = int(powerKW.Int64)
}
if powerPS.Valid {
vd.Specs["power_hp"] = int(powerPS.Int64)
}
if capacityTech.Valid {
vd.Specs["capacity_cc"] = int(capacityTech.Int64)
}
if capacityLt.Valid {
vd.Specs["capacity_liters"] = capacityLt.Float64
}
if cylinders.Valid {
vd.Specs["cylinders"] = int(cylinders.Int64)
}
if valves.Valid {
vd.Specs["valves"] = int(valves.Int64)
}
if voltage.Valid {
vd.Specs["voltage"] = voltage.Float64
}
if fuelTank.Valid {
vd.Specs["fuel_tank_liters"] = fuelTank.Float64
}
if engineType.Valid && engineType.String != "" {
vd.Specs["engine_type"] = engineType.String
}
if fuelType.Valid && fuelType.String != "" {
vd.Specs["fuel_type"] = fuelType.String
}
if fuelMixture.Valid && fuelMixture.String != "" {
vd.Specs["fuel_mixture"] = fuelMixture.String
}
if driveType.Valid && driveType.String != "" {
vd.Specs["drive_type"] = driveType.String
}
if mtbType.Valid && mtbType.String != "" {
vd.Specs["motorcycle_type"] = mtbType.String
}

// Parse engine codes
if engineCodes.Valid && engineCodes.String != "" {
vd.EngineCodes = strings.Split(engineCodes.String, ",")
}

return &vd, nil
}
