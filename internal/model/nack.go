package model

import "strings"

func NackDetail(detail string) string {
	detail = strings.Join(strings.Fields(detail), " ")
	if strings.Contains(detail, "-----BEGIN") {
		return "ayrıntı gizlendi"
	}
	runes := []rune(detail)
	if len(runes) > 180 {
		return string(runes[:180])
	}
	return detail
}
