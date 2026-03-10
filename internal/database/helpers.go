package database

import (
"fmt"
"strings"
)

// parseYear парсить рік з дати формату "2010-01-01"
func parseYear(dateStr string) (int, error) {
if len(dateStr) < 4 {
return 0, fmt.Errorf("invalid date format")
}
var year int
_, err := fmt.Sscanf(dateStr[:4], "%d", &year)
return year, err
}

// splitAndClean розділяє і очищує строку
func splitAndClean(s, sep string) []string {
if s == "" {
return []string{}
}
parts := strings.Split(s, sep)
result := make([]string, 0, len(parts))
for _, p := range parts {
trimmed := strings.TrimSpace(p)
if trimmed != "" {
result = append(result, trimmed)
}
}
return result
}
