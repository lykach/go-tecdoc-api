package database

import (
	"strings"
	"unicode"
)

// normalizeSearchNumber converts a user-facing article or OEM number to the
// canonical TecDoc lookup representation. Display separators such as spaces,
// slash, dash and dot must not change the result of a search.
func normalizeSearchNumber(value string) string {
	var result strings.Builder
	result.Grow(len(value))
	for _, r := range strings.TrimSpace(value) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			result.WriteRune(unicode.ToUpper(r))
		}
	}
	return result.String()
}
