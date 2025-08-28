package pagination

import (
	"github.com/gompdf/gompdf/internal/layout"
)

// Options represents options for the pagination engine
type Options struct {
	PageWidth    float64
	PageHeight   float64
	MarginTop    float64
	MarginRight  float64
	MarginBottom float64
	MarginLeft   float64
}

// Engine handles the pagination process
type Engine struct {
	options Options
}

// NewEngine creates a new pagination engine
func NewEngine() *Engine {
	return &Engine{
		options: Options{
			PageWidth:    595.28, // Default A4 width in points
			PageHeight:   841.89, // Default A4 height in points
			MarginTop:    72,     // Default 1-inch margins
			MarginRight:  72,
			MarginBottom: 72,
			MarginLeft:   72,
		},
	}
}

// SetOptions sets the options for the pagination engine
func (e *Engine) SetOptions(options Options) {
	e.options = options
}

// Paginate breaks content into pages
func (e *Engine) Paginate(rootBox *layout.BlockBox) []*Page {
	paginator := NewPaginator(
		PageSize{
			Width:  e.options.PageWidth,
			Height: e.options.PageHeight,
			Name:   "Custom",
		},
		Margins{
			Top:    e.options.MarginTop,
			Right:  e.options.MarginRight,
			Bottom: e.options.MarginBottom,
			Left:   e.options.MarginLeft,
		},
	)

	return paginator.Paginate(rootBox)
}
