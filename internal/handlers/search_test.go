package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSearchAnalogsRejectsNumberOnlyLookup(t *testing.T) {
	t.Parallel()
	h := New(nil)
	req := httptest.NewRequest(http.MethodGet, "/search/analog?search_number=WK8422", nil)
	res := httptest.NewRecorder()

	h.SearchAnalogs(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusBadRequest)
	}
	if !strings.Contains(res.Body.String(), "requires supplier_id") {
		t.Fatalf("body = %s", res.Body.String())
	}
}

func TestSearchAnalogsRejectsMixedSourceContracts(t *testing.T) {
	t.Parallel()
	h := New(nil)
	req := httptest.NewRequest(http.MethodGet, "/search/analog?art_id=101&search_number=WK8422&supplier_id=9", nil)
	res := httptest.NewRecorder()

	h.SearchAnalogs(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusBadRequest)
	}
	if !strings.Contains(res.Body.String(), "not both") {
		t.Fatalf("body = %s", res.Body.String())
	}
}

func TestSearchAnalogsRejectsInvalidArticleIDBeforeDatabaseAccess(t *testing.T) {
	t.Parallel()
	h := New(nil)
	req := httptest.NewRequest(http.MethodGet, "/search/analog?art_id=invalid", nil)
	res := httptest.NewRecorder()

	h.SearchAnalogs(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusBadRequest)
	}
	if !strings.Contains(res.Body.String(), "Invalid art_id") {
		t.Fatalf("body = %s", res.Body.String())
	}
}

func TestClampSearchLimit(t *testing.T) {
	t.Parallel()
	if got := clampSearchLimit(0); got != defaultSearchLimit {
		t.Fatalf("clampSearchLimit(0) = %d", got)
	}
	if got := clampSearchLimit(maxSearchLimit + 1); got != maxSearchLimit {
		t.Fatalf("clampSearchLimit(max+1) = %d", got)
	}
	if got := clampSearchLimit(25); got != 25 {
		t.Fatalf("clampSearchLimit(25) = %d", got)
	}
}
