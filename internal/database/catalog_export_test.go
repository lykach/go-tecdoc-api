package database

import (
	"strings"
	"testing"
)

func TestCrossReferenceExportPagesARTLookupBeforeJoiningArticles(t *testing.T) {
	upper := strings.ToUpper(exportCrossReferencesSQL)
	fromDerived := strings.Index(upper, "FROM (\n")
	limit := strings.Index(upper, "LIMIT ? OFFSET ?")
	joinArticles := strings.Index(upper, "INNER JOIN ARTICLES")
	if fromDerived < 0 || limit < 0 || joinArticles < 0 {
		t.Fatalf("export query is missing derived paging shape: %s", exportCrossReferencesSQL)
	}
	if !(fromDerived < limit && limit < joinArticles) {
		t.Fatalf("ART_LOOKUP must be paged before ARTICLES join: %s", exportCrossReferencesSQL)
	}
	if !strings.Contains(upper, "ART_LOOKUP FORCE INDEX (PRIMARY)") {
		t.Fatalf("cross-reference export must keep the live-verified primary-key scan: %s", exportCrossReferencesSQL)
	}
}

func TestEANExportUsesTypeIndexBeforeJoiningArticles(t *testing.T) {
	upper := strings.ToUpper(exportEANSQL)
	fromDerived := strings.Index(upper, "FROM (\n")
	limit := strings.Index(upper, "LIMIT ? OFFSET ?")
	joinArticles := strings.Index(upper, "INNER JOIN ARTICLES")
	if fromDerived < 0 || limit < 0 || joinArticles < 0 {
		t.Fatalf("EAN export query is missing derived paging shape: %s", exportEANSQL)
	}
	if !(fromDerived < limit && limit < joinArticles) {
		t.Fatalf("EAN ART_LOOKUP rows must be paged before ARTICLES join: %s", exportEANSQL)
	}
	if !strings.Contains(upper, "ART_LOOKUP FORCE INDEX (ARL_TYPE_INDX)") {
		t.Fatalf("EAN export must seek by ARL_TYPE_INDX instead of scanning the 216M-row primary key: %s", exportEANSQL)
	}
	if !strings.Contains(upper, "WHERE ARL_TYPE = 'EAN'") {
		t.Fatalf("EAN export must constrain ARL_TYPE before pagination: %s", exportEANSQL)
	}
	if !strings.Contains(upper, "ORDER BY ARL_TYPE, ARL_ART_ID, ARL_DISPLAY_NR, ARL_BRA_ID") {
		t.Fatalf("EAN export order must match the ARL_TYPE secondary index plus its InnoDB primary-key suffix: %s", exportEANSQL)
	}
	if strings.Contains(upper, "ART_LOOKUP FORCE INDEX (PRIMARY)") {
		t.Fatalf("EAN export must not force the primary index: %s", exportEANSQL)
	}
}
