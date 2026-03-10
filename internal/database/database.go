package database

import (
"database/sql"
"fmt"
)

type Queries struct {
db *sql.DB
}

func New(db *sql.DB) *Queries {
return &Queries{db: db}
}

// GetText - функція для отримання перекладеного тексту
func (q *Queries) GetText(desID int, lngID int) (string, error) {
var result sql.NullString

query := `
SELECT COALESCE(
(SELECT DES_TEXTS.TEX_TEXT 
 FROM TEXT_DESIGNATIONS 
 INNER JOIN DES_TEXTS ON DES_TEXTS.TEX_ID = TEXT_DESIGNATIONS.DES_TEX_ID
 WHERE TEXT_DESIGNATIONS.DES_ID = ? AND TEXT_DESIGNATIONS.DES_LNG_ID = ?
 GROUP BY TEXT_DESIGNATIONS.DES_ID),
(SELECT DES_TEXTS.TEX_TEXT 
 FROM TEXT_DESIGNATIONS 
 INNER JOIN DES_TEXTS ON DES_TEXTS.TEX_ID = TEXT_DESIGNATIONS.DES_TEX_ID
 WHERE TEXT_DESIGNATIONS.DES_ID = ? AND TEXT_DESIGNATIONS.DES_LNG_ID = 4
 GROUP BY TEXT_DESIGNATIONS.DES_ID),
(SELECT DES_TEXTS.TEX_TEXT 
 FROM TEXT_DESIGNATIONS 
 INNER JOIN DES_TEXTS ON DES_TEXTS.TEX_ID = TEXT_DESIGNATIONS.DES_TEX_ID
 WHERE TEXT_DESIGNATIONS.DES_ID = ? AND TEXT_DESIGNATIONS.DES_LNG_ID = 1
 GROUP BY TEXT_DESIGNATIONS.DES_ID)
) AS result
`

err := q.db.QueryRow(query, desID, lngID, desID, desID).Scan(&result)
if err != nil {
if err == sql.ErrNoRows {
return "", nil
}
return "", err
}

return result.String, nil
}

// Helper function to handle NULL strings
func nullString(s sql.NullString) string {
if s.Valid {
return s.String
}
return ""
}

// Helper function to handle NULL ints
func nullInt(i sql.NullInt64) *int {
if i.Valid {
val := int(i.Int64)
return &val
}
return nil
}

// Helper function to create NULL string
func toNullString(s string) sql.NullString {
if s == "" {
return sql.NullString{Valid: false}
}
return sql.NullString{String: s, Valid: true}
}

// Helper function to create NULL int
func toNullInt(i *int) sql.NullInt64 {
if i == nil {
return sql.NullInt64{Valid: false}
}
return sql.NullInt64{Int64: int64(*i), Valid: true}
}

// BuildGetTextSubquery - helper для побудови підзапиту get_text в SQL
func BuildGetTextSubquery(desIDColumn string, lngID int) string {
return fmt.Sprintf(`
COALESCE(
(SELECT DES_TEXTS.TEX_TEXT 
 FROM TEXT_DESIGNATIONS 
 INNER JOIN DES_TEXTS ON DES_TEXTS.TEX_ID = TEXT_DESIGNATIONS.DES_TEX_ID
 WHERE TEXT_DESIGNATIONS.DES_ID = %s AND TEXT_DESIGNATIONS.DES_LNG_ID = %d
 GROUP BY TEXT_DESIGNATIONS.DES_ID LIMIT 1),
(SELECT DES_TEXTS.TEX_TEXT 
 FROM TEXT_DESIGNATIONS 
 INNER JOIN DES_TEXTS ON DES_TEXTS.TEX_ID = TEXT_DESIGNATIONS.DES_TEX_ID
 WHERE TEXT_DESIGNATIONS.DES_ID = %s AND TEXT_DESIGNATIONS.DES_LNG_ID = 4
 GROUP BY TEXT_DESIGNATIONS.DES_ID LIMIT 1),
(SELECT DES_TEXTS.TEX_TEXT 
 FROM TEXT_DESIGNATIONS 
 INNER JOIN DES_TEXTS ON DES_TEXTS.TEX_ID = TEXT_DESIGNATIONS.DES_TEX_ID
 WHERE TEXT_DESIGNATIONS.DES_ID = %s AND TEXT_DESIGNATIONS.DES_LNG_ID = 1
 GROUP BY TEXT_DESIGNATIONS.DES_ID LIMIT 1)
)
`, desIDColumn, lngID, desIDColumn, desIDColumn)
}
