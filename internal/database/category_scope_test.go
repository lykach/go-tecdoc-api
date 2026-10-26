package database

import (
	"strings"
	"testing"
)

func TestProductGroupScopePreservesCountryAndBoundCriteria(t *testing.T) {
	query, args := productGroupArticleScope(100480, 214, "PC", 258, map[int]string{92: "9", 6: "x' OR 1=1 --"})
	for _, field := range []string{"ARTICLES.ART_CTM", "LINK_ART.LA_CTM", "LINK_LA_TYP.LAT_CTM", "ARTICLE_CRITERIA.ACR_CTM"} {
		if !strings.Contains(query, field+" = 0 OR EXISTS") || !strings.Contains(query, "CNTR_CTM_ID = "+field) {
			t.Fatalf("country scope missing for %s", field)
		}
	}
	if strings.Contains(query, " OR 1=1") || strings.Contains(query, " & ") {
		t.Fatal("unsafe criterion or bit-mask CTM")
	}
	if len(args) != 16 || args[10] != 6 || args[11] != "x' OR 1=1 --" || args[13] != 92 || args[14] != "9" {
		t.Fatalf("non-deterministic or missing bound filters: %#v", args)
	}
	if strings.Count(query, "?") != len(args) {
		t.Fatal("placeholder/argument mismatch")
	}
	plain, plainArgs := productGroupArticleScope(100480, 214, "PC", 258, nil)
	if strings.Count(plain, "?") != len(plainArgs) || len(plainArgs) != 10 {
		t.Fatal("unfiltered country arguments missing")
	}
}
