package database

import (
	"testing"

	"go-tecdoc-api/internal/models"
)

func TestNormalizeSearchNumber(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"WK 842/2":       "WK8422",
		"WK 8422":        "WK8422",
		"WK8422":         "WK8422",
		"W-712.95":       "W71295",
		" 7700 115-294 ": "7700115294",
	}
	for input, want := range tests {
		input, want := input, want
		t.Run(input, func(t *testing.T) {
			t.Parallel()
			if got := normalizeSearchNumber(input); got != want {
				t.Fatalf("normalizeSearchNumber(%q) = %q, want %q", input, got, want)
			}
		})
	}
}

func TestPaginateSearchResults(t *testing.T) {
	t.Parallel()
	items := []models.SearchResult{{ArticleID: 1}, {ArticleID: 2}, {ArticleID: 3}}
	got := paginateSearchResults(items, 1, 1)
	if len(got) != 1 || got[0].ArticleID != 2 {
		t.Fatalf("paginateSearchResults() = %#v", got)
	}
	if got := paginateSearchResults(items, 10, 5); len(got) != 0 {
		t.Fatalf("out-of-range pagination = %#v", got)
	}
}

func TestUniquePositiveInts(t *testing.T) {
	t.Parallel()
	got := uniquePositiveInts([]int{0, 18, 18, -1, 19, 20, 19})
	want := []int{18, 19, 20}
	if len(got) != len(want) {
		t.Fatalf("uniquePositiveInts() = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("uniquePositiveInts() = %#v, want %#v", got, want)
		}
	}
}
