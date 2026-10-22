package handlers

import "testing"

func TestParseCriteriaFiltersStrict(t *testing.T) {
	got, err := parseCriteriaFilters("6:12,92:9")
	if err != nil {
		t.Fatal(err)
	}
	if got[6] != "12" || got[92] != "9" || len(got) != 2 {
		t.Fatalf("unexpected filters: %#v", got)
	}

	for _, raw := range []string{"6", "x:12", "0:12", "6:", ":12", "6:12,broken"} {
		if _, err := parseCriteriaFilters(raw); err == nil {
			t.Fatalf("%q must fail", raw)
		}
	}
}
