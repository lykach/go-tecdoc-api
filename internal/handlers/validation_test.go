package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

func TestParsePageParamsStrict(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		wantL   int
		wantO   int
		wantErr bool
	}{
		{name: "defaults", query: "", wantL: 100, wantO: 0},
		{name: "valid", query: "limit=25&offset=50", wantL: 25, wantO: 50},
		{name: "zero limit", query: "limit=0", wantErr: true},
		{name: "too large", query: "limit=501", wantErr: true},
		{name: "bad limit", query: "limit=nope", wantErr: true},
		{name: "negative offset", query: "offset=-1", wantErr: true},
		{name: "bad offset", query: "offset=nope", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/?"+tc.query, nil)
			got, err := parsePageParams(r, 100, 500)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %+v", got)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got.Limit != tc.wantL || got.Offset != tc.wantO {
				t.Fatalf("got limit=%d offset=%d", got.Limit, got.Offset)
			}
		})
	}
}

func TestParseBoolQueryStrict(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/?include_hidden=definitely", nil)
	if _, err := parseBoolQuery(r, "include_hidden", false); err == nil {
		t.Fatal("expected invalid boolean to fail")
	}
}

func TestParseResponseFormatStrict(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/?format=xml", nil)
	if _, err := parseResponseFormat(r, "json", "csv"); err == nil {
		t.Fatal("expected unsupported format to fail")
	}
}

func TestCommonQueryValidationMiddleware(t *testing.T) {
	h := New(nil)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
	wrapped := h.ValidateCommonQueryMiddleware(next)

	for _, rawURL := range []string{"/?language_id=abc", "/?language_id=0", "/?country_id=-1"} {
		rr := httptest.NewRecorder()
		wrapped.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, rawURL, nil))
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("%s: expected 400, got %d", rawURL, rr.Code)
		}
		if !strings.Contains(rr.Body.String(), "invalid query parameter") {
			t.Fatalf("%s: unexpected body %s", rawURL, rr.Body.String())
		}
	}
}

func TestGetPathIntRequiresPositiveID(t *testing.T) {
	for _, raw := range []string{"0", "-1", "abc"} {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r = mux.SetURLVars(r, map[string]string{"id": raw})
		if _, err := getPathInt(r, "id"); err == nil {
			t.Fatalf("id=%q must fail", raw)
		}
	}

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = mux.SetURLVars(r, map[string]string{"id": "42"})
	if got, err := getPathInt(r, "id"); err != nil || got != 42 {
		t.Fatalf("positive id: got %d, err=%v", got, err)
	}
}

func TestParseImportVehicleIDs(t *testing.T) {
	ids, err := parseImportVehicleIDs("10,20,10,30")
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 3 || ids[0] != 10 || ids[1] != 20 || ids[2] != 30 {
		t.Fatalf("unexpected ids: %#v", ids)
	}
	for _, raw := range []string{"", "0", "-1", "1,nope"} {
		if _, err := parseImportVehicleIDs(raw); err == nil {
			t.Fatalf("vehicle_ids=%q must fail", raw)
		}
	}
}

func TestNormalizeImportProductGroupVehicleType(t *testing.T) {
	tests := map[string]string{
		"PC":         "PC",
		"LCV":        "PC",
		"E-PC":       "PC",
		"E-LCV":      "PC",
		"CV":         "CV",
		"Motorcycle": "Motorcycle",
	}
	for input, want := range tests {
		got, err := normalizeImportProductGroupVehicleType(input)
		if err != nil || got != want {
			t.Fatalf("%s: got %q err=%v; want %q", input, got, err, want)
		}
	}
	for _, value := range []string{"", "Engine", "Axle", "BOAT"} {
		if _, err := normalizeImportProductGroupVehicleType(value); err == nil {
			t.Fatalf("%s must fail", value)
		}
	}
}

func TestNormalizeProductGroupTreeType(t *testing.T) {
	tests := map[string]string{
		"PC":              "PC",
		"LCV":             "PC",
		"E-PC":            "PC",
		"E-LCV":           "PC",
		"CV":              "CV",
		"motorcycle":      "Motorcycle",
		"Engine":          "Engine",
		"Axle":            "Axle",
		"Universal Parts": "Universal Parts",
	}
	for input, want := range tests {
		got, err := normalizeProductGroupTreeType(input)
		if err != nil || got != want {
			t.Fatalf("%s: got %q err=%v; want %q", input, got, err, want)
		}
	}
	if _, err := normalizeProductGroupTreeType("BOAT"); err == nil {
		t.Fatal("BOAT must fail")
	}
}
