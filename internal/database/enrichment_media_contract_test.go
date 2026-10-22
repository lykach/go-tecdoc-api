package database

import (
	"os"
	"strings"
	"testing"
)

func TestArticleMediaBatchUsesCountryRestrictionAnd2024Q1Dimensions(t *testing.T) {
	raw, err := os.ReadFile("enrichment.go")
	if err != nil {
		t.Fatal(err)
	}
	source := strings.ReplaceAll(string(raw), "\t", " ")
	for _, want := range []string{
		"GetArticleMediaBatch(ctx context.Context, articleIDs []int, languageID, countryID int)",
		"COALESCE(ART_MEDIA_INFO.ART_MEDIA_CTM,0)",
		"C.CNTR_CTM_ID=ART_MEDIA_INFO.ART_MEDIA_CTM AND C.CNTR_COU_ID=?",
		"ART_MEDIA_INFO.ART_MEDIA_CONTENT_TYPE",
		"args = append(args, countryID)",
	} {
		if !strings.Contains(source, want) {
			t.Fatalf("article-media 2024Q1/country contract missing %q", want)
		}
	}
}

func TestStandaloneArticleMediaUsesCountryRestriction(t *testing.T) {
	raw, err := os.ReadFile("articles.go")
	if err != nil {
		t.Fatal(err)
	}
	source := strings.ReplaceAll(string(raw), "\t", " ")
	for _, want := range []string{
		"GetArticleMedia(articleID, langID, countryID int)",
		"C.CNTR_CTM_ID=ART_MEDIA_INFO.ART_MEDIA_CTM AND C.CNTR_COU_ID=?",
		"q.db.Query(query, articleID, countryID)",
	} {
		if !strings.Contains(source, want) {
			t.Fatalf("standalone article-media country contract missing %q", want)
		}
	}
}
