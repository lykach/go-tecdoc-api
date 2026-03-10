package models

// Supplier представляє постачальника/бренд запчастин
type Supplier struct {
	ID       int     `json:"id" db:"SUP_ID"`
	Brand    string  `json:"brand" db:"SUP_BRAND"`
	FullName *string `json:"full_name,omitempty" db:"SUP_FULL_NAME"`
	LogoPNG  *string `json:"logo_png,omitempty" db:"SUP_LOGO_NAME"`
	LogoWEBP *string `json:"logo_webp,omitempty" db:"WEBP_LOGO"`
}

// SuppliersResponse - відповідь зі списком постачальників
type SuppliersResponse struct {
	Suppliers []Supplier `json:"suppliers"`
}