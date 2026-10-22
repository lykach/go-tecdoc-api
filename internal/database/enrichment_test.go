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

	"go-tecdoc-api/internal/models"
)

type staticRows struct {
	columns []string
	rows    [][]driver.Value
	index   int
}

func (r *staticRows) Columns() []string { return r.columns }
func (r *staticRows) Close() error      { return nil }
func (r *staticRows) Next(dest []driver.Value) error {
	if r.index >= len(r.rows) {
		return io.EOF
	}
	copy(dest, r.rows[r.index])
	r.index++
	return nil
}

type enrichmentTestDriver struct{}
type enrichmentTestConn struct{}

var registerEnrichmentTestDriver sync.Once

func (enrichmentTestDriver) Open(string) (driver.Conn, error) { return &enrichmentTestConn{}, nil }
func (*enrichmentTestConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("not implemented")
}
func (*enrichmentTestConn) Close() error              { return nil }
func (*enrichmentTestConn) Begin() (driver.Tx, error) { return nil, errors.New("not implemented") }
func (*enrichmentTestConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	switch {
	case strings.Contains(query, "FROM ARTICLE_CRITERIA"):
		if !strings.Contains(query, "CRITERIA.CRI_UNIT_DES_ID") || !strings.Contains(query, "ARTICLE_CRITERIA.ACR_DISPLAY") || !strings.Contains(query, "ARTICLE_CRITERIA.ACR_DES_ID") {
			return nil, errors.New("criteria query lost unit/display fields")
		}
		if strings.Contains(query, "AND ARTICLE_CRITERIA.ACR_DISPLAY = 1") {
			return nil, errors.New("includeHidden=true unexpectedly filtered hidden criteria")
		}
		return &staticRows{
			columns: []string{"ART_ID", "CRI_ID", "NAME", "SHORT", "VALUE", "VALUE_DES_ID", "VALUE_DES", "UNIT", "TYPE", "DISPLAY"},
			rows: [][]driver.Value{
				{int64(3575879), int64(636), "Number of pins", "Number of pins", "2", int64(7001), nil, "-pin connector", "Numerical", int64(1)},
				{int64(3575879), int64(637), "Supplementary Article", "Supplementary Article", "with central slave cylinder", int64(0), nil, "", "Alphanumerical", int64(0)},
			},
		}, nil
	case strings.Contains(query, "FROM ARTICLE_INFO"):
		if !strings.Contains(query, "ARTICLE_INFO.AIN_CTM") || !strings.Contains(query, "COUNTRY_RESTRICTIONS") {
			return nil, errors.New("article info query lost country restriction")
		}
		if strings.Contains(query, "AND COALESCE(ARTICLE_INFO.AIN_DISPLAY,0)<>0") {
			return nil, errors.New("includeHidden=true unexpectedly filtered detail-only article info")
		}
		return &staticRows{
			columns: []string{"ART_ID", "TYPE", "TEXT", "SORT", "DISPLAY", "TEXT_ID"},
			rows: [][]driver.Value{
				{int64(3575879), "General", "First page note", int64(1), int64(1), int64(9001)},
				{int64(3575879), "General", "Detail page note", int64(2), int64(0), int64(9002)},
			},
		}, nil
	case strings.Contains(query, "WITH ranked_accessories AS"):
		return &staticRows{
			columns: []string{"PARENT_ID", "ART_ID", "ART_ARTICLE_NR", "ART_SUP_ID", "ART_SUP_BRAND", "ART_PRODUCT_NAME", "GROUP_NAME", "ART_ACCS_TYPE", "ART_ACCS_TYP_ID", "QUANTITY", "PT_ID", "SORT_NO", "TOTAL_COUNT"},
			rows:    [][]driver.Value{{int64(3575879), int64(404), "ACC1", int64(10), "ATE", "Sensor", "Accessories", "PC", int64(123), int64(1), int64(55), int64(2), int64(1)}},
		}, nil
	case strings.Contains(query, "FROM ART_ACCS_LIST_CRITERIA AAC"):
		return &staticRows{
			columns: []string{"ART_ID", "TYPE", "TYPE_ID", "CRI_ID", "NAME", "SHORT", "CRI_TYPE", "UNIT", "RAW", "VALUE_DES"},
			rows: [][]driver.Value{
				{int64(404), "PC", int64(123), int64(100), "Fitting Position", "Fitting Position", "Alphanumerical", "", "Rear axle", ""},
				{int64(404), "PC", int64(123), int64(100), "Fitting Position", "Fitting Position", "Alphanumerical", "", "Front axle", ""},
			},
		}, nil
	case strings.Contains(query, "FROM LINK_ART_PT LAP"):
		return &staticRows{
			columns: []string{"ART_ID", "PT_ID", "PT_NAME", "STR_ID", "STR_TYPE", "LEVEL", "LEAF", "PARENT", "STR_NAME"},
			rows: [][]driver.Value{
				{int64(3575879), int64(6343), "Cable Repair Set, direction indicator", int64(100010), "PC,Motorcycle", int64(1), int64(0), nil, "Electrics"},
				{int64(3575879), int64(6343), "Cable Repair Set, direction indicator", int64(100480), "PC,Motorcycle", int64(4), int64(1), int64(100145), "Indicator Parts"},
			},
		}, nil
	case strings.Contains(query, "COUNT(DISTINCT LINK_ART.LA_ID)"):
		if !strings.Contains(query, "LINK_ART.LA_CTM = 0") || !strings.Contains(query, "LINK_LA_TYP.LAT_CTM = 0") {
			return nil, errors.New("classification usage lost country-context filters")
		}
		return &staticRows{
			columns: []string{"ART_ID", "PT_ID", "LINKAGE_COUNT", "TARGET_COUNT"},
			rows:    [][]driver.Value{{int64(3575879), int64(6343), int64(2), int64(176)}},
		}, nil
	default:
		return nil, errors.New("unexpected query: " + query)
	}
}

func openEnrichmentTestDB(t *testing.T) *sql.DB {
	t.Helper()
	registerEnrichmentTestDriver.Do(func() {
		sql.Register("tecdoc-enrichment-test", enrichmentTestDriver{})
	})
	db, err := sql.Open("tecdoc-enrichment-test", "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestGetArticleCriteriaBatchReturnsUnitRawValueAndDisplay(t *testing.T) {
	queries := &Queries{db: openEnrichmentTestDB(t)}
	items, err := queries.GetArticleCriteriaBatch(context.Background(), []int{3575879}, 48, 258, true)
	if err != nil {
		t.Fatal(err)
	}
	criteria := items[3575879]
	if len(criteria) != 2 {
		t.Fatalf("criteria = %#v, want display and detail-only rows", criteria)
	}
	got := criteria[0]
	if got.Value != "2" || got.RawValue != "2" || got.ValueDescriptionID != 7001 || got.Unit != "-pin connector" || !got.Display {
		t.Fatalf("criterion = %#v", got)
	}
	if criteria[1].Display || criteria[1].CriteriaID != 637 {
		t.Fatalf("detail criterion = %#v, want ACR_DISPLAY=0 retained", criteria[1])
	}
}

func TestGetArticleClassificationsBatchPreservesScopesAndUsage(t *testing.T) {
	queries := &Queries{db: openEnrichmentTestDB(t)}
	items, err := queries.GetArticleClassificationsBatch(context.Background(), []int{3575879}, 48, 258)
	if err != nil {
		t.Fatal(err)
	}
	classifications := items[3575879]
	if len(classifications) != 1 {
		t.Fatalf("classifications = %#v", classifications)
	}
	got := classifications[0]
	if got.ProductTypeID != 6343 || got.LinkageCount != 2 || got.TargetCount != 176 {
		t.Fatalf("classification = %#v", got)
	}
	if len(got.Scopes) != 2 || got.Scopes[0] != "Motorcycle" || got.Scopes[1] != "PC" {
		t.Fatalf("scopes = %#v", got.Scopes)
	}
	if len(got.TreeNodes) != 2 || !got.TreeNodes[1].IsLeaf {
		t.Fatalf("tree_nodes = %#v", got.TreeNodes)
	}
}

func TestGetArticleAccessoriesBatchMarksEmptyRequestedArticlesComplete(t *testing.T) {
	queries := &Queries{db: openEnrichmentTestDB(t)}
	items, err := queries.GetArticleAccessoriesBatch(context.Background(), []int{3575879, 999999}, 48, 258, 250)
	if err != nil {
		t.Fatal(err)
	}
	empty, ok := items[999999]
	if !ok {
		t.Fatal("empty requested ART_ID is missing from batch result")
	}
	if !empty.CriteriaIncluded || empty.Total != 0 || empty.Returned != 0 || empty.Truncated || len(empty.Items) != 0 {
		t.Fatalf("empty page = %#v, want an explicit complete empty page", empty)
	}
}

func TestGetArticleAccessoriesBatchIncludesCriteriaDetails(t *testing.T) {
	queries := &Queries{db: openEnrichmentTestDB(t)}
	items, err := queries.GetArticleAccessoriesBatch(context.Background(), []int{3575879}, 48, 258, 250)
	if err != nil {
		t.Fatal(err)
	}
	page := items[3575879]
	if page.Total != 1 || page.Returned != 1 || page.Truncated || len(page.Items) != 1 {
		t.Fatalf("page = %#v", page)
	}
	accessory := page.Items[0]
	if len(accessory.CriteriaDetails) != 2 {
		t.Fatalf("criteria_details = %#v", accessory.CriteriaDetails)
	}
	if got := accessory.Criteria["Fitting Position"]; got != "Rear axle; Front axle" {
		t.Fatalf("criteria map value = %q", got)
	}
}

func TestGetArticleInfoBatchKeepsDetailPageOnlyRows(t *testing.T) {
	queries := &Queries{db: openEnrichmentTestDB(t)}
	items, err := queries.GetArticleInfoBatch(context.Background(), []int{3575879}, 48, 258, true)
	if err != nil {
		t.Fatal(err)
	}
	info := items[3575879]
	if len(info) != 2 || !info[0].Display || info[1].Display {
		t.Fatalf("article info = %#v, want first-page and detail-page-only rows", info)
	}
}

func TestFlattenArticleCriteriaKeepsRepeatedValues(t *testing.T) {
	items := []models.ArticleCriterion{
		{CriteriaID: 100, Name: "Fitting Position", RawValue: "Transmission side", Type: "Alphanumerical", Display: true},
		{CriteriaID: 100, Name: "Fitting Position", RawValue: "Wheel side", Type: "Alphanumerical", Display: false},
	}
	flat := flattenArticleCriteria(items)
	if got := flat["Fitting Position"]; got != "Transmission side; Wheel side" {
		t.Fatalf("flattened value = %q", got)
	}
}

type exactLookupTestDriver struct{}
type exactLookupTestConn struct{}

var registerExactLookupTestDriver sync.Once

func (exactLookupTestDriver) Open(string) (driver.Conn, error) { return &exactLookupTestConn{}, nil }
func (*exactLookupTestConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("not implemented")
}
func (*exactLookupTestConn) Close() error              { return nil }
func (*exactLookupTestConn) Begin() (driver.Tx, error) { return nil, errors.New("not implemented") }
func (*exactLookupTestConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if !strings.Contains(query, "ART_LOOKUP.ARL_SEARCH_NUMBER = ? AND ARTICLES.ART_SUP_ID = ?") {
		return nil, errors.New("exact batch lookup is not supplier-qualified")
	}
	if strings.Contains(query, "ARL_SEARCH_NUMBER IN (") {
		return nil, errors.New("exact batch lookup fanned out by bare article number")
	}
	values := make([]interface{}, 0, len(args))
	for _, arg := range args {
		values = append(values, arg.Value)
	}
	if len(values) != 3 || values[0] != int64(258) || values[1] != "50390266" || values[2] != int64(72) {
		return nil, errors.New("unexpected exact lookup bound arguments")
	}
	return &staticRows{
		columns: []string{"SEARCH_NUMBER", "ART_ID", "ARTICLE_NR", "SUP_ID", "BRAND", "NAME"},
		rows:    [][]driver.Value{{"50390266", int64(3575879), "50390266", int64(72), "HERTH+BUSS ELPARTS", "Cable Repair Set"}},
	}, nil
}

func TestFindExactArticlesBatchQualifiesOwnNumberBySupplier(t *testing.T) {
	registerExactLookupTestDriver.Do(func() {
		sql.Register("tecdoc-exact-lookup-test", exactLookupTestDriver{})
	})
	db, err := sql.Open("tecdoc-exact-lookup-test", "")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	queries := &Queries{db: db}
	supplierID := 72
	items, err := queries.FindExactArticlesBatch(context.Background(), []ExactArticleBatchLookup{{
		SearchNumber: "50 390-266",
		SupplierID:   &supplierID,
	}}, 48, 258)
	if err != nil {
		t.Fatal(err)
	}
	got := items["50390266"]
	if len(got) != 1 || got[0].ArticleID != 3575879 || got[0].SupplierID != 72 {
		t.Fatalf("items = %#v", items)
	}
}

type applicabilityTestDriver struct{}
type applicabilityTestConn struct{}

var registerApplicabilityTestDriver sync.Once

func (applicabilityTestDriver) Open(string) (driver.Conn, error) {
	return &applicabilityTestConn{}, nil
}
func (*applicabilityTestConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("not implemented")
}
func (*applicabilityTestConn) Close() error              { return nil }
func (*applicabilityTestConn) Begin() (driver.Tx, error) { return nil, errors.New("not implemented") }
func (*applicabilityTestConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	switch {
	case strings.Contains(query, "WITH ranked_applicability"):
		if !strings.Contains(query, "LINK_LA_TYP.LAT_PT_ID = LINK_ART.LA_PT_ID") {
			return nil, errors.New("applicability query missing PT_ID join")
		}
		return &staticRows{
			columns: []string{"ART_ID", "LA_ID", "PT_ID", "LAT_TYPE", "LAT_TYP_ID", "TOTAL"},
			rows:    [][]driver.Value{{int64(3575879), int64(13039151), int64(6343), "PC", int64(214), int64(1)}},
		}, nil
	case strings.Contains(query, "FROM LA_CRITERIA"):
		return &staticRows{
			columns: []string{"LA_ID", "CRI_ID", "NAME", "SHORT", "TYPE", "UNIT", "VALUE", "VALUE_DES"},
			rows:    [][]driver.Value{{int64(13039151), int64(25), "Vehicle Identification Number (VIN) from", "VIN from", "Alphanumerical", nil, "3C-B-200 001", nil}},
		}, nil
	default:
		return nil, errors.New("unexpected query")
	}
}

func TestApplicabilityBatchKeepsLinkageProductTypeAndRestrictions(t *testing.T) {
	registerApplicabilityTestDriver.Do(func() {
		sql.Register("tecdoc-applicability-test", applicabilityTestDriver{})
	})
	db, err := sql.Open("tecdoc-applicability-test", "")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	queries := &Queries{db: db}
	items, err := queries.GetArticleApplicabilityLinkagesBatch(context.Background(), []int{3575879}, 48, 258, 100)
	if err != nil {
		t.Fatal(err)
	}
	page := items[3575879]
	if len(page.Linkages) != 1 || page.Linkages[0].ProductType != 6343 || page.Linkages[0].TargetType != "PC" {
		t.Fatalf("page = %#v", page)
	}
	if len(page.Linkages[0].Restrictions) != 1 || page.Linkages[0].Restrictions[0].CriteriaID != 25 {
		t.Fatalf("restrictions = %#v", page.Linkages[0].Restrictions)
	}
}

type criteriaFilterTestDriver struct{}
type criteriaFilterTestConn struct{}

var registerCriteriaFilterTestDriver sync.Once

func (criteriaFilterTestDriver) Open(string) (driver.Conn, error) {
	return &criteriaFilterTestConn{}, nil
}
func (*criteriaFilterTestConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("not implemented")
}
func (*criteriaFilterTestConn) Close() error              { return nil }
func (*criteriaFilterTestConn) Begin() (driver.Tx, error) { return nil, errors.New("not implemented") }
func (*criteriaFilterTestConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	const attack = "x' OR 1=1 --"
	if strings.Contains(query, attack) {
		return nil, errors.New("criterion value was interpolated into SQL")
	}
	if !strings.Contains(query, "AC0.ACR_VALUE = ?") || !strings.Contains(query, "AC0.ACR_CRI_ID = ?") {
		return nil, errors.New("criterion filters are not parameterized")
	}
	if !strings.Contains(query, "LINK_LA_TYP.LAT_PT_ID = LINK_ART.LA_PT_ID") {
		return nil, errors.New("vehicle linkage missing PT_ID guard")
	}
	foundAttack := false
	foundType := false
	for _, arg := range args {
		if value, ok := arg.Value.(string); ok && value == attack {
			foundAttack = true
		}
		if value, ok := arg.Value.(string); ok && value == "PC" {
			foundType = true
		}
	}
	if !foundAttack || !foundType {
		return nil, errors.New("expected bound values missing")
	}
	return &staticRows{
		columns: []string{"ART_ID", "ARTICLE_NR", "BRAND", "NAME"},
		rows:    [][]driver.Value{{int64(1), "A1", "TEST", "Test item"}},
	}, nil
}

func TestProductGroupCriteriaValuesAreBoundParameters(t *testing.T) {
	registerCriteriaFilterTestDriver.Do(func() {
		sql.Register("tecdoc-criteria-filter-test", criteriaFilterTestDriver{})
	})
	db, err := sql.Open("tecdoc-criteria-filter-test", "")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	queries := &Queries{db: db}
	items, err := queries.GetProductGroupArticlesWithCriteria(
		100480, 214, "PC", 48, 258, map[int]string{596: "x' OR 1=1 --"}, 50, 0,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ArticleNr != "A1" {
		t.Fatalf("items = %#v", items)
	}
}

type crossBatchTestDriver struct{}
type crossBatchTestConn struct{}

var registerCrossBatchTestDriver sync.Once

func (crossBatchTestDriver) Open(string) (driver.Conn, error) { return &crossBatchTestConn{}, nil }
func (*crossBatchTestConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("not implemented")
}
func (*crossBatchTestConn) Close() error              { return nil }
func (*crossBatchTestConn) Begin() (driver.Tx, error) { return nil, errors.New("not implemented") }
func (*crossBatchTestConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if !strings.Contains(query, "SOURCE_LOOKUP.ARL_ART_ID IN (?,?)") {
		return nil, errors.New("cross-reference batch lost set-based source ID filter")
	}
	if strings.Contains(query, "SOURCE_LOOKUP.ARL_TYPE = 'IAMNumber'") && strings.Contains(query, "TARGET_LOOKUP.ARL_TYPE = 'ArticleNumber'") {
		if len(args) != 3 || args[0].Value != int64(258) {
			return nil, errors.New("forward IAM query lost country argument")
		}
		return &staticRows{
			columns: []string{"SOURCE_ART_ID", "ART_ID", "ARTICLE_NR", "LEGACY_BRAND", "SUPPLIER_ID", "SUPPLIER_NAME", "MANUFACTURER_ID", "MANUFACTURER_NAME", "NAME", "TYPE"},
			rows:    [][]driver.Value{{int64(10), int64(20), "WK842/2", "MANN-FILTER", int64(4), "MANN-FILTER", nil, nil, "Fuel filter", "IAM"}},
		}, nil
	}
	if strings.Contains(query, "SOURCE_LOOKUP.ARL_TYPE = 'ArticleNumber'") && strings.Contains(query, "TARGET_LOOKUP.ARL_TYPE = 'IAMNumber'") {
		return &staticRows{columns: []string{"SOURCE_ART_ID", "ART_ID", "ARTICLE_NR", "LEGACY_BRAND", "SUPPLIER_ID", "SUPPLIER_NAME", "MANUFACTURER_ID", "MANUFACTURER_NAME", "NAME", "TYPE"}}, nil
	}
	if strings.Contains(query, "SOURCE_LOOKUP.ARL_TYPE = 'OENumber'") {
		if len(args) != 2 {
			return nil, errors.New("OEM cross query should only bind source IDs")
		}
		return &staticRows{
			columns: []string{"SOURCE_ART_ID", "ART_ID", "ARTICLE_NR", "LEGACY_BRAND", "SUPPLIER_ID", "SUPPLIER_NAME", "MANUFACTURER_ID", "MANUFACTURER_NAME", "NAME", "TYPE"},
			rows:    [][]driver.Value{{int64(10), nil, "03L115562", "AUDI", nil, nil, int64(5), "AUDI", "Oil filter", "OEM"}},
		}, nil
	}
	return nil, errors.New("unexpected cross-reference query")
}

func TestGetArticleCrossReferencesBatchPreservesSourceAndRoles(t *testing.T) {
	registerCrossBatchTestDriver.Do(func() {
		sql.Register("tecdoc-cross-batch-test", crossBatchTestDriver{})
	})
	db, err := sql.Open("tecdoc-cross-batch-test", "")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	queries := &Queries{db: db}
	items, err := queries.GetArticleCrossReferencesBatch(context.Background(), []int{10, 11}, 48, 258)
	if err != nil {
		t.Fatal(err)
	}
	if len(items[10]) != 2 {
		t.Fatalf("cross references = %#v", items)
	}
	if items[10][0].Type != "IAM" || items[10][0].SupplierID == nil || *items[10][0].SupplierID != 4 {
		t.Fatalf("IAM cross = %#v", items[10][0])
	}
	if items[10][1].Type != "OEM" || items[10][1].ManufacturerID == nil || *items[10][1].ManufacturerID != 5 {
		t.Fatalf("OEM cross = %#v", items[10][1])
	}
}

func TestExactLookupBrandNamesIncludesCanonicalAndAliases(t *testing.T) {
	got := exactLookupBrandNames(ExactArticleBatchLookup{
		Brand:        "MANN",
		BrandAliases: []string{" MANN-FILTER ", "mann", "MANN-FILTER"},
	})
	if len(got) != 2 || got[0] != "MANN" || got[1] != "MANN-FILTER" {
		t.Fatalf("brands = %#v", got)
	}
}
