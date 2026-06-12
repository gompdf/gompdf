package layout

import (
	"github.com/gompdf/gompdf/internal/parser/html"
)

type Box interface {
	Layout(containingBlock *BlockBox)
	GetX() float64
	GetY() float64
	GetWidth() float64
	GetHeight() float64
	GetMarginTop() float64
	GetMarginBottom() float64
	GetMarginLeft() float64
	GetMarginRight() float64
	SetPosition(x, y float64)
	GetNode() *html.Node
}
