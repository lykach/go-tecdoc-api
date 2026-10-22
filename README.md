# Go TecDoc API

Production-oriented read-only REST API for the supplied **TecDoc 2024Q1** MySQL/MariaDB schema.

This revision has two complementary access layers:

1. **Domain API** — normalized endpoints for vehicles, articles, applicability, search tree, criteria, crosses, KBA/Kenteken, CV chassis, prices, text modules, diagrams, etc.
2. **Complete dataset API** — safe read-only access to every TecDoc source table through `/api/v1/datasets/*`.

## Highlights

- PC / CV / Motorcycle / Engine / Axle catalog data.
- Ukrainian/localized text support with language fallback.
- Country-aware `CTM` restrictions.
- Exact and broad article-number lookup.
- OEM/IAM cross references and safe analog resolution.
- `TYPE_NUMBER_LOOKUP`: KBA, Kenteken, TypeMine, Swiss type numbers, CV manufacturer numbers, engine codes.
- Full article specifications: `ARTICLE_CRITERIA` + `LA_CRITERIA` without flattening linkage restrictions into global product data.
- TecDoc `ACR_DISPLAY=0` / `AIN_DISPLAY=0` means detail-page-only, not hidden. Full enrichment keeps these rows; structured filter projection remains an explicit consumer policy.
- Article info (`ARTICLE_INFO`) and linkage info (`LA_INFO`).
- Accessory criteria (`ART_ACCS_LIST_CRITERIA`).
- Search Tree / `STR_LOOKUP` / Quick Start / criterion KeyValue dictionaries.
- CV wheelbases, suspension, driver cabs and axles.
- Axle models and brake sizes.
- Parts-diagram coordinates using `LINK_PL_SC` + `SENSITIVE_COORDINATES`.
- TecDoc price table access.
- Paged JSON/CSV CROSS and EAN exports.
- Read-only raw dataset access with `INFORMATION_SCHEMA` validation of dynamic table/column identifiers.
- API-key auth, allowlists, expiry, rate limits and usage logging.
- Production hardening: strict query validation, bounded heavy endpoints, deterministic pagination, exact-country-over-universal fallback, and a 72/72 OpenAPI/live contract gate.
- Passenger-family subtype contract: `PASSENGER_CARS.PC_TYPE` is a multi-value `SET('PC','E-PC','LCV','E-LCV')`. `/models/{id}/cars?vehicle_type=...` filters by SET membership (`FIND_IN_SET`), while each returned `vehicle_type` preserves the complete source membership string (for example `PC,LCV`). `/engines` exposes the paged global `ENGINES` master for resumable downstream imports.

## Documentation

- [`API_REFERENCE.md`](API_REFERENCE.md) — full endpoint reference with request/response examples.
- [`API_EXAMPLES.md`](API_EXAMPLES.md) — task-oriented request examples.
- [`OPENAPI.yaml`](OPENAPI.yaml) — OpenAPI 3.0.3, all registered routes.
- [`SQL_REVIEW_2024Q1.md`](SQL_REVIEW_2024Q1.md) — review/corrections of supplied SQL examples.
- [`SCHEMA_COVERAGE.md`](SCHEMA_COVERAGE.md) — mapping of all 56 TecDoc domain tables to API coverage.
- [`PRODUCTION_BASELINE.md`](PRODUCTION_BASELINE.md) — final release gate, deployment and rollback checklist.

## Requirements

- Go `1.21.5+`.
- MySQL/MariaDB containing the `tecdoc2024q1` schema.
- API auth tables from `migrations/000001_api_keys.sql`.

## Configuration

`.env`:

```dotenv
DB_HOST=127.0.0.1
DB_PORT=3306
DB_USER=tecdoc_api
DB_PASSWORD=change-me
DB_NAME=tecdoc2024q1
SERVER_PORT=8081
DEFAULT_LANGUAGE_ID=48
DEFAULT_COUNTRY_ID=258
ALLOWED_ORIGINS=*
```

For production, do not use `ALLOWED_ORIGINS=*` unless the API is genuinely intended for arbitrary browser origins.

## Build and test

```bash
go test ./...
go build ./...
```

Then start:

```bash
go run ./cmd/server
# or
mkdir -p bin
go build -o bin/server ./cmd/server
./bin/server
```

Health check:

```bash
curl -sS http://localhost:8081/health
```

## API key setup

Apply auth migration once:

```bash
mysql -u <user> -p tecdoc2024q1 < migrations/000001_api_keys.sql
```

Generate a new key:

```bash
go run ./cmd/apikey -name autoparts-platform -rate-limit 1000
```

Register the current test key supplied for this environment:

```bash
go run ./cmd/apikey \
  -name tecdoc-test \
  -key '<dedicated-test-key>' \
  -rate-limit 1000
```

Only SHA-256 hash + public prefix are stored in `api_keys`; plaintext is not stored.

Every `/api/v1/*` request then includes:

```http
X-API-Key: <dedicated-test-key>
```

## Key endpoint groups

```text
GET  /api/v1
GET  /api/v1/languages
GET  /api/v1/countries

GET  /api/v1/suppliers
GET  /api/v1/suppliers/{id}/addresses
GET  /api/v1/suppliers/{id}/logos
GET  /api/v1/manufacturers
GET  /api/v1/manufacturers/{id}/models

GET  /api/v1/models/{id}/cars
GET  /api/v1/cars/{id}
GET  /api/v1/cars/{id}/type-numbers
GET  /api/v1/cars/{id}/criteria

GET  /api/v1/models/{id}/cv
GET  /api/v1/cv/{id}
GET  /api/v1/cv/{id}/type-numbers
GET  /api/v1/cv/{id}/criteria
GET  /api/v1/cv/{id}/chassis

GET  /api/v1/models/{id}/mc
GET  /api/v1/mc/{id}
GET  /api/v1/mc/{id}/type-numbers
GET  /api/v1/mc/{id}/criteria

GET  /api/v1/engines/{id}
GET  /api/v1/axles/{id}

GET  /api/v1/articles/search
GET  /api/v1/articles/{id}
GET  /api/v1/articles/{id}/applicability
GET  /api/v1/articles/{id}/cross-references
GET  /api/v1/articles/{id}/media
GET  /api/v1/articles/{id}/components
GET  /api/v1/articles/{id}/accessories
GET  /api/v1/articles/{id}/oem
GET  /api/v1/articles/{id}/coordinates
GET  /api/v1/articles/{id}/parts-diagram
GET  /api/v1/articles/{id}/criteria
GET  /api/v1/articles/{id}/specifications
GET  /api/v1/articles/{id}/classification
GET  /api/v1/articles/{id}/info
GET  /api/v1/articles/{id}/linkage-info
GET  /api/v1/articles/{id}/prices

GET  /api/v1/quick-start
GET  /api/v1/product-groups
GET  /api/v1/product-groups/search
GET  /api/v1/product-groups/{id}/children
GET  /api/v1/product-groups/{id}/articles
GET  /api/v1/product-types/{id}/criteria
GET  /api/v1/criteria/{id}/values

GET  /api/v1/search/kba
GET  /api/v1/search/kenteken
GET  /api/v1/search/type-number
GET  /api/v1/search/article
GET  /api/v1/search/article-exact
GET  /api/v1/search/oem
GET  /api/v1/search/analog
GET  /api/v1/search/oem-oem

POST /api/v1/enrichment/articles/batch

Stage 4 batch facts (explicit `include` values): `barcodes`, `trade_numbers`, `media`, `components`, `accessories`, `article_info`, `supersessions`, `country_metadata`. Extended sections are opt-in; omitting `include` preserves the legacy batch sections. Returned accessory rows include `ART_ACCS_LIST_CRITERIA`; `criteria_included=true` makes this explicit, while `truncated=true` requires paging `/articles/{id}/accessories` to complete the relation set.

GET  /api/v1/datasets
GET  /api/v1/datasets/{table}/schema
GET  /api/v1/datasets/{table}
GET  /api/v1/exports/cross-references
GET  /api/v1/exports/ean

GET  /health
```

See `API_REFERENCE.md` for every parameter and response example.

## Complete source-table access

List exposed TecDoc tables:

```bash
curl -H "X-API-Key: $API_KEY" \
  'http://localhost:8081/api/v1/datasets'
```

Read rows safely:

```bash
curl -G -H "X-API-Key: $API_KEY" \
  --data-urlencode 'select=ARL_ART_ID,ARL_DISPLAY_NR,ARL_TYPE' \
  --data-urlencode 'filter=ARL_TYPE:OENumber,IAMNumber' \
  --data-urlencode 'limit=100' \
  'http://localhost:8081/api/v1/datasets/ART_LOOKUP'
```

The dataset API never accepts arbitrary SQL. Table and column identifiers must exist in the active database schema, and `api_keys`, `api_key_usage_logs`, `pma__*` are denied.

## Full article technical data

For import/enrichment, prefer:

```bash
curl -H "X-API-Key: $API_KEY" \
  'http://localhost:8081/api/v1/articles/28/specifications?language_id=48&country_id=258&include_hidden=true'
```

Do not flatten all `source=linkage` values globally onto a product. They are applicability restrictions tied to a particular `LA_ID`.

Batch enrichment preserves technical-article provenance: an exact result can contain multiple `article_ids`, and merged classification/OEM/cross-reference/criteria/applicability records expose `source_article_ids`. Do not treat a merged multi-article applicability linkage as target-level provenance; page `/articles/{id}/applicability` per technical `ART_ID` when exact target attribution is required. `truncated=true` or `total_targets_exact=false` is never a complete applicability snapshot.

A caller promoting an immutable preview can reuse the complete reviewed technical identity by sending `article_ids` plus the asserted `article_number` and `supplier_id`. The batch handler revalidates every `ART_ID` against those assertions before loading facts, so this path avoids a second article-number lookup without trusting stale IDs blindly.

## Live smoke test

```bash
export API_KEY='<dedicated-test-key>'
export API='http://localhost:8081'
./test-api.sh
```

The script is the final live release gate: it executes **all 72 OpenAPI operations** and negative auth/validation cases. CV, motorcycle, engine and an unrestricted `ART_CTM=0` article fixture are resolved dynamically from the read-only dataset layer. The unrestricted article is verified through exact search and the commercial-identity batch resolver so country-filter regressions fail the release gate. `test-api.sh` contains no default credential: `API_KEY` must be supplied explicitly. The test key should allow at least ~120 requests/minute for a single full run.

## Deployment

Example systemd flow:

```bash
sudo cp go-tecdoc-api.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now go-tecdoc-api
sudo systemctl status go-tecdoc-api
sudo journalctl -u go-tecdoc-api -f
```

## Database/query notes

- The supplied `tecdoc2024q1.sql` is a structure dump, so local static verification cannot prove production row cardinality/query plans.
- Search/list endpoints use strict bounded pagination; general search max is `500`, heavy article relation pages max `500`, KeyValue dictionaries max `1000`, and raw dataset/export pages max `5000`.
- `limit=0`, values above an endpoint maximum, negative `offset`, malformed booleans/localization IDs, invalid `format` and invalid dataset `order` return HTTP `400`.
- Article detail embeds at most `related_limit` accessories (`50` default, `100` maximum); use `/articles/{id}/accessories` for full pagination. Accessory pages include lossless `criteria_details` plus the compatibility `criteria` map, and report `criteria_included=true`.
- Country-specific rows deterministically prefer the exact requested country over TecDoc universal `COU_ID=0`.
- TecDoc CTM country restrictions preserve unrestricted rows: `*_CTM=0` is accepted without a `COUNTRY_RESTRICTIONS` membership row; non-zero CTM values require explicit membership in the requested country.
- Every paged SQL list has a stable ID tie-breaker to prevent page drift between repeated requests.
- Large source exports are paged; max dataset/export page is `5000` rows.
- `include_total=true` on `/datasets/{table}` runs exact `COUNT(*)`; avoid it on every page of very large ETL jobs.
- The raw source layer is read-only by implementation; there is no arbitrary query endpoint and no write endpoint for TecDoc catalog tables.

## Project status

- API version: `1.2.6`
- TecDoc target: `2024Q1`
- Revision date: `2026-09-01`
- Baseline: `tecdoc2024q1-final`

### Trusted internal bulk importer key

`rate_limit_per_minute=0` means unlimited **for a trusted server-to-server key only**. Do not use an unlimited key in browser/client-side code. The `autoparts-platform` bulk importer is expected to use a private internal key and can be provisioned with:

```bash
go run ./cmd/apikey -name autoparts-platform-importer -rate-limit 0 -allowed-ips 127.0.0.1,::1
```

For an existing private importer key, set `api_keys.rate_limit_per_minute` to `0`. Public or third-party keys should retain a finite rate limit. HTTP 429 responses include `Retry-After` so well-behaved clients can resume instead of failing immediately.

Bulk-import preview uses `GET /api/v1/import/vehicle-stats?vehicle_types=...`, which aggregates source-table counts in one request instead of enumerating the entire catalog.

Bulk category synchronization uses `GET /api/v1/import/product-groups?vehicle_types=PC,CV,Motorcycle,Engine&limit=500&offset=0`, a flat paged SEARCH_TREE view that avoids recursive per-node HTTP fan-out.
