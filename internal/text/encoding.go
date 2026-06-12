package text

import (
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
)

// EncodeForPDF converts UTF-8 text into Windows-1252, which is what the
// built-in PDF fonts in fpdf expect. Characters that cannot be represented
// are replaced with '?'.
func EncodeForPDF(s string) string {
	if s == "" {
		return s
	}

	var b strings.Builder
	b.Grow(len(s))

	for len(s) > 0 {
		r, size := utf8.DecodeRuneInString(s)
		s = s[size:]

		if r == utf8.RuneError && size == 1 {
			b.WriteByte('?')
			continue
		}

		if encoded, ok := charmap.Windows1252.EncodeRune(r); ok {
			b.WriteByte(encoded)
			continue
		}

		b.WriteByte('?')
	}

	return b.String()
}
