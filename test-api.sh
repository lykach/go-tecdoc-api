#!/bin/bash

API="http://localhost:8081"

echo "🧪 Testing Go TecDoc API..."
echo ""

echo "1️⃣ Health Check"
curl -s "$API/health"
echo -e "\n"

echo "2️⃣ Search Article BOSCH"
curl -s "$API/api/v1/search/article?number=0001106017&limit=2"
echo -e "\n"

echo "3️⃣ Article Details"
curl -s "$API/api/v1/articles/29"
echo -e "\n"

echo "4️⃣ Article Criteria"
curl -s "$API/api/v1/articles/29/criteria"
echo -e "\n"

echo "5️⃣ OEM Cross Toyota→Lexus"
curl -s "$API/api/v1/search/oem-oem?oem_number=4853009T50&limit=5"
echo -e "\n"

echo "6️⃣ IAM Analogs"
curl -s "$API/api/v1/search/analog?art_id=29&limit=5"
echo -e "\n"

echo "7️⃣ Supplier Products BOSCH"
curl -s "$API/api/v1/suppliers/9/products?limit=5"
echo -e "\n"

echo "8️⃣ Engine Details Honda J30A4"
curl -s "$API/api/v1/engines/19243"
echo -e "\n"

echo "9️⃣ Article Coordinates"
curl -s "$API/api/v1/articles/5100724/coordinates"
echo -e "\n"

echo "🔟 Filtered Starters (12V)"
curl -s "$API/api/v1/product-groups/100359/articles?car_id=2694&criteria=6:12&limit=3"
echo -e "\n"

echo "✅ All tests completed!"
