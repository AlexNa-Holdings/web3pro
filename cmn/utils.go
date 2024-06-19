package cmn

import "strings"

func Contains(s string, subststr string) bool {
	return strings.Contains(
		strings.ToLower(s),
		strings.ToLower(subststr),
	)
}
