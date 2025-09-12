package layout

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"codeberg.org/go-pdf/fpdf"
	"github.com/gompdf/gompdf/internal/parser/html"
	"github.com/gompdf/gompdf/internal/style"
	xhtml "golang.org/x/net/html"
)

// Singleton PDF instance for text measurement using go-pdf/fpdf metrics
var (
	measureOnce sync.Once
	measurePDF  *fpdf.Fpdf
	measureMu   sync.Mutex
)

// orientation is a package variable to control PDF orientation for measurement
var orientation = "P" // Default to portrait

// SetMeasurementOrientation sets the orientation for text measurement
func SetMeasurementOrientation(o string) {
	if o == "L" || o == "P" {
		orientation = o
	}
}

// computeTableColumnWidths determines consistent column widths for a table row.
// It prefers widths declared on the first header row (<thead> > <tr>) if present.
// Otherwise it uses the current row's cells. It honors percentage and px widths
// and supports colspan by dividing the declared width evenly across spanned columns.
func (e *Engine) computeTableColumnWidths(row *BlockBox, totalWidth, gap float64) ([]float64, int) {
    if row == nil || row.Node == nil {
        return nil, 0
    }
    // Find the ancestor <table>
    t := row.Node.Parent
    for t != nil && !strings.EqualFold(t.Data, "table") {
        t = t.Parent
    }
    if t == nil {
        // Not inside a table
        cells := 0
        for _, ch := range row.Children {
            if bb, ok := ch.(*BlockBox); ok && bb.Node != nil {
                tag := strings.ToLower(bb.Node.Data)
                if tag == "td" || tag == "th" { cells++ }
            }
        }
        if cells == 0 { return nil, 0 }
        eff := totalWidth - gap*math.Max(0, float64(cells-1))
        w := eff / float64(cells)
        out := make([]float64, cells)
        for i := range out { out[i] = w }
        return out, cells
    }

    // Helper to scan a <tr> node's children for widths/colspans using computed styles
    type colSpec struct{ width float64; span int; hasWidth bool }
    scanTR := func(tr *html.Node) ([]colSpec, int) {
        specs := []colSpec{}
        colCount := 0
        for c := tr.FirstChild; c != nil; c = c.NextSibling {
            if c.Type != xhtml.ElementNode { continue }
            tag := strings.ToLower(c.Data)
            if tag != "th" && tag != "td" { continue }
            span := 1
            for _, a := range c.Attr {
                if strings.EqualFold(a.Key, "colspan") {
                    if n, err := strconv.Atoi(strings.TrimSpace(a.Val)); err == nil && n > 1 { span = n }
                }
            }
            wv := 0.0
            hasW := false
            if st, ok := e.styles[c]; ok {
                if wp, ok2 := st["width"]; ok2 && strings.TrimSpace(wp.Value) != "" {
                    wv = parseLength(wp.Value, totalWidth, 0)
                    if wv > 0 { hasW = true }
                }
            }
            if !hasW {
                for _, a := range c.Attr {
                    if strings.EqualFold(a.Key, "width") {
                        v := strings.TrimSpace(a.Val)
                        // Support percentage or pixels
                        if strings.HasSuffix(v, "%") || strings.HasSuffix(v, "px") {
                            wv = parseLength(v, totalWidth, 0)
                            if wv > 0 { hasW = true }
                        } else if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
                            wv = f
                            hasW = true
                        }
                    }
                }
            }
            specs = append(specs, colSpec{width: wv, span: span, hasWidth: hasW})
            colCount += span
        }
        return specs, colCount
    }

    // Prefer header row specs
    var specs []colSpec
    cols := 0
    // Locate the first <tr> within <thead>
    for n := t.FirstChild; n != nil && cols == 0; n = n.NextSibling {
        if n.Type == xhtml.ElementNode && strings.EqualFold(n.Data, "thead") {
            for tr := n.FirstChild; tr != nil && cols == 0; tr = tr.NextSibling {
                if tr.Type == xhtml.ElementNode && strings.EqualFold(tr.Data, "tr") {
                    specs, cols = scanTR(tr)
                }
            }
        }
    }
    // If no thead widths, use current row
    if cols == 0 {
        specs, cols = scanTR(row.Node)
    }
    if cols == 0 {
        return nil, 0
    }

    effective := totalWidth - gap*math.Max(0, float64(cols-1))
    colWidths := make([]float64, cols)

    // First, assign declared widths
    idx := 0
    totalDeclared := 0.0
    undeclaredCols := 0
    for _, s := range specs {
        if s.hasWidth {
            // divide width evenly across spanned columns
            share := s.width / float64(s.span)
            for j := 0; j < s.span && idx < cols; j++ {
                colWidths[idx] = share
                totalDeclared += share
                idx++
            }
        } else {
            for j := 0; j < s.span && idx < cols; j++ {
                // mark as undeclared
                undeclaredCols++
                idx++
            }
        }
    }

    remaining := effective - totalDeclared
    if remaining < 0 { remaining = 0 }
    // Count how many zeros remain
    zeroCount := 0
    for i := 0; i < cols; i++ { if colWidths[i] == 0 { zeroCount++ } }
    if zeroCount > 0 {
        each := remaining / float64(zeroCount)
        for i := 0; i < cols; i++ {
            if colWidths[i] == 0 { colWidths[i] = each }
        }
    }
    return colWidths, cols
}

func initMeasurePDF() {
	measurePDF = fpdf.New(orientation, "pt", "", "")
	measurePDF.SetFont("Helvetica", "", 12)
}

// measureTextWidth returns a font-aware width using fpdf metrics
func measureTextWidth(text string, fontSize float64, st style.ComputedStyle) float64 {
	if text == "" || fontSize <= 0 {
		return 0
	}
	measureOnce.Do(initMeasurePDF)
	measureMu.Lock()
	defer measureMu.Unlock()
	fam, sty := resolveFontFromStyle(st)
	measurePDF.SetFont(fam, sty, fontSize)
	return measurePDF.GetStringWidth(text)
}

// resolveFontFromStyle maps CSS-like style to core PDF font family and style
func resolveFontFromStyle(st style.ComputedStyle) (string, string) {
	family := "Helvetica"
	if ff, ok := st["font-family"]; ok && strings.TrimSpace(ff.Value) != "" {
		first := strings.Split(ff.Value, ",")[0]
		first = strings.TrimSpace(strings.Trim(first, "'\""))
		switch strings.ToLower(first) {
		case "arial", "helvetica", "sans-serif":
			family = "Helvetica"
		case "times", "times new roman", "serif":
			family = "Times"
		case "courier", "courier new", "monospace":
			family = "Courier"
		}
	}
	styleStr := ""
	if fw, ok := st["font-weight"]; ok {
		v := strings.TrimSpace(fw.Value)
		if v == "bold" || v == "700" || v == "800" || v == "900" {
			styleStr += "B"
		}
	}
	if fs, ok := st["font-style"]; ok {
		if strings.TrimSpace(fs.Value) == "italic" {
			styleStr += "I"
		}
	}
	return family, styleStr
}

// Options represents options for the layout engine
type Options struct {
	Width  float64
	Height float64
	DPI    float64
}

// layoutTableRow arranges the direct children <td>/<th> of a <tr> horizontally
// with either explicit CSS widths or equal-width distribution, and sets the row height
// to the max of the cell heights. It also shifts cell descendants when repositioning.
func (e *Engine) layoutTableRow(row *BlockBox) {
	if row == nil {
		return
	}
	// Collect cell boxes
	var cells []*BlockBox
	for _, ch := range row.Children {
		if bb, ok := ch.(*BlockBox); ok && bb.Node != nil {
			tag := strings.ToLower(bb.Node.Data)
			if tag == "td" || tag == "th" {
				cells = append(cells, bb)
			}
		}
	}
	if len(cells) == 0 {
		return
	}

	totalWidth := row.Width
	// Determine horizontal gap between cells. Prefer explicit border-spacing from the nearest table ancestor
	// or the row itself. If not set, try 'gap' or 'column-gap'. If nothing set, default to 0.
	cellGapX := 0.0
	// Helper to extract spacing from a style map
	extractGap := func(st style.ComputedStyle) (float64, bool) {
		if st == nil {
			return 0, false
		}
		if bs, ok := st["border-spacing"]; ok && strings.TrimSpace(bs.Value) != "" {
			parts := strings.Fields(bs.Value)
			if len(parts) > 0 {
				return parseLength(parts[0], row.Width, 0), true
			}
		}
		if g, ok := st["gap"]; ok && strings.TrimSpace(g.Value) != "" {
			return parseLength(g.Value, row.Width, 0), true
		}
		if cg, ok := st["column-gap"]; ok && strings.TrimSpace(cg.Value) != "" {
			return parseLength(cg.Value, row.Width, 0), true
		}
		return 0, false
	}
	// 1) Check the row's own style
	if v, ok := extractGap(row.Style); ok {
		cellGapX = v
	} else if row != nil && row.Node != nil {
		// 2) Walk up to find the nearest table's style
		findTable := row.Node.Parent
		for findTable != nil && !strings.EqualFold(findTable.Data, "table") {
			findTable = findTable.Parent
		}
		if findTable != nil {
			if st, ok := e.styles[findTable]; ok {
				if v, ok2 := extractGap(st); ok2 {
					cellGapX = v
				}
			}
		}
	}
	if cellGapX < 0 { cellGapX = 0 }
	// Build per-column widths for the table, respecting widths from a header row when present
    colWidths, colCount := e.computeTableColumnWidths(row, totalWidth, cellGapX)
    if colCount == 0 {
        // Fallback: treat each cell as one column
        colCount = len(cells)
        effective := totalWidth - cellGapX*math.Max(0, float64(colCount-1))
        w := 0.0
        if colCount > 0 { w = effective / float64(colCount) }
        colWidths = make([]float64, colCount)
        for i := 0; i < colCount; i++ { colWidths[i] = w }
    }

    // Column positions
    colX := make([]float64, colCount)
    cx := row.X
    for i := 0; i < colCount; i++ {
        colX[i] = cx
        cx += colWidths[i]
        if i < colCount-1 { cx += cellGapX }
    }

    // Place cells using colspan
    x := row.X
    maxH := 0.0
    colIdx := 0
    for _, cell := range cells {
        span := 1
        if cell.Node != nil {
            for _, a := range cell.Node.Attr {
                if strings.EqualFold(a.Key, "colspan") {
                    if n, err := strconv.Atoi(strings.TrimSpace(a.Val)); err == nil && n > 1 {
                        span = n
                    }
                }
            }
        }

        if colIdx >= len(colX) {
            colIdx = len(colX) - 1
        }

        // Compute width across spanned columns + inner gaps
        w := 0.0
        for j := 0; j < span && colIdx+j < len(colWidths); j++ {
            w += colWidths[colIdx+j]
        }
        if span > 1 {
            w += cellGapX * float64(span-1)
        }

        oldX, oldY := cell.X, cell.Y
        newX, newY := colX[colIdx], row.Y
        dx, dy := newX-oldX, newY-oldY

        // Apply position and width
        cell.X = newX
        cell.Y = newY
        cell.Width = w

        e.shiftDescendants(cell, dx, dy)

        if len(cell.Children) > 0 {
            last := cell.Children[len(cell.Children)-1]
            calcH := last.GetY() + last.GetHeight() - cell.Y
            if calcH < 20 {
                calcH = 20
            }
            cell.Height = calcH
        } else if cell.Height == 0 {
            cell.Height = 20
        }

        if cell.Height > maxH {
            maxH = cell.Height
        }

        // Advance by spanned columns
        x = newX + w
        if colIdx+span < colCount {
            x += cellGapX
        }
        colIdx += span
    }
	if maxH < 20 {
		maxH = 20
	}
	row.Height = maxH
}

// shiftDescendants shifts all descendant boxes of the given block by (dx, dy)
func (e *Engine) shiftDescendants(b *BlockBox, dx, dy float64) {
	if b == nil {
		return
	}
	for _, ch := range b.Children {
		ch.SetPosition(ch.GetX()+dx, ch.GetY()+dy)
		switch c := ch.(type) {
		case *BlockBox:
			e.shiftDescendants(c, dx, dy)
		case *InlineBox:
			for _, gc := range c.Children {
				gc.SetPosition(gc.GetX()+dx, gc.GetY()+dy)
				if bb, ok := gc.(*BlockBox); ok {
					e.shiftDescendants(bb, dx, dy)
				}
			}
		}
	}
}

// Engine handles the layout process
type Engine struct {
	options Options
	styles  map[*html.Node]style.ComputedStyle
	Debug   bool
	Width   float64
	Height  float64
	Margin  float64
}

// NewEngine creates a new layout engine
func NewEngine() *Engine {
	return &Engine{
		options: Options{
			Width:  595.28, // Default A4 width in points
			Height: 841.89, // Default A4 height in points
			DPI:    96,     // Default DPI
		},
		styles: make(map[*html.Node]style.ComputedStyle),
		Debug:  true,
		Width:  595.28, // Default A4 width in points
		Height: 841.89, // Default A4 height in points
		Margin: 50,     // Default margin in points
	}
}

// SetOptions sets the options for the layout engine
func (e *Engine) SetOptions(options Options) {
	e.options = options
	e.Width = options.Width
	e.Height = options.Height
	if e.Margin == 0 {
		e.Margin = 50 // Default margin
	}
}

// SetStyles sets the computed styles for the layout engine
func (e *Engine) SetStyles(styles map[*html.Node]style.ComputedStyle) {
	e.styles = styles
}

// Layout creates a layout tree from a document
func (e *Engine) Layout(doc interface{}) *BlockBox {
	// Create the root box
	rootBox := &BlockBox{
		X:        e.Margin,
		Y:        e.Margin,
		Width:    e.Width - (2 * e.Margin),
		Height:   e.Height - (2 * e.Margin),
		Children: []Box{},
	}

	if e.Debug {
		fmt.Printf("Creating layout with root box: x=%.2f, y=%.2f, width=%.2f, height=%.2f\n",
			rootBox.X, rootBox.Y, rootBox.Width, rootBox.Height)
	}

	var htmlNode *html.Node
	if htmlDoc, ok := doc.(*html.Document); ok {
		if e.Debug {
			fmt.Println("Processing standard HTML document")
		}
		htmlNode = htmlDoc.Root
	} else if node, ok := doc.(*html.Node); ok {
		if e.Debug {
			fmt.Println("Processing HTML node directly")
		}
		htmlNode = node
	} else {
		if e.Debug {
			fmt.Printf("Unknown document type: %T", doc)
		}
		return rootBox
	}

	if e.Debug {
		e.debugDocumentStructure(htmlNode, 0)
	}
	var htmlElement, bodyElement *html.Node

	if htmlNode.Type == xhtml.DocumentNode { // DocumentNode
		for child := htmlNode.FirstChild; child != nil; child = child.NextSibling {
			if child.Type == xhtml.ElementNode && strings.ToLower(child.Data) == "html" {
				htmlElement = child
				break
			}
		}
	} else if htmlNode.Type == xhtml.ElementNode && strings.ToLower(htmlNode.Data) == "html" {
		htmlElement = htmlNode
	}

	if htmlElement != nil {
		if e.Debug {
			fmt.Println("Found HTML element, looking for BODY")
		}

		// Look for BODY element in the HTML element's children
		for child := htmlElement.FirstChild; child != nil; child = child.NextSibling {
			if child.Type == xhtml.ElementNode && strings.ToLower(child.Data) == "body" {
				bodyElement = child
				break
			}
		}
	} else {
		// If we didn't find the HTML element, look for BODY directly
		if e.Debug {
			fmt.Println("No HTML element found, looking for BODY directly")
		}

		// Look for BODY element in the document's children
		for child := htmlNode.FirstChild; child != nil; child = child.NextSibling {
			if child.Type == xhtml.ElementNode && strings.ToLower(child.Data) == "body" {
				bodyElement = child
				break
			}
		}
	}

	// Create HTML box if found
	var htmlBox *BlockBox
	if htmlElement != nil {
		htmlBox = &BlockBox{
			Node:     htmlElement,
			X:        rootBox.X,
			Y:        rootBox.Y,
			Width:    rootBox.Width,
			Height:   rootBox.Height,
			Children: []Box{},
		}

		// Add HTML box to root
		rootBox.Children = append(rootBox.Children, htmlBox)

		if e.Debug {
			fmt.Println("Created HTML box")
		}
	} else {
		// Use root box as HTML box
		htmlBox = rootBox

		if e.Debug {
			fmt.Println("Using root box as HTML box")
		}
	}

	// Create BODY box if found
	var bodyBox *BlockBox
	if bodyElement != nil {
		bodyBox = &BlockBox{
			Node:     bodyElement,
			X:        htmlBox.X,
			Y:        htmlBox.Y,
			Width:    htmlBox.Width,
			Height:   htmlBox.Height,
			Children: []Box{},
		}

		// Add BODY box to HTML box
		htmlBox.Children = append(htmlBox.Children, bodyBox)

		if e.Debug {
			fmt.Println("Created BODY box")
		}

		// Process all children of the BODY element
		for child := bodyElement.FirstChild; child != nil; child = child.NextSibling {
			e.processNode(child, bodyBox, 1)
		}
	} else {
		// Use HTML box as BODY box
		bodyBox = htmlBox

		if e.Debug {
			fmt.Println("No BODY element found, using HTML box as BODY box")
		}

		// Process all children of the HTML element or document
		var contentNode *html.Node
		if htmlElement != nil {
			contentNode = htmlElement
		} else {
			contentNode = htmlNode
		}

		// Process all children, skipping HEAD element
		for child := contentNode.FirstChild; child != nil; child = child.NextSibling {
			// Skip HEAD element and its children
			if child.Type == xhtml.ElementNode && strings.ToLower(child.Data) == "head" {
				if e.Debug {
					fmt.Println("Skipping HEAD element")
				}
				continue
			}

			// Skip text nodes that are just whitespace
			if child.Type == xhtml.TextNode && strings.TrimSpace(child.Data) == "" {
				continue
			}

			// Process other children
			e.processNode(child, bodyBox, 1)
		}
	}

	// Adjust box heights based on children
	if len(bodyBox.Children) > 0 {
		lastChild := bodyBox.Children[len(bodyBox.Children)-1]
		bodyBox.Height = lastChild.GetY() + lastChild.GetHeight() - bodyBox.Y
	}

	if htmlBox != rootBox && len(htmlBox.Children) > 0 {
		lastChild := htmlBox.Children[len(htmlBox.Children)-1]
		htmlBox.Height = lastChild.GetY() + lastChild.GetHeight() - htmlBox.Y
	}

	// Debug output
	if e.Debug {
		fmt.Printf("Final layout tree:\n")
		fmt.Printf("Root box has %d children\n", len(rootBox.Children))

		for i, child := range rootBox.Children {
			fmt.Printf("  Child %d: type=%T, x=%.2f, y=%.2f, width=%.2f, height=%.2f\n",
				i, child, child.GetX(), child.GetY(), child.GetWidth(), child.GetHeight())

			// If it's a block box, check its children too
			if blockChild, ok := child.(*BlockBox); ok {
				fmt.Printf("    Block child has %d children\n", len(blockChild.Children))

				for j, grandchild := range blockChild.Children {
					fmt.Printf("      Grandchild %d: type=%T, x=%.2f, y=%.2f, width=%.2f, height=%.2f\n",
						j, grandchild, grandchild.GetX(), grandchild.GetY(), grandchild.GetWidth(), grandchild.GetHeight())
				}
			}
		}
	}

	return rootBox
}

// processNode processes an HTML node and creates appropriate layout boxes
func (e *Engine) processNode(node *html.Node, parentBox *BlockBox, depth int) {
	if node == nil {
		if e.Debug {
			fmt.Printf("Skipping nil node\n")
		}
		return
	}

	// Debug output
	if e.Debug {
		indent := strings.Repeat("  ", depth)
		fmt.Printf("%sProcessing node: type=%d, data='%s', parent=%T\n",
			indent, node.Type, node.Data, parentBox)

		// Print attributes for element nodes
		if node.Type == xhtml.ElementNode { // ElementNode
			for _, attr := range node.Attr {
				fmt.Printf("%s  Attr: %s='%s'\n", indent, attr.Key, attr.Val)
			}
		}
	}

	// Handle different node types
	if node.Type == xhtml.CommentNode { // CommentNode
		if e.Debug {
			fmt.Printf("Skipping comment node\n")
		}
		return
	}

	if node.Type == xhtml.DoctypeNode { // DoctypeNode
		if e.Debug {
			fmt.Printf("Skipping doctype node\n")
		}
		return
	}

	if node.Type == xhtml.DocumentNode { // DocumentNode
		if e.Debug {
			fmt.Printf("Processing document node\n")
		}
		// Process all children of the document node
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			e.processNode(child, parentBox, depth+1)
		}
		return
	}

	if node.Type == xhtml.TextNode { // TextNode
		if strings.TrimSpace(node.Data) == "" {
			if e.Debug {
				fmt.Printf("Skipping whitespace-only text node\n")
			}
			return
		}

		if e.Debug {
			fmt.Printf("Processing text node: '%s'\n", strings.TrimSpace(node.Data))
		}

		effectiveStyle := style.ComputedStyle{}
		if parentBox != nil && parentBox.GetNode() != nil {
			if ps, ok := e.styles[parentBox.GetNode()]; ok {
				effectiveStyle = ps
				if e.Debug {
					fmt.Printf("Found parent box style for text node: %v\n", effectiveStyle)
				}
			}
		}
		if node.Parent != nil {
			if ps, ok := e.styles[node.Parent]; ok {
				merged := make(style.ComputedStyle)
				for k, v := range effectiveStyle {
					merged[k] = v
				}
				for k, v := range ps {
					merged[k] = v
				}
				effectiveStyle = merged
				if e.Debug {
					fmt.Printf("Merged parent element style for text node: %v\n", ps)
				}
			}
		}

		// Apply default text styles if needed
		if _, ok := effectiveStyle["color"]; !ok {
			effectiveStyle["color"] = style.StyleProperty{
				Name:  "color",
				Value: "#000000",
			}
		}

		if _, ok := effectiveStyle["font-size"]; !ok {
			effectiveStyle["font-size"] = style.StyleProperty{
				Name:  "font-size",
				Value: "16px",
			}
		}

		// Determine font-size to size the inline text box correctly
		fontSizeVal := "16"
		if fs, ok := effectiveStyle["font-size"]; ok && fs.Value != "" {
			fontSizeVal = fs.Value
		}
		fontSize := parseLength(fontSizeVal, 0, 16)

		// Determine vertical position below the previous sibling; include parent padding/border for first line
		childY := parentBox.Y
		if len(parentBox.Children) > 0 {
			last := parentBox.Children[len(parentBox.Children)-1]
			childY = last.GetY() + last.GetHeight()
		} else {
			childY = parentBox.Y + parentBox.PaddingTop + parentBox.BorderTop
		}

		lineHeight := 1.25 * fontSize
		if lhProp, ok := effectiveStyle["line-height"]; ok && strings.TrimSpace(lhProp.Value) != "" {
			lineHeight = parseLength(lhProp.Value, 0, lineHeight)
		}

		// Respect parent content box (padding/border) for X/Width so padding works in TD/TH
		contentX := parentBox.X + parentBox.PaddingLeft + parentBox.BorderLeft
		contentW := parentBox.Width - parentBox.PaddingLeft - parentBox.PaddingRight - parentBox.BorderLeft - parentBox.BorderRight
		if contentW < 0 {
			contentW = 0
		}
		inlineBox := &InlineBox{
			Node:   node,
			Style:  effectiveStyle, // Use merged effective style (captures strong/em)
			X:      contentX,
			Y:      childY,
			Width:  contentW,
			Height: lineHeight, // add leading to avoid clipping descenders
			Text:   strings.TrimSpace(node.Data),
		}

		parentBox.Children = append(parentBox.Children, inlineBox)

		if e.Debug {
			fmt.Printf("Created inline box for text: x=%.2f, y=%.2f, width=%.2f, height=%.2f, text='%s'\n",
				inlineBox.X, inlineBox.Y, inlineBox.Width, inlineBox.Height, inlineBox.Text)
		}
		return
	}

	if node.Type == xhtml.ElementNode { // ElementNode
		// Skip script and style elements
		if strings.ToLower(node.Data) == "script" || strings.ToLower(node.Data) == "style" {
			if e.Debug {
				fmt.Printf("Skipping %s element\n", node.Data)
			}
			return
		}

		tagName := strings.ToLower(node.Data)
		isBlock := e.isBlockTag(tagName)

		var nodeStyle style.ComputedStyle // Default empty style

		parentStyle := style.ComputedStyle{}
		if parentBox != nil && parentBox.GetNode() != nil {
			if ps, ok := e.styles[parentBox.GetNode()]; ok {
				parentStyle = ps
			}
		}

		thisNodeStyle, hasStyle := e.styles[node]

		if hasStyle {
			nodeStyle = e.mergeStyles(parentStyle, thisNodeStyle)
		} else {
			nodeStyle = parentStyle
		}

		if display, ok := nodeStyle["display"]; ok {
			switch display.Value {
			case "block", "flex", "grid":
				isBlock = true
			case "inline", "inline-block":
				isBlock = false
			}
		}
		if e.Debug {
			fmt.Printf("Element '%s' is block: %v\n", node.Data, isBlock)
		}

		childContainer := parentBox

		// Special-case inline replaced element: <img>
		if tagName == "img" {
			// Determine merged style for the element
			nodeStyle := style.ComputedStyle{}
			parentStyle := style.ComputedStyle{}
			if parentBox != nil && parentBox.GetNode() != nil {
				if ps, ok := e.styles[parentBox.GetNode()]; ok {
					parentStyle = ps
				}
			}
			if thisNodeStyle, ok := e.styles[node]; ok {
				nodeStyle = e.mergeStyles(parentStyle, thisNodeStyle)
			} else {
				nodeStyle = parentStyle
			}

			// Position just like inline
			childY := parentBox.Y
			if len(parentBox.Children) > 0 {
				last := parentBox.Children[len(parentBox.Children)-1]
				childY = last.GetY() + last.GetHeight()
			}

			// Extract src attribute
			src := ""
			for _, a := range node.Attr {
				if strings.EqualFold(a.Key, "src") {
					src = a.Val
					break
				}
			}

			img := &ImageBox{
				Node:  node,
				Style: nodeStyle,
				X:     parentBox.X,
				Y:     childY,
				Src:   src,
			}
			// Let the image compute its own size based on styles/defaults
			img.Layout(parentBox)
			parentBox.Children = append(parentBox.Children, img)
			if e.Debug {
				fmt.Printf("Created image box: src='%s' at x=%.2f y=%.2f w=%.2f h=%.2f\n", src, img.X, img.Y, img.Width, img.Height)
			}
			return
		}

		if isBlock {
			childY := parentBox.Y
			if len(parentBox.Children) > 0 {
				last := parentBox.Children[len(parentBox.Children)-1]
				childY = last.GetY() + last.GetHeight()
			}
			blockBox := &BlockBox{
				Node:     node,
				Style:    nodeStyle,
				X:        parentBox.X,
				Y:        childY,
				Width:    parentBox.Width,
				Height:   30, // Default height, will be adjusted later
				Children: []Box{},
			}

			parentBox.Children = append(parentBox.Children, blockBox)
			childContainer = blockBox

			if e.Debug {
				fmt.Printf("Created block box for element %s: x=%.2f, y=%.2f, width=%.2f, height=%.2f\n",
					node.Data, blockBox.X, blockBox.Y, blockBox.Width, blockBox.Height)
			}
			if strings.EqualFold(node.Data, "p") {
				e.layoutParagraphInline(node, blockBox, nodeStyle)
				return
			}
			// Lay out table cell inline content with wrapping just like a paragraph
			// if strings.EqualFold(node.Data, "td") || strings.EqualFold(node.Data, "th") {
			// 	e.layoutParagraphInline(node, blockBox, nodeStyle)
			// 	return
			// }
		} else {
			childY := parentBox.Y
			if len(parentBox.Children) > 0 {
				last := parentBox.Children[len(parentBox.Children)-1]
				childY = last.GetY() + last.GetHeight()
			}
			inlineBox := &InlineBox{
				Node:     node,
				Style:    nodeStyle,
				X:        parentBox.X,
				Y:        childY,
				Width:    parentBox.Width,
				Height:   20,
				Text:     "", // Empty for element nodes
				Children: []Box{},
			}

			parentBox.Children = append(parentBox.Children, inlineBox)

			if e.Debug {
				fmt.Printf("Created inline box for element %s: x=%.2f, y=%.2f, width=%.2f, height=%.2f\n",
					node.Data, inlineBox.X, inlineBox.Y, inlineBox.Width, inlineBox.Height)
			}
		}

		for child := node.FirstChild; child != nil; child = child.NextSibling {
			e.processNode(child, childContainer, depth+1)
		}
		didRowLayout := false
		if childContainer != parentBox && strings.EqualFold(node.Data, "tr") {
			e.layoutTableRow(childContainer)
			didRowLayout = true
			if e.Debug {
				fmt.Printf("Applied horizontal layout for table row\n")
			}
		}

		if !didRowLayout {
			if childContainer != parentBox && len(childContainer.Children) > 0 {
				lastChild := childContainer.Children[len(childContainer.Children)-1]
				childContainer.Height = lastChild.GetY() + lastChild.GetHeight() - childContainer.Y

				if e.Debug {
					fmt.Printf("Adjusted block box height for %s: height=%.2f\n", node.Data, childContainer.Height)
				}
			} else if childContainer != parentBox {
				childContainer.Height = 20

				if e.Debug {
					fmt.Printf("Set minimum height for empty block box %s: height=%.2f\n", node.Data, childContainer.Height)
				}
			}
		}
	}

	if len(parentBox.Children) == 0 {
		return
	}
	lastChild := parentBox.Children[len(parentBox.Children)-1]
	if parentBox.Y+parentBox.Height < lastChild.GetY()+lastChild.GetHeight() {
		parentBox.Height = lastChild.GetY() + lastChild.GetHeight() - parentBox.Y
	}
}

// mergeStyles combines parent and child styles with child styles taking precedence
func (e *Engine) mergeStyles(parentStyle, childStyle style.ComputedStyle) style.ComputedStyle {
	mergedStyle := make(style.ComputedStyle)

	for key, value := range parentStyle {
		mergedStyle[key] = value
	}

	for key, value := range childStyle {
		mergedStyle[key] = value
	}

	return mergedStyle
}

// isBlockTag reports whether a tag name is treated as block-level
func (e *Engine) isBlockTag(tag string) bool {
	switch strings.ToLower(tag) {
	case "div", "p", "h1", "h2", "h3", "h4", "h5", "h6",
		"ul", "ol", "li", "table", "thead", "tbody", "tfoot",
		"tr", "td", "th", "header", "footer", "section", "article",
		"form", "fieldset", "hr", "blockquote", "address", "main",
		"nav", "aside":
		return true
	default:
		return false
	}
}

// inlineRun represents a contiguous text run with a specific style
type inlineRun struct {
	text  string
	style style.ComputedStyle
}

// layoutParagraphInline lays out inline content of a <p> with wrapping and shared baseline per line
func (e *Engine) layoutParagraphInline(pNode *html.Node, container *BlockBox, baseStyle style.ComputedStyle) {
	runs := []inlineRun{}
	e.collectInlineRuns(pNode, baseStyle, &runs)

	normalizeInlineRuns(&runs)

	type tkn struct {
		text    string
		style   style.ComputedStyle
		width   float64
		isSpace bool    // Whether this token is a space
		drop    bool    // Whether to drop this token during layout
		fs      float64 // Font size
		lh      float64 // Line height
	}

	raw := []tkn{}
	for _, run := range runs {
		if run.text == "" {
			continue
		}
		fs := 16.0
		if prop, ok := run.style["font-size"]; ok && strings.TrimSpace(prop.Value) != "" {
			fs = parseLength(prop.Value, 0, 16)
		}
		lh := 1.2 * fs
		if prop, ok := run.style["line-height"]; ok && strings.TrimSpace(prop.Value) != "" {
			lh = parseLength(prop.Value, 0, 1.2*fs)
		}

		tokens := splitTokens(run.text)
		for _, t := range tokens {
			isSpace := isAllSpace(t)
			w := 0.0
			if isSpace {
				// Measure space width using font metrics to avoid over/under spacing
				w = measureTextWidth(" ", fs, run.style)
			} else {
				t = strings.TrimSpace(t)
				if t != "" {
					w = measureTextWidth(t, fs, run.style)
				}
			}
			if t != "" {
				raw = append(raw, tkn{
					text:    t,
					isSpace: isSpace,
					style:   run.style,
					fs:      fs,
					lh:      lh,
					width:   w,
				})
			}
		}
	}

	// Start within the content box of the container (respect padding/border)
	startX := container.X + container.PaddingLeft + container.BorderLeft
	maxWidth := container.Width
	curY := container.Y + container.PaddingTop + container.BorderTop
	line := []tkn{}
	lineWidth := 0.0
	maxAscent := 0.0
	maxDescent := 0.0

	emitLine := func() {
		if len(line) == 0 {
			return
		}
		if len(line) > 0 && line[len(line)-1].isSpace {
			line[len(line)-1].drop = true
		}
		maxAscent, maxDescent = 0, 0
		for _, tk := range line {
			if tk.drop {
				continue
			}
			if tk.fs > maxAscent {
				maxAscent = tk.fs
			}
			if tk.lh-tk.fs > maxDescent {
				maxDescent = tk.lh - tk.fs
			}
		}
		baselineY := curY + maxAscent
		// Compute alignment offset for the entire line
		// total lineWidth has been accumulated while building the line
		offsetX := 0.0
		align := "left"
		if prop, ok := container.Style["text-align"]; ok && strings.TrimSpace(prop.Value) != "" {
			align = strings.ToLower(strings.TrimSpace(prop.Value))
		}
		if align == "right" || align == "end" {
			if lineWidth < maxWidth { offsetX = maxWidth - lineWidth }
		} else if align == "center" {
			if lineWidth < maxWidth { offsetX = (maxWidth - lineWidth) / 2 }
		}
		x := offsetX
		for _, tk := range line {
			if tk.drop {
				continue
			}
			// Use the precomputed token width (font-aware for both words and spaces)
			w := tk.width
			ib := &InlineBox{
				Node:   nil,
				Style:  tk.style,
				X:      startX + x,
				Y:      baselineY - tk.fs,
				Width:  w,
				Height: maxAscent + maxDescent,
				Text:   map[bool]string{true: " ", false: tk.text}[tk.isSpace],
			}
			container.Children = append(container.Children, ib)
			x += w
		}
		curY += (maxAscent + maxDescent)
		line = line[:0]
		lineWidth = 0
	}

	pendingSpace := false
	for i := 0; i < len(raw); i++ {
		tk := raw[i]
		if tk.isSpace {
			if !pendingSpace {
				pendingSpace = true
			}
			continue
		}

		if pendingSpace {
			if r, _ := utf8.DecodeRuneInString(tk.text); r != utf8.RuneError && strings.ContainsRune(",.;:!?)]}»", r) {
			} else {
				fs, lh := tk.fs, tk.lh
				// Use font-aware space width
				spw := measureTextWidth(" ", fs, tk.style)
				if lineWidth+spw+tk.width > maxWidth && len(line) > 0 {
					emitLine()
					pendingSpace = false
					continue
				}
				if len(line) > 0 {
					line = append(line, tkn{text: " ", style: tk.style, fs: fs, lh: lh, width: spw, isSpace: true})
					lineWidth += spw
				}
			}
			pendingSpace = false
		}

		if tk.width > maxWidth { // extremely long word: place on new line anyway
			if len(line) > 0 {
				emitLine()
			}
		} else if lineWidth+tk.width > maxWidth && len(line) > 0 {
			emitLine()
		}

		line = append(line, tk)
		lineWidth += tk.width
	}
	if len(line) > 0 {
		emitLine()
	}

	if len(container.Children) > 0 {
		last := container.Children[len(container.Children)-1]
		container.Height = (last.GetY() + last.GetHeight()) - container.Y
	} else {
		container.Height = 0
	}
}

// collectInlineRuns traverses children, collecting text with merged inline styles
func (e *Engine) collectInlineRuns(n *html.Node, inherited style.ComputedStyle, out *[]inlineRun) {
	if n == nil {
		return
	}
	for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
		switch ch.Type {
		case xhtml.TextNode:
			txt := ch.Data
			if txt == "" {
				continue
			}

			isFirstNode := ch.PrevSibling == nil || (ch.PrevSibling.Type != xhtml.TextNode && ch.PrevSibling.Type != xhtml.ElementNode)
			isLastNode := ch.NextSibling == nil || (ch.NextSibling.Type != xhtml.TextNode && ch.NextSibling.Type != xhtml.ElementNode)

			txt = normalizeWhitespace(txt)

			if isFirstNode {
				txt = strings.TrimLeftFunc(txt, unicode.IsSpace)
			}
			if isLastNode {
				txt = strings.TrimRightFunc(txt, unicode.IsSpace)
			}

			if txt == "" {
				continue
			}

			eff := make(style.ComputedStyle)
			for k, v := range inherited {
				eff[k] = v
			}
			if ch.Parent != nil {
				if ps, ok := e.styles[ch.Parent]; ok {
					for k, v := range ps {
						eff[k] = v
					}
				}
			}
			if _, ok := eff["color"]; !ok {
				eff["color"] = style.StyleProperty{Name: "color", Value: "#000000"}
			}
			if _, ok := eff["font-size"]; !ok {
				eff["font-size"] = style.StyleProperty{Name: "font-size", Value: "16px"}
			}
			*out = append(*out, inlineRun{text: txt, style: eff})
		case xhtml.ElementNode:
			tag := strings.ToLower(ch.Data)
			if e.isBlockTag(tag) {
				// stop at block-level elements inside a paragraph
				continue
			}
			eff := inherited
			if thisStyle, ok := e.styles[ch]; ok {
				eff = e.mergeStyles(inherited, thisStyle)
			}
			e.collectInlineRuns(ch, eff, out)
		default:
			// ignore
		}
	}
}

// splitTokens splits text into tokens of words and spaces
func splitTokens(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}

	tokens := []string{}
	var cur []rune
	var curIsSpace *bool

	for _, r := range s {
		isSp := unicode.IsSpace(r)
		if curIsSpace == nil {
			curIsSpace = new(bool)
			*curIsSpace = isSp
		}

		switch {
		case *curIsSpace != isSp:
			if len(cur) > 0 {
				tokens = append(tokens, string(cur))
			}
			cur = []rune{}

			if isSp {
				cur = append(cur, ' ')
			} else {
				cur = append(cur, r)
			}
			*curIsSpace = isSp
		case isSp:
			if len(cur) == 0 {
				cur = append(cur, ' ')
			}
		default:
			cur = append(cur, r)
		}
	}
	if len(cur) > 0 {
		tokens = append(tokens, string(cur))
	}
	return tokens
}

func isAllSpace(s string) bool {
	for _, r := range s {
		if !unicode.IsSpace(r) {
			return false
		}
	}
	return true
}

// normalizeWhitespace preserves single spaces but collapses multiple consecutive spaces
// into a single space. Unlike strings.TrimSpace, it doesn't remove leading/trailing spaces.
func normalizeWhitespace(s string) string {
	var result []rune
	var lastWasSpace bool

	for _, r := range s {
		isSpace := unicode.IsSpace(r)

		if isSpace {
			if !lastWasSpace {
				result = append(result, ' ')
			}
			lastWasSpace = true
		} else {
			result = append(result, r)
			lastWasSpace = false
		}
	}

	return string(result)
}

// normalizeInlineRuns ensures proper spacing between inline elements
// by examining adjacent runs and adding spaces where needed
func normalizeInlineRuns(runs *[]inlineRun) {
	if runs == nil || len(*runs) <= 1 {
		return
	}

	result := make([]inlineRun, 0, len(*runs))

	for i, run := range *runs {
		result = append(result, run)

		if i < len(*runs)-1 {
			currentEndsWithSpace := len(run.text) > 0 && unicode.IsSpace(rune(run.text[len(run.text)-1]))
			nextStartsWithSpace := len((*runs)[i+1].text) > 0 && unicode.IsSpace(rune((*runs)[i+1].text[0]))
			if !currentEndsWithSpace && !nextStartsWithSpace {
				if len(run.text) > 0 && len((*runs)[i+1].text) > 0 {
					spaceRun := inlineRun{
						text:  " ",
						style: run.style,
					}
					result = append(result, spaceRun)
				}
			}
		}
	}

	*runs = result
}

// debugDocumentStructure prints the HTML document structure for debugging
func (e *Engine) debugDocumentStructure(node *html.Node, depth int) {
	if node == nil {
		return
	}

	indent := strings.Repeat("  ", depth)
	switch node.Type {
	case xhtml.ElementNode: // ElementNode
		fmt.Printf("%s[ElementNode] %s\n", indent, node.Data)
	case xhtml.TextNode: // TextNode
		fmt.Printf("%s[TextNode] %s\n", indent, node.Data)
	case xhtml.DocumentNode: // DocumentNode
		fmt.Printf("%s[DocumentNode] %s\n", indent, node.Data)
	case xhtml.CommentNode: // CommentNode
		fmt.Printf("%s[CommentNode] %s\n", indent, node.Data)
	case xhtml.DoctypeNode: // DoctypeNode
		fmt.Printf("%s[DoctypeNode] %s\n", indent, node.Data)
	default:
		fmt.Printf("%s[unknown] %s\n", indent, node.Data)
	}

	for child := node.FirstChild; child != nil; child = child.NextSibling {
		e.debugDocumentStructure(child, depth+1)
	}
}
