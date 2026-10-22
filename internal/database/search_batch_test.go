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

var registerAnalogBatchDriver sync.Once

type analogBatchDriver struct{}
type analogBatchConn struct{}
type analogBatchRows struct{ emitted bool }

func (analogBatchDriver) Open(string) (driver.Conn, error) { return &analogBatchConn{}, nil }
func (*analogBatchConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("not implemented")
}
func (*analogBatchConn) Close() error              { return nil }
func (*analogBatchConn) Begin() (driver.Tx, error) { return nil, errors.New("not implemented") }
func (*analogBatchConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if !strings.Contains(query, "SOURCE_LOOKUP.ARL_ART_ID IN (?,?,?)") {
		return nil, errors.New("missing three-source batch IN clause")
	}
	if strings.Count(query, "ARTICLES.ART_CTM = 0 OR EXISTS") != 2 {
		return nil, errors.New("analog batch must preserve unrestricted CTM=0 target articles in both IAM directions")
	}
	if len(args) != 16 {
		return nil, errors.New("unexpected batch argument count")
	}
	return &analogBatchRows{}, nil
}
func (*analogBatchRows) Columns() []string {
	return []string{
		"ART_ID", "ART_ARTICLE_NR", "ART_SUP_BRAND", "ART_SUP_ID",
		"ART_PRODUCT_NAME", "FOUND_VIA", "RELATION_TYPE",
	}
}
func (*analogBatchRows) Close() error { return nil }
func (rows *analogBatchRows) Next(dest []driver.Value) error {
	if rows.emitted {
		return io.EOF
	}
	rows.emitted = true
	dest[0] = int64(8422)
	dest[1] = "WK 842/2"
	dest[2] = "MANN-FILTER"
	dest[3] = int64(4)
	dest[4] = "Fuel filter"
	dest[5] = "IAM"
	dest[6] = "IAM"
	return nil
}

func TestSearchAnalogsByArticleIDsBuildsSingleBatchQuery(t *testing.T) {
	registerAnalogBatchDriver.Do(func() {
		sql.Register("tecdoc-analog-batch-test", analogBatchDriver{})
	})
	db, err := sql.Open("tecdoc-analog-batch-test", "")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	queries := &Queries{db: db}
	items, err := queries.SearchAnalogsByArticleIDs(
		context.Background(), []int{181, 182, 183}, 48, 258, 200, 0,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ArticleNr != "WK 842/2" || items[0].Brand != "MANN-FILTER" {
		t.Fatalf("items = %#v", items)
	}
}
