package layout

import (
	"github.com/gompdf/gompdf/internal/parser/html"
	"github.com/gompdf/gompdf/internal/style"
)

// ImageBox represents an <img> element laid out as an inline replaced element
// It implements the Box interface.
// For simplicity we treat it as inline-level and size it from CSS width/height or a default.

type ImageBox struct {
	Node   *html.Node
	Style  style.ComputedStyle

	X      float64
	Y      float64
	Width  float64
	Height float64

	MarginTop    float64
	MarginRight  float64
	MarginBottom float64
	MarginLeft   float64

	PaddingTop    float64
	PaddingRight  float64
	PaddingBottom float64
	PaddingLeft   float64

	BorderTop    float64
	BorderRight  float64
	BorderBottom float64
	BorderLeft   float64

	Src string // resolved later by renderer via Loader; stores the attribute value
}

func (b *ImageBox) Layout(containingBlock *BlockBox) {
	// Size from CSS width/height if present, else default square 40px
	w := 40.0
	h := 40.0
	if prop, ok := b.Style["width"]; ok && prop.Value != "" {
		if v := parseLength(prop.Value, containingBlock.Width, w); v > 0 {
			w = v
		}
	}
	if prop, ok := b.Style["height"]; ok && prop.Value != "" {
		if v := parseLength(prop.Value, containingBlock.Width, h); v > 0 {
			h = v
		}
	}
	b.Width = w
	b.Height = h
}

func (b *ImageBox) GetX() float64      { return b.X }
func (b *ImageBox) GetY() float64      { return b.Y }
func (b *ImageBox) GetWidth() float64  { return b.Width }
func (b *ImageBox) GetHeight() float64 { return b.Height }

func (b *ImageBox) GetMarginTop() float64    { return b.MarginTop }
func (b *ImageBox) GetMarginBottom() float64 { return b.MarginBottom }
func (b *ImageBox) GetMarginLeft() float64   { return b.MarginLeft }
func (b *ImageBox) GetMarginRight() float64  { return b.MarginRight }

func (b *ImageBox) SetPosition(x, y float64) { b.X, b.Y = x, y }

func (b *ImageBox) GetNode() *html.Node { return b.Node }
