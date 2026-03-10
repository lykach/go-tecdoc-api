package database

import (
	"database/sql"
	"go-tecdoc-api/internal/models"
)

// GetModelSeries повертає список серій моделей для виробника
func (q *Queries) GetModelSeries(mfaID int, vehicleType string, languageID int, countryID int, limit, offset int) ([]models.ModelSeries, error) {
	query := `
		SELECT  
			MODELS_SERIES.MS_ID,
			MODELS_SERIES.MS_MFA_ID,
			MODELS_SERIES.MS_CI_FROM,
			MODELS_SERIES.MS_CI_TO,
			MODELS_SERIES.MS_TYPE,
			COALESCE(
				(SELECT DES_TEXTS.TEX_TEXT 
				 FROM TEXT_DESIGNATIONS 
				 INNER JOIN DES_TEXTS ON DES_TEXTS.TEX_ID = TEXT_DESIGNATIONS.DES_TEX_ID
				 WHERE TEXT_DESIGNATIONS.DES_ID = IFNULL(MS_COUNTRY_SPECIFICS.MSCS_NAME_DES, MODELS_SERIES.MS_NAME_DES) 
				 AND TEXT_DESIGNATIONS.DES_LNG_ID = ?
				 GROUP BY TEXT_DESIGNATIONS.DES_ID LIMIT 1),
				(SELECT DES_TEXTS.TEX_TEXT 
				 FROM TEXT_DESIGNATIONS 
				 INNER JOIN DES_TEXTS ON DES_TEXTS.TEX_ID = TEXT_DESIGNATIONS.DES_TEX_ID
				 WHERE TEXT_DESIGNATIONS.DES_ID = IFNULL(MS_COUNTRY_SPECIFICS.MSCS_NAME_DES, MODELS_SERIES.MS_NAME_DES) 
				 AND TEXT_DESIGNATIONS.DES_LNG_ID = 4
				 GROUP BY TEXT_DESIGNATIONS.DES_ID LIMIT 1),
				'Unknown'
			) AS MS_NAME
		FROM 
			MODELS_SERIES
			LEFT OUTER JOIN MS_COUNTRY_SPECIFICS
				ON MS_COUNTRY_SPECIFICS.MSCS_ID = MODELS_SERIES.MS_ID
				AND MS_COUNTRY_SPECIFICS.MSCS_COU_ID = ? 
		WHERE  
			MODELS_SERIES.MS_MFA_ID = ?
			AND FIND_IN_SET(?, MODELS_SERIES.MS_TYPE) > 0
		ORDER BY 
			MS_NAME
		LIMIT ? OFFSET ?
	`

	rows, err := q.db.Query(query, languageID, countryID, mfaID, vehicleType, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var modelSeries []models.ModelSeries
	for rows.Next() {
		var ms models.ModelSeries
		var yearFrom, yearTo sql.NullString

		err := rows.Scan(
			&ms.ID,
			&ms.MfaID,
			&yearFrom,
			&yearTo,
			&ms.Type,
			&ms.Name,
		)
		if err != nil {
			return nil, err
		}

		// Parse year from date string (YYYY-MM-DD)
		if yearFrom.Valid && len(yearFrom.String) >= 4 {
			if year, err := parseYear(yearFrom.String); err == nil { ms.YearFrom = year }
		}
		if yearTo.Valid && len(yearTo.String) >= 4 {
			if year, err := parseYear(yearTo.String); err == nil { ms.YearTo = year }
		}

		modelSeries = append(modelSeries, ms)
	}

	return modelSeries, rows.Err()
}

// GetModelSeriesByID повертає детальну інформацію про серію моделі
func (q *Queries) GetModelSeriesByID(msID int, languageID int, countryID int) (*models.ModelDetail, error) {
	query := `
		SELECT  
			MODELS_SERIES.MS_ID,
			MODELS_SERIES.MS_MFA_ID,
			MODELS_SERIES.MS_CI_FROM,
			MODELS_SERIES.MS_CI_TO,
			MODELS_SERIES.MS_TYPE,
			MANUFACTURERS.MFA_BRAND,
			COALESCE(
				(SELECT DES_TEXTS.TEX_TEXT 
				 FROM TEXT_DESIGNATIONS 
				 INNER JOIN DES_TEXTS ON DES_TEXTS.TEX_ID = TEXT_DESIGNATIONS.DES_TEX_ID
				 WHERE TEXT_DESIGNATIONS.DES_ID = IFNULL(MS_COUNTRY_SPECIFICS.MSCS_NAME_DES, MODELS_SERIES.MS_NAME_DES) 
				 AND TEXT_DESIGNATIONS.DES_LNG_ID = ?
				 GROUP BY TEXT_DESIGNATIONS.DES_ID LIMIT 1),
				(SELECT DES_TEXTS.TEX_TEXT 
				 FROM TEXT_DESIGNATIONS 
				 INNER JOIN DES_TEXTS ON DES_TEXTS.TEX_ID = TEXT_DESIGNATIONS.DES_TEX_ID
				 WHERE TEXT_DESIGNATIONS.DES_ID = IFNULL(MS_COUNTRY_SPECIFICS.MSCS_NAME_DES, MODELS_SERIES.MS_NAME_DES) 
				 AND TEXT_DESIGNATIONS.DES_LNG_ID = 4
				 GROUP BY TEXT_DESIGNATIONS.DES_ID LIMIT 1),
				'Unknown'
			) AS MS_NAME
		FROM 
			MODELS_SERIES
			INNER JOIN MANUFACTURERS ON MANUFACTURERS.MFA_ID = MODELS_SERIES.MS_MFA_ID
			LEFT OUTER JOIN MS_COUNTRY_SPECIFICS
				ON MS_COUNTRY_SPECIFICS.MSCS_ID = MODELS_SERIES.MS_ID
				AND MS_COUNTRY_SPECIFICS.MSCS_COU_ID = ? 
		WHERE  
			MODELS_SERIES.MS_ID = ?
	`

	var md models.ModelDetail
	var yearFrom, yearTo sql.NullString

	err := q.db.QueryRow(query, languageID, countryID, msID).Scan(
		&md.ID,
		&md.MfaID,
		&yearFrom,
		&yearTo,
		&md.Type,
		&md.Manufacturer,
		&md.Name,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Parse years
	if yearFrom.Valid && len(yearFrom.String) >= 4 {
		if year, err := parseYear(yearFrom.String); err == nil { md.YearFrom = year }
	}
	if yearTo.Valid && len(yearTo.String) >= 4 {
		if year, err := parseYear(yearTo.String); err == nil { md.YearTo = year }
	}

	return &md, nil
}

// CountModelSeries повертає загальну кількість серій моделей для виробника
func (q *Queries) CountModelSeries(mfaID int, vehicleType string) (int, error) {
	query := `
		SELECT 
			COUNT(*)
		FROM 
			MODELS_SERIES
		WHERE 
			MS_MFA_ID = ?
			AND FIND_IN_SET(?, MS_TYPE) > 0
	`

	var count int
	err := q.db.QueryRow(query, mfaID, vehicleType).Scan(&count)
	return count, err
}

