module github.com/gompdf/gompdf

go 1.24

require (
golang.org/x/net v0.30.0 // html parser
github.com/andybalholm/cascadia v1.3.2 // CSS selectors
github.com/tdewolff/parse/v2 v2.7.12 // CSS lexer/parser
github.com/go-text/typesetting v0.0.0-20250101-xxxxxxxxxx // harfbuzz shaping (replace with latest)
golang.org/x/text v0.18.0 // bidi, width, language
github.com/tdewolff/canvas v0.0.0-2025xxxxxxx // PDF backend (pin latest)
)