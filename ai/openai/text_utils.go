package openai

import "strings"

// scrubString removes punctuation and trims whitespace from text.
func scrubString(s string) string {
	// Remove common punctuation
	s = strings.Map(func(r rune) rune {
		if strings.ContainsRune(".,!?;:\"'()[]{}—–-", r) {
			return -1
		}
		return r
	}, s)
	// Trim leading and trailing whitespace
	return strings.TrimSpace(s)
}

// isLetter returns true if the rune is an ASCII letter.
func isLetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}
