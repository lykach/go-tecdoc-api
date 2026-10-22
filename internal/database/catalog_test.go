package database

import (
	"reflect"
	"strings"
	"testing"

	"go-tecdoc-api/internal/models"
)

func TestNormalizeVehicleType(t *testing.T) {
	cases := map[string]string{
		"PC":         "PC",
		"car":        "PC",
		"CV":         "CV",
		"commercial": "CV",
		"mc":         "Motorcycle",
		"motorbike":  "Motorcycle",
	}
	for input, want := range cases {
		got, ok := normalizeVehicleType(input)
		if !ok || got != want {
			t.Fatalf("normalizeVehicleType(%q) = %q, %v; want %q, true", input, got, ok, want)
		}
	}
	if _, ok := normalizeVehicleType("Engine"); ok {
		t.Fatal("Engine must not be accepted as TYPE_NUMBER_LOOKUP vehicle type")
	}
}

func TestNormalizeCatalogType(t *testing.T) {
	for _, input := range []string{"PC", "CV", "Motorcycle", "Engine", "Axle", "Universal Parts"} {
		if _, ok := normalizeCatalogType(input); !ok {
			t.Fatalf("catalog type %q should be accepted", input)
		}
	}
	if _, ok := normalizeCatalogType("unknown"); ok {
		t.Fatal("unknown catalog type must be rejected")
	}
}

func TestInternalDatasetNamesAreNotExposed(t *testing.T) {
	for _, name := range []string{"api_keys", "API_KEY_USAGE_LOGS", "pma__userconfig"} {
		if !isInternalDatasetName(name) {
			t.Fatalf("%q must be internal", name)
		}
	}
	for _, name := range []string{"ARTICLES", "TYPE_NUMBER_LOOKUP", "KEY_VALUES"} {
		if isInternalDatasetName(name) {
			t.Fatalf("%q must be public TecDoc dataset", name)
		}
	}
}

func TestCreateAPIKeyFromPlainValidatesBeforeDB(t *testing.T) {
	q := &Queries{}
	_, err := q.CreateAPIKeyFromPlain(CreateAPIKeyParams{Name: "test"}, "td_live_not-hex")
	if err == nil {
		t.Fatal("malformed key must be rejected before database access")
	}
}

func TestDatasetOrderColumnsUsesPrimaryKeyTieBreakers(t *testing.T) {
	columns := []models.DatasetColumn{
		{Name: "A", Key: "PRI"},
		{Name: "B", Key: "PRI"},
		{Name: "C"},
	}
	got := datasetOrderColumns(columns, "C")
	want := []string{"C", "A", "B"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("datasetOrderColumns() = %v, want %v", got, want)
	}
}

func TestDatasetOrderColumnsFallsBackToAllColumnsWithoutPrimaryKey(t *testing.T) {
	columns := []models.DatasetColumn{
		{Name: "A"},
		{Name: "B"},
		{Name: "C"},
	}
	got := datasetOrderColumns(columns, "B")
	want := []string{"B", "A", "C"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("datasetOrderColumns() = %v, want %v", got, want)
	}
}

func TestVehicleCriterionSelectColumnsExportsDerivedAliases(t *testing.T) {
	projection := vehicleCriterionSelectColumns(4)
	for _, alias := range []string{"CRI_ID", "CRI_NAME", "CRI_SHORT", "CRI_UNIT", "CRI_TYPE", "SUGGESTED"} {
		if !strings.Contains(projection, " AS "+alias) {
			t.Fatalf("vehicle criterion projection must export alias %s; SQL: %s", alias, projection)
		}
	}
}
