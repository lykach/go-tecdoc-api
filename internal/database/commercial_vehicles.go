package database

import (
	"database/sql"
	"go-tecdoc-api/internal/models"
	"strings"
)

// GetCommercialVehicles повертає список вантажівок для серії моделі
func (q *Queries) GetCommercialVehicles(msID int, languageID int, countryID int, limit, offset int) ([]models.CommercialVehicle, error) {
	query := `
		SELECT
			COMMERCIAL_VEHICLES.CV_ID,
			COMMERCIAL_VEHICLES.CV_MFA_ID,
			COMMERCIAL_VEHICLES.CV_MS_ID,
			CV_COUNTRY_SPECIFICS.CCS_CI_FROM,
			CV_COUNTRY_SPECIFICS.CCS_CI_TO,
			CV_COUNTRY_SPECIFICS.CCS_POWER_KW_START,
			CV_COUNTRY_SPECIFICS.CCS_POWER_PS_START,
			CV_COUNTRY_SPECIFICS.CCS_CAPACITY_TECH,
			CV_COUNTRY_SPECIFICS.CCS_TONNAGE,
			CONCAT_WS(' ', 
				MFA_BRAND,
				(` + BuildGetTextSubquery("IFNULL(MS_COUNTRY_SPECIFICS.MSCS_NAME_DES, MODELS_SERIES.MS_NAME_DES)", languageID) + `),
				(` + BuildGetTextSubquery("COMMERCIAL_VEHICLES.CV_MODEL_DES", languageID) + `)
			) AS TYPE_NAME,
			(` + BuildGetTextSubquery("CV_COUNTRY_SPECIFICS.CCS_ENGINE_TYPE", languageID) + `) AS ENGINE_TYPE,
			(` + BuildGetTextSubquery("CV_COUNTRY_SPECIFICS.CCS_PLATFORM_TYPE", languageID) + `) AS PLATFORM_TYPE,
			(SELECT GROUP_CONCAT(ENGINES.ENG_CODE)
			 FROM ENGINES
			 JOIN LINK_ENGINE_TYPE ON ENGINES.ENG_ID = LINK_ENGINE_TYPE.LET_ENG_ID
			 WHERE LINK_ENGINE_TYPE.LET_TYPE_ID = COMMERCIAL_VEHICLES.CV_ID 
			 AND LINK_ENGINE_TYPE.LET_TYPE = 'CV') AS ENG_CODES
		FROM 
			COMMERCIAL_VEHICLES
			INNER JOIN CV_COUNTRY_SPECIFICS ON CV_COUNTRY_SPECIFICS.CCS_CV_ID = COMMERCIAL_VEHICLES.CV_ID
				AND (CV_COUNTRY_SPECIFICS.CCS_COU_ID = ? OR CV_COUNTRY_SPECIFICS.CCS_COU_ID = 0)
			INNER JOIN MODELS_SERIES ON MODELS_SERIES.MS_ID = ?
			INNER JOIN MANUFACTURERS ON MANUFACTURERS.MFA_ID = COMMERCIAL_VEHICLES.CV_MFA_ID
			LEFT OUTER JOIN MS_COUNTRY_SPECIFICS
				ON MS_COUNTRY_SPECIFICS.MSCS_ID = MODELS_SERIES.MS_ID
				AND MS_COUNTRY_SPECIFICS.MSCS_COU_ID = ? 
		WHERE 
			COMMERCIAL_VEHICLES.CV_MS_ID = ?
			AND FIND_IN_SET('CV', MODELS_SERIES.MS_TYPE) > 0
		ORDER BY TYPE_NAME
		LIMIT ? OFFSET ?
	`

	rows, err := q.db.Query(query, countryID, msID, countryID, msID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vehicles []models.CommercialVehicle
	for rows.Next() {
		var cv models.CommercialVehicle
		var yearFrom, yearTo sql.NullString
		var powerKW, powerPS sql.NullFloat64
		var capacity, tonnage sql.NullFloat64
		var engineType, platformType sql.NullString
		var engineCodes sql.NullString

		err := rows.Scan(
			&cv.ID,
			&cv.MfaID,
			&cv.MsID,
			&yearFrom,
			&yearTo,
			&powerKW,
			&powerPS,
			&capacity,
			&tonnage,
			&cv.TypeName,
			&engineType,
			&platformType,
			&engineCodes,
		)
		if err != nil {
			return nil, err
		}

		// Parse years
		if yearFrom.Valid && len(yearFrom.String) >= 4 {
			if year, err := parseYear(yearFrom.String); err == nil {
				cv.YearFrom = year
			}
		}
		if yearTo.Valid && len(yearTo.String) >= 4 {
			if year, err := parseYear(yearTo.String); err == nil {
				cv.YearTo = year
			}
		}

		// Parse power
		if powerKW.Valid {
			cv.PowerKW = int(powerKW.Float64)
		}
		if powerPS.Valid {
			cv.PowerHP = int(powerPS.Float64)
		}

		// Parse capacity and tonnage
		if capacity.Valid {
			cv.Capacity = int(capacity.Float64)
		}
		if tonnage.Valid {
			cv.Tonnage = tonnage.Float64
		}

		cv.EngineType = nullString(engineType)
		cv.PlatformType = nullString(platformType)

		// Parse engine codes
		if engineCodes.Valid && engineCodes.String != "" {
			cv.EngineCodes = strings.Split(engineCodes.String, ",")
		}

		vehicles = append(vehicles, cv)
	}

	return vehicles, rows.Err()
}

// CountCommercialVehicles повертає кількість вантажівок для серії моделі
func (q *Queries) CountCommercialVehicles(msID int) (int, error) {
	query := `
		SELECT 
			COUNT(*)
		FROM 
			COMMERCIAL_VEHICLES
			INNER JOIN MODELS_SERIES ON MODELS_SERIES.MS_ID = ?
		WHERE 
			COMMERCIAL_VEHICLES.CV_MS_ID = ?
			AND FIND_IN_SET('CV', MODELS_SERIES.MS_TYPE) > 0
	`

	var count int
	err := q.db.QueryRow(query, msID, msID).Scan(&count)
	return count, err
}
// GetCommercialVehicleByID повертає детальну інформацію про вантажівку
func (q *Queries) GetCommercialVehicleByID(cvID int, languageID int, countryID int) (*models.VehicleDetail, error) {
query := `
SELECT
COMMERCIAL_VEHICLES.CV_ID,
CONCAT_WS(' ', 
MANUFACTURERS.MFA_BRAND,
(` + BuildGetTextSubquery("IFNULL(MS_COUNTRY_SPECIFICS.MSCS_NAME_DES, MODELS_SERIES.MS_NAME_DES)", languageID) + `),
(` + BuildGetTextSubquery("COMMERCIAL_VEHICLES.CV_MODEL_DES", languageID) + `)
) AS TYPE_NAME,
MANUFACTURERS.MFA_BRAND AS MANUFACTURER,
(` + BuildGetTextSubquery("IFNULL(MS_COUNTRY_SPECIFICS.MSCS_NAME_DES, MODELS_SERIES.MS_NAME_DES)", languageID) + `) AS MODEL_SERIES,
CV_COUNTRY_SPECIFICS.CCS_CI_FROM,
CV_COUNTRY_SPECIFICS.CCS_CI_TO,
CV_COUNTRY_SPECIFICS.CCS_POWER_KW_START,
CV_COUNTRY_SPECIFICS.CCS_POWER_PS_START,
CV_COUNTRY_SPECIFICS.CCS_CAPACITY_TECH,
CV_COUNTRY_SPECIFICS.CCS_TONNAGE,
(` + BuildGetTextSubquery("CV_COUNTRY_SPECIFICS.CCS_ENGINE_TYPE", languageID) + `) AS ENGINE_TYPE,
(` + BuildGetTextSubquery("CV_COUNTRY_SPECIFICS.CCS_PLATFORM_TYPE", languageID) + `) AS PLATFORM_TYPE,
(` + BuildGetTextSubquery("CV_COUNTRY_SPECIFICS.CCS_AXLE_CONFIGURATION", languageID) + `) AS AXLE_CONFIGURATION,
(SELECT GROUP_CONCAT(ENGINES.ENG_CODE)
 FROM ENGINES
 JOIN LINK_ENGINE_TYPE ON ENGINES.ENG_ID = LINK_ENGINE_TYPE.LET_ENG_ID
 WHERE LINK_ENGINE_TYPE.LET_TYPE_ID = COMMERCIAL_VEHICLES.CV_ID 
 AND LINK_ENGINE_TYPE.LET_TYPE = 'CV') AS ENG_CODES
FROM 
COMMERCIAL_VEHICLES
INNER JOIN CV_COUNTRY_SPECIFICS ON CV_COUNTRY_SPECIFICS.CCS_CV_ID = COMMERCIAL_VEHICLES.CV_ID
AND (CV_COUNTRY_SPECIFICS.CCS_COU_ID = ? OR CV_COUNTRY_SPECIFICS.CCS_COU_ID = 0)
INNER JOIN MODELS_SERIES ON MODELS_SERIES.MS_ID = COMMERCIAL_VEHICLES.CV_MS_ID
INNER JOIN MANUFACTURERS ON MANUFACTURERS.MFA_ID = COMMERCIAL_VEHICLES.CV_MFA_ID
LEFT OUTER JOIN MS_COUNTRY_SPECIFICS
ON MS_COUNTRY_SPECIFICS.MSCS_ID = MODELS_SERIES.MS_ID
AND MS_COUNTRY_SPECIFICS.MSCS_COU_ID = ? 
WHERE 
COMMERCIAL_VEHICLES.CV_ID = ?
`

var vd models.VehicleDetail
var yearFrom, yearTo sql.NullString
var powerKW, powerPS sql.NullFloat64
var capacity, tonnage sql.NullFloat64
var engineType, platformType, axleConfig sql.NullString
var engineCodes sql.NullString

err := q.db.QueryRow(query, countryID, countryID, cvID).Scan(
&vd.ID,
&vd.TypeName,
&vd.Manufacturer,
&vd.ModelSeries,
&yearFrom,
&yearTo,
&powerKW,
&powerPS,
&capacity,
&tonnage,
&engineType,
&platformType,
&axleConfig,
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
vd.Specs["power_kw"] = int(powerKW.Float64)
}
if powerPS.Valid {
vd.Specs["power_hp"] = int(powerPS.Float64)
}
if capacity.Valid {
vd.Specs["capacity_cc"] = int(capacity.Float64)
}
if tonnage.Valid {
vd.Specs["tonnage"] = tonnage.Float64
}
if engineType.Valid && engineType.String != "" {
vd.Specs["engine_type"] = engineType.String
}
if platformType.Valid && platformType.String != "" {
vd.Specs["platform_type"] = platformType.String
}
if axleConfig.Valid && axleConfig.String != "" {
vd.Specs["axle_configuration"] = axleConfig.String
}

// Parse engine codes
if engineCodes.Valid && engineCodes.String != "" {
vd.EngineCodes = strings.Split(engineCodes.String, ",")
}

return &vd, nil
}
