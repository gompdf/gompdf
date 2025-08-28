package text

import (
	"unicode"
)

// TextShaper handles text shaping operations
type TextShaper struct {
	// Configuration options could be added here
}

// ShapedText represents shaped text ready for rendering
type ShapedText struct {
	Text    string
	Glyphs  []Glyph
	Width   float64
	Height  float64
	Ascent  float64
	Descent float64
	LineGap float64
}

// Glyph represents a single glyph in shaped text
type Glyph struct {
	Rune    rune
	Index   uint16
	X       float64
	Y       float64
	Width   float64
	Height  float64
	Advance float64
}

// Font represents a font used for text shaping
type Font struct {
	Family     string
	Style      string
	Weight     int
	Size       float64
	LineHeight float64
}

// NewTextShaper creates a new text shaper
func NewTextShaper() *TextShaper {
	return &TextShaper{}
}

// ShapeText shapes text for rendering
func (s *TextShaper) ShapeText(text string, font *Font, maxWidth float64) *ShapedText {
	shaped := &ShapedText{
		Text:   text,
		Glyphs: make([]Glyph, 0, len(text)),
	}

	charWidth := font.Size * 0.6 // Approximate width for monospace
	lineHeight := font.Size * font.LineHeight
	ascent := font.Size * 0.8
	descent := font.Size * 0.2

	x := 0.0
	y := ascent

	for _, r := range text {
		if r == '\n' {
			x = 0
			y += lineHeight
			continue
		}

		if unicode.IsSpace(r) {
			x += charWidth
			continue
		}

		glyph := Glyph{
			Rune:    r,
			Index:   0, // In a real implementation, this would be the glyph index
			X:       x,
			Y:       y,
			Width:   charWidth,
			Height:  font.Size,
			Advance: charWidth,
		}

		shaped.Glyphs = append(shaped.Glyphs, glyph)

		x += charWidth

		if maxWidth > 0 && x > maxWidth {
			x = 0
			y += lineHeight
		}
	}

	shaped.Width = maxWidth
	shaped.Height = y + descent
	shaped.Ascent = ascent
	shaped.Descent = descent
	shaped.LineGap = lineHeight - (ascent + descent)

	return shaped
}

// MeasureText measures text without shaping it
func (s *TextShaper) MeasureText(text string, font *Font) (width, height float64) {
	// This is a simplified implementation
	// In a real implementation, we would use a proper text measurement library

	charWidth := font.Size * 0.6 // Approximate width for monospace
	lineHeight := font.Size * font.LineHeight

	maxWidth := 0.0
	currentWidth := 0.0
	lines := 1

	for _, r := range text {
		if r == '\n' {
			maxWidth = max(maxWidth, currentWidth)
			currentWidth = 0
			lines++
			continue
		}

		currentWidth += charWidth
	}

	maxWidth = max(maxWidth, currentWidth)
	height = float64(lines) * lineHeight

	return maxWidth, height
}

// SplitTextToLines splits text into lines based on a maximum width
func (s *TextShaper) SplitTextToLines(text string, font *Font, maxWidth float64) []string {
	// This is a simplified implementation
	// In a real implementation, you would use a proper line breaking algorithm

	if maxWidth <= 0 {
		return []string{text}
	}

	charWidth := font.Size * 0.6 // Approximate width for monospace
	charsPerLine := int(maxWidth / charWidth)

	if charsPerLine <= 0 {
		charsPerLine = 1
	}

	var lines []string
	var currentLine string

	words := splitIntoWords(text)

	for _, word := range words {
		if len(currentLine)+len(word)+1 > charsPerLine && currentLine != "" {
			lines = append(lines, currentLine)
			currentLine = word
		} else {
			if currentLine != "" {
				currentLine += " "
			}
			currentLine += word
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return lines
}

// splitIntoWords splits text into words
func splitIntoWords(text string) []string {
	var words []string
	var currentWord string

	for _, r := range text {
		if unicode.IsSpace(r) {
			if currentWord != "" {
				words = append(words, currentWord)
				currentWord = ""
			}
		} else {
			currentWord += string(r)
		}
	}

	if currentWord != "" {
		words = append(words, currentWord)
	}

	return words
}
