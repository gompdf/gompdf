package pagination

import (
	"testing"

	"github.com/gompdf/gompdf/internal/layout"
)

func TestPaginateSplitsContentAcrossPages(t *testing.T) {
	root := &layout.BlockBox{
		X:        0,
		Y:        0,
		Width:    500,
		Height:   900,
		Children: []layout.Box{},
	}

	first := &layout.BlockBox{X: 20, Y: 10, Width: 100, Height: 120}
	second := &layout.BlockBox{X: 20, Y: 420, Width: 100, Height: 120}
	root.Children = []layout.Box{first, second}

	paginator := NewPaginator(
		PageSize{Width: 400, Height: 500, Name: "Test"},
		Margins{Top: 20, Right: 20, Bottom: 20, Left: 20},
	)

	pages := paginator.Paginate(root)
	if got, want := len(pages), 2; got != want {
		t.Fatalf("expected %d pages, got %d", want, got)
	}
	if got, want := len(pages[0].Boxes), 1; got != want {
		t.Fatalf("expected first page to contain %d box, got %d", want, got)
	}
	if got, want := len(pages[1].Boxes), 1; got != want {
		t.Fatalf("expected second page to contain %d box, got %d", want, got)
	}
}
