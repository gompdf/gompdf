package layout

import (
	"github.com/gompdf/gompdf/internal/parser/html"
	"github.com/gompdf/gompdf/internal/style"
	"strings"
)

// BlockBox represents a block-level box in the layout
type BlockBox struct {
	Node          *html.Node
	Style         style.ComputedStyle
	X             float64
	Y             float64
	Width         float64
	Height        float64
	MarginTop     float64
	MarginRight   float64
	MarginBottom  float64
	MarginLeft    float64
	PaddingTop    float64
	PaddingRight  float64
	PaddingBottom float64
	PaddingLeft   float64
	BorderTop     float64
	BorderRight   float64
	BorderBottom  float64
	BorderLeft    float64
	Children      []Box
}

// parseBoxShorthand parses CSS shorthand like:
//  - "10px"
//  - "10px 20px"
//  - "10px 15px 8px"
//  - "10px 12px 8px 6px"
// and returns (top, right, bottom, left) values.
func parseBoxShorthand(value string, containerSize float64, def float64) (float64, float64, float64, float64) {
    v := strings.TrimSpace(value)
    if v == "" {
        return def, def, def, def
    }
    parts := strings.Fields(v)
    to := func(s string) float64 { return parseLength(s, containerSize, def) }
    switch len(parts) {
    case 1:
        a := to(parts[0])
        return a, a, a, a
    case 2:
        vtb := to(parts[0])
        vrl := to(parts[1])
        return vtb, vrl, vtb, vrl
    case 3:
        t := to(parts[0])
        r := to(parts[1])
        b := to(parts[2])
        return t, r, b, r
    default:
        t := to(parts[0])
        r := to(parts[1])
        b := to(parts[2])
        l := to(parts[3])
        return t, r, b, l
    }
}

// NewBlockBox creates a new block box for an element
func NewBlockBox(node *html.Node, computedStyle style.ComputedStyle) *BlockBox {
	return &BlockBox{
		Node:     node,
		Style:    computedStyle,
		Children: []Box{},
	}
}

// Layout performs layout for this block box and its children
func (b *BlockBox) Layout(containingBlock *BlockBox) {
	if containingBlock != nil {
		b.X = containingBlock.X + containingBlock.PaddingLeft
		b.Y = containingBlock.Y + containingBlock.PaddingTop

		availableWidth := containingBlock.Width -
			containingBlock.PaddingLeft -
			containingBlock.PaddingRight

		b.Width = availableWidth
	}

	b.parseBoxModel()
	b.layoutChildren()
	b.calculateHeight()
}

// parseBoxModel parses margin, padding, and border properties
func (b *BlockBox) parseBoxModel() {
	// Margin shorthand support
	if m, ok := b.Style["margin"]; ok && strings.TrimSpace(m.Value) != "" {
		t, r, bt, l := parseBoxShorthand(m.Value, b.Width, 0)
		b.MarginTop, b.MarginRight, b.MarginBottom, b.MarginLeft = t, r, bt, l
	} else {
		b.MarginTop = parseLength(b.Style["margin-top"].Value, b.Width, 0)
		b.MarginRight = parseLength(b.Style["margin-right"].Value, b.Width, 0)
		b.MarginBottom = parseLength(b.Style["margin-bottom"].Value, b.Width, 0)
		b.MarginLeft = parseLength(b.Style["margin-left"].Value, b.Width, 0)
	}

	// Padding shorthand support
	if p, ok := b.Style["padding"]; ok && strings.TrimSpace(p.Value) != "" {
		t, r, bt, l := parseBoxShorthand(p.Value, b.Width, 0)
		b.PaddingTop, b.PaddingRight, b.PaddingBottom, b.PaddingLeft = t, r, bt, l
	} else {
		b.PaddingTop = parseLength(b.Style["padding-top"].Value, b.Width, 0)
		b.PaddingRight = parseLength(b.Style["padding-right"].Value, b.Width, 0)
		b.PaddingBottom = parseLength(b.Style["padding-bottom"].Value, b.Width, 0)
		b.PaddingLeft = parseLength(b.Style["padding-left"].Value, b.Width, 0)
	}

	b.BorderTop = parseLength(b.Style["border-top-width"].Value, b.Width, 0)
	b.BorderRight = parseLength(b.Style["border-right-width"].Value, b.Width, 0)
	b.BorderBottom = parseLength(b.Style["border-bottom-width"].Value, b.Width, 0)
	b.BorderLeft = parseLength(b.Style["border-left-width"].Value, b.Width, 0)

	b.Width = b.Width - b.PaddingLeft - b.PaddingRight - b.BorderLeft - b.BorderRight
}

// layoutChildren performs layout for all children
func (b *BlockBox) layoutChildren() {
	y := b.Y + b.MarginTop + b.BorderTop + b.PaddingTop

	for _, child := range b.Children {
		child.SetPosition(b.X+b.MarginLeft+b.BorderLeft+b.PaddingLeft, y)
		child.Layout(b)

		y += child.GetHeight() + child.GetMarginTop() + child.GetMarginBottom()
	}
}

// calculateHeight calculates the height of the block box
func (b *BlockBox) calculateHeight() {
	if heightProp, exists := b.Style["height"]; exists {
		b.Height = parseLength(heightProp.Value, b.Width, 0)
		return
	}

	if len(b.Children) > 0 {
		lastChild := b.Children[len(b.Children)-1]
		b.Height = lastChild.GetY() + lastChild.GetHeight() + lastChild.GetMarginBottom() - b.Y
	} else {
		b.Height = 0
	}

	b.Height += b.PaddingTop + b.PaddingBottom + b.BorderTop + b.BorderBottom
}

// GetX returns the x position of the box
func (b *BlockBox) GetX() float64 {
	return b.X
}

// GetY returns the y position of the box
func (b *BlockBox) GetY() float64 {
	return b.Y
}

// GetWidth returns the width of the box
func (b *BlockBox) GetWidth() float64 {
	return b.Width
}

// GetHeight returns the height of the box
func (b *BlockBox) GetHeight() float64 {
	return b.Height
}

// GetMarginTop returns the top margin of the box
func (b *BlockBox) GetMarginTop() float64 {
	return b.MarginTop
}

// GetMarginBottom returns the bottom margin of the box
func (b *BlockBox) GetMarginBottom() float64 {
	return b.MarginBottom
}

// GetMarginLeft returns the left margin of the box
func (b *BlockBox) GetMarginLeft() float64 {
	return b.MarginLeft
}

// GetMarginRight returns the right margin of the box
func (b *BlockBox) GetMarginRight() float64 {
	return b.MarginRight
}

// SetPosition sets the position of the box
func (b *BlockBox) SetPosition(x, y float64) {
	b.X = x
	b.Y = y
}

// AddChild adds a child box
func (b *BlockBox) AddChild(child Box) {
	b.Children = append(b.Children, child)
}

// GetNode returns the HTML node associated with this box
func (b *BlockBox) GetNode() *html.Node {
	return b.Node
}
