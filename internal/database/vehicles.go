package database

import (
	"database/sql"
	"encoding/json"
	"go-tecdoc-api/internal/models"
)

// GetPassengerCars повертає список легкових автомобілів для серії моделі
func (q *Queries) GetPassengerCars(msID int, languageID int, countryID int, limit, offset int) ([]models.PassengerCar, error) {
	query := `
		SELECT
			PASSENGER_CARS.PC_ID,
			PASSENGER_CARS.PC_MFA_ID,
			PASSENGER_CARS.PC_MS_ID,
			PC_COUNTRY_SPECIFICS.PCS_CI_FROM,
			PC_COUNTRY_SPECIFICS.PCS_CI_TO,
			PC_COUNTRY_SPECIFICS.PCS_POWER_KW,
			PC_COUNTRY_SPECIFICS.PCS_POWER_PS,
			PC_COUNTRY_SPECIFICS.PCS_CAPACITY_TECH,
			CONCAT_WS(' ',
				MFA_BRAND,
				COALESCE(
					(SELECT DES_TEXTS.TEX_TEXT
					 FROM TEXT_DESIGNATIONS
					 INNER JOIN DES_TEXTS ON DES_TEXTS.TEX_ID = TEXT_DESIGNATIONS.DES_TEX_ID
					 WHERE TEXT_DESIGNATIONS.DES_ID = IFNULL(MS_COUNTRY_SPECIFICS.MSCS_NAME_DES, MODELS_SERIES.MS_NAME_DES)
					   AND TEXT_DESIGNATIONS.DES_LNG_ID = ?
					 GROUP BY TEXT_DESIGNATIONS.DES_ID
					 LIMIT 1),
					(SELECT DES_TEXTS.TEX_TEXT
					 FROM TEXT_DESIGNATIONS
					 INNER JOIN DES_TEXTS ON DES_TEXTS.TEX_ID = TEXT_DESIGNATIONS.DES_TEX_ID
					 WHERE TEXT_DESIGNATIONS.DES_ID = IFNULL(MS_COUNTRY_SPECIFICS.MSCS_NAME_DES, MODELS_SERIES.MS_NAME_DES)
					   AND TEXT_DESIGNATIONS.DES_LNG_ID = 4
					 GROUP BY TEXT_DESIGNATIONS.DES_ID
					 LIMIT 1),
					'Unknown'
				),
				COALESCE(
					(SELECT DES_TEXTS.TEX_TEXT
					 FROM TEXT_DESIGNATIONS
					 INNER JOIN DES_TEXTS ON DES_TEXTS.TEX_ID = TEXT_DESIGNATIONS.DES_TEX_ID
					 WHERE TEXT_DESIGNATIONS.DES_ID = PASSENGER_CARS.PC_MODEL_DES
					   AND TEXT_DESIGNATIONS.DES_LNG_ID = ?
					 GROUP BY TEXT_DESIGNATIONS.DES_ID
					 LIMIT 1),
					(SELECT DES_TEXTS.TEX_TEXT
					 FROM TEXT_DESIGNATIONS
					 INNER JOIN DES_TEXTS ON DES_TEXTS.TEX_ID = TEXT_DESIGNATIONS.DES_TEX_ID
					 WHERE TEXT_DESIGNATIONS.DES_ID = PASSENGER_CARS.PC_MODEL_DES
					   AND TEXT_DESIGNATIONS.DES_LNG_ID = 4
					 GROUP BY TEXT_DESIGNATIONS.DES_ID
					 LIMIT 1),
					'Unknown'
				)
			) AS TYPE_NAME,
			COALESCE(
				(SELECT DES_TEXTS.TEX_TEXT
				 FROM TEXT_DESIGNATIONS
				 INNER JOIN DES_TEXTS ON DES_TEXTS.TEX_ID = TEXT_DESIGNATIONS.DES_TEX_ID
				 WHERE TEXT_DESIGNATIONS.DES_ID = PC_COUNTRY_SPECIFICS.PCS_BODY_TYPE
				   AND TEXT_DESIGNATIONS.DES_LNG_ID = ?
				 GROUP BY TEXT_DESIGNATIONS.DES_ID
				 LIMIT 1),
				(SELECT DES_TEXTS.TEX_TEXT
				 FROM TEXT_DESIGNATIONS
				 INNER JOIN DES_TEXTS ON DES_TEXTS.TEX_ID = TEXT_DESIGNATIONS.DES_TEX_ID
				 WHERE TEXT_DESIGNATIONS.DES_ID = PC_COUNTRY_SPECIFICS.PCS_BODY_TYPE
				   AND TEXT_DESIGNATIONS.DES_LNG_ID = 4
				 GROUP BY TEXT_DESIGNATIONS.DES_ID
				 LIMIT 1),
				''
			) AS BODY_TYPE,
			COALESCE(
				(SELECT DES_TEXTS.TEX_TEXT
				 FROM TEXT_DESIGNATIONS
				 INNER JOIN DES_TEXTS ON DES_TEXTS.TEX_ID = TEXT_DESIGNATIONS.DES_TEX_ID
				 WHERE TEXT_DESIGNATIONS.DES_ID = PC_COUNTRY_SPECIFICS.PCS_ENGINE_TYPE
				   AND TEXT_DESIGNATIONS.DES_LNG_ID = ?
				 GROUP BY TEXT_DESIGNATIONS.DES_ID
				 LIMIT 1),
				(SELECT DES_TEXTS.TEX_TEXT
				 FROM TEXT_DESIGNATIONS
				 INNER JOIN DES_TEXTS ON DES_TEXTS.TEX_ID = TEXT_DESIGNATIONS.DES_TEX_ID
				 WHERE TEXT_DESIGNATIONS.DES_ID = PC_COUNTRY_SPECIFICS.PCS_ENGINE_TYPE
				   AND TEXT_DESIGNATIONS.DES_LNG_ID = 4
				 GROUP BY TEXT_DESIGNATIONS.DES_ID
				 LIMIT 1),
				''
			) AS ENGINE_TYPE,
			COALESCE(
				(SELECT DES_TEXTS.TEX_TEXT
				 FROM TEXT_DESIGNATIONS
				 INNER JOIN DES_TEXTS ON DES_TEXTS.TEX_ID = TEXT_DESIGNATIONS.DES_TEX_ID
				 WHERE TEXT_DESIGNATIONS.DES_ID = PC_COUNTRY_SPECIFICS.PCS_FUEL_TYPE
				   AND TEXT_DESIGNATIONS.DES_LNG_ID = ?
				 GROUP BY TEXT_DESIGNATIONS.DES_ID
				 LIMIT 1),
				(SELECT DES_TEXTS.TEX_TEXT
				 FROM TEXT_DESIGNATIONS
				 INNER JOIN DES_TEXTS ON DES_TEXTS.TEX_ID = TEXT_DESIGNATIONS.DES_TEX_ID
				 WHERE TEXT_DESIGNATIONS.DES_ID = PC_COUNTRY_SPECIFICS.PCS_FUEL_TYPE
				   AND TEXT_DESIGNATIONS.DES_LNG_ID = 4
				 GROUP BY TEXT_DESIGNATIONS.DES_ID
				 LIMIT 1),
				''
			) AS FUEL_TYPE,
			(
				SELECT GROUP_CONCAT(DISTINCT ENGINES.ENG_CODE ORDER BY ENGINES.ENG_CODE SEPARATOR ',')
				FROM ENGINES
				INNER JOIN LINK_ENGINE_TYPE ON ENGINES.ENG_ID = LINK_ENGINE_TYPE.LET_ENG_ID
				WHERE LINK_ENGINE_TYPE.LET_TYPE_ID = PASSENGER_CARS.PC_ID
				  AND LINK_ENGINE_TYPE.LET_TYPE = 'PC'
			) AS ENG_CODES,
			COALESCE(
				CONCAT(
					'[',
					(
						SELECT GROUP_CONCAT(
							DISTINCT JSON_OBJECT(
								'id', ENGINES.ENG_ID,
								'code', ENGINES.ENG_CODE
							)
							ORDER BY ENGINES.ENG_CODE
							SEPARATOR ','
						)
						FROM ENGINES
						INNER JOIN LINK_ENGINE_TYPE ON ENGINES.ENG_ID = LINK_ENGINE_TYPE.LET_ENG_ID
						WHERE LINK_ENGINE_TYPE.LET_TYPE_ID = PASSENGER_CARS.PC_ID
						  AND LINK_ENGINE_TYPE.LET_TYPE = 'PC'
					),
					']'
				),
				'[]'
			) AS ENGINES_JSON
		FROM
			PASSENGER_CARS
			INNER JOIN PC_COUNTRY_SPECIFICS
				ON PC_COUNTRY_SPECIFICS.PCS_PC_ID = PASSENGER_CARS.PC_ID
				AND (PC_COUNTRY_SPECIFICS.PCS_COU_ID = ? OR PC_COUNTRY_SPECIFICS.PCS_COU_ID = 0)
			INNER JOIN MODELS_SERIES
				ON MODELS_SERIES.MS_ID = ?
			INNER JOIN MANUFACTURERS
				ON MANUFACTURERS.MFA_ID = PASSENGER_CARS.PC_MFA_ID
			LEFT OUTER JOIN MS_COUNTRY_SPECIFICS
				ON MS_COUNTRY_SPECIFICS.MSCS_ID = MODELS_SERIES.MS_ID
				AND MS_COUNTRY_SPECIFICS.MSCS_COU_ID = ?
		WHERE
			PASSENGER_CARS.PC_MS_ID = ?
			AND FIND_IN_SET('PC', MODELS_SERIES.MS_TYPE) > 0
		ORDER BY TYPE_NAME
		LIMIT ? OFFSET ?
	`

	rows, err := q.db.Query(
		query,
		languageID, languageID, languageID, languageID, languageID,
		countryID, msID, countryID, msID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cars []models.PassengerCar

	for rows.Next() {
		var pc models.PassengerCar
		var yearFrom, yearTo sql.NullString
		var powerKW, powerPS sql.NullFloat64
		var capacity sql.NullFloat64
		var engineCodes sql.NullString
		var enginesJSON sql.NullString

		err := rows.Scan(
			&pc.ID,
			&pc.MfaID,
			&pc.MsID,
			&yearFrom,
			&yearTo,
			&powerKW,
			&powerPS,
			&capacity,
			&pc.TypeName,
			&pc.BodyType,
			&pc.EngineType,
			&pc.FuelType,
			&engineCodes,
			&enginesJSON,
		)
		if err != nil {
			return nil, err
		}

		if yearFrom.Valid && len(yearFrom.String) >= 4 {
			if year, err := parseYear(yearFrom.String); err == nil {
				pc.YearFrom = year
			}
		}
		if yearTo.Valid && len(yearTo.String) >= 4 {
			if year, err := parseYear(yearTo.String); err == nil {
				pc.YearTo = year
			}
		}

		if powerKW.Valid {
			pc.PowerKW = int(powerKW.Float64)
		}
		if powerPS.Valid {
			pc.PowerHP = int(powerPS.Float64)
		}
		if capacity.Valid {
			pc.Capacity = int(capacity.Float64)
		}

		if engineCodes.Valid && engineCodes.String != "" {
			pc.EngineCodes = splitCommaSeparated(engineCodes.String)
		} else {
			pc.EngineCodes = []string{}
		}

		if enginesJSON.Valid && enginesJSON.String != "" {
			pc.Engines = parseEnginesJSON(enginesJSON.String)
		} else {
			pc.Engines = []models.Engine{}
		}

		cars = append(cars, pc)
	}

	return cars, rows.Err()
}

// CountPassengerCars повертає загальну кількість автомобілів для серії моделі
func (q *Queries) CountPassengerCars(msID int) (int, error) {
	query := `
		SELECT
			COUNT(*)
		FROM
			PASSENGER_CARS
			INNER JOIN MODELS_SERIES ON MODELS_SERIES.MS_ID = ?
		WHERE
			PASSENGER_CARS.PC_MS_ID = ?
			AND FIND_IN_SET('PC', MODELS_SERIES.MS_TYPE) > 0
	`

	var count int
	err := q.db.QueryRow(query, msID, msID).Scan(&count)
	return count, err
}

// GetPassengerCarByID повертає детальну інформацію про автомобіль
func (q *Queries) GetPassengerCarByID(pcID int, languageID int, countryID int) (*models.VehicleDetail, error) {
	query := `
		SELECT
			PASSENGER_CARS.PC_ID,
			CONCAT_WS(' ',
				MFA_BRAND,
				COALESCE(
					(SELECT DES_TEXTS.TEX_TEXT
					 FROM TEXT_DESIGNATIONS
					 INNER JOIN DES_TEXTS ON DES_TEXTS.TEX_ID = TEXT_DESIGNATIONS.DES_TEX_ID
					 WHERE TEXT_DESIGNATIONS.DES_ID = IFNULL(MS_COUNTRY_SPECIFICS.MSCS_NAME_DES, MODELS_SERIES.MS_NAME_DES)
					   AND TEXT_DESIGNATIONS.DES_LNG_ID = ?
					 GROUP BY TEXT_DESIGNATIONS.DES_ID
					 LIMIT 1),
					'Unknown'
				),
				COALESCE(
					(SELECT DES_TEXTS.TEX_TEXT
					 FROM TEXT_DESIGNATIONS
					 INNER JOIN DES_TEXTS ON DES_TEXTS.TEX_ID = TEXT_DESIGNATIONS.DES_TEX_ID
					 WHERE TEXT_DESIGNATIONS.DES_ID = PASSENGER_CARS.PC_MODEL_DES
					   AND TEXT_DESIGNATIONS.DES_LNG_ID = ?
					 GROUP BY TEXT_DESIGNATIONS.DES_ID
					 LIMIT 1),
					'Unknown'
				)
			) AS TYPE_NAME,
			MFA_BRAND AS MANUFACTURER,
			COALESCE(
				(SELECT DES_TEXTS.TEX_TEXT
				 FROM TEXT_DESIGNATIONS
				 INNER JOIN DES_TEXTS ON DES_TEXTS.TEX_ID = TEXT_DESIGNATIONS.DES_TEX_ID
				 WHERE TEXT_DESIGNATIONS.DES_ID = IFNULL(MS_COUNTRY_SPECIFICS.MSCS_NAME_DES, MODELS_SERIES.MS_NAME_DES)
				   AND TEXT_DESIGNATIONS.DES_LNG_ID = ?
				 GROUP BY TEXT_DESIGNATIONS.DES_ID
				 LIMIT 1),
				'Unknown'
			) AS MODEL_SERIES,
			PC_COUNTRY_SPECIFICS.PCS_CI_FROM,
			PC_COUNTRY_SPECIFICS.PCS_CI_TO,
			PC_COUNTRY_SPECIFICS.PCS_POWER_KW,
			PC_COUNTRY_SPECIFICS.PCS_POWER_PS,
			PC_COUNTRY_SPECIFICS.PCS_CAPACITY_TECH,
			PC_COUNTRY_SPECIFICS.PCS_CAPACITY_LT,
			PC_COUNTRY_SPECIFICS.PCS_NUMBER_OF_CYLINDERS,
			PC_COUNTRY_SPECIFICS.PCS_NUMBER_OF_VALVES,
			COALESCE(
				(SELECT DES_TEXTS.TEX_TEXT
				 FROM TEXT_DESIGNATIONS
				 INNER JOIN DES_TEXTS ON DES_TEXTS.TEX_ID = TEXT_DESIGNATIONS.DES_TEX_ID
				 WHERE TEXT_DESIGNATIONS.DES_ID = PC_COUNTRY_SPECIFICS.PCS_BODY_TYPE
				   AND TEXT_DESIGNATIONS.DES_LNG_ID = ?
				 GROUP BY TEXT_DESIGNATIONS.DES_ID
				 LIMIT 1),
				''
			) AS BODY_TYPE,
			COALESCE(
				(SELECT DES_TEXTS.TEX_TEXT
				 FROM TEXT_DESIGNATIONS
				 INNER JOIN DES_TEXTS ON DES_TEXTS.TEX_ID = TEXT_DESIGNATIONS.DES_TEX_ID
				 WHERE TEXT_DESIGNATIONS.DES_ID = PC_COUNTRY_SPECIFICS.PCS_ENGINE_TYPE
				   AND TEXT_DESIGNATIONS.DES_LNG_ID = ?
				 GROUP BY TEXT_DESIGNATIONS.DES_ID
				 LIMIT 1),
				''
			) AS ENGINE_TYPE,
			COALESCE(
				(SELECT DES_TEXTS.TEX_TEXT
				 FROM TEXT_DESIGNATIONS
				 INNER JOIN DES_TEXTS ON DES_TEXTS.TEX_ID = TEXT_DESIGNATIONS.DES_TEX_ID
				 WHERE TEXT_DESIGNATIONS.DES_ID = PC_COUNTRY_SPECIFICS.PCS_FUEL_TYPE
				   AND TEXT_DESIGNATIONS.DES_LNG_ID = ?
				 GROUP BY TEXT_DESIGNATIONS.DES_ID
				 LIMIT 1),
				''
			) AS FUEL_TYPE,
			COALESCE(
				(SELECT DES_TEXTS.TEX_TEXT
				 FROM TEXT_DESIGNATIONS
				 INNER JOIN DES_TEXTS ON DES_TEXTS.TEX_ID = TEXT_DESIGNATIONS.DES_TEX_ID
				 WHERE TEXT_DESIGNATIONS.DES_ID = PC_COUNTRY_SPECIFICS.PCS_GEAR_TYPE
				   AND TEXT_DESIGNATIONS.DES_LNG_ID = ?
				 GROUP BY TEXT_DESIGNATIONS.DES_ID
				 LIMIT 1),
				''
			) AS GEAR_TYPE,
			COALESCE(
				(SELECT DES_TEXTS.TEX_TEXT
				 FROM TEXT_DESIGNATIONS
				 INNER JOIN DES_TEXTS ON DES_TEXTS.TEX_ID = TEXT_DESIGNATIONS.DES_TEX_ID
				 WHERE TEXT_DESIGNATIONS.DES_ID = PC_COUNTRY_SPECIFICS.PCS_DRIVE_TYPE
				   AND TEXT_DESIGNATIONS.DES_LNG_ID = ?
				 GROUP BY TEXT_DESIGNATIONS.DES_ID
				 LIMIT 1),
				''
			) AS DRIVE_TYPE,
			(
				SELECT GROUP_CONCAT(DISTINCT ENGINES.ENG_CODE ORDER BY ENGINES.ENG_CODE SEPARATOR ',')
				FROM ENGINES
				INNER JOIN LINK_ENGINE_TYPE ON ENGINES.ENG_ID = LINK_ENGINE_TYPE.LET_ENG_ID
				WHERE LINK_ENGINE_TYPE.LET_TYPE_ID = PASSENGER_CARS.PC_ID
				  AND LINK_ENGINE_TYPE.LET_TYPE = 'PC'
			) AS ENG_CODES,
			COALESCE(
				CONCAT(
					'[',
					(
						SELECT GROUP_CONCAT(
							DISTINCT JSON_OBJECT(
								'id', ENGINES.ENG_ID,
								'code', ENGINES.ENG_CODE
							)
							ORDER BY ENGINES.ENG_CODE
							SEPARATOR ','
						)
						FROM ENGINES
						INNER JOIN LINK_ENGINE_TYPE ON ENGINES.ENG_ID = LINK_ENGINE_TYPE.LET_ENG_ID
						WHERE LINK_ENGINE_TYPE.LET_TYPE_ID = PASSENGER_CARS.PC_ID
						  AND LINK_ENGINE_TYPE.LET_TYPE = 'PC'
					),
					']'
				),
				'[]'
			) AS ENGINES_JSON
		FROM
			PASSENGER_CARS
			INNER JOIN PC_COUNTRY_SPECIFICS
				ON PC_COUNTRY_SPECIFICS.PCS_PC_ID = PASSENGER_CARS.PC_ID
				AND (PC_COUNTRY_SPECIFICS.PCS_COU_ID = ? OR PC_COUNTRY_SPECIFICS.PCS_COU_ID = 0)
			INNER JOIN MODELS_SERIES
				ON MODELS_SERIES.MS_ID = PASSENGER_CARS.PC_MS_ID
			INNER JOIN MANUFACTURERS
				ON MANUFACTURERS.MFA_ID = PASSENGER_CARS.PC_MFA_ID
			LEFT OUTER JOIN MS_COUNTRY_SPECIFICS
				ON MS_COUNTRY_SPECIFICS.MSCS_ID = MODELS_SERIES.MS_ID
				AND MS_COUNTRY_SPECIFICS.MSCS_COU_ID = ?
		WHERE
			PASSENGER_CARS.PC_ID = ?
	`

	var vd models.VehicleDetail
	var yearFrom, yearTo sql.NullString
	var powerKW, powerPS sql.NullFloat64
	var capacityTech, capacityLt sql.NullFloat64
	var cylinders, valves sql.NullInt64
	var bodyType, engineType, fuelType, gearType, driveType sql.NullString
	var engineCodes sql.NullString
	var enginesJSON sql.NullString

	err := q.db.QueryRow(
		query,
		languageID, languageID, languageID,
		languageID, languageID, languageID, languageID, languageID,
		countryID, countryID, pcID,
	).Scan(
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
		&bodyType,
		&engineType,
		&fuelType,
		&gearType,
		&driveType,
		&engineCodes,
		&enginesJSON,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

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

	vd.Specs = make(map[string]interface{})

	if powerKW.Valid {
		vd.Specs["power_kw"] = int(powerKW.Float64)
	}
	if powerPS.Valid {
		vd.Specs["power_hp"] = int(powerPS.Float64)
	}
	if capacityTech.Valid {
		vd.Specs["capacity_cc"] = int(capacityTech.Float64)
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
	if bodyType.Valid && bodyType.String != "" {
		vd.Specs["body_type"] = bodyType.String
	}
	if engineType.Valid && engineType.String != "" {
		vd.Specs["engine_type"] = engineType.String
	}
	if fuelType.Valid && fuelType.String != "" {
		vd.Specs["fuel_type"] = fuelType.String
	}
	if gearType.Valid && gearType.String != "" {
		vd.Specs["gear_type"] = gearType.String
	}
	if driveType.Valid && driveType.String != "" {
		vd.Specs["drive_type"] = driveType.String
	}

	if engineCodes.Valid && engineCodes.String != "" {
		vd.EngineCodes = splitCommaSeparated(engineCodes.String)
	} else {
		vd.EngineCodes = []string{}
	}

	if enginesJSON.Valid && enginesJSON.String != "" {
		vd.Specs["engines"] = parseEnginesJSON(enginesJSON.String)
	} else {
		vd.Specs["engines"] = []models.Engine{}
	}

	return &vd, nil
}

func splitCommaSeparated(value string) []string {
	if value == "" {
		return []string{}
	}

	parts := make([]string, 0)
	current := make([]rune, 0)

	for _, r := range value {
		if r == ',' {
			item := trimSpaces(string(current))
			if item != "" {
				parts = append(parts, item)
			}
			current = current[:0]
			continue
		}
		current = append(current, r)
	}

	item := trimSpaces(string(current))
	if item != "" {
		parts = append(parts, item)
	}

	return parts
}

func trimSpaces(s string) string {
	start := 0
	end := len(s)

	for start < end && (s[start] == ' ' || s[start] == '\n' || s[start] == '\t' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\n' || s[end-1] == '\t' || s[end-1] == '\r') {
		end--
	}

	return s[start:end]
}

func parseEnginesJSON(value string) []models.Engine {
	if value == "" {
		return []models.Engine{}
	}

	var engines []models.Engine
	if err := json.Unmarshal([]byte(value), &engines); err != nil {
		return []models.Engine{}
	}

	return engines
}