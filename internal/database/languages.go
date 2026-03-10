package database

import (
"go-tecdoc-api/internal/models"
)

// GetLanguages повертає список всіх мов
func (q *Queries) GetLanguages() ([]models.Language, error) {
query := `
SELECT 
LNG_ID,
LNG_DESCRIPTION,
LNG_ISO2,
LNG_CODEPAGE
FROM 
LANGUAGES
ORDER BY 
LNG_DESCRIPTION
`

rows, err := q.db.Query(query)
if err != nil {
return nil, err
}
defer rows.Close()

var languages []models.Language
for rows.Next() {
var lang models.Language
if err := rows.Scan(&lang.ID, &lang.Description, &lang.ISO2, &lang.Codepage); err != nil {
return nil, err
}
languages = append(languages, lang)
}

return languages, rows.Err()
}

// GetLanguageByID повертає деталі мови за ID
func (q *Queries) GetLanguageByID(id int) (*models.Language, error) {
query := `
SELECT 
LNG_ID,
LNG_DESCRIPTION,
LNG_ISO2,
LNG_CODEPAGE
FROM 
LANGUAGES
WHERE 
LNG_ID = ?
`

var lang models.Language
err := q.db.QueryRow(query, id).Scan(&lang.ID, &lang.Description, &lang.ISO2, &lang.Codepage)
if err != nil {
return nil, err
}

return &lang, nil
}
