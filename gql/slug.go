package gql

import (
	"strings"
	"unicode"
)

// GameNameToSlug converts a display name to the Twitch directory slug
// (same rules as TwitchDropsMiner utils.Game.slug).
func GameNameToSlug(name string) string {
	lower := strings.ToLower(name)
	var b strings.Builder
	prevDash := false
	for _, ch := range lower {
		if ch == '\'' {
			continue
		}
		if unicode.IsLetter(ch) || unicode.IsDigit(ch) {
			b.WriteRune(ch)
			prevDash = false
			continue
		}
		if !prevDash {
			b.WriteByte('-')
			prevDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}
