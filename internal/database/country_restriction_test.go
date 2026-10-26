package database

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
)

type countryRestrictionDriver struct{}
type countryRestrictionConn struct{}
type countryRestrictionRows struct {
	columns []string
	rows    [][]driver.Value
	index   int
}

var registerCountryRestrictionDriver sync.Once

func (countryRestrictionDriver) Open(string) (driver.Conn, error) {
	return &countryRestrictionConn{}, nil
}
func (*countryRestrictionConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("not implemented")
}
func (*countryRestrictionConn) Close() error              { return nil }
func (*countryRestrictionConn) Begin() (driver.Tx, error) { return nil, errors.New("not implemented") }
func (*countryRestrictionConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	if strings.Contains(query, "INNER JOIN COUNTRY_RESTRICTIONS") {
		return nil, errors.New("article country filtering must not drop unrestricted CTM=0 rows")
	}
	if strings.Contains(query, "ARTICLES.ART_CTM") && !strings.Contains(query, "ARTICLES.ART_CTM = 0 OR EXISTS") {
		return nil, errors.New("article country filtering must include the unrestricted CTM=0 branch")
	}
	if strings.Contains(query, "SELECT EXISTS") {
		return &countryRestrictionRows{
			columns: []string{"exists"},
			rows:    [][]driver.Value{{true}},
		}, nil
	}
	return &countryRestrictionRows{columns: []string{"unused"}}, nil
}

func (r *countryRestrictionRows) Columns() []string { return r.columns }
func (r *countryRestrictionRows) Close() error      { return nil }
func (r *countryRestrictionRows) Next(dest []driver.Value) error {
	if r.index >= len(r.rows) {
		return io.EOF
	}
	copy(dest, r.rows[r.index])
	r.index++
	return nil
}

func openCountryRestrictionTestDB(t *testing.T) *sql.DB {
	t.Helper()
	registerCountryRestrictionDriver.Do(func() {
		sql.Register("tecdoc-country-restriction-test", countryRestrictionDriver{})
	})
	db, err := sql.Open("tecdoc-country-restriction-test", "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestArticleCountryQueriesPreserveUnrestrictedCTM(t *testing.T) {
	ctx := context.Background()
	queries := &Queries{db: openCountryRestrictionTestDB(t)}
	if _, err := queries.GetProductGroupArticles(100480, 214, "PC", 48, 258, 10, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := queries.GetProductGroupArticlesWithCriteria(100480, 214, "PC", 48, 258, map[int]string{6: "12"}, 10, 0); err != nil {
		t.Fatal(err)
	}

	if _, err := queries.SearchArticlesByNumber(ctx, "1900", 48, 258, 20, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := queries.SearchExactArticlesByNumber(ctx, "1900", 48, 258, 20, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := queries.SearchByOEMNumber(ctx, "7700115294", 48, 258, 20, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := queries.ArticleExistsForCountry(ctx, 28, 258); err != nil {
		t.Fatal(err)
	}
	if _, err := queries.SearchAnalogsByArticleIDs(ctx, []int{28, 29}, 48, 258, 20, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := queries.FindExactArticleIDsBySupplier(ctx, "1900", 30, 258, 10); err != nil {
		t.Fatal(err)
	}
	if _, err := queries.GetArticleCrossReferences(ctx, 28, 48, 258); err != nil {
		t.Fatal(err)
	}
	if _, err := queries.FindExactArticlesBatch(ctx, []ExactArticleBatchLookup{{SearchNumber: "1900", Brand: "BOSCH"}}, 48, 258); err != nil {
		t.Fatal(err)
	}
	if _, err := queries.GetArticleIdentitiesByIDsBatch(ctx, []int{28, 29}, 48, 258); err != nil {
		t.Fatal(err)
	}
	if _, err := queries.GetArticleCrossReferencesBatch(ctx, []int{28, 29}, 48, 258); err != nil {
		t.Fatal(err)
	}
}
