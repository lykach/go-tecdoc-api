# 📚 TecDoc Go API — Приклади запитів

Базовий URL:

```
http://localhost:8081/api/v1
```

---

# 🌐 Параметри локалізації

## 1. Тільки мова

```
?language_id=48
```

## 2. Мова + країна

```
?language_id=48&country_id=258
```

---

# 🔎 SEARCH (ПОШУК)

## Пошук по артикулу

### Широкий пошук для діагностики

```
GET /api/v1/search/article?number=OC90&language_id=48&country_id=258
```

Цей endpoint може повертати збіги через ArticleNumber, IAMNumber, OENumber або
інші TecDoc lookup-типи. Його не можна використовувати для вибору джерела
ланцюжка аналогів.

### Точний власний ArticleNumber виробника

```
GET /api/v1/search/article-exact?number=OC90&language_id=48&country_id=258
```

Результат містить `supplier_id`. Клієнт повинен вибрати рівно одну статтю
потрібного виробника, після чого використовувати її `article_id`.

---

## Пошук по OEM

### тільки language_id

```
GET /api/v1/search/oem?number=7700115294&language_id=48
```

### language_id + country_id

```
GET /api/v1/search/oem?number=7700115294&language_id=48&country_id=258
```

---

## Пошук аналогів

### Рекомендовано: за точним ART_ID

```
GET /api/v1/search/analog?art_id=29&language_id=48&country_id=258
```

### За власним артикулом + TecDoc supplier ID

```
GET /api/v1/search/analog?search_number=OC90&supplier_id=9&language_id=48&country_id=258
```

Запит лише з `search_number` навмисно повертає HTTP 400. Однаковий номер у
різних виробників не доводить, що товари є аналогами.

---

## OEM → OEM (без локалізації)

```
GET /api/v1/search/oem-oem?oem_number=4853009T50
```

---

# 📦 ARTICLES

## Пошук статей

### тільки language_id

```
GET /api/v1/articles/search?number=OC90&language_id=48
```

### language_id + country_id

```
GET /api/v1/articles/search?number=OC90&language_id=48&country_id=258
```

---

## Деталі товару

### тільки language_id

```
GET /api/v1/articles/12345?language_id=48
```

### language_id + country_id

```
GET /api/v1/articles/12345?language_id=48&country_id=258
```

---

## Крос-референси

```
GET /api/v1/articles/12345/cross-references?language_id=48&country_id=258
```

---

## OEM номери

```
GET /api/v1/articles/12345/oem?language_id=48&country_id=258
```

---

## Фото / медіа

```
GET /api/v1/articles/12345/media?language_id=48
```

---

## Характеристики

```
GET /api/v1/articles/12345/criteria?language_id=48
GET /api/v1/articles/12345/criteria?language_id=48&include_hidden=true
```

`include_hidden=true` додає технічні TecDoc criteria з `ACR_DISPLAY=0`. У
відповіді зберігається сумісне поле `value` і додаються `raw_value`, `unit` та
`display`.

---

## TecDoc classification статті

```
GET /api/v1/articles/12345/classification?language_id=48&country_id=258
```

Повертає всі `PT_ID`, Search Tree nodes/scopes і usage evidence. Endpoint не
обирає локальну категорію автоматично.

---

## Structured applicability

```
GET /api/v1/articles/12345/applicability?language_id=48&country_id=258&target_type=PC&limit=1000&offset=0
```

Підтримувані `target_type`: `PC`, `CV`, `Motorcycle`, `Engine`, `Axle`.
`linkages[]` містить `la_id`, `pt_id`, target IDs і `restrictions[]`.

---

## Batch catalog enrichment

Для `article_number` обов’язково передавайте `supplier_id` (рекомендовано) або точну ідентичність виробника через `brand`/`brand_aliases`; голий номер не є глобально унікальним у TecDoc. `brand_aliases` призначений для явних локальних синонімів того самого виробника. Якщо різні aliases ведуть до різних TecDoc supplier ID, результат буде `ambiguous`.

```http
POST /api/v1/enrichment/articles/batch
Content-Type: application/json
X-API-Key: ...

{
  "language_id": 48,
  "country_id": 258,
  "include": ["classification", "oem", "cross_references", "criteria", "applicability"],
  "max_applicability_targets": 2000,
  "items": [
    {
      "request_id": "product-uuid",
      "article_number": "50390266",
      "supplier_id": 72
    },
    {
      "request_id": "product-with-local-brand-alias",
      "article_number": "WK8422",
      "brand": "MANN",
      "brand_aliases": ["MANN-FILTER"]
    }
  ]
}
```

Status кожного item: `exact`, `ambiguous`, `not_found`, `error`. Один commercial article
може мати кілька технічних `article_ids`; вони збагачуються як одна identity.

Якщо consumer вже має immutable/reviewed identity snapshot, повторно шукати той самий номер не потрібно. Передайте весь перевірений набір `article_ids` разом з тим самим `article_number` і `supplier_id`:

```json
{
  "language_id": 48,
  "country_id": 258,
  "include": ["classification", "oem", "criteria", "applicability"],
  "items": [
    {
      "request_id": "promoted-product-uuid",
      "article_ids": [3575879, 3575880],
      "article_number": "50390266",
      "supplier_id": 72
    }
  ]
}
```

Це server-to-server stale fence: API заново перевіряє кожний `ART_ID` проти заявленої commercial identity. `article_ids` повинні бути унікальними й позитивними (до 200 на item), не комбінуються з `article_id`; відсутній ID повертає `not_found`, а зміна supplier/article identity — `error`.

Для `exact` response поле `article_ids` містить усі технічні `ART_ID`. Об’єднані `classification`, `oem_numbers`, `criteria` та `applicability.linkages` повертають `source_article_ids`, щоб consumer не втрачав provenance. Якщо потрібна повна target-level applicability для multi-`ART_ID`, дочитуйте `/articles/{id}/applicability` окремо для кожного `ART_ID`; embedded page з `truncated=true` або `total_targets_exact=false` не можна застосовувати як повний snapshot.

---

# 🚗 VEHICLES

## Виробники

### тільки language_id

```
GET /api/v1/manufacturers?language_id=48
```

### language_id + country_id

```
GET /api/v1/manufacturers?language_id=48&country_id=258
```

---

## Моделі виробника

```
GET /api/v1/manufacturers/1/models?language_id=48&country_id=258
```

---

## Автомобілі

```
GET /api/v1/models/1/cars?language_id=48&country_id=258
```

---

## Деталі авто

```
GET /api/v1/cars/1?language_id=48&country_id=258
```

---

## Групи товарів авто

```
GET /api/v1/cars/1/product-groups?language_id=48
```

---

# 📂 PRODUCT GROUPS

## Категорії

```
GET /api/v1/product-groups?language_id=48
```

---

## Дочірні категорії

```
GET /api/v1/product-groups/1/children?language_id=48
```

---

## Товари категорії

```
GET /api/v1/product-groups/1/articles?car_id=123&language_id=48&country_id=258
```

---

# 🏭 SUPPLIERS

```
GET /api/v1/suppliers?language_id=48
GET /api/v1/suppliers/1?language_id=48
GET /api/v1/suppliers/1/products?language_id=48&country_id=258
```

---

# 🌍 LOCALIZATION

## Мови

```
GET /api/v1/languages
GET /api/v1/languages/48
```

## Країни

```
GET /api/v1/countries
GET /api/v1/countries/258
```

---

# 💚 HEALTH CHECK

```
GET /health
```

---

# ⚠️ ВАЖЛИВО

* Якщо `language_id` не переданий → використовується `.env`
* Якщо `country_id` не переданий → використовується `.env`
* Рекомендовано **завжди передавати обидва параметри**

---

# 🧠 РЕКОМЕНДАЦІЯ

Для багатомовного сайту:

```
uk → language_id=48
en → language_id=4
ru → language_id=16
```

```
country_id=258 (Україна)
```

---

# 🚀 ПРИКЛАД (PRODUCTION)

```
GET /api/v1/search/article?number=OC90&language_id=48&country_id=258&limit=20&offset=0
```

---

# 🆕 TECdoc 2024Q1 — повне покриття

Усі `/api/v1/*` приклади нижче потребують:

```http
X-API-Key: <dedicated-test-key>
```

## KBA

```bash
curl -H 'X-API-Key: <dedicated-test-key>' \
  'http://localhost:8081/api/v1/search/kba?number=0588549&vehicle_type=PC'
```

```json
{"matches":[{"vehicle_id":1014,"vehicle_type":"PC","number":"0588549","number_type":"German KBA Number","manufacturer_id":35}],"total":1}
```

## Netherlands Kenteken

```bash
curl -H 'X-API-Key: <dedicated-test-key>' \
  'http://localhost:8081/api/v1/search/kenteken?number=2VXT04&vehicle_type=PC'
```

## Усі типи номерів

```bash
curl -G -H 'X-API-Key: <dedicated-test-key>' \
  --data-urlencode 'number=0588549' \
  --data-urlencode 'number_type=German KBA Number' \
  --data-urlencode 'vehicle_type=PC' \
  'http://localhost:8081/api/v1/search/type-number'
```

## Повні специфікації артикула

```bash
curl -H 'X-API-Key: <dedicated-test-key>' \
  'http://localhost:8081/api/v1/articles/28/specifications?language_id=4&country_id=62&include_hidden=true'
```

`source=article` — глобальна характеристика; `source=linkage` — обмеження конкретного `LA_ID`.

## ARTICLE_INFO

```bash
curl -H 'X-API-Key: <dedicated-test-key>' \
  'http://localhost:8081/api/v1/articles/28/info?language_id=48&include_hidden=true'
```

## LA_INFO

```bash
curl -H 'X-API-Key: <dedicated-test-key>' \
  'http://localhost:8081/api/v1/articles/28/linkage-info?language_id=48&country_id=258'
```

## Ціни TecDoc

```bash
curl -H 'X-API-Key: <dedicated-test-key>' \
  'http://localhost:8081/api/v1/articles/28/prices?language_id=48&country_id=258&include_expired=false'
```

## Схема деталей на зображенні

```bash
curl -H 'X-API-Key: <dedicated-test-key>' \
  'http://localhost:8081/api/v1/articles/124926/parts-diagram'
```

## Quick Start

```bash
curl -H 'X-API-Key: <dedicated-test-key>' \
  'http://localhost:8081/api/v1/quick-start?vehicle_type=PC&language_id=48'
```

## Пошук товарної групи за текстом

```bash
curl -G -H 'X-API-Key: <dedicated-test-key>' \
  --data-urlencode 'q=VALVE' \
  --data-urlencode 'vehicle_type=PC' \
  --data-urlencode 'language_id=4' \
  'http://localhost:8081/api/v1/product-groups/search'
```

## Критерії Product Type

```bash
curl -H 'X-API-Key: <dedicated-test-key>' \
  'http://localhost:8081/api/v1/product-types/1/criteria?language_id=48&suggested_only=false'
```

## KeyValue словник критерію

```bash
curl -H 'X-API-Key: <dedicated-test-key>' \
  'http://localhost:8081/api/v1/criteria/1/values?language_id=48'
```

## CV chassis

```bash
curl -H 'X-API-Key: <dedicated-test-key>' \
  'http://localhost:8081/api/v1/cv/17582/chassis?language_id=48'
```

## Axle detail

```bash
curl -H 'X-API-Key: <dedicated-test-key>' \
  'http://localhost:8081/api/v1/axles/1?language_id=48'
```

## Повний raw dataset

```bash
curl -H 'X-API-Key: <dedicated-test-key>' \
  'http://localhost:8081/api/v1/datasets'
```

```bash
curl -G -H 'X-API-Key: <dedicated-test-key>' \
  --data-urlencode 'select=ART_ID,ART_ARTICLE_NR,ART_SUP_ID,ART_SUP_BRAND' \
  --data-urlencode 'order_by=ART_ID' \
  --data-urlencode 'limit=100' \
  --data-urlencode 'offset=0' \
  'http://localhost:8081/api/v1/datasets/ARTICLES'
```

## CROSS CSV

```bash
curl -o crosses.csv \
  -H 'X-API-Key: <dedicated-test-key>' \
  'http://localhost:8081/api/v1/exports/cross-references?format=csv&limit=5000&offset=0'
```

## EAN CSV

```bash
curl -o ean.csv \
  -H 'X-API-Key: <dedicated-test-key>' \
  'http://localhost:8081/api/v1/exports/ean?format=csv&limit=5000&offset=0'
```

Повна документація з response shapes: `API_REFERENCE.md` та `OPENAPI.yaml`.

---

# 🔒 FINAL PRODUCTION HARDENING (v1.2.6)

## Великі article relations — тільки сторінками

```http
GET /api/v1/articles/1307505/accessories?language_id=4&country_id=62&limit=25&offset=0
X-API-Key: ...
```

Відповідь містить `total`, `limit`, `offset`, `returned`, `has_more`. Для `accessories` використовуйте `criteria_details` як lossless-джерело: одна й та сама TecDoc-характеристика може мати кілька значень. `criteria` — лише compatibility-map з одним детермінованим значенням на назву.

```http
GET /api/v1/articles/28/specifications?language_id=4&country_id=62&include_hidden=true&limit=100&offset=0
GET /api/v1/articles/28/linkage-info?language_id=4&country_id=62&limit=100&offset=0
GET /api/v1/articles/124926/parts-diagram?limit=100&offset=0
GET /api/v1/articles/28/applicability?language_id=4&country_id=62&limit=100&offset=0
```

`limit` для цих контрактів: `1..500`. `offset` — ціле число `>=0`.

## Article detail без мегабайтного nested payload

```http
GET /api/v1/articles/28?language_id=4&country_id=62&related_limit=25
```

`related_limit`: `1..100`, default `50`. Для повного списку аксесуарів використовуйте окремий `/articles/{id}/accessories`.

## Strict validation

Ці запити навмисно повертають `400`:

```http
GET /api/v1/articles/1307505/accessories?limit=0
GET /api/v1/articles/1307505/accessories?limit=501
GET /api/v1/articles/1307505/accessories?offset=-1
GET /api/v1/articles/28/specifications?include_hidden=maybe
GET /api/v1/datasets/ARTICLES?order_by=ART_ID&order=SIDEWAYS
GET /api/v1/datasets/ARTICLES?format=xml
GET /api/v1/languages?language_id=0
```

## Final release gate — 72/72 operations + ART_CTM=0 regression

```bash
export API=http://localhost:8081
export API_KEY='<dedicated-test-key>'
./test-api.sh
```

Успішний фінал:

```text
Unique OpenAPI operationIds exercised live: 72/72
Negative/auth/validation checks: PASS
All 72 OpenAPI operations completed successfully.
FINAL PRODUCTION CONTRACT TEST: PASS
```
