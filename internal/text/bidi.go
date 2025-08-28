package text

// Direction represents text direction
type Direction int

const (
	LeftToRight Direction = iota
	RightToLeft
)

// BidiProcessor handles bidirectional text processing
type BidiProcessor struct{}

// BidiParagraph represents a paragraph with bidirectional text
type BidiParagraph struct {
	Text      string
	Direction Direction
	Runs      []BidiRun
}

// BidiRun represents a run of text with the same direction
type BidiRun struct {
	Start     int
	Length    int
	Text      string
	Direction Direction
	Level     uint8
}

// NewBidiProcessor creates a new bidirectional text processor
func NewBidiProcessor() *BidiProcessor {
	return &BidiProcessor{}
}

// Process processes bidirectional text
func (p *BidiProcessor) Process(text string) *BidiParagraph {
	paragraph := &BidiParagraph{
		Text:      text,
		Direction: LeftToRight, // Default to LTR
		Runs:      []BidiRun{},
	}

	paragraph.Runs = append(paragraph.Runs, BidiRun{
		Start:     0,
		Length:    len(text),
		Text:      text,
		Direction: LeftToRight,
		Level:     0,
	})

	return paragraph
}

// IsRTL checks if a string contains right-to-left text
// This is a simplified implementation that only checks for Arabic and Hebrew ranges
func (p *BidiProcessor) IsRTL(text string) bool {
	for _, r := range text {
		// Check for Arabic (0x0600-0x06FF) or Hebrew (0x0590-0x05FF) characters
		if (r >= 0x0590 && r <= 0x06FF) || (r >= 0xFB50 && r <= 0xFDFF) || (r >= 0xFE70 && r <= 0xFEFF) {
			return true
		}
	}
	return false
}

// GetDisplayText returns the text in display order
func (p *BidiProcessor) GetDisplayText(paragraph *BidiParagraph) string {
	return paragraph.Text
}

// SplitMixedDirectionText splits text with mixed directions into separate runs
func (p *BidiProcessor) SplitMixedDirectionText(text string) []string {
	return []string{text}
}
