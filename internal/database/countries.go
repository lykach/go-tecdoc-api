package database

import (
	"database/sql"
	"fmt"
	"go-tecdoc-api/internal/models"
)

// GetCountries повертає список країн з перекладами
func (q *Queries) GetCountries(langID int, includeGroups bool) ([]models.Country, error) {
	whereClause := ""
	if !includeGroups {
		whereClause = "WHERE COU_IS_GROUP = 0"
	}

	query := fmt.Sprintf(`
		SELECT 
			COU_ID,
			COU_CODE,
			(%s) AS COU_NAME,
			COU_IS_GROUP,
			COU_CURRENCY_CODE,
			COU_FLAG_IMGNAME,
			COU_ISOCODE2,
			COU_ISOCODE3,
			COU_ISOCODENO
		FROM 
			COUNTRIES
		%s
		ORDER BY 
			COU_NAME
	`, BuildGetTextSubquery("COU_DES", langID), whereClause)

	rows, err := q.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var countries []models.Country
	for rows.Next() {
		var c models.Country
		var name, code, flagImage, iso2, iso3 sql.NullString
		var currencyCode sql.NullString
		var isGroup int
		var isoNumber sql.NullInt64

		err := rows.Scan(
			&c.ID,
			&code,
			&name,
			&isGroup,
			&currencyCode,
			&flagImage,
			&iso2,
			&iso3,
			&isoNumber,
		)
		if err != nil {
			return nil, err
		}

		c.Code = nullString(code)
		c.Name = nullString(name)
		c.IsGroup = isGroup == 1
		if currencyCode.Valid {
			c.CurrencyCode = &currencyCode.String
		}
		if flagImage.Valid {
			c.FlagImage = &flagImage.String
		}
		if iso2.Valid {
			c.ISOCode2 = &iso2.String
		}
		if iso3.Valid {
			c.ISOCode3 = &iso3.String
		}
		if isoNumber.Valid {
			num := int(isoNumber.Int64)
			c.ISOCodeNumber = &num
		}

		countries = append(countries, c)
	}

	return countries, rows.Err()
}

// GetCountryByID повертає деталі країни за ID
func (q *Queries) GetCountryByID(id, langID int) (*models.Country, error) {
	query := fmt.Sprintf(`
		SELECT 
			COU_ID,
			COU_CODE,
			(%s) AS COU_NAME,
			COU_IS_GROUP,
			COU_CURRENCY_CODE,
			(%s) AS COU_CURRENCY_NAME,
			COU_FLAG_IMGNAME,
			COU_ISOCODE2,
			COU_ISOCODE3,
			COU_ISOCODENO
		FROM 
			COUNTRIES
		WHERE 
			COU_ID = ?
	`, BuildGetTextSubquery("COU_DES", langID), BuildGetTextSubquery("COU_CURRENCY_CODE_DES", langID))

	var c models.Country
	var name, code, flagImage, iso2, iso3 sql.NullString
	var currencyCode, currencyName sql.NullString
	var isGroup int
	var isoNumber sql.NullInt64

	err := q.db.QueryRow(query, id).Scan(
		&c.ID,
		&code,
		&name,
		&isGroup,
		&currencyCode,
		&currencyName,
		&flagImage,
		&iso2,
		&iso3,
		&isoNumber,
	)
	if err != nil {
		return nil, err
	}

	c.Code = nullString(code)
	c.Name = nullString(name)
	c.IsGroup = isGroup == 1
	if currencyCode.Valid {
		c.CurrencyCode = &currencyCode.String
	}
	if currencyName.Valid {
		c.CurrencyName = &currencyName.String
	}
	if flagImage.Valid {
		c.FlagImage = &flagImage.String
	}
	if iso2.Valid {
		c.ISOCode2 = &iso2.String
	}
	if iso3.Valid {
		c.ISOCode3 = &iso3.String
	}
	if isoNumber.Valid {
		num := int(isoNumber.Int64)
		c.ISOCodeNumber = &num
	}

	return &c, nil
}