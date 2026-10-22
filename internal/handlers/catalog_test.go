package handlers

import (
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestParseDatasetFilters(t *testing.T) {
	r := httptest.NewRequest("GET", "/api/v1/datasets/ART_LOOKUP?filter=ARL_TYPE:OENumber,IAMNumber&filter=ARL_BRA_ID:30", nil)
	got, err := parseDatasetFilters(r)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string][]string{
		"ARL_TYPE":   {"OENumber", "IAMNumber"},
		"ARL_BRA_ID": {"30"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("filters = %#v; want %#v", got, want)
	}
}

func TestParseDatasetFiltersRejectsMalformedFilter(t *testing.T) {
	r := httptest.NewRequest("GET", "/api/v1/datasets/ARTICLES?filter=broken", nil)
	if _, err := parseDatasetFilters(r); err == nil {
		t.Fatal("malformed filter must fail")
	}
}
