package layout

import (
	"math"
	"strconv"
	"strings"

	"github.com/gompdf/gompdf/internal/parser/html"
	"github.com/gompdf/gompdf/internal/style"
)

// InlineBox implements the Box interface for inline-level elements

// InlineBox represents an inline-level box in the layout
type InlineBox struct {
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
	Text          string
}

// NewInlineBox creates a new inline box for an element
func NewInlineBox(node *html.Node, computedStyle style.ComputedStyle) *InlineBox {
	return &InlineBox{
		Node:     node,
		Style:    computedStyle,
		Children: []Box{},
	}
}

// NewTextBox creates a new inline box for text content
func NewTextBox(node *html.Node, computedStyle style.ComputedStyle, text string) *InlineBox {
	return &InlineBox{
		Node:  node,
		Style: computedStyle,
		Text:  text,
	}
}

// Layout performs layout for this inline box and its children
func (b *InlineBox) Layout(containingBlock *BlockBox) {
	if containingBlock != nil {
		if b.X == 0 && b.Y == 0 {
			b.X = containingBlock.X + containingBlock.PaddingLeft
			b.Y = containingBlock.Y + containingBlock.PaddingTop
		}
	}

	b.parseBoxModel()

	if b.Text != "" {
		b.calculateTextDimensions()
	} else {
		b.layoutChildren()
		b.calculateDimensions()
	}
}

// parseBoxModel parses margin, padding, and border properties
func (b *InlineBox) parseBoxModel() {
	b.MarginTop = parseLength(b.Style["margin-top"].Value, b.Width, 0)
	b.MarginRight = parseLength(b.Style["margin-right"].Value, b.Width, 0)
	b.MarginBottom = parseLength(b.Style["margin-bottom"].Value, b.Width, 0)
	b.MarginLeft = parseLength(b.Style["margin-left"].Value, b.Width, 0)

	b.PaddingTop = parseLength(b.Style["padding-top"].Value, b.Width, 0)
	b.PaddingRight = parseLength(b.Style["padding-right"].Value, b.Width, 0)
	b.PaddingBottom = parseLength(b.Style["padding-bottom"].Value, b.Width, 0)
	b.PaddingLeft = parseLength(b.Style["padding-left"].Value, b.Width, 0)

	b.BorderTop = parseLength(b.Style["border-top-width"].Value, b.Width, 0)
	b.BorderRight = parseLength(b.Style["border-right-width"].Value, b.Width, 0)
	b.BorderBottom = parseLength(b.Style["border-bottom-width"].Value, b.Width, 0)
	b.BorderLeft = parseLength(b.Style["border-left-width"].Value, b.Width, 0)
}

// calculateTextDimensions calculates dimensions for text content
func (b *InlineBox) calculateTextDimensions() {
	fontSize := parseLength(b.Style["font-size"].Value, 0, 16)

	charWidth := fontSize * 0.5
	b.Width = float64(len(b.Text)) * charWidth

	b.Height = fontSize

	b.Width += b.PaddingLeft + b.PaddingRight + b.BorderLeft + b.BorderRight
	b.Height += b.PaddingTop + b.PaddingBottom + b.BorderTop + b.BorderBottom
}

// layoutChildren performs layout for all children
func (b *InlineBox) layoutChildren() {
	x := b.X + b.MarginLeft + b.BorderLeft + b.PaddingLeft
	y := b.Y + b.MarginTop + b.BorderTop + b.PaddingTop

	for _, child := range b.Children {
		x += child.GetMarginLeft()
		child.SetPosition(x, y)
		child.Layout(nil)
		x += child.GetWidth() + child.GetMarginRight()
	}
}

// calculateDimensions calculates dimensions based on children
func (b *InlineBox) calculateDimensions() {
	if len(b.Children) == 0 {
		b.Width = b.PaddingLeft + b.PaddingRight + b.BorderLeft + b.BorderRight
		b.Height = b.PaddingTop + b.PaddingBottom + b.BorderTop + b.BorderBottom
		return
	}

	width := 0.0
	height := 0.0

	for _, child := range b.Children {
		width += child.GetWidth() + child.GetMarginLeft() + child.GetMarginRight()
		height = math.Max(height, child.GetHeight())
	}

	b.Width = width
	b.Height = height

	b.Width += b.PaddingLeft + b.PaddingRight + b.BorderLeft + b.BorderRight
	b.Height += b.PaddingTop + b.PaddingBottom + b.BorderTop + b.BorderBottom
}

// GetX returns the x position of the box
func (b *InlineBox) GetX() float64 {
	return b.X
}

// GetY returns the y position of the box
func (b *InlineBox) GetY() float64 {
	return b.Y
}

// GetWidth returns the width of the box
func (b *InlineBox) GetWidth() float64 {
	return b.Width
}

// GetHeight returns the height of the box
func (b *InlineBox) GetHeight() float64 {
	return b.Height
}

// GetMarginTop returns the top margin of the box
func (b *InlineBox) GetMarginTop() float64 {
	return b.MarginTop
}

// GetMarginBottom returns the bottom margin of the box
func (b *InlineBox) GetMarginBottom() float64 {
	return b.MarginBottom
}

// GetMarginLeft returns the left margin of the box
func (b *InlineBox) GetMarginLeft() float64 {
	return b.MarginLeft
}

// GetMarginRight returns the right margin of the box
func (b *InlineBox) GetMarginRight() float64 {
	return b.MarginRight
}

// SetPosition sets the position of the box
func (b *InlineBox) SetPosition(x, y float64) {
	b.X = x
	b.Y = y
}

// AddChild adds a child box
func (b *InlineBox) AddChild(child Box) {
	b.Children = append(b.Children, child)
}

// GetNode returns the HTML node associated with this box
func (b *InlineBox) GetNode() *html.Node {
	return b.Node
}

// parseLength parses a CSS length value
func parseLength(value string, containerSize float64, defaultValue float64) float64 {
	if value == "" {
		return defaultValue
	}

	if strings.HasSuffix(value, "%") {
		percentage, err := strconv.ParseFloat(value[:len(value)-1], 64)
		if err != nil {
			return defaultValue
		}
		return containerSize * percentage / 100
	}

	if strings.HasSuffix(value, "px") {
		pixels, err := strconv.ParseFloat(value[:len(value)-2], 64)
		if err != nil {
			return defaultValue
		}
		return pixels
	}

	if strings.HasSuffix(value, "em") {
		ems, err := strconv.ParseFloat(value[:len(value)-2], 64)
		if err != nil {
			return defaultValue
		}
		return ems * 16
	}

	if strings.HasSuffix(value, "rem") {
		rems, err := strconv.ParseFloat(value[:len(value)-3], 64)
		if err != nil {
			return defaultValue
		}
		return rems * 16
	}

	pixels, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return defaultValue
	}
	return pixels
}
