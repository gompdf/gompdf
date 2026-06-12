package layout

import (
	"reflect"
	"testing"

	"github.com/gompdf/gompdf/internal/parser/css"
	htmlparser "github.com/gompdf/gompdf/internal/parser/html"
	"github.com/gompdf/gompdf/internal/style"
	xhtml "golang.org/x/net/html"
)

func TestLayoutParagraphWrapsText(t *testing.T) {
	doc, err := htmlparser.NewParser().ParseString(`<!doctype html>
<html>
  <body>
    <p>One two three four five six seven eight nine ten eleven twelve</p>
  </body>
</html>`)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	cssParser := css.NewParser()
	sheet, err := cssParser.ParseString(`p { font-size: 12px; }`)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	styleEngine := style.NewStyleEngine()
	styleEngine.AddStylesheet(sheet)
	styles := styleEngine.ComputeStyles(doc)

	SetMeasurementOrientation("P")

	engine := NewEngine()
	engine.Debug = false
	engine.SetOptions(Options{Width: 220, Height: 400, DPI: 96})
	engine.SetStyles(styles)

	root := engine.Layout(doc)
	paragraph := findBlockByTag(root, "p")
	if paragraph == nil {
		t.Fatal("expected to find a paragraph block in the layout tree")
	}
	if got := len(paragraph.Children); got < 2 {
		t.Fatalf("expected the paragraph to wrap into multiple inline boxes, got %d", got)
	}

	firstLine, ok := paragraph.Children[0].(*InlineBox)
	if !ok {
		t.Fatalf("expected first child of paragraph to be an InlineBox, got %T", paragraph.Children[0])
	}
	if firstLine.Width <= 0 || firstLine.Height <= 0 {
		t.Fatalf("expected positive dimensions for the first line, got width=%v height=%v", firstLine.Width, firstLine.Height)
	}
}

func TestLayoutParagraphPreservesInlineSpacing(t *testing.T) {
	doc, err := htmlparser.NewParser().ParseString(`<!doctype html>
<html>
  <body>
    <p>Loads <code>styles.css</code> and renders</p>
  </body>
</html>`)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	cssParser := css.NewParser()
	sheet, err := cssParser.ParseString(`p { font-size: 12px; }`)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	styleEngine := style.NewStyleEngine()
	styleEngine.AddStylesheet(sheet)
	styles := styleEngine.ComputeStyles(doc)

	SetMeasurementOrientation("P")

	engine := NewEngine()
	engine.Debug = false
	engine.SetOptions(Options{Width: 500, Height: 400, DPI: 96})
	engine.SetStyles(styles)

	root := engine.Layout(doc)
	paragraph := findBlockByTag(root, "p")
	if paragraph == nil {
		t.Fatal("expected to find a paragraph block in the layout tree")
	}

	got := make([]string, 0, len(paragraph.Children))
	for _, child := range paragraph.Children {
		if ib, ok := child.(*InlineBox); ok {
			got = append(got, ib.Text)
		}
	}

	want := []string{"Loads", " ", "styles.css", " ", "and", " ", "renders"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected inline token sequence: got %v want %v", got, want)
	}
}

func findBlockByTag(b Box, tag string) *BlockBox {
	if b == nil {
		return nil
	}
	if block, ok := b.(*BlockBox); ok && block.Node != nil && block.Node.Type == xhtml.ElementNode && block.Node.Data == tag {
		return block
	}
	switch cur := b.(type) {
	case *BlockBox:
		for _, child := range cur.Children {
			if found := findBlockByTag(child, tag); found != nil {
				return found
			}
		}
	case *InlineBox:
		for _, child := range cur.Children {
			if found := findBlockByTag(child, tag); found != nil {
				return found
			}
		}
	}
	return nil
}
