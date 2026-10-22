# Go TecDoc API — API Reference (TecDoc 2024Q1)

Production-oriented REST API for the `tecdoc2024q1` schema.

## Base URL and authentication

```text
http://localhost:8081/api/v1
```

Every `/api/v1/*` request requires:

```http
X-API-Key: <dedicated-test-key>
```

The test key can be registered without storing plaintext in MySQL:

```bash
go run ./cmd/apikey \
  -name tecdoc-test \
  -key '<dedicated-test-key>' \
  -rate-limit 1000
```

`/health` is intentionally outside API-key middleware.

## Common request context and strict validation

Most localized endpoints accept:

- `language_id` — positive TecDoc language (`LANGUAGES.LNG_ID`), default from `DEFAULT_LANGUAGE_ID`, project default `48`.
- `country_id` — positive TecDoc country (`COUNTRIES.COU_ID`), default from `DEFAULT_COUNTRY_ID`, project default `258`.
- `limit`, `offset` — page size and zero-based offset where supported.

Production hardening rules:

- malformed `language_id` / `country_id`, zero/negative IDs, invalid booleans, invalid `format`, invalid `order`, `limit=0`, limits above the endpoint maximum, and negative `offset` return **HTTP 400** instead of being silently coerced;
- authenticated routes can return **401** (missing/invalid key), **403** (inactive/expired key or IP/route restriction), and **429** (rate limit);
- all SQL-backed paged lists have deterministic ordering with stable numeric-ID tie-breakers; raw datasets append composite primary-key columns as tie-breakers, and the three TecDoc 2024Q1 tables without a primary key fall back to the remaining schema columns;
- where TecDoc contains both an exact country-specific row and `COU_ID=0`, the exact requested country is selected first and universal `0` is only fallback;
- for TecDoc CTM restriction fields, `*_CTM=0` means unrestricted and is valid for every requested country; non-zero CTM values require a matching `COUNTRY_RESTRICTIONS` membership row.

Default/max page sizes for the heaviest contracts:

| Contract | Default | Maximum |
|---|---:|---:|
| article/search/type-number/OEM/analog searches | 100 | 500 |
| article applicability/accessories/specifications/linkage-info/parts-diagram | 100 | 500 |
| criterion KeyValues | 100 | 1000 |
| raw datasets | 100 | 5000 |
| CROSS / EAN exports | 1000 | 5000 |

Paged normalized responses expose `total`, `limit`, `offset`, `returned`, and `has_more` where an exact total is available.

Example:

```bash
curl -sS \
  -H 'X-API-Key: <dedicated-test-key>' \
  'http://localhost:8081/api/v1/search/article-exact?number=0001106017&language_id=48&country_id=258'
```

Representative response:

```json
{
  "results": [
    {
      "article_id": 29,
      "article_nr": "0 001 106 017",
      "brand": "BOSCH",
      "supplier_id": 30,
      "name": "Starter"
    }
  ],
  "total": 1
}
```

> Response values above are examples of shape, not a promise that the same row exists in every TecDoc import.

## Endpoint catalog

### Discovery and localization

| Method | Endpoint | Purpose | Main response |
|---|---|---|---|
| GET | `/api/v1` | API discovery | `name`, `version`, `database`, `capabilities` |
| GET | `/api/v1/languages` | Languages | `languages[]` |
| GET | `/api/v1/languages/{id}` | One language | `language` |
| GET | `/api/v1/countries` | Countries | `countries[]` |
| GET | `/api/v1/countries/{id}` | One country | `country` |
| GET | `/health` | Database liveness | `status`, `database` |

Example:

```bash
curl -H 'X-API-Key: <dedicated-test-key>' \
  'http://localhost:8081/api/v1/languages'
```

```json
{"languages":[{"id":48,"code":"UA","name":"Ukrainian"}]}
```

### Suppliers and manufacturers

| Method | Endpoint | Query | Main response |
|---|---|---|---|
| GET | `/suppliers` | `page`, `limit`, optional `brand` | `suppliers[]`, paging |
| GET | `/suppliers/{id}` | — | `supplier` |
| GET | `/suppliers/{id}/products` | `limit`, `offset`, localization | product types supplied by supplier |
| GET | `/suppliers/{id}/addresses` | `language_id` | `addresses[]` |
| GET | `/suppliers/{id}/logos` | — | `logos[]` |
| GET | `/manufacturers` | `vehicle_type`, `limit`, `offset` | `manufacturers[]` |
| GET | `/manufacturers/{id}` | — | `manufacturer` |
| GET | `/manufacturers/{id}/models` | `vehicle_type`, localization, paging | `models[]` |

Address example:

```bash
curl -H 'X-API-Key: <dedicated-test-key>' \
  'http://localhost:8081/api/v1/suppliers/30/addresses?language_id=48'
```

```json
{
  "addresses": [
    {
      "supplier_id": 30,
      "address_type": "Head office",
      "country_id": 62,
      "country": "Germany",
      "city1": "...",
      "homepage": "..."
    }
  ],
  "total": 1
}
```

### Model series and vehicles

| Method | Endpoint | Query | Main response |
|---|---|---|---|
| GET | `/models/{id}` | localization | `model` |
| GET | `/models/{id}/cars` | `vehicle_type=PC|LCV|E-PC|E-LCV`, `limit`, `offset`, localization | `vehicles[]`; query matches `PC_TYPE` SET membership and each row preserves its full membership value (for example `PC,LCV`) |
| GET | `/cars/{id}` | localization | `vehicle` |
| GET | `/cars/{id}/product-groups` | `vehicle_type=PC|LCV|E-PC|E-LCV`, paging | `categories[]`; passenger subtypes resolve to TecDoc target family `PC` |
| GET | `/cars/{id}/type-numbers` | — | `matches[]` from `TYPE_NUMBER_LOOKUP` |
| GET | `/cars/{id}/criteria` | `product_type_id`, `supplier_id`, `suggested_only` | aggregate valid criteria |
| GET | `/models/{id}/cv` | paging, localization | `vehicles[]` |
| GET | `/cv/{id}` | localization | commercial vehicle detail |
| GET | `/cv/{id}/type-numbers` | — | type/manufacturer numbers |
| GET | `/cv/{id}/criteria` | same filters as car criteria | valid criteria |
| GET | `/cv/{id}/chassis` | `language_id` | wheelbases, suspensions, cabs, axles |
| GET | `/models/{id}/mc` | paging, localization | motorcycles |
| GET | `/mc/{id}` | localization | motorcycle detail |
| GET | `/mc/{id}/type-numbers` | — | motorcycle type numbers |
| GET | `/mc/{id}/criteria` | criteria filters | valid criteria |
| GET | `/engines` | `manufacturer_id`, `limit`, `offset`, localization | paged global engine master |
| GET | `/engines/{id}` | localization | engine detail |
| GET | `/axles/{id}` | `language_id` | axle + models + brake sizes |

Commercial chassis example:

```bash
curl -H 'X-API-Key: <dedicated-test-key>' \
  'http://localhost:8081/api/v1/cv/17582/chassis?language_id=48'
```

```json
{
  "vehicle_id": 17582,
  "wheelbases": [{"axle_group":"...","wheelbase_mm":3600}],
  "suspensions": [{"axle_position":"1","description":"..."}],
  "driver_cabs": [{"id":1,"name":"..."}],
  "axles": [{"id":12,"code":"...","position":"1"}]
}
```

Axle example:

```bash
curl -H 'X-API-Key: <dedicated-test-key>' \
  'http://localhost:8081/api/v1/axles/12?language_id=48'
```

```json
{
  "axle": {
    "id": 12,
    "manufacturer_id": 1,
    "code": "...",
    "models": [{"country_id":62,"sort":1,"model":"..."}],
    "brake_sizes": [{"sequence":1,"size":"...","description":"..."}]
  }
}
```

### Product groups, Quick Start and criteria dictionaries

| Method | Endpoint | Query | Main response |
|---|---|---|---|
| GET | `/quick-start` | optional `vehicle_type` | `items[]` (`QUICK_START` + `LINK_QS_ST`) |
| GET | `/product-groups` | `vehicle_type`, paging, localization | root/tree categories |
| GET | `/product-groups/search` | required `q`, `vehicle_type`, paging | localized search-tree paths |
| GET | `/product-groups/{id}/children` | `vehicle_type`, localization | children |
| GET | `/product-groups/{id}/articles` | `car_id`, `vehicle_type`, `criteria`, paging | articles in node |
| GET | `/product-types/{id}/criteria` | `suggested_only`, `language_id` | allowable criteria (`LINK_PT_CRI`) |
| GET | `/criteria/{id}/values` | `language_id` | KeyValue dictionary (`KEY_VALUES`) |

Product-tree target normalization: `PC`, `LCV`, `E-PC` and `E-LCV` resolve to the TecDoc `PC` search-tree family. `CV` and `Motorcycle` remain separate. Generic tree navigation additionally accepts `Engine`, `Axle` and `Universal Parts`. `SEARCH_TREE.STR_TYPE` / `LINK_PT_STR.STR_TYPE` are treated as multi-value memberships, not as strings that must be exactly equal.

For bulk vehicle synchronization, `GET /api/v1/import/vehicle-product-groups?...&ids_only=true` returns only `category_ids` per vehicle. This path avoids text translation and uses request-context cancellation; it is the recommended production mode for Full Import. The default `ids_only=false` response remains backward compatible and returns category metadata.

Search tree by description:

```bash
curl -H 'X-API-Key: <dedicated-test-key>' \
  'http://localhost:8081/api/v1/product-groups/search?q=VALVE&vehicle_type=PC&language_id=4&limit=50'
```

```json
{
  "results": [
    {
      "category_id": 100470,
      "parent_id": 100400,
      "level": 3,
      "name": "Valve",
      "path": "Engine > ... > Valve",
      "vehicle_type": "PC",
      "product_type_id": 123,
      "product_type": "..."
    }
  ],
  "total": 1
}
```

Criteria dictionary:

```bash
curl -H 'X-API-Key: <dedicated-test-key>' \
  'http://localhost:8081/api/v1/criteria/123/values?language_id=48'
```

```json
{
  "criteria_id": 123,
  "values": [
    {"criteria_id":123,"key_table_id":7,"value":"1","name":"...","description":"..."}
  ],
  "total": 1
}
```

### Articles

| Method | Endpoint | Purpose |
|---|---|---|
| GET | `/articles/search?number=...` | broad number lookup |
| GET | `/articles/{id}` | article detail; includes lossless `criteria_details`; `related_limit=1..100` bounds embedded accessories |
| GET | `/articles/{id}/applicability` | paged fitment (`limit<=500`) + normalized linkages/restrictions |
| GET | `/articles/{id}/cross-references` | IAM/OE/article cross refs |
| GET | `/articles/{id}/media` | country-aware images, PDF, URL and other media (`language_id`, `country_id`) |
| GET | `/articles/{id}/components` | parts list (`ARTICLES_PART_LIST`) |
| GET | `/articles/{id}/accessories` | paged accessories (`limit<=500`) + `ART_ACCS_LIST_CRITERIA` |
| GET | `/articles/{id}/oem` | OEM numbers |
| GET | `/articles/{id}/coordinates` | raw own sensitive coordinates |
| GET | `/articles/{id}/parts-diagram` | paged component-to-image mapping (`LINK_PL_SC`) |
| GET | `/articles/{id}/criteria` | article-only `ARTICLE_CRITERIA`, storefront-compatible |
| GET | `/articles/{id}/specifications` | **complete**, paged article + linkage criteria |
| GET | `/articles/{id}/classification` | Product Type + Search Tree classification |
| GET | `/articles/{id}/info` | structured country-aware `ARTICLE_INFO` text modules (`country_id`, `language_id`, `include_hidden`) |
| GET | `/articles/{id}/linkage-info` | paged `LA_INFO` text modules by LA_ID |
| GET | `/articles/{id}/prices` | TecDoc prices, country-aware |

Complete specifications example (recommended when importing all technical data):

```bash
curl -H 'X-API-Key: <dedicated-test-key>' \
  'http://localhost:8081/api/v1/articles/28/specifications?language_id=4&country_id=62&include_hidden=true&limit=100&offset=0'
```

```json
{
  "article_id": 28,
  "specifications": [
    {
      "article_id": 28,
      "source": "article",
      "criteria_id": 6,
      "name": "Voltage [V]",
      "type": "Numerical",
      "raw_value": "12",
      "display": true
    },
    {
      "article_id": 28,
      "source": "linkage",
      "linkage_id": 123456,
      "product_type_id": 1,
      "criteria_id": 92,
      "name": "Number of Teeth",
      "type": "Numerical",
      "raw_value": "9"
    }
  ],
  "total": 2,
  "limit": 100,
  "offset": 0,
  "returned": 2,
  "has_more": false
}
```

Why there are two sources:

- `source=article`: characteristic belongs to the article itself (`ARTICLE_CRITERIA`).
- `source=linkage`: condition/restriction belongs to one applicability link (`LA_CRITERIA`) and must **not** be flattened globally into the product.

Accessories example:

```bash
curl -H 'X-API-Key: <dedicated-test-key>' \
  'http://localhost:8081/api/v1/articles/1307505/accessories?language_id=4&country_id=62&limit=25&offset=0'
```

```json
{
  "accessories": [
    {
      "article_id": 123,
      "article_nr": "...",
      "brand": "...",
      "name": "...",
      "group_name": "...",
      "relation_type": "PC",
      "relation_type_id": 9877,
      "quantity": 1,
      "product_type_id": 123,
      "criteria": {"...":"..."},
      "criteria_details": [{"criteria_id":1,"name":"...","raw_value":"...","display":true}]
    }
  ],
  "total": 694,
  "limit": 25,
  "offset": 0,
  "returned": 25,
  "has_more": true
}
```

`criteria_details` is the **lossless** representation: the same criterion may appear more than once for one relation. `criteria` is retained for compatibility and deterministically aggregates repeated display values with `; `. Importers that need stable IDs/raw values must still use `criteria_details`.

Parts diagram example:

```bash
curl -H 'X-API-Key: <dedicated-test-key>' \
  'http://localhost:8081/api/v1/articles/124926/parts-diagram?limit=100&offset=0'
```

```json
{
  "article_id": 124926,
  "components": [
    {
      "parent_article_id":124926,
      "component_article_id":125000,
      "article_number":"...",
      "brand":"ELRING",
      "media_id":1,
      "media_type":"JPEG",
      "media_source":"IMAGE/123/file.jpg",
      "coordinate_id":1,
      "x":100,
      "y":120,
      "width":40,
      "height":40,
      "type":"Square"
    }
  ],
  "total":1,
  "limit":100,
  "offset":0,
  "returned":1,
  "has_more":false
}
```

Article info:

```bash
curl -H 'X-API-Key: <dedicated-test-key>' \
  'http://localhost:8081/api/v1/articles/28/info?language_id=48&include_hidden=true'
```

```json
{"items":[{"type":"Information","text":"...","sort":1,"display":true,"text_id":123}],"total":1}
```

Prices:

```bash
curl -H 'X-API-Key: <dedicated-test-key>' \
  'http://localhost:8081/api/v1/articles/28/prices?country_id=258&language_id=48&include_expired=false'
```

```json
{
  "prices":[{"article_id":28,"country_id":258,"type":"...","price":123.45,"currency":"EUR","valid_from":"2024-01-01"}],
  "total":1
}
```

### Number search

| Method | Endpoint | Required / optional parameters |
|---|---|---|
| GET | `/search/kba` | `number`; optional `vehicle_type`, paging |
| GET | `/search/kenteken` | `number`; optional `vehicle_type`, paging |
| GET | `/search/type-number` | `number`; optional exact `number_type`, `vehicle_type`, paging |
| GET | `/search/article` | `number`, localization, paging |
| GET | `/search/article-exact` | `number`, localization, paging |
| GET | `/search/oem` | `number`, localization, paging |
| GET | `/search/analog` | preferred `art_id`; or `search_number` + `supplier_id` |
| GET | `/search/oem-oem` | `oem_number`, paging |

Supported `TYPE_NUMBER_LOOKUP.LTN_NUMBER_TYPE` values in this 2024Q1 schema:

- `CV Manufacturers Number`
- `German KBA Number`
- `French TypeMine`
- `Netherlands Kenteken`
- `Swiss Typenumber`
- `Engine Code`

KBA example:

```bash
curl -H 'X-API-Key: <dedicated-test-key>' \
  'http://localhost:8081/api/v1/search/kba?number=0588549&vehicle_type=PC'
```

```json
{
  "matches": [
    {
      "vehicle_id": 1014,
      "vehicle_type": "PC",
      "number": "0588549",
      "number_type": "German KBA Number",
      "manufacturer_id": 35
    }
  ],
  "total": 1
}
```

Kenteken example:

```bash
curl -H 'X-API-Key: <dedicated-test-key>' \
  'http://localhost:8081/api/v1/search/kenteken?number=2VXT04&vehicle_type=PC'
```

Generic type-number example:

```bash
curl -H 'X-API-Key: <dedicated-test-key>' \
  'http://localhost:8081/api/v1/search/type-number?number=0588549&number_type=German%20KBA%20Number&vehicle_type=PC'
```

### Server-to-server batch enrichment

```http
POST /api/v1/enrichment/articles/batch
Content-Type: application/json
X-API-Key: <dedicated-test-key>
```

```json
{
  "language_id":48,
  "country_id":258,
  "include":["classification","oem","cross_references","criteria","applicability","barcodes","trade_numbers","media","components","accessories","article_info","supersessions","country_metadata"],
  "max_applicability_targets":2000,
  "max_accessories":250,
  "items":[
    {"request_id":"product-1","article_number":"50390266","supplier_id":72}
  ]
}
```

For a previously reviewed immutable commercial identity, callers may bypass a second article-number resolution while still fencing against stale identity by supplying the complete reviewed technical `article_ids` together with the asserted `article_number` and positive `supplier_id`:

```json
{
  "language_id":48,
  "country_id":258,
  "include":["classification","oem","criteria","applicability"],
  "items":[
    {
      "request_id":"product-1-promoted-preview",
      "article_ids":[123,456],
      "article_number":"50390266",
      "supplier_id":72
    }
  ]
}
```

`article_ids` is a trusted server-to-server reuse contract, not an unchecked direct-ID shortcut. The list must contain unique positive IDs (maximum 200 per item), cannot be combined with `article_id`, and requires both commercial-identity assertions. The API reloads every `ART_ID` and returns `error` if any ID now belongs to another supplier/article identity or `not_found` if an asserted ID no longer exists. This keeps preview promotion free of a second article-number identity lookup without allowing a stale snapshot to bind unrelated technical articles.

Representative response:

```json
{
  "language_id":48,
  "country_id":258,
  "results":[
    {
      "request_id":"product-1",
      "status":"exact",
      "resolved":{"article_id":123,"article_number":"50390266","supplier_id":72,"brand":"...","name":"..."},
      "article_ids":[123,456],
      "classification":[{"source_article_ids":[123,456],"product_type_id":1,"name":"...","scopes":["PC"],"tree_nodes":[],"linkage_count":2,"target_count":15}],
      "oem_numbers":[{"brand":"...","number":"...","source_article_ids":[123]}],
      "cross_references":[],
      "criteria":[{"criteria_id":209,"name":"Height","display":true,"raw_value":"79","value_description_id":0,"source_article_ids":[123]}],
      "applicability":{"linkages":[{"source_article_ids":[123],"la_id":10,"pt_id":1,"target_type":"PC","targets":[],"restrictions":[]}],"total_targets":0,"returned_targets":0,"truncated":false,"total_targets_exact":true}
    }
  ],
  "total":1
}
```

`criteria[].value_description_id` exposes the stable TecDoc `ARTICLE_CRITERIA.ACR_DES_ID` when present. Use it for source identity when `raw_value` is empty; `value_description` is localized display text and must not be the primary cross-language identity.

`display=false` is **not hidden data** in TecDoc. `ARTICLE_CRITERIA.ACR_DISPLAY=0` and `ARTICLE_INFO.AIN_DISPLAY=0` mean “show only on the detailed/second article page”. The enrichment batch therefore returns both display variants. Consumers may still use the flag to decide whether a fact belongs in compact listings or only in a detailed product view.

`article_ids` lists every technical TecDoc `ART_ID` contributing to the resolved commercial identity. Merged classification, OEM, cross-reference, criteria and applicability records expose `source_article_ids` so downstream enrichment can retain provenance instead of attributing data to an arbitrary primary `ART_ID`. For complete fitment, consumers must page `/articles/{id}/applicability` for each technical `ART_ID` whenever target-level provenance is required; `total_targets_exact=false` or `truncated=true` means the embedded batch page is not a complete snapshot.

Stage-4 extended sections are opt-in to preserve the legacy batch performance contract. When `article_info` is requested, the batch returns both first-page and detail-page-only `ARTICLE_INFO` rows. Supported values are `barcodes`, `trade_numbers`, `media`, `components`, `accessories`, `article_info`, `supersessions` and `country_metadata`. `accessories` is bounded per technical article by `max_accessories` (default 100, maximum 500); every returned relation is hydrated from `ART_ACCS_LIST_CRITERIA`, and the page exposes `total`, `returned`, `truncated` and `criteria_included=true`. If `truncated=true`, page `/articles/{id}/accessories` for every contributing technical `ART_ID` until `has_more=false`; the dedicated endpoint also returns `criteria_included=true`. `media` exposes TecDoc media identity separately from its relative IMAGE/PDF path or external URL. For TecDoc 2024Q1 it is filtered by `ART_MEDIA_CTM` for the requested `country_id`; the response also carries `country_marker`, `language_id` and `content_type` so applicable provider variants are not silently collapsed. Filename/hyperlink remain mutable payload, not durable media identity. `country_specifics` is returned as a list because multiple technical `ART_ID`s of one commercial identity can disagree and callers must not silently merge conflicting packaging/status facts.

### Complete read-only source-table access

These endpoints guarantee that every TecDoc source table remains retrievable even when no purpose-built domain endpoint is needed.

| Method | Endpoint | Purpose |
|---|---|---|
| GET | `/datasets` | all exposed TecDoc base tables |
| GET | `/datasets/{table}/schema` | exact columns/types/keys from `INFORMATION_SCHEMA` |
| GET | `/datasets/{table}` | read rows with safe column selection/filter/order/paging |

Internal tables `api_keys`, `api_key_usage_logs` and `pma__*` are deliberately excluded.

List datasets:

```bash
curl -H 'X-API-Key: <dedicated-test-key>' \
  'http://localhost:8081/api/v1/datasets'
```

```json
{
  "datasets":[
    {"name":"ARTICLES","estimated_rows":1234567},
    {"name":"TYPE_NUMBER_LOOKUP","estimated_rows":123456}
  ],
  "total":56
}
```

Table schema:

```bash
curl -H 'X-API-Key: <dedicated-test-key>' \
  'http://localhost:8081/api/v1/datasets/ART_LOOKUP/schema'
```

```json
{
  "table":"ART_LOOKUP",
  "columns":[
    {"name":"ARL_ART_ID","data_type":"int","column_type":"int(10) unsigned","nullable":false,"key":"...","ordinal":1}
  ]
}
```

Read selected rows:

```bash
curl -G \
  -H 'X-API-Key: <dedicated-test-key>' \
  --data-urlencode 'select=ARL_ART_ID,ARL_DISPLAY_NR,ARL_BRA_BRAND,ARL_TYPE' \
  --data-urlencode 'filter=ARL_TYPE:OENumber,IAMNumber' \
  --data-urlencode 'order_by=ARL_ART_ID' \
  --data-urlencode 'order=ASC' \
  --data-urlencode 'limit=100' \
  --data-urlencode 'offset=0' \
  --data-urlencode 'include_total=true' \
  'http://localhost:8081/api/v1/datasets/ART_LOOKUP'
```

```json
{
  "table":"ART_LOOKUP",
  "rows":[{"ARL_ART_ID":28,"ARL_DISPLAY_NR":"...","ARL_BRA_BRAND":"...","ARL_TYPE":"OENumber"}],
  "limit":100,
  "offset":0,
  "total":123456
}
```

CSV uses the same safe query contract:

```bash
curl -o art_lookup.csv -G \
  -H 'X-API-Key: <dedicated-test-key>' \
  --data-urlencode 'filter=ARL_TYPE:EAN' \
  --data-urlencode 'format=csv' \
  --data-urlencode 'limit=5000' \
  'http://localhost:8081/api/v1/datasets/ART_LOOKUP'
```

Rules:

- `select=COLUMN1,COLUMN2`
- repeat `filter=COLUMN:value1,value2` for AND between different columns; values within one filter are `IN (...)`
- `order_by=COLUMN`, `order=ASC|DESC`
- `limit` maximum `5000`
- `offset>=0`
- `include_total=true` performs an exact `COUNT(*)`, which can be expensive on huge tables
- `format=json|csv`
- table and column names are accepted only after exact validation against `INFORMATION_SCHEMA`

### Paged global exports

The canonical TecDoc examples use `INTO OUTFILE`, which requires database-server filesystem privileges. The API exposes the same data safely over HTTP in pages. Export paging is executed against `ART_LOOKUP` before joining `ARTICLES`. CROSS uses deterministic primary-key order; EAN uses the existing `ARL_TYPE_INDX` and its InnoDB extended-key order so the database does not scan the full primary index to find the much rarer `EAN` rows. Very large offsets still require skipped-row work, so consumers should process pages sequentially and avoid unnecessarily large jumps.

Cross references:

```bash
curl -H 'X-API-Key: <dedicated-test-key>' \
  'http://localhost:8081/api/v1/exports/cross-references?limit=1000&offset=0&format=json'
```

```json
{"rows":[{"article_number":"...","article_brand":"...","cross_number":"...","cross_brand":"...","cross_type":"OENumber"}],"limit":1000,"offset":0}
```

CSV:

```bash
curl -o crosses.csv \
  -H 'X-API-Key: <dedicated-test-key>' \
  'http://localhost:8081/api/v1/exports/cross-references?limit=5000&offset=0&format=csv'
```

EAN/GTIN:

```bash
curl -H 'X-API-Key: <dedicated-test-key>' \
  'http://localhost:8081/api/v1/exports/ean?limit=1000&offset=0'
```

```json
{"rows":[{"article_number":"...","article_brand":"...","ean":"4011558..."}],"limit":1000,"offset":0}
```

## Error contract

Normal handler errors:

```json
{"error":"Invalid article ID"}
```

API-key middleware errors:

```json
{"error":"invalid_api_key","message":"API key is invalid"}
```

Relevant status codes: `400`, `401`, `403`, `404`, `422`, `429`, `500`, `503`, `504`. Export queries return `504` when their bounded database context expires. A request rejected by strict validation never reaches the expensive TecDoc SQL query.

## Final contract verification

`go test ./...` includes a route/OpenAPI contract test that requires the production mux registry and `OPENAPI.yaml` to match exactly and asserts **72 operations**. `test-api.sh` then executes all 72 operations against the live database, plus negative auth/validation cases. The script dynamically resolves CV, motorcycle, engine, and an unrestricted `ART_CTM=0` article fixture from the read-only dataset layer so it is less brittle across imports. It verifies that the unrestricted article survives exact search and exact commercial-identity batch resolution. `API_KEY` is mandatory and is never embedded as a default in the script.

```bash
API_KEY="$API_KEY" API=http://localhost:8081 ./test-api.sh
```

A successful release ends with:

```text
Unique OpenAPI operationIds exercised live: 72/72
Negative/auth/validation checks: PASS
All 72 OpenAPI operations completed successfully.
FINAL PRODUCTION CONTRACT TEST: PASS
```

## Importing the full database

For large source tables, do not request an unbounded response. Iterate `offset` in steps up to `limit=5000`. For restartable ETL, persist the last page/primary key in your importer. `SCHEMA_COVERAGE.md` maps all 56 TecDoc tables to domain/raw endpoints.
