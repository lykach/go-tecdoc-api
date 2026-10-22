# Review of supplied TecDoc 2024Q1 SQL examples

Basis:

- supplied `SQL examples 2024Q1` Markdown;
- supplied `tecdoc2024q1.sql` structure dump;
- current API SQL after this revision.

## Structural validation

Automated static validation extracted explicit `TABLE.COLUMN` references from the supplied examples and checked them against the supplied structure dump:

- explicit references checked: 569 occurrences / 187 unique pairs;
- unknown real TecDoc `TABLE.COLUMN` references: **0**;
- apparent unknown prefixes `ART_LOOKUP_CROSS` and `TEXT_MODULE_TEXTS_UNI` are valid SQL aliases;
- recursive CTE names are not physical tables.

This confirms that the examples target the supplied 2024Q1 schema at the column-name level. It does **not** mean every example is ideal production SQL.

## Production corrections implemented

### 1. `SUPERSEDED` alias shadowing

The supplied article-detail example has an inner `JOIN ARTICLES` while the outer query also uses `ARTICLES`. In the second supersession subquery this shadows the outer table and makes the intended correlation ambiguous/wrong.

API fix:

- inner old article is explicitly aliased as `OLD_ARTICLES`;
- `SUPERSEDED_ARTICLES.SUA_NEW_ART_ID = ARTICLES.ART_ID` now references the outer current article;
- country restriction supports both `SUA_CTM=0` and explicit country membership.

### 2. Country restrictions

Several examples use an unconditional `INNER JOIN COUNTRY_RESTRICTIONS`. In TecDoc, a CTM value of `0` is commonly the unrestricted case. Production queries therefore use the safer pattern:

```sql
CTM = 0 OR EXISTS (
  SELECT 1
  FROM COUNTRY_RESTRICTIONS
  WHERE CNTR_CTM_ID = CTM
    AND CNTR_COU_ID = :country_id
)
```

This pattern is used consistently for article existence/search/exact resolution, IAM cross-reference targets, article criteria, linkage criteria/info, supersessions and other CTM-scoped relations. This matters because an unconditional `INNER JOIN COUNTRY_RESTRICTIONS` can silently drop valid unrestricted `CTM=0` rows.

### 3. Full article specifications

The supplied “all specifications” example correctly demonstrates that full technical data is the union of:

- `ARTICLE_CRITERIA` — article-global characteristics;
- `LA_CRITERIA` via `LINK_ART` — applicability/linkage restrictions.

The API no longer forces these into one lossy flat map. Use:

```text
GET /api/v1/articles/{id}/specifications
```

Each row contains `source=article|linkage`; linkage rows also contain `linkage_id` and `product_type_id`.

### 4. Accessory criteria

The supplied accessory example joins `ART_ACCS_LIST_CRITERIA`; the previous API had a TODO and returned an empty criteria map. It now returns both:

- backward-compatible flattened `criteria`;
- structured `criteria_details`.

### 5. Parts-list image coordinates

The supplied coordinate example uses `LINK_PL_SC` to map a component article to a coordinate on a parent article image. The historical `/coordinates` endpoint only exposed sensitive coordinates of the parent.

The exact relation is now available as:

```text
GET /api/v1/articles/{id}/parts-diagram
```

The legacy `/coordinates` endpoint remains available and its media join is tightened by article ID + media ID + media type.

### 6. Search-tree text search

The supplied recursive search-tree example contains a hard-coded English language (`get_text(..., 4)`) inside one recursive branch while using `@LNGID` elsewhere.

The API consistently uses the requested `language_id` and fallback text resolution.

### 7. KBA and Kenteken

The examples use `TYPE_NUMBER_LOOKUP`. The old route `/search/kba` existed but returned HTTP 501. It is now implemented, and generic type-number search exposes all 2024Q1 number types.

### 8. Global CROSS/EAN export

The supplied SQL uses `INTO OUTFILE`, which requires database-server filesystem access and FILE privilege. HTTP API endpoints are safer for application integration:

```text
GET /api/v1/exports/cross-references?format=csv
GET /api/v1/exports/ean?format=csv
```

Both are paged (max 5000 rows/request), so large exports do not require one huge DB result in memory.

### 9. Raw source access

To avoid creating dozens of brittle one-off wrappers for every source table, `/datasets/*` exposes every TecDoc base table read-only. Table/column identifiers are validated against `INFORMATION_SCHEMA`; auth/phpMyAdmin tables are explicitly denied.

## What cannot be proven from a structure-only dump

The supplied `.sql` file contains schema definitions but no TecDoc row data. Therefore static review can prove table/column compatibility and Go compilation, but not actual row counts, sample IDs, query plans, cardinality, collation edge cases or exact response payloads from the production database.

Run `./test-api.sh` on the server after deployment for live-data verification. The v1.2.6 live gate dynamically selects an `ARTICLES.ART_CTM=0` row and asserts that both exact article search and the batch commercial-identity resolver retain that unrestricted article.
