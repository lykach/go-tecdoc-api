package database

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

// Explicit opt-in: uses the configured source database, SELECT statements only.
// All queries share one connection forced into session read-only mode.
func TestLiveCatalogReadOnly(t *testing.T) {
	if os.Getenv("TECDOC_AUDIT_READONLY") != "yes" {
		t.Skip("set TECDOC_AUDIT_READONLY=yes for source SELECT verification")
	}
	env, err := godotenv.Read("../../.env")
	if err != nil {
		t.Fatal("source configuration unavailable")
	}
	cfg := mysql.NewConfig()
	cfg.User = env["DB_USER"]
	cfg.Passwd = env["DB_PASSWORD"]
	cfg.Net = "tcp"
	cfg.Addr = env["DB_HOST"] + ":" + env["DB_PORT"]
	cfg.DBName = env["DB_NAME"]
	cfg.ParseTime = true
	cfg.Timeout = 5 * time.Second
	cfg.ReadTimeout = 30 * time.Second
	cfg.Params = map[string]string{"charset": "utf8mb4"}
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		t.Fatal("source connection unavailable")
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if _, err := db.ExecContext(ctx, "SET SESSION TRANSACTION READ ONLY"); err != nil {
		t.Fatal(err)
	}
	q := New(db)
	var language, country string
	if err := db.QueryRowContext(ctx, "SELECT LNG_ISO2 FROM LANGUAGES WHERE LNG_ID=48").Scan(&language); err != nil || language != "uk" {
		t.Fatal("Ukrainian language mismatch", err)
	}
	if err := db.QueryRowContext(ctx, "SELECT COU_ISOCODE2 FROM COUNTRIES WHERE COU_ID=258").Scan(&country); err != nil || country != "UA" {
		t.Fatal("Ukrainian country mismatch", err)
	}
	t.Log("language 48=uk; country 258=UA")
	for _, modelID := range []int{5114, 40960} {
		model, err := q.GetModelSeriesByID(modelID, 48, 258)
		if err != nil || model == nil {
			t.Fatal("source model unavailable", modelID, err)
		}
		var from, to sql.NullString
		if err := db.QueryRowContext(ctx, "SELECT MS_CI_FROM,MS_CI_TO FROM MODELS_SERIES WHERE MS_ID=?", modelID).Scan(&from, &to); err != nil {
			t.Fatal(err)
		}
		t.Logf("model=%d name=%s source dates=%s/%s API years=%d/%d", modelID, model.Name, from.String, to.String, model.YearFrom, model.YearTo)
	}
	for _, scope := range []string{"PC", "CV", "Motorcycle", "Engine"} {
		groups, err := q.GetProductGroups(scope, 48, 500, 0)
		if err != nil {
			t.Fatal(err)
		}
		var expected int
		if err := db.QueryRowContext(ctx, "SELECT count(*) FROM SEARCH_TREE WHERE STR_ID_PARENT IS NULL AND FIND_IN_SET(?,STR_TYPE)>0", scope).Scan(&expected); err != nil {
			t.Fatal(err)
		}
		if len(groups) != expected {
			t.Fatalf("%s tree lost roots: %d/%d", scope, len(groups), expected)
		}
		for _, g := range groups {
			if g.Name == "" || !strings.Contains(","+g.Type+",", ","+scope+",") {
				t.Fatal("tree scope/name lost")
			}
		}
		t.Logf("%s roots: %d", scope, len(groups))
	}
	for _, family := range []struct{ name, table, id, model, specific, specificID, country string }{
		{"PC", "PASSENGER_CARS", "PC_ID", "PC_MS_ID", "PC_COUNTRY_SPECIFICS", "PCS_PC_ID", "PCS_COU_ID"},
		{"CV", "COMMERCIAL_VEHICLES", "CV_ID", "CV_MS_ID", "CV_COUNTRY_SPECIFICS", "CCS_CV_ID", "CCS_COU_ID"},
		{"Motorcycle", "MOTORBIKES", "MTB_ID", "MTB_MS_ID", "MTB_COUNTRY_SPECIFICS", "MCS_MTB_ID", "MCS_COU_ID"},
	} {
		var id, model int
		query := "SELECT " + family.id + "," + family.model + " FROM " + family.table + " ORDER BY " + family.id + " LIMIT 1"
		if err := db.QueryRowContext(ctx, query).Scan(&id, &model); err != nil {
			t.Fatal(err)
		}
		count := 0
		size := 0
		switch family.name {
		case "PC":
			items, e := q.GetPassengerCars(model, "PC", 48, 258, 100, 0)
			if e != nil {
				t.Fatal(e)
			}
			size = len(items)
			count, err = q.CountPassengerCars(model, "PC", 258)
		case "CV":
			items, e := q.GetCommercialVehicles(model, 48, 258, 100, 0)
			if e != nil {
				t.Fatal(e)
			}
			size = len(items)
			count, err = q.CountCommercialVehicles(model, 258)
		case "Motorcycle":
			items, e := q.GetMotorcycles(model, 48, 258, 100, 0)
			if e != nil {
				t.Fatal(e)
			}
			size = len(items)
			count, err = q.CountMotorcycles(model, 258)
		}
		if err != nil {
			t.Fatal(err)
		}
		var expected int
		query = "SELECT count(*) FROM " + family.table + " v WHERE v." + family.model + "=? AND EXISTS(SELECT 1 FROM " + family.specific + " s WHERE s." + family.specificID + "=v." + family.id + " AND s." + family.country + " IN(258,0))"
		if family.name == "PC" {
			query += " AND FIND_IN_SET('PC',v.PC_TYPE)>0"
		}
		if err := db.QueryRowContext(ctx, query, model).Scan(&expected); err != nil {
			t.Fatal(err)
		}
		if count != expected || size > count || (count > 0 && size == 0) {
			t.Fatalf("%s count/list/source disagree: %d/%d/%d", family.name, count, size, expected)
		}
		t.Logf("%s model=%d returned=%d total=%d source=%d", family.name, model, size, count, expected)
	}
	var car, group int
	if err := db.QueryRowContext(ctx, `SELECT LAT_TYP_ID FROM LINK_LA_TYP WHERE LAT_TYPE='PC' ORDER BY LAT_TYP_ID LIMIT 1`).Scan(&car); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, `SELECT p.STR_ID FROM LINK_LA_TYP l JOIN LINK_PT_STR p ON p.PT_ID=l.LAT_PT_ID WHERE l.LAT_TYP_ID=? AND l.LAT_TYPE='PC' AND FIND_IN_SET('PC',p.STR_TYPE)>0 LIMIT 1`, car).Scan(&group); err != nil {
		t.Fatal(err)
	}
	items, err := q.GetProductGroupArticles(group, car, "PC", 48, 258, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	total, err := q.CountProductGroupArticles(group, car, "PC", 258, nil)
	if err != nil || len(items) > total {
		t.Fatal("article list/count mismatch", err)
	}
	t.Logf("vehicle=%d group=%d articles=%d total=%d", car, group, len(items), total)
	if len(items) == 0 {
		t.Fatal("sample article query empty; choose a populated source sample")
	}
	var criterion int
	var value string
	err = db.QueryRowContext(ctx, `SELECT ACR_CRI_ID,ACR_VALUE FROM ARTICLE_CRITERIA WHERE ACR_ART_ID=? AND (ACR_CTM=0 OR EXISTS(SELECT 1 FROM COUNTRY_RESTRICTIONS WHERE CNTR_CTM_ID=ACR_CTM AND CNTR_COU_ID=258)) LIMIT 1`, items[0].ID).Scan(&criterion, &value)
	if err != nil {
		t.Fatal("sample global criterion unavailable", err)
	}
	filtered, err := q.GetProductGroupArticlesWithCriteria(group, car, "PC", 48, 258, map[int]string{criterion: value}, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	filteredTotal, err := q.CountProductGroupArticles(group, car, "PC", 258, map[int]string{criterion: value})
	if err != nil || len(filtered) > filteredTotal || filteredTotal > total || len(filtered) == 0 {
		t.Fatal("filtered article scope/count mismatch", err)
	}
	for _, item := range filtered {
		if item.SupplierID == 0 {
			t.Fatal("filtered result lost supplier identity")
		}
	}
	t.Logf("criterion=%d filtered=%d total=%d; supplier identities retained", criterion, len(filtered), filteredTotal)
}
