package cmn

import (
	"regexp"
	"strings"
)

func Contains(s string, subststr string) bool {
	return strings.Contains(
		strings.ToLower(s),
		strings.ToLower(subststr),
	)
}

// Split splits the input string into a slice of strings. Guarantees that the
// result has at least 3 elements.
func Split(input string) []string {
	re := regexp.MustCompile(`'[^']*'|"[^"]*"|\b[^'\s"]+\b`)
	matches := re.FindAllString(input, -1)

	var result []string
	for _, match := range matches {
		// Remove quotes if the match is a quoted string
		if (match[0] == '"' && match[len(match)-1] == '"') || (match[0] == '\'' && match[len(match)-1] == '\'') {
			match = match[1 : len(match)-1]
		}
		result = append(result, match)
	}

	// make sure the result has at least 3 elements
	for len(result) < 3 {
		result = append(result, "")
	}

	return result
}

func IsInArray(slice []string, item string) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}
