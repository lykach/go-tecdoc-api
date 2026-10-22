package models

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCrossReferenceJSONExposesUnambiguousRoles(t *testing.T) {
	t.Parallel()
	supplierID := 30
	manufacturerID := 5
	payload := CrossReferencesResponse{Crosses: []CrossReference{
		{
			ArticleNr:  "F 026 407 023",
			Brand:      "BOSCH",
			SupplierID: &supplierID,
			Supplier:   "BOSCH",
			Type:       "IAM",
		},
		{
			ArticleNr:      "03L 115 562",
			Brand:          "AUDI",
			ManufacturerID: &manufacturerID,
			Manufacturer:   "AUDI",
			Type:           "OEM",
		},
	}}

	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	text := string(raw)
	for _, expected := range []string{
		`"supplier_id":30`,
		`"supplier":"BOSCH"`,
		`"manufacturer_id":5`,
		`"manufacturer":"AUDI"`,
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("payload %s does not contain %s", text, expected)
		}
	}
}

func TestOEMNumberKeepsLegacyAndExplicitManufacturerFields(t *testing.T) {
	t.Parallel()
	item := OEMNumber{
		Number:         "03L 115 562",
		Brand:          "AUDI",
		BrandID:        5,
		Manufacturer:   "AUDI",
		ManufacturerID: 5,
	}
	raw, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	text := string(raw)
	for _, expected := range []string{
		`"brand":"AUDI"`,
		`"brand_id":5`,
		`"manufacturer":"AUDI"`,
		`"manufacturer_id":5`,
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("payload %s does not contain %s", text, expected)
		}
	}
}

func TestSearchResultSeparatesLookupPathFromReturnedEntity(t *testing.T) {
	t.Parallel()
	item := SearchResult{
		ArticleID:    6926850,
		ArticleNr:    "F 026 407 023",
		Brand:        "BOSCH",
		SupplierID:   30,
		FoundVia:     "OENumber",
		RelationType: "ArticleNumber",
	}
	raw, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	text := string(raw)
	if !strings.Contains(text, `"found_via":"OENumber"`) || !strings.Contains(text, `"relation_type":"ArticleNumber"`) {
		t.Fatalf("payload = %s", text)
	}
}
