package models

// Language представляє мову в системі TecDoc
type Language struct {
	ID          int    `json:"id" db:"LNG_ID"`
	Description string `json:"description" db:"LNG_DESCRIPTION"`
	ISO2        string `json:"iso2" db:"LNG_ISO2"`
	Codepage    string `json:"codepage" db:"LNG_CODEPAGE"`
}

// LanguagesResponse - відповідь зі списком мов
type LanguagesResponse struct {
	Languages []Language `json:"languages"`
}