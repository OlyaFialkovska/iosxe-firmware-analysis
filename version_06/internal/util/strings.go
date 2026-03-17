package util

import "strings"

func ExtractASCIIStrings(data []byte, minLen int) []string {
	var result []string
	var current []byte

	for _, b := range data {
		if b >= 32 && b <= 126 {
			current = append(current, b)
		} else {
			if len(current) >= minLen {
				result = append(result, string(current))
			}
			current = nil
		}
	}

	if len(current) >= minLen {
		result = append(result, string(current))
	}

	return result
}

func IsPrintable(b byte) bool {
	return b >= 32 && b <= 126
}

func ContainsAny(s string, words []string) bool {
	for _, w := range words {
		if strings.Contains(s, w) {
			return true
		}
	}
	return false
}

func SafeSubstring(s string, start int, end int) string {
	if start < 0 {
		start = 0
	}
	if end > len(s) {
		end = len(s)
	}
	if start >= end {
		return ""
	}
	return s[start:end]
}

func TrimNulls(s string) string {
	return strings.Trim(s, "\x00")
}
