# TecDoc 2024Q1 schema coverage

The supplied dump contains 77 base tables in total. This service deliberately excludes application-auth tables (`api_keys`, `api_key_usage_logs`) and phpMyAdmin `pma__*` tables from source-data export. The remaining **56 TecDoc domain tables are all referenced by domain SQL and/or are retrievable through `/api/v1/datasets/{table}`**.

| TecDoc table | Primary API coverage |
|---|---|
| `ARTICLES` | articles, search, datasets |
| `ARTICLES_PART_LIST` | `/articles/{id}/components` |
| `ARTICLE_CRITERIA` | `/articles/{id}/criteria`, `/specifications` |
| `ARTICLE_INFO` | `/articles/{id}/info` |
| `ART_ACCS_LIST` | `/articles/{id}/accessories` |
| `ART_ACCS_LIST_CRITERIA` | `/articles/{id}/accessories` |
| `ART_COUNTRY_SPECIFICS` | article country context |
| `ART_LOOKUP` | search/cross/OEM/EAN/exports |
| `ART_MEDIA_INFO` | country-aware `/articles/{id}/media`, enrichment media, diagrams |
| `AXLES` | `/axles/{id}`, CV chassis |
| `AXLE_MODELS` | `/axles/{id}` |
| `AXL_BRAKE_SIZES` | `/axles/{id}` |
| `COMMERCIAL_VEHICLES` | `/cv/*` |
| `COUNTRIES` | `/countries`, country restrictions |
| `COUNTRY_RESTRICTIONS` | all country-aware TecDoc relations |
| `CRITERIA` | criteria endpoints |
| `CV_COUNTRY_SPECIFICS` | CV details |
| `CV_SUSPENSIONS` | `/cv/{id}/chassis` |
| `DES_TEXTS` | localization helper |
| `DRIVERS_CABS` | `/cv/{id}/chassis` |
| `ENGINES` | `/engines/{id}`, applicability |
| `KEY_VALUES` | `/criteria/{id}/values` |
| `LANGUAGES` | `/languages` |
| `LA_CRITERIA` | applicability, `/specifications` |
| `LA_INFO` | `/articles/{id}/linkage-info` |
| `LINK_ART` | article/product/linkage relations |
| `LINK_ART_PT` | classification/product type relations |
| `LINK_AXLES_CV` | `/cv/{id}/chassis` |
| `LINK_DRIVERS_CABS` | `/cv/{id}/chassis` |
| `LINK_ENGINE_TYPE` | vehicle/engine details |
| `LINK_LA_TYP` | fitment/applicability |
| `LINK_PL_SC` | `/articles/{id}/parts-diagram` |
| `LINK_PT_CRI` | `/product-types/{id}/criteria`, vehicle criteria |
| `LINK_PT_STR` | Search Tree/product type mapping |
| `LINK_QS_ST` | `/quick-start` |
| `MANUFACTURERS` | `/manufacturers`, cross resolution |
| `MODELS_SERIES` | `/models/*` |
| `MOTORBIKES` | `/mc/*` |
| `MS_COUNTRY_SPECIFICS` | localized model series |
| `MTB_COUNTRY_SPECIFICS` | motorcycle details |
| `PASSENGER_CARS` | `/cars/*` |
| `PC_COUNTRY_SPECIFICS` | passenger-car details |
| `PRICES` | `/articles/{id}/prices` |
| `PRODUCTS` | product types/groups/classification |
| `QUICK_START` | `/quick-start` |
| `SEARCH_TREE` | `/product-groups/*` |
| `SENSITIVE_COORDINATES` | `/coordinates`, `/parts-diagram` |
| `STR_LOOKUP` | `/product-groups/search` |
| `SUPERSEDED_ARTICLES` | article detail supersession |
| `SUPPLIERS` | `/suppliers/*` |
| `SUPPLIERS_LOGOS` | `/suppliers/{id}/logos` |
| `SUPPLIER_ADDRESSES` | `/suppliers/{id}/addresses` |
| `TEXT_DESIGNATIONS` | localization helper |
| `TEXT_MODULE_TEXTS` | article/linkage info |
| `TYPE_NUMBER_LOOKUP` | KBA/Kenteken/type-number endpoints |
| `WHEELBASES` | `/cv/{id}/chassis` |

For any source row not represented in a purpose-built response, use `/datasets/{table}`. This is intentional: domain endpoints provide normalized application contracts, while dataset endpoints preserve complete raw TecDoc access.
