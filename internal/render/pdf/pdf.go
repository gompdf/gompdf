package pdf

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"codeberg.org/go-pdf/fpdf"
	"github.com/gompdf/gompdf/internal/layout"
	"github.com/gompdf/gompdf/internal/pagination"
)

// Renderer handles rendering to PDF
type Renderer struct {
	// Configuration options
	FontDirs []string
	DPI      float64
	// Debug enables verbose logging to stdout
	Debug bool
	// RenderBackgrounds controls whether box backgrounds are painted
	RenderBackgrounds bool
	// RenderBorders controls whether box borders are painted
	RenderBorders bool
	// DebugDrawBoxes controls drawing of debug overlays (outlines/placeholder fills)
	DebugDrawBoxes bool
	// listStack tracks nested list contexts while rendering
	listStack []listContext
	// renderedTexts tracks which text boxes have been rendered to avoid duplicates
	renderedTexts map[string]bool
}

// listContext represents an active list (ul/ol) while rendering
type listContext struct {
	kind    string // "ul" or "ol"
	style   string // list-style-type
	counter int    // for ordered lists
}

// RenderOptions contains options for rendering
type RenderOptions struct {
	Title       string
	Author      string
	Subject     string
	Keywords    string
	Creator     string
	Producer    string
	Orientation string // "P" for portrait, "L" for landscape
}

// NewRenderer creates a new PDF renderer
func NewRenderer() *Renderer {
	return &Renderer{
		FontDirs:          []string{},
		DPI:               96,
		Debug:             false,
		RenderBackgrounds: true,
		RenderBorders:     true,
		DebugDrawBoxes:    false,
		renderedTexts:     make(map[string]bool),
	}
}

// AddFontDirectory adds a directory to search for fonts
func (r *Renderer) AddFontDirectory(dir string) {
	r.FontDirs = append(r.FontDirs, dir)
}

// Render renders pages to a PDF file
func (r *Renderer) Render(pages []*pagination.Page, outputPath string, options RenderOptions) error {
	// Reset the rendered texts map to ensure clean state for each rendering
	r.renderedTexts = make(map[string]bool)

	// Always use the orientation from options
	orient := options.Orientation
	if orient == "" {
		orient = "P" // Default to portrait if not specified
	}

	pdf := fpdf.New(orient, "pt", "", "")

	pdf.SetAutoPageBreak(true, 2)
	pdf.SetTitle(options.Title, true)
	pdf.SetAuthor(options.Author, true)
	pdf.SetSubject(options.Subject, true)
	pdf.SetKeywords(options.Keywords, true)
	pdf.SetCreator(options.Creator, true)
	pdf.SetProducer(options.Producer, true)
	r.registerFonts(pdf)

	// Process each page - skip truly empty pages
	fmt.Printf("Rendering %d pages\n", len(pages))
	for i, page := range pages {
		// Skip pages with no boxes at all
		if len(page.Boxes) == 0 {
			fmt.Printf("Skipping empty page %d (no boxes)\n", i)
			continue
		}

		// Check if page has any meaningful content
		hasContent := false
		for _, box := range page.Boxes {
			if blockBox, ok := box.(*layout.BlockBox); ok {
				// Consider content if box has children, height, or is a table/structural element
				if len(blockBox.Children) > 0 || blockBox.Height > 0 ||
					(blockBox.Node != nil && (blockBox.Node.Data == "table" || blockBox.Node.Data == "div" || blockBox.Node.Data == "section")) {
					hasContent = true
					break
				}
			} else {
				// Non-block boxes (like InlineBox) are always considered content
				hasContent = true
				break
			}
		}

		if !hasContent {
			fmt.Printf("Skipping empty page %d (no meaningful content)\n", i)
			continue
		}
		pdf.AddPage()

		for _, box := range page.Boxes {
			// Skip rendering boxes with no content
			if blockBox, ok := box.(*layout.BlockBox); ok && len(blockBox.Children) == 0 && blockBox.Height < 1 {
				continue
			}
			r.renderBox(pdf, box)
		}
	}

	outputDir := filepath.Dir(outputPath)
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	return pdf.OutputFileAndClose(outputPath)
}

// registerFonts registers fonts with the PDF document
func (r *Renderer) registerFonts(pdf *fpdf.Fpdf) {
	pdf.SetFont("Helvetica", "", 12)

}

// renderBox renders a box to the PDF
func (r *Renderer) renderBox(pdf *fpdf.Fpdf, box layout.Box) {

	switch b := box.(type) {
	case *layout.BlockBox:
		r.renderBlockBox(pdf, b)
	case *layout.InlineBox:
		r.renderInlineBox(pdf, b)
	default:
		if r.Debug {
			fmt.Printf("Unknown box type: %T\n", box)
		}
	}
}

// renderBlockBox renders a block box to the PDF
func (r *Renderer) renderBlockBox(pdf *fpdf.Fpdf, box *layout.BlockBox) {
	r.renderBackground(pdf, box)

	// Special handling for table elements
	if box != nil && box.Node != nil {
		tag := strings.ToLower(box.Node.Data)
		if tag == "table" || tag == "td" || tag == "th" {
			r.renderTableElement(pdf, box, tag)
		} else {
			// Standard border rendering for non-table elements
			r.renderBorders(pdf, box)
		}
	} else {
		r.renderBorders(pdf, box)
	}

	enteringList := false
	if box != nil && box.Node != nil {
		tag := strings.ToLower(box.Node.Data)
		if tag == "ul" || tag == "ol" {
			enteringList = true
			lc := listContext{kind: tag}
			if prop, ok := box.Style["list-style-type"]; ok && prop.Value != "" {
				lc.style = strings.ToLower(strings.TrimSpace(prop.Value))
			}
			if lc.style == "" {
				if tag == "ul" {
					lc.style = "disc"
				} else {
					lc.style = "decimal"
				}
			}
			r.listStack = append(r.listStack, lc)
		}
	}

	for _, child := range box.Children {
		if len(r.listStack) > 0 {
			if cb, ok := child.(*layout.BlockBox); ok && cb.Node != nil {
				ctag := strings.ToLower(cb.Node.Data)
				if ctag == "li" {
					top := &r.listStack[len(r.listStack)-1]
					if top.kind == "ol" {
						top.counter++
					}
					r.renderListMarker(pdf, cb, *top)
				}
			}
		}
		r.renderBox(pdf, child)
	}

	if enteringList && len(r.listStack) > 0 {
		r.listStack = r.listStack[:len(r.listStack)-1]
	}

	if r.DebugDrawBoxes {
		pdf.SetDrawColor(200, 0, 0)
		pdf.SetLineWidth(0.5)
		pdf.Rect(
			box.X,
			box.Y,
			box.Width+box.PaddingLeft+box.PaddingRight+box.BorderLeft+box.BorderRight,
			box.Height+box.PaddingTop+box.PaddingBottom+box.BorderTop+box.BorderBottom,
			"D",
		)
	}
}

// renderInlineBox renders an inline box to the PDF
func (r *Renderer) renderInlineBox(pdf *fpdf.Fpdf, box *layout.InlineBox) {
	r.renderBackground(pdf, box)
	r.renderBorders(pdf, box)

	if box.Text != "" {
		r.renderText(pdf, box)
	}

	for _, child := range box.Children {
		r.renderBox(pdf, child)
	}
	if r.DebugDrawBoxes {
		pdf.SetDrawColor(0, 0, 200)
		pdf.SetLineWidth(0.5)
		pdf.Rect(
			box.X,
			box.Y,
			box.Width,
			box.Height,
			"D",
		)
	}
}

// renderBackground renders the background of a box
func (r *Renderer) renderBackground(pdf *fpdf.Fpdf, box layout.Box) {
	if !r.RenderBackgrounds {
		return
	}
	hasCustomBg := false

	switch b := box.(type) {
	case *layout.BlockBox:
		if bgColor, exists := b.Style["background-color"]; exists && bgColor.Value != "" {
			color := parseColor(bgColor.Value)
			pdf.SetFillColor(color[0], color[1], color[2])
			pdf.Rect(box.GetX(), box.GetY(), box.GetWidth(), box.GetHeight(), "F")
			hasCustomBg = true
			if r.Debug {
				fmt.Printf("Applied background color %v to block box\n", color)
			}
		}
	case *layout.InlineBox:
		if bgColor, exists := b.Style["background-color"]; exists && bgColor.Value != "" {
			color := parseColor(bgColor.Value)
			pdf.SetFillColor(color[0], color[1], color[2])
			pdf.Rect(box.GetX(), box.GetY(), box.GetWidth(), box.GetHeight(), "F")
			hasCustomBg = true
			if r.Debug {
				fmt.Printf("Applied background color %v to inline box\n", color)
			}
		}
	}

	if r.DebugDrawBoxes && !hasCustomBg {
		pdf.SetFillColor(240, 240, 240)
		pdf.Rect(box.GetX(), box.GetY(), box.GetWidth(), box.GetHeight(), "F")
		pdf.SetFont("Helvetica", "", 8)
		pdf.SetTextColor(150, 150, 150)
		dimensionText := fmt.Sprintf("%.0fx%.0f", box.GetWidth(), box.GetHeight())
		pdf.Text(box.GetX()+2, box.GetY()+10, dimensionText)
	}
}

// renderBorders renders the borders of a box
func (r *Renderer) renderBorders(pdf *fpdf.Fpdf, box layout.Box) {
	if !r.RenderBorders {
		return
	}
	hasCustomBorder := false

	switch b := box.(type) {
	case *layout.BlockBox:
		if borderColor, exists := b.Style["border-color"]; exists && borderColor.Value != "" {
			color := parseColor(borderColor.Value)
			pdf.SetDrawColor(color[0], color[1], color[2])

			width := 1.0
			if borderWidth, exists := b.Style["border-width"]; exists {
				width = parseFloat(borderWidth.Value, 1.0)
			}
			pdf.SetLineWidth(width)

			pdf.Rect(box.GetX(), box.GetY(), box.GetWidth(), box.GetHeight(), "D")
			hasCustomBorder = true

			if r.Debug {
				fmt.Printf("Applied border color %v with width %.1f to block box\n", color, width)
			}
		}
	case *layout.InlineBox:
		if borderColor, exists := b.Style["border-color"]; exists && borderColor.Value != "" {
			color := parseColor(borderColor.Value)
			pdf.SetDrawColor(color[0], color[1], color[2])
			width := 1.0
			if borderWidth, exists := b.Style["border-width"]; exists {
				width = parseFloat(borderWidth.Value, 1.0)
			}
			pdf.SetLineWidth(width)

			pdf.Rect(box.GetX(), box.GetY(), box.GetWidth(), box.GetHeight(), "D")
			hasCustomBorder = true

			if r.Debug {
				fmt.Printf("Applied border color %v with width %.1f to inline box\n", color, width)
			}
		}
	}

	if r.DebugDrawBoxes && !hasCustomBorder {
		pdf.SetDrawColor(200, 200, 200)
		pdf.SetLineWidth(0.5)
		pdf.Rect(box.GetX(), box.GetY(), box.GetWidth(), box.GetHeight(), "D")
	}
}

// renderText renders text to the PDF
func (r *Renderer) renderText(pdf *fpdf.Fpdf, box *layout.InlineBox) {
	if box.Text == "" {
		if r.Debug {
			fmt.Printf("Skipping empty text box\n")
		}
		return
	}

	// Generate a unique ID for this text box to avoid duplicate rendering
	// Include position, size, and the box pointer to prevent false positives
	textID := fmt.Sprintf("%s-%.2f-%.2f-%.2f-%.2f-%p",
		box.Text, box.X, box.Y, box.Width, box.Height, box)

	// Check if we've already rendered this text
	if r.renderedTexts[textID] {
		// Skip if already rendered
		if r.Debug {
			fmt.Printf("Skipping duplicate text: '%s' at (%.2f, %.2f)\n", box.Text, box.X, box.Y)
		}
		return
	}

	// Mark as rendered
	r.renderedTexts[textID] = true

	fontSize := 12.0
	if fontSizeProp, exists := box.Style["font-size"]; exists {
		fontSize = parseFloat(fontSizeProp.Value, 12)
		if r.Debug {
			fmt.Printf("Using font size: %.1f\n", fontSize)
		}
	}

	fontFamily := "Helvetica"
	if fontFamilyProp, exists := box.Style["font-family"]; exists {
		fontFamilies := strings.Split(fontFamilyProp.Value, ",")
		if len(fontFamilies) > 0 {
			firstFont := strings.TrimSpace(fontFamilies[0])
			firstFont = strings.Trim(firstFont, "'\"")

			switch strings.ToLower(firstFont) {
			case "arial", "helvetica", "sans-serif":
				fontFamily = "Helvetica"
			case "times", "times new roman", "serif":
				fontFamily = "Times"
			case "courier", "courier new", "monospace":
				fontFamily = "Courier"
			default:
				// Keep default Helvetica
			}
		}
		if r.Debug {
			fmt.Printf("Using font family: %s\n", fontFamily)
		}
	}

	fontStyle := ""
	if fontWeightProp, exists := box.Style["font-weight"]; exists {
		if fontWeightProp.Value == "bold" || fontWeightProp.Value == "700" || fontWeightProp.Value == "800" || fontWeightProp.Value == "900" {
			fontStyle += "B"
			if r.Debug {
				fmt.Printf("Using bold font\n")
			}
		}
	}
	if fontStyleProp, exists := box.Style["font-style"]; exists {
		if fontStyleProp.Value == "italic" {
			fontStyle += "I"
			if r.Debug {
				fmt.Printf("Using italic font\n")
			}
		}
	}

	textColor := [3]int{0, 0, 0}
	if colorProp, exists := box.Style["color"]; exists {
		textColor = parseColor(colorProp.Value)
	}
	pdf.SetTextColor(textColor[0], textColor[1], textColor[2])

	pdf.SetFont(fontFamily, fontStyle, fontSize)

	text := box.Text

	align := "left"
	if alignProp, exists := box.Style["text-align"]; exists && alignProp.Value != "" {
		align = strings.ToLower(strings.TrimSpace(alignProp.Value))
	}
	dir := "ltr"
	if dirProp, exists := box.Style["direction"]; exists && dirProp.Value != "" {
		dir = strings.ToLower(strings.TrimSpace(dirProp.Value))
	}
	if align == "left" && dir == "rtl" {
		align = "right"
	}

	textWidth := pdf.GetStringWidth(text)
	var startX float64
	switch align {
	case "center":
		startX = box.X + (box.Width-textWidth)/2
	case "right", "end":
		startX = box.X + box.Width - textWidth
	default:
		startX = box.X
	}
	if startX < box.X {
		startX = box.X
	}
	if startX > box.X+box.Width {
		startX = box.X + box.Width
	}

    // Compute baseline Y. Inline tokens produced by layoutParagraphInline() have Node == nil
    // and their Y was set to (baseline - fontSize), so baseline is simply Y + fontSize.
    var baselineY float64
    if box.Node == nil {
        baselineY = box.Y + fontSize
    } else {
        // For standalone inline boxes with real nodes, derive baseline using ascent/descent and half-leading.
        paddingTop := box.PaddingTop
        paddingBottom := box.PaddingBottom
        borderTop := box.BorderTop
        borderBottom := box.BorderBottom
        contentHeight := box.Height - paddingTop - paddingBottom - borderTop - borderBottom
        if contentHeight < 0 {
            contentHeight = 0
        }
        // Approximate ascent/descent
        ascent := 0.80 * fontSize
        descent := 0.20 * fontSize
        if ascent+descent > contentHeight {
            // Clamp if line-height is smaller than font bounds
            scale := contentHeight / (ascent + descent)
            if scale < 0 {
                scale = 0
            }
            ascent *= scale
            descent *= scale
        }
        leading := contentHeight - (ascent + descent)
        if leading < 0 {
            leading = 0
        }
        baselineOffset := ascent + (leading / 2.0)
        baselineY = box.Y + borderTop + paddingTop + baselineOffset
    }

	if r.Debug {
		fmt.Printf("Rendering text: '%s' at (%.2f, %.2f) with font %s %.0fpt, color: %v\n",
			text, startX, baselineY, fontFamily, fontSize, textColor)
	}

	pdf.Text(startX, baselineY, text)

	if r.DebugDrawBoxes {
		pdf.SetDrawColor(255, 0, 0)
		pdf.SetLineWidth(0.1)

		x, y := startX, baselineY
		// Crosshair at baseline position
		pdf.Line(x-2, y, x+2, y)
		pdf.Line(x, y-2, x, y+2)

		// Approximate text box around the rendered text
		pdf.Rect(x, y-fontSize*0.8, textWidth, fontSize*1.2, "D")

		// Additional guides: baseline and bottom of line box
		// Baseline guide
		pdf.SetDrawColor(0, 180, 0)
		pdf.Line(box.X, baselineY, box.X+box.Width, baselineY)
		// Bottom of line box guide
		bottomY := box.Y + box.Height
		pdf.SetDrawColor(0, 0, 180)
		pdf.Line(box.X, bottomY, box.X+box.Width, bottomY)
	}

	if r.DebugDrawBoxes {
		pdf.SetDrawColor(255, 0, 0)
		pdf.SetLineWidth(0.1)
		pdf.Rect(box.X, box.Y, box.Width, box.Height, "D")
	}
}

// parseFloat parses a float value with a default
func parseFloat(value string, defaultValue float64) float64 {
	var result float64
	_, err := fmt.Sscanf(value, "%f", &result)
	if err != nil {
		return defaultValue
	}
	return result
}

// parseColor parses a CSS color value
func parseColor(value string) [3]int {
	if strings.HasPrefix(value, "#") {
		if r, g, b, ok := parseHexColor(value); ok {
			return [3]int{r, g, b}
		}
	}

	var r, g, b int
	if _, err := fmt.Sscanf(value, "rgb(%d,%d,%d)", &r, &g, &b); err == nil {
		return [3]int{r, g, b}
	}
	if _, err := fmt.Sscanf(value, "rgb(%d, %d, %d)", &r, &g, &b); err == nil {
		return [3]int{r, g, b}
	}

	return [3]int{0, 0, 0}
}

// parseHexColor parses #RRGGBB or #RGB into r,g,b
func parseHexColor(s string) (int, int, int, bool) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	switch len(s) {
	case 6:
		if rv, err := strconv.ParseUint(s[0:2], 16, 8); err == nil {
			if gv, err := strconv.ParseUint(s[2:4], 16, 8); err == nil {
				if bv, err := strconv.ParseUint(s[4:6], 16, 8); err == nil {
					return int(rv), int(gv), int(bv), true
				}
			}
		}
	case 3:
		r := string([]byte{s[0], s[0]})
		g := string([]byte{s[1], s[1]})
		b := string([]byte{s[2], s[2]})
		if rv, err := strconv.ParseUint(r, 16, 8); err == nil {
			if gv, err := strconv.ParseUint(g, 16, 8); err == nil {
				if bv, err := strconv.ParseUint(b, 16, 8); err == nil {
					return int(rv), int(gv), int(bv), true
				}
			}
		}
	}
	return 0, 0, 0, false
}

// renderListMarker draws the bullet/number for a list item based on current list context
func (r *Renderer) renderListMarker(pdf *fpdf.Fpdf, li *layout.BlockBox, ctx listContext) {
	fontSize := 16.0
	if ib := firstInlineChild(li); ib != nil {
		if fs, ok := ib.Style["font-size"]; ok && fs.Value != "" {
			val := strings.TrimSpace(fs.Value)
			val = strings.TrimSuffix(val, "px")
			if v := parseFloat(val, fontSize); v > 0 {
				fontSize = v
			}
		}
	}
	color := [3]int{0, 0, 0}
	if ib := firstInlineChild(li); ib != nil {
		if cprop, ok := ib.Style["color"]; ok && strings.TrimSpace(cprop.Value) != "" {
			color = parseColor(cprop.Value)
		}
	}

	cx := li.X - fontSize      // approx 1em to the left
	cy := li.Y + fontSize*0.75 // closer to visual middle of the text

	if ctx.kind == "ul" {
		style := ctx.style
		if style == "" {
			style = "disc"
		}
		// Basic sizing based on font size
		rbullet := fontSize * 0.18
		if rbullet < 1.2 {
			rbullet = 1.2
		}
		pdf.SetDrawColor(color[0], color[1], color[2])
		pdf.SetFillColor(color[0], color[1], color[2])
		switch strings.ToLower(style) {
		case "none":
			return
		case "circle":
			pdf.SetLineWidth(0.8)
			pdf.Circle(cx, cy, rbullet, "D")
		case "square":
			side := rbullet * 2
			pdf.Rect(cx-rbullet, cy-rbullet, side, side, "F")
		default: // disc
			pdf.Circle(cx, cy, rbullet, "F")
		}
		return
	}

	if ctx.kind == "ol" {
		if ctx.style == "none" {
			return
		}
		marker := ""
		switch ctx.style {
		case "decimal", "":
			marker = fmt.Sprintf("%d.", ctx.counter)
		case "lower-alpha":
			marker = fmt.Sprintf("%s.", toAlpha(ctx.counter, false))
		case "upper-alpha":
			marker = fmt.Sprintf("%s.", toAlpha(ctx.counter, true))
		default:
			marker = fmt.Sprintf("%d.", ctx.counter)
		}

		pdf.SetTextColor(color[0], color[1], color[2])
		pdf.SetFont("Helvetica", "", fontSize)

		markerWidth := pdf.GetStringWidth(marker)
		startX := li.X - markerWidth - fontSize*0.2
		if startX < 0 {
			startX = 0
		}
		pdf.Text(startX, li.Y+fontSize, marker)
		return
	}
}

// firstInlineChild returns the first InlineBox found within the list item
func firstInlineChild(b *layout.BlockBox) *layout.InlineBox {
	for _, ch := range b.Children {
		if ib, ok := ch.(*layout.InlineBox); ok {
			return ib
		}
		if bb, ok := ch.(*layout.BlockBox); ok {
			for _, gc := range bb.Children {
				if ib2, ok := gc.(*layout.InlineBox); ok {
					return ib2
				}
			}
		}
	}
	return nil
}

// toAlpha converts 1-based index to alphabetic sequence (a..z, aa..zz, ...)
func toAlpha(n int, upper bool) string {
	if n <= 0 {
		return ""
	}
	letters := []rune{}
	for n > 0 {
		n--
		rem := n % 26
		ch := rune('a' + rem)
		if upper {
			ch = rune('A' + rem)
		}
		letters = append([]rune{ch}, letters...)
		n /= 26
	}
	return string(letters)
}

// renderTableElement handles special rendering for table elements
func (r *Renderer) renderTableElement(pdf *fpdf.Fpdf, box *layout.BlockBox, tag string) {
	if !r.RenderBorders {
		return
	}

	hasBorder := false
	borderWidth := 0.0
	borderColor := [3]int{0, 0, 0}

	if width, exists := box.Style["border-width"]; exists && width.Value != "" && width.Value != "0" {
		borderWidth = parseFloat(width.Value, 0.0)
		hasBorder = borderWidth > 0
	}

	if color, exists := box.Style["border-color"]; exists && color.Value != "" {
		borderColor = parseColor(color.Value)
	}

	if hasBorder {
		pdf.SetDrawColor(borderColor[0], borderColor[1], borderColor[2])
		pdf.SetLineWidth(borderWidth)
		pdf.Rect(box.X, box.Y, box.Width, box.Height, "D")

		if r.Debug {
			fmt.Printf("Rendered border for %s: x=%.2f, y=%.2f, w=%.2f, h=%.2f\n",
				tag, box.X, box.Y, box.Width, box.Height)
		}
	}

	if tag == "th" {
		hasCustomBg := false
		if bgColor, exists := box.Style["background-color"]; exists && bgColor.Value != "" {
			hasCustomBg = true
		}

		if !hasCustomBg {
			pdf.SetFillColor(240, 240, 240)
			pdf.Rect(box.X, box.Y, box.Width, box.Height, "F")
		}
	}
}
