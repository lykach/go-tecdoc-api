# go-tecdoc-api — TecDoc 2024Q1 Production Baseline

Baseline: `tecdoc2024q1-final`
API documentation version: `1.2.6`
Date: `2026-09-01`

Patch `1.2.6` keeps the public API shape from `1.2.5` and hardens the production release process. `test-api.sh` no longer contains a fallback credential: `API_KEY` must be supplied explicitly. The live gate now discovers a real unrestricted `ART_CTM=0` source row and verifies that exact search and the commercial-identity batch resolver preserve it. Runtime article/enrichment behavior remains backward compatible with `1.2.5`.

## Release guarantees

This baseline is read-only for TecDoc catalog data and is built against the supplied `tecdoc2024q1` schema.

- Production mux registry and `OPENAPI.yaml` are mechanically checked for an exact **72-operation** match.
- All 56 TecDoc source tables remain available through the validated read-only `/api/v1/datasets/*` layer.
- General search requests are bounded to `1..500` rows.
- Heavy article relations (`applicability`, `accessories`, `specifications`, `linkage-info`, `parts-diagram`) are SQL-paged, default `100`, maximum `500`.
- Criterion KeyValue pages are maximum `1000`; raw dataset and CROSS/EAN export pages are maximum `5000`.
- Article detail embeds only `related_limit` accessories (`50` default, `100` maximum).
- Invalid pagination, localization IDs, boolean values, export formats and dataset sort directions fail with HTTP `400` before expensive SQL work.
- Exact requested country-specific data is preferred over TecDoc universal `COU_ID=0` fallback.
- TecDoc country restriction semantics are consistent across article search, exact identity validation, IAM traversal and enrichment: `*_CTM=0` is unrestricted; non-zero CTM requires membership in `COUNTRY_RESTRICTIONS` for the requested country.
- Paged list queries use deterministic `ORDER BY` plus stable ID tie-breakers; raw datasets append composite PK columns and use full-row schema tie-breakers for TecDoc tables without a PK.
- CROSS and EAN exports retain the live-tested optimized index plans introduced before this hardening baseline.
- `criteria_details` is the lossless accessory criteria representation. The legacy `criteria` map is deterministic but cannot represent repeated values for the same criterion name.

## Pre-deployment gates

Run from the project root:

```bash
gofmt -w cmd internal
go test ./...
go vet ./...
go build ./...
bash -n test-api.sh
```

`go test ./...` includes `TestOpenAPIExactlyMatchesProductionRouteRegistry`, which fails if a production route is missing from OpenAPI, a stale OpenAPI operation has no route, or the documented operation count is not exactly 72.

## Production build and restart

`go build ./...` is a compile gate only. The systemd unit runs `bin/server`, so deploy the actual binary explicitly:

```bash
cd /var/www/go-tecdoc-api
cp -a bin/server "bin/server.backup-$(date +%Y%m%d-%H%M%S)"
go build -trimpath -o bin/server ./cmd/server
systemctl restart go-tecdoc-api
systemctl --no-pager --full status go-tecdoc-api
journalctl -u go-tecdoc-api -n 100 --no-pager
```

## Final live release gate

Use a dedicated test key with enough quota for at least 120 requests/minute. The script has no embedded fallback key:

```bash
export API=http://localhost:8081
export API_KEY='<dedicated-test-key>'
./test-api.sh
```

The script performs:

- live missing-key and invalid-key checks (`401`);
- strict validation checks (`400`), unknown dataset (`404`), and unsafe analog contract rejection;
- dynamic fixture discovery for CV, motorcycle and engine IDs;
- dynamic unrestricted `ART_CTM=0` discovery plus exact-search and batch-identity regression assertions;
- one successful live request for **every OpenAPI operation**;
- final unique-operation assertion `72/72`.

Successful completion must end with:

```text
Unique OpenAPI operationIds exercised live: 72/72
Negative/auth/validation checks: PASS
All 72 OpenAPI operations completed successfully.
FINAL PRODUCTION CONTRACT TEST: PASS
```

Live `403` and `429` are not forced with the normal smoke key because that would require deliberately restricting the key or exhausting its quota. Their route/IP/rate-limiter mechanics are covered by `internal/auth` tests; use a dedicated restricted key if an environment-specific end-to-end check is required.

## Rollback

If the service fails after deployment:

```bash
systemctl stop go-tecdoc-api
cp -a <known-good-backup> bin/server
systemctl start go-tecdoc-api
systemctl --no-pager --full status go-tecdoc-api
```

Do not roll back the TecDoc database schema as part of this code baseline: this release does not require a new TecDoc catalog migration.

## SEARCH_TREE target-family invariant (2026-08-28)

TecDoc `SEARCH_TREE.STR_TYPE` and `LINK_PT_STR.STR_TYPE` are multi-value target-family fields. Category/product-group SQL must test membership with `FIND_IN_SET(...)`; it must not require the two complete SET strings to be textually equal.

Vehicle-family normalization for product-group endpoints is:

- `PC`, `LCV`, `E-PC`, `E-LCV` -> `PC`
- `CV` -> `CV`
- `Motorcycle` -> `Motorcycle`

Generic search-tree endpoints may additionally expose `Engine`, `Axle` and `Universal Parts`. Vehicle-specific product-group applicability remains limited to `PC`, `CV` and `Motorcycle` target families.

## Vehicle full-import product-group performance invariant (2026-08-28)

Server-to-server vehicle Full Import must call `/api/v1/import/vehicle-product-groups` with `ids_only=true`. The compact mode returns only canonical root `SEARCH_TREE` IDs per vehicle and must not perform description translation inside the applicability join.

The database query for this endpoint must be executed with the incoming HTTP request context (`QueryContext`). If the caller times out or cancels the request, the in-flight MySQL query must receive cancellation instead of continuing as orphaned work.

The default `ids_only=false` response remains available for interactive/API consumers that require translated category metadata. Metadata lookup is performed only after the applicable unique root IDs have been resolved.

The production contract test includes the nine-PC-vehicle batch that previously exceeded the 30-second client timeout; keep that fixture as a regression gate for future SQL changes.
