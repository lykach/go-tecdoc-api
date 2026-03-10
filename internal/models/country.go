package models

// Country представляє країну в системі TecDoc
type Country struct {
	ID            int     `json:"id" db:"COU_ID"`
	Code          string  `json:"code" db:"COU_CODE"`
	Name          string  `json:"name" db:"COU_NAME"`
	IsGroup       bool    `json:"is_group" db:"COU_IS_GROUP"`
	CurrencyCode  *string `json:"currency_code,omitempty" db:"COU_CURRENCY_CODE"`
	CurrencyName  *string `json:"currency_name,omitempty" db:"COU_CURRENCY_NAME"`
	FlagImage     *string `json:"flag_image,omitempty" db:"COU_FLAG_IMGNAME"`
	ISOCode2      *string `json:"iso_code_2,omitempty" db:"COU_ISOCODE2"`
	ISOCode3      *string `json:"iso_code_3,omitempty" db:"COU_ISOCODE3"`
	ISOCodeNumber *int    `json:"iso_code_number,omitempty" db:"COU_ISOCODENO"`
}

// CountriesResponse - відповідь зі списком країн
type CountriesResponse struct {
	Countries []Country `json:"countries"`
}