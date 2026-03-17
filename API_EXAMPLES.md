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

### тільки language_id

```
GET /api/v1/search/article?number=OC90&language_id=48
```

### language_id + country_id

```
GET /api/v1/search/article?number=OC90&language_id=48&country_id=258
```

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

### тільки language_id

```
GET /api/v1/search/analog?search_number=OC90&language_id=48
```

### language_id + country_id

```
GET /api/v1/search/analog?search_number=OC90&language_id=48&country_id=258
```

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
```

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
