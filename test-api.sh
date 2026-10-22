#!/usr/bin/env bash
set -euo pipefail

API="${API:-http://localhost:8081}"
API_KEY="${API_KEY:-td_live_c74f19f882469e4177d18ab49975d9a3c8d904311663608b7b352427cf49621c}"
MAX_PREVIEW_BYTES="${MAX_PREVIEW_BYTES:-1600}"
EXPECTED_OPENAPI_OPERATIONS=72

if [[ -z "$API_KEY" ]]; then
  echo "API_KEY environment variable is required for live contract tests." >&2
  exit 2
fi

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

declare -A COVERED=()

show_response() {
  local file="$1" size
  size="$(wc -c < "$file" | tr -d '[:space:]')"
  if (( size <= MAX_PREVIEW_BYTES )); then
    cat "$file"
  else
    head -c "$MAX_PREVIEW_BYTES" "$file"
    printf '\n... [response truncated: %s bytes total]\n' "$size"
  fi
}

mark_covered() {
  COVERED["$1"]=1
}

request_op_capture() {
  local op="$1" name="$2" path="$3" outfile="$4"
  local status
  status="$(curl -sS --connect-timeout 5 --max-time 40 -o "$outfile" -w '%{http_code}' \
    -H "X-API-Key: $API_KEY" "$API$path")"
  printf '\n[%s] %s HTTP %s %s\n' "$op" "$name" "$status" "$path"
  show_response "$outfile"; printf '\n'
  if [[ ! "$status" =~ ^2 ]]; then
    echo "FAILED: $op ($name) returned HTTP $status" >&2
    return 1
  fi
  mark_covered "$op"
}

request_op() {
  local op="$1" name="$2" path="$3"
  request_op_capture "$op" "$name" "$path" "$TMP_DIR/${op}.out"
}

post_op() {
  local op="$1" name="$2" path="$3" body="$4"
  local outfile="$TMP_DIR/${op}.out" status
  status="$(curl -sS --connect-timeout 5 --max-time 40 -o "$outfile" -w '%{http_code}' \
    -H "X-API-Key: $API_KEY" -H 'Content-Type: application/json' -X POST "$API$path" -d "$body")"
  printf '\n[%s] %s HTTP %s %s\n' "$op" "$name" "$status" "$path"
  show_response "$outfile"; printf '\n'
  if [[ ! "$status" =~ ^2 ]]; then
    echo "FAILED: $op ($name) returned HTTP $status" >&2
    return 1
  fi
  mark_covered "$op"
}

fixture_request() {
  local name="$1" path="$2" outfile="$3" status
  status="$(curl -sS --connect-timeout 5 --max-time 40 -o "$outfile" -w '%{http_code}' \
    -H "X-API-Key: $API_KEY" "$API$path")"
  if [[ ! "$status" =~ ^2 ]]; then
    echo "FAILED to resolve fixture $name: HTTP $status $path" >&2
    show_response "$outfile" >&2
    return 1
  fi
}

json_field() {
  local file="$1" field="$2"
  python3 - "$file" "$field" <<'PY'
import json, sys
with open(sys.argv[1], encoding='utf-8') as f:
    value=json.load(f)
for part in sys.argv[2].split('.'):
    value = value[int(part)] if isinstance(value, list) else value[part]
print(value)
PY
}

json_optional_field() {
  local file="$1" field="$2"
  python3 - "$file" "$field" <<'PY'
import json, sys
with open(sys.argv[1], encoding='utf-8') as f:
    value=json.load(f)
try:
    for part in sys.argv[2].split('.'):
        if isinstance(value, list):
            idx=int(part)
            if idx < 0 or idx >= len(value):
                raise LookupError
            value=value[idx]
        elif isinstance(value, dict):
            if part not in value:
                raise LookupError
            value=value[part]
        else:
            raise LookupError
except (LookupError, ValueError, TypeError):
    print('')
else:
    print('' if value is None else value)
PY
}

expect_status() {
  local name="$1" expected="$2" path="$3" key_mode="${4:-valid}" outfile="$TMP_DIR/negative.out"
  local args=(-sS --connect-timeout 5 --max-time 20 -o "$outfile" -w '%{http_code}')
  case "$key_mode" in
    valid) args+=(-H "X-API-Key: $API_KEY") ;;
    invalid) args+=(-H 'X-API-Key: td_invalid_contract_test_key') ;;
    none) ;;
    *) echo "unknown key mode $key_mode" >&2; return 1 ;;
  esac
  local status
  status="$(curl "${args[@]}" "$API$path")"
  printf '\n[negative] %s HTTP %s (expected %s) %s\n' "$name" "$status" "$expected" "$path"
  show_response "$outfile"; printf '\n'
  if [[ "$status" != "$expected" ]]; then
    echo "FAILED negative test: $name expected HTTP $expected, got $status" >&2
    return 1
  fi
}

echo "Go TecDoc API final production contract test"
echo "Target: $API"
echo "Expected OpenAPI operations: $EXPECTED_OPENAPI_OPERATIONS"

# ---------------------------------------------------------------------------
# Negative/auth/validation contract. 403/429 branch mechanics are unit-tested
# by internal/auth; live 403/429 needs a deliberately restricted/rate-limited
# key and is intentionally not forced with the production smoke key.
# ---------------------------------------------------------------------------
expect_status "missing API key" 401 "/api/v1" none
expect_status "invalid API key" 401 "/api/v1" invalid
expect_status "invalid language_id" 400 "/api/v1/languages?language_id=0"
expect_status "invalid country_id" 400 "/api/v1/countries?country_id=nope"
expect_status "invalid path integer" 400 "/api/v1/articles/not-an-id"
expect_status "non-positive path integer" 400 "/api/v1/articles/-1"
expect_status "zero page size" 400 "/api/v1/articles/1307505/accessories?limit=0"
expect_status "page size above hard limit" 400 "/api/v1/articles/1307505/accessories?limit=501"
expect_status "negative offset" 400 "/api/v1/articles/1307505/accessories?offset=-1"
expect_status "invalid boolean" 400 "/api/v1/articles/28/specifications?include_hidden=maybe"
expect_status "invalid dataset order" 400 "/api/v1/datasets/ARTICLES?order_by=ART_ID&order=SIDEWAYS&limit=1"
expect_status "invalid dataset format" 400 "/api/v1/datasets/ARTICLES?format=xml&limit=1"
expect_status "unknown dataset" 404 "/api/v1/datasets/THIS_TABLE_DOES_NOT_EXIST?limit=1"
expect_status "unsafe analog number-only lookup" 400 "/api/v1/search/analog?search_number=1900"
expect_status "missing KBA number" 400 "/api/v1/search/kba"
expect_status "malformed criteria filter" 400 "/api/v1/product-groups/100470/articles?car_id=9877&criteria=broken"
expect_status "invalid passenger subtype" 400 "/api/v1/models/348/cars?vehicle_type=CV&limit=1&offset=0"
expect_status "invalid vehicle import stats type" 400 "/api/v1/import/vehicle-stats?vehicle_types=PC,BOAT"
expect_status "invalid bulk vehicle product-group type" 400 "/api/v1/import/vehicle-product-groups?vehicle_type=BOAT&vehicle_ids=9877"

# ---------------------------------------------------------------------------
# Resolve live fixtures. These calls deliberately use the raw read-only dataset
# layer so the contract test stays valid when numeric vehicle IDs change in a
# future imported TecDoc quarter.
# ---------------------------------------------------------------------------
fixture_request "CV 17582 model" "/api/v1/datasets/COMMERCIAL_VEHICLES?select=CV_ID,CV_MS_ID&filter=CV_ID:17582&order_by=CV_ID&limit=1" "$TMP_DIR/cv.json"
CV_MS_ID="$(json_field "$TMP_DIR/cv.json" 'rows.0.CV_MS_ID')"

fixture_request "motorcycle with country/universal data" "/api/v1/datasets/MTB_COUNTRY_SPECIFICS?select=MCS_MTB_ID,MCS_COU_ID&filter=MCS_COU_ID:62,0&order_by=MCS_MTB_ID&limit=1" "$TMP_DIR/mtb_country.json"
MTB_ID="$(json_field "$TMP_DIR/mtb_country.json" 'rows.0.MCS_MTB_ID')"
fixture_request "resolved motorcycle model" "/api/v1/datasets/MOTORBIKES?select=MTB_ID,MTB_MS_ID&filter=MTB_ID:${MTB_ID}&order_by=MTB_ID&limit=1" "$TMP_DIR/mtb.json"
MTB_MS_ID="$(json_field "$TMP_DIR/mtb.json" 'rows.0.MTB_MS_ID')"

fixture_request "first engine" "/api/v1/datasets/ENGINES?select=ENG_ID,ENG_MFA_ID&order_by=ENG_ID&limit=1" "$TMP_DIR/engine.json"
ENG_ID="$(json_field "$TMP_DIR/engine.json" 'rows.0.ENG_ID')"
ENG_MFA_ID="$(json_field "$TMP_DIR/engine.json" 'rows.0.ENG_MFA_ID')"

# Resolve a real unrestricted TecDoc article. ART_CTM=0 is the source-schema
# representation for an article without a country restriction; this fixture is
# used below to prevent regressions back to an unconditional
# COUNTRY_RESTRICTIONS join. Select several rows so a nullable article number in
# one source row cannot make the live release gate brittle.
fixture_request "unrestricted ART_CTM=0 article" "/api/v1/datasets/ARTICLES?select=ART_ID,ART_ARTICLE_NR,ART_SUP_ID,ART_SUP_BRAND,ART_CTM&filter=ART_CTM:0&order_by=ART_ID&limit=50" "$TMP_DIR/unrestricted_article.json"
mapfile -t UNRESTRICTED_ARTICLE < <(python3 - "$TMP_DIR/unrestricted_article.json" <<'PY3'
import json, sys
payload=json.load(open(sys.argv[1], encoding='utf-8'))
for row in payload.get('rows') or []:
    number=str(row.get('ART_ARTICLE_NR') or '').strip()
    article_id=int(row.get('ART_ID') or 0)
    supplier_id=int(row.get('ART_SUP_ID') or 0)
    ctm=int(row.get('ART_CTM') or 0)
    if article_id > 0 and supplier_id > 0 and number and ctm == 0:
        print(article_id)
        print(supplier_id)
        print(number)
        break
else:
    raise SystemExit('no usable unrestricted ART_CTM=0 article fixture found')
PY3
)
UNRESTRICTED_ART_ID="${UNRESTRICTED_ARTICLE[0]:-}"
UNRESTRICTED_SUPPLIER_ID="${UNRESTRICTED_ARTICLE[1]:-}"
UNRESTRICTED_ARTICLE_NUMBER="${UNRESTRICTED_ARTICLE[2]:-}"
if [[ -z "$UNRESTRICTED_ART_ID" || -z "$UNRESTRICTED_SUPPLIER_ID" || -z "$UNRESTRICTED_ARTICLE_NUMBER" ]]; then
  echo "FAILED: unrestricted ART_CTM=0 fixture is incomplete" >&2
  exit 1
fi
UNRESTRICTED_ARTICLE_NUMBER_URL="$(python3 - "$UNRESTRICTED_ARTICLE_NUMBER" <<'PY3'
import sys, urllib.parse
print(urllib.parse.quote(sys.argv[1], safe=''))
PY3
)"
fixture_request "unrestricted exact article search" "/api/v1/search/article-exact?number=${UNRESTRICTED_ARTICLE_NUMBER_URL}&language_id=4&country_id=62&limit=50&offset=0" "$TMP_DIR/unrestricted_exact_search.json"
python3 - "$TMP_DIR/unrestricted_exact_search.json" "$UNRESTRICTED_ART_ID" <<'PY3'
import json, sys
payload=json.load(open(sys.argv[1], encoding='utf-8'))
expected=int(sys.argv[2])
ids={int(item.get('article_id') or 0) for item in payload.get('results') or []}
if expected not in ids:
    raise SystemExit(f'unrestricted ART_CTM=0 article {expected} was filtered out by exact search')
print(f"[country restriction] exact search preserves unrestricted ART_CTM=0 article {expected}")
PY3

# PASSENGER_CARS.PC_TYPE is a MySQL SET, so a row such as "PC,LCV" belongs
# to both passenger subtypes. Generic /datasets equality filters cannot express
# SET membership and would incorrectly report zero for LCV/E-* in that case.
# Use the import stats endpoint, whose counts are based on FIND_IN_SET(), and
# verify the public route accepts each subtype. Any returned row must contain
# the requested token in its full PC_TYPE membership value.
fixture_request "passenger subtype membership stats" "/api/v1/import/vehicle-stats?vehicle_types=PC,LCV,E-PC,E-LCV" "$TMP_DIR/passenger_membership_stats.json"
for PASSENGER_TYPE in PC LCV E-PC E-LCV; do
  SAFE_TYPE="${PASSENGER_TYPE//-/_}"
  PASSENGER_TOTAL="$(json_optional_field "$TMP_DIR/passenger_membership_stats.json" "by_type.${PASSENGER_TYPE}")"
  fixture_request "${PASSENGER_TYPE} public passenger contract" "/api/v1/models/348/cars?vehicle_type=${PASSENGER_TYPE}&language_id=4&country_id=62&limit=2&offset=0" "$TMP_DIR/passenger_${SAFE_TYPE}_public.json"
  python3 - "$TMP_DIR/passenger_${SAFE_TYPE}_public.json" "$PASSENGER_TYPE" "${PASSENGER_TOTAL:-0}" <<'PY2'
import json, sys
payload=json.load(open(sys.argv[1], encoding='utf-8'))
expected=sys.argv[2]
source_total=sys.argv[3]
for key in ('vehicles', 'total', 'limit', 'offset'):
    if key not in payload:
        raise SystemExit(f"passenger subtype response for {expected} is missing {key}")
items=payload.get('vehicles') or []
wrong=[]
for item in items:
    memberships={part.strip() for part in str(item.get('vehicle_type') or '').split(',') if part.strip()}
    if expected not in memberships:
        wrong.append(item.get('vehicle_type'))
if wrong:
    raise SystemExit(f"passenger subtype membership mismatch for {expected}: {wrong}")
print(f"[passenger subtype] {expected}: membership rows={source_total}; public fixture rows={len(items)}")
PY2
done

printf '\nLive fixtures: CV_MS_ID=%s MTB_ID=%s MTB_MS_ID=%s ENG_ID=%s ENG_MFA_ID=%s\n' "$CV_MS_ID" "$MTB_ID" "$MTB_MS_ID" "$ENG_ID" "$ENG_MFA_ID"

# ---------------------------------------------------------------------------
# Every OpenAPI operation exactly once for coverage purposes. Additional fixture
# lookups above do not affect the unique-operation count.
# ---------------------------------------------------------------------------
request_op "Health" "health" "/health"
request_op "GetAPIIndex" "API discovery" "/api/v1"
request_op "GetLanguages" "languages" "/api/v1/languages"
request_op "GetLanguageByID" "language detail" "/api/v1/languages/4"
request_op "GetCountries" "countries" "/api/v1/countries?language_id=4"
request_op "GetCountryByID" "country detail" "/api/v1/countries/62?language_id=4"

request_op "GetSuppliers" "suppliers" "/api/v1/suppliers?page=1&limit=2"
request_op "GetSupplierByID" "supplier BOSCH" "/api/v1/suppliers/30"
request_op "GetSupplierProducts" "supplier product types" "/api/v1/suppliers/30/products?language_id=4&limit=2&offset=0"
request_op "GetSupplierAddresses" "supplier addresses" "/api/v1/suppliers/30/addresses?language_id=4"
request_op "GetSupplierLogos" "supplier logos" "/api/v1/suppliers/30/logos"

request_op "GetVehicleImportStats" "vehicle import aggregate stats" "/api/v1/import/vehicle-stats?vehicle_types=PC,CV,Motorcycle,Engine"
request_op "GetImportProductGroups" "flat product-group import page" "/api/v1/import/product-groups?vehicle_types=PC,CV,Motorcycle,Engine&language_id=4&limit=2&offset=0"
request_op "GetImportVehicleProductGroups" "bulk vehicle product groups" "/api/v1/import/vehicle-product-groups?vehicle_type=PC&vehicle_ids=9877&language_id=4"
fixture_request "bulk vehicle product group ids_only batch" "/api/v1/import/vehicle-product-groups?vehicle_type=PC&vehicle_ids=28251,126961,1052,33667,58731,1053,120080,120081,121446&language_id=48&ids_only=true" "$TMP_DIR/import_vehicle_product_group_ids.out"
python3 - "$TMP_DIR/import_vehicle_product_group_ids.out" <<'PY'
import json, sys
with open(sys.argv[1], encoding='utf-8') as f:
    payload=json.load(f)
assert payload.get('ids_only') is True, payload
items=payload.get('items') or []
assert len(items) == 9, f"expected 9 items, got {len(items)}"
assert all('category_ids' in item for item in items), payload
print("[bulk ids_only regression] 9-vehicle PC batch: OK")
PY
request_op "GetManufacturers" "PC manufacturers" "/api/v1/manufacturers?vehicle_type=PC&limit=2&offset=0"
request_op "GetManufacturerByID" "FIAT manufacturer" "/api/v1/manufacturers/35"
request_op "GetModelSeries" "FIAT model series" "/api/v1/manufacturers/35/models?vehicle_type=PC&language_id=4&country_id=62&limit=2&offset=0"
request_op "GetModelSeriesDetails" "model series 348" "/api/v1/models/348?language_id=4&country_id=62"

request_op "GetCommercialVehicles" "commercial vehicles in resolved series" "/api/v1/models/${CV_MS_ID}/cv?language_id=4&country_id=62&limit=2&offset=0"
request_op "GetCommercialVehicleDetails" "commercial vehicle 17582" "/api/v1/cv/17582?language_id=4&country_id=62"
request_op "GetCommercialVehicleTypeNumbers" "commercial type numbers" "/api/v1/cv/17582/type-numbers"
request_op "GetCommercialVehicleCriteria" "commercial criteria" "/api/v1/cv/17582/criteria?language_id=4&product_type_id=1&suggested_only=true"
request_op "GetCommercialVehicleChassis" "commercial chassis" "/api/v1/cv/17582/chassis?language_id=4"

request_op "GetMotorcycles" "motorcycles in resolved series" "/api/v1/models/${MTB_MS_ID}/mc?language_id=4&country_id=62&limit=2&offset=0"
request_op "GetMotorcycleDetails" "motorcycle detail" "/api/v1/mc/${MTB_ID}?language_id=4&country_id=62"
request_op "GetMotorcycleTypeNumbers" "motorcycle type numbers" "/api/v1/mc/${MTB_ID}/type-numbers"
request_op "GetMotorcycleCriteria" "motorcycle criteria" "/api/v1/mc/${MTB_ID}/criteria?language_id=4&product_type_id=1&suggested_only=true"

request_op "GetPassengerCars" "FIAT cars" "/api/v1/models/348/cars?language_id=4&country_id=62&limit=2&offset=0"
request_op "GetCarDetails" "passenger car 9877" "/api/v1/cars/9877?language_id=4&country_id=62"
request_op "GetCarProductGroups" "car product groups" "/api/v1/cars/9877/product-groups?vehicle_type=PC&language_id=4&limit=2&offset=0"
request_op "GetCarTypeNumbers" "car type numbers" "/api/v1/cars/9877/type-numbers"
request_op "GetCarCriteria" "car criteria" "/api/v1/cars/9877/criteria?language_id=4&product_type_id=1&suggested_only=true"

request_op "GetEngines" "engine master listing" "/api/v1/engines?manufacturer_id=${ENG_MFA_ID}&language_id=4&limit=2&offset=0"
request_op "GetEngineDetails" "engine detail" "/api/v1/engines/${ENG_ID}?language_id=4"
request_op "GetAxleDetail" "axle detail" "/api/v1/axles/5779?language_id=4"

request_op "SearchArticles" "article search" "/api/v1/articles/search?number=0001106017&language_id=4&country_id=62&limit=2&offset=0"
request_op "GetArticleDetails" "article detail bounded nested relations" "/api/v1/articles/28?language_id=4&country_id=62&related_limit=2"
request_op "GetArticleApplicability" "article applicability paged" "/api/v1/articles/28/applicability?language_id=4&country_id=62&limit=2&offset=0"
request_op "GetArticleCrossReferences" "article cross references" "/api/v1/articles/1878343/cross-references?language_id=4&country_id=62"
request_op "GetArticleMedia" "article media" "/api/v1/articles/1830460/media?language_id=4"
request_op "GetArticleComponents" "article components" "/api/v1/articles/10445689/components?language_id=4&country_id=62"
request_op "GetArticleAccessories" "article accessories paged" "/api/v1/articles/1307505/accessories?language_id=4&country_id=62&limit=2&offset=0"
request_op "GetArticleOEMNumbers" "article OEM" "/api/v1/articles/28/oem"
request_op "GetArticleCoordinates" "article coordinates" "/api/v1/articles/124926/coordinates"
request_op "GetArticleCriteria" "article criteria" "/api/v1/articles/28/criteria?language_id=4&include_hidden=true"
request_op "GetArticleClassification" "article classification" "/api/v1/articles/28/classification?language_id=4"
request_op "GetArticleInfo" "article info" "/api/v1/articles/28/info?language_id=4&include_hidden=true"
request_op "GetArticlePrices" "article prices" "/api/v1/articles/28/prices?language_id=4&country_id=62&include_expired=true"
request_op "GetArticleSpecifications" "article specifications paged" "/api/v1/articles/28/specifications?language_id=4&country_id=62&include_hidden=true&limit=2&offset=0"
request_op "GetArticleLinkageInfo" "article linkage info paged" "/api/v1/articles/28/linkage-info?language_id=4&country_id=62&limit=2&offset=0"
request_op "GetPartsDiagram" "parts diagram paged" "/api/v1/articles/124926/parts-diagram?limit=2&offset=0"

ENRICHMENT_BATCH_BODY="$(python3 - "$UNRESTRICTED_ARTICLE_NUMBER" "$UNRESTRICTED_SUPPLIER_ID" <<'PY3'
import json, sys
print(json.dumps({
    "language_id": 4,
    "country_id": 62,
    "include": ["classification", "oem", "criteria"],
    "items": [
        {"request_id": "final-contract-1", "article_id": 28},
        {
            "request_id": "ctm-zero-contract",
            "article_number": sys.argv[1],
            "supplier_id": int(sys.argv[2]),
        },
    ],
}, separators=(",", ":")))
PY3
)"
post_op "EnrichArticlesBatch" "batch enrichment + unrestricted ART_CTM=0 identity" "/api/v1/enrichment/articles/batch" "$ENRICHMENT_BATCH_BODY"
python3 - "$TMP_DIR/EnrichArticlesBatch.out" "$UNRESTRICTED_ART_ID" <<'PY3'
import json, sys
payload=json.load(open(sys.argv[1], encoding='utf-8'))
expected=int(sys.argv[2])
results={item.get('request_id'): item for item in payload.get('results') or []}
item=results.get('ctm-zero-contract')
if not item:
    raise SystemExit('batch enrichment did not return the ART_CTM=0 fixture result')
if item.get('status') != 'exact':
    raise SystemExit(f"ART_CTM=0 batch identity expected exact, got {item.get('status')!r}")
ids={int(v) for v in item.get('article_ids') or []}
if expected not in ids:
    raise SystemExit(f'ART_CTM=0 batch identity lost technical ART_ID {expected}: {sorted(ids)}')
print(f"[country restriction] batch exact identity preserves unrestricted ART_CTM=0 article {expected}")
PY3

request_op "GetQuickStart" "quick start" "/api/v1/quick-start?vehicle_type=PC&language_id=4"
request_op "GetProductTypeCriteria" "product type criteria" "/api/v1/product-types/1/criteria?language_id=4&suggested_only=true"
request_op "GetCriterionKeyValues" "criterion values paged" "/api/v1/criteria/468/values?language_id=4&limit=5&offset=0"
request_op "SearchProductGroups" "product-group text search" "/api/v1/product-groups/search?q=VALVE&vehicle_type=PC&language_id=4&limit=2&offset=0"
request_op "GetProductGroups" "product group roots" "/api/v1/product-groups?vehicle_type=PC&language_id=4&limit=2&offset=0"
request_op "GetCategoryChildren" "product group children" "/api/v1/product-groups/100002/children?vehicle_type=PC&language_id=4"
request_op "GetProductGroupArticles" "product-group articles" "/api/v1/product-groups/100470/articles?car_id=9877&vehicle_type=PC&language_id=4&country_id=62&limit=2&offset=0"

request_op "SearchByKBA" "KBA" "/api/v1/search/kba?number=0588549&vehicle_type=PC&limit=2&offset=0"
request_op "SearchByKenteken" "Kenteken" "/api/v1/search/kenteken?number=2VXT04&vehicle_type=PC&limit=2&offset=0"
request_op "SearchByTypeNumber" "generic type number" "/api/v1/search/type-number?number=0588549&number_type=German%20KBA%20Number&vehicle_type=PC&limit=2&offset=0"
request_op "SearchArticleByNumber" "broad article search" "/api/v1/search/article?number=0001106017&language_id=4&country_id=62&limit=2&offset=0"
request_op "SearchExactArticleByNumber" "exact article search" "/api/v1/search/article-exact?number=0001106017&language_id=4&country_id=62&limit=2&offset=0"
request_op "SearchByOEM" "OEM search" "/api/v1/search/oem?number=000%201477%20V006&language_id=4&country_id=62&limit=2&offset=0"
request_op "SearchAnalogs" "analog search by exact ART_ID" "/api/v1/search/analog?art_id=1878343&language_id=4&country_id=62&limit=2&offset=0"
request_op "SearchOEMByOEM" "OEM-to-OEM" "/api/v1/search/oem-oem?oem_number=000%201477%20V006&limit=2&offset=0"

request_op "GetDatasets" "dataset list" "/api/v1/datasets"
request_op "GetDatasetSchema" "ARTICLES schema" "/api/v1/datasets/ARTICLES/schema"
request_op_capture "GetDatasetRows" "deterministic raw row page" "/api/v1/datasets/ARTICLES?select=ART_ID,ART_ARTICLE_NR,ART_SUP_ID,ART_SUP_BRAND&order_by=ART_ID&order=ASC&limit=2&offset=0" "$TMP_DIR/GetDatasetRows.out"
request_op "ExportCrossReferences" "cross export JSON" "/api/v1/exports/cross-references?limit=2&offset=0&format=json"
request_op "ExportEAN" "EAN export JSON" "/api/v1/exports/ean?limit=2&offset=0&format=json"

covered_count="${#COVERED[@]}"
printf '\nUnique OpenAPI operationIds exercised live: %s/%s\n' "$covered_count" "$EXPECTED_OPENAPI_OPERATIONS"
if (( covered_count != EXPECTED_OPENAPI_OPERATIONS )); then
  echo "FAILED: live coverage is not 72/72" >&2
  printf '%s\n' "${!COVERED[@]}" | sort >&2
  exit 1
fi

echo "Negative/auth/validation checks: PASS"
echo "All 72 OpenAPI operations completed successfully."
echo "FINAL PRODUCTION CONTRACT TEST: PASS"
