package database

import (
	"os"
	"strings"
	"testing"
)

func TestExtendedFactQueriesHonorTecDoc2024Q1CountryMarkers(t *testing.T) {
	enrichment, err := os.ReadFile("enrichment.go")
	if err != nil {
		t.Fatal(err)
	}
	articles, err := os.ReadFile("articles.go")
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := os.ReadFile("catalog.go")
	if err != nil {
		t.Fatal(err)
	}

	e := string(enrichment)
	for _, want := range []string{
		"C.CNTR_CTM_ID=ART_LOOKUP.ARL_CTM AND C.CNTR_COU_ID=?",
		"C.CNTR_CTM_ID=ARTICLES_PART_LIST.APL_CTM AND C.CNTR_COU_ID=?",
		"C.CNTR_CTM_ID=ARTICLE_INFO.AIN_CTM AND C.CNTR_COU_ID=?",
	} {
		if !strings.Contains(e, want) {
			t.Fatalf("batch country restriction contract missing %q", want)
		}
	}
	if !strings.Contains(string(articles), "C.CNTR_CTM_ID=ARTICLES_PART_LIST.APL_CTM AND C.CNTR_COU_ID=?") {
		t.Fatal("standalone parts-list query ignores APL_CTM")
	}
	if !strings.Contains(string(catalog), "C.CNTR_CTM_ID=ARTICLE_INFO.AIN_CTM AND C.CNTR_COU_ID=?") {
		t.Fatal("standalone ARTICLE_INFO query ignores AIN_CTM")
	}
}
