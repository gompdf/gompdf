package layout

import (
	"reflect"
	"strings"
	"testing"

	"github.com/gompdf/gompdf/internal/parser/css"
	htmlparser "github.com/gompdf/gompdf/internal/parser/html"
	"github.com/gompdf/gompdf/internal/style"
	xhtml "golang.org/x/net/html"
)

func TestLayoutUsesConfiguredMargins(t *testing.T) {
	doc, err := htmlparser.NewParser().ParseString(`<!doctype html>
<html>
  <body>
    <div>Hello</div>
  </body>
</html>`)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	engine := NewEngine()
	engine.Debug = false
	engine.SetOptions(Options{Width: 200, Height: 100, DPI: 96})
	engine.SetMargins(10, 20, 30, 40)

	root := engine.Layout(doc)
	if root == nil {
		t.Fatal("expected a root box")
	}

	if root.X != 40 || root.Y != 10 {
		t.Fatalf("unexpected root origin: got (%.2f, %.2f), want (40, 10)", root.X, root.Y)
	}

	if root.Width != 140 || root.Height != 60 {
		t.Fatalf("unexpected root size: got (%.2f, %.2f), want (140, 60)", root.Width, root.Height)
	}
}

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

func TestLayoutTableCellsResetInheritedVisualStylesAndWidths(t *testing.T) {
	doc, err := htmlparser.NewParser().ParseString(`<!doctype html>
<html>
  <body>
    <table style="width: 100%; border-spacing: 0;">
      <tr>
        <td width="60%" style="padding: 0; border: 0; vertical-align: top;">
          <div class="card" style="padding: 12px; border: 1px solid #e5e7eb; background: #f9fafb;">
            <div class="label">Bill to</div>
            <div class="value">Acme, Inc.</div>
          </div>
        </td>
        <td width="40%" class="right" style="padding: 0; border: 0; vertical-align: top; text-align: right;">
          <div style="padding-left: 16px;">
            <div class="label">Summary</div>
            <div class="value">Invoice date 2025-09-12</div>
          </div>
        </td>
      </tr>
    </table>
  </body>
</html>`)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	cssParser := css.NewParser()
	sheet, err := cssParser.ParseString(`
.card { background: #f9fafb; border: 1px solid #e5e7eb; }
.label { font-size: 10px; font-weight: bold; }
.value { color: #6b7280; }
.right { text-align: right; }`)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	styleEngine := style.NewStyleEngine()
	styleEngine.AddStylesheet(sheet)
	styles := styleEngine.ComputeStyles(doc)

	engine := NewEngine()
	engine.Debug = false
	engine.SetOptions(Options{Width: 400, Height: 200, DPI: 96})
	engine.SetMargins(0, 0, 0, 0)
	engine.SetStyles(styles)

	root := engine.Layout(doc)
	if root == nil {
		t.Fatal("expected a root box")
	}

	card := findBlockByClass(root, "card")
	if card == nil {
		t.Fatal("expected to find the card block")
	}
	if got := card.Style["background-color"].Value; got != "#f9fafb" {
		t.Fatalf("expected card background to stay on the card, got %q", got)
	}
	if got := card.Style["border-width"].Value; got != "1px" {
		t.Fatalf("expected card border to stay on the card, got %q", got)
	}

	cardLabel := findInlineByText(root, "Bill to")
	if cardLabel == nil {
		t.Fatal("expected to find the card label text box")
	}
	if _, ok := cardLabel.Style["background-color"]; ok {
		t.Fatalf("did not expect background-color on text box style: %#v", cardLabel.Style)
	}
	if _, ok := cardLabel.Style["border-width"]; ok {
		t.Fatalf("did not expect border-width on text box style: %#v", cardLabel.Style)
	}
	if got := cardLabel.Style["font-size"].Value; got != "10px" {
		t.Fatalf("expected font-size to inherit, got %q", got)
	}

	rightValue := findInlineByText(root, "Invoice date 2025-09-12")
	if rightValue == nil {
		t.Fatal("expected to find the right-column text box")
	}
	if got := rightValue.Style["text-align"].Value; got != "right" {
		t.Fatalf("expected text-align to inherit, got %q", got)
	}
	if rightValue.Width > 170 {
		t.Fatalf("expected right-column text box width to fit the cell, got %.2f", rightValue.Width)
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

func findBlockByClass(b Box, className string) *BlockBox {
	if b == nil {
		return nil
	}
	if block, ok := b.(*BlockBox); ok && block.Node != nil && hasClass(block.Node, className) {
		return block
	}
	switch cur := b.(type) {
	case *BlockBox:
		for _, child := range cur.Children {
			if found := findBlockByClass(child, className); found != nil {
				return found
			}
		}
	case *InlineBox:
		for _, child := range cur.Children {
			if found := findBlockByClass(child, className); found != nil {
				return found
			}
		}
	}
	return nil
}

func findInlineByText(b Box, text string) *InlineBox {
	if b == nil {
		return nil
	}
	if inline, ok := b.(*InlineBox); ok && inline.Text == text {
		return inline
	}
	switch cur := b.(type) {
	case *BlockBox:
		for _, child := range cur.Children {
			if found := findInlineByText(child, text); found != nil {
				return found
			}
		}
	case *InlineBox:
		for _, child := range cur.Children {
			if found := findInlineByText(child, text); found != nil {
				return found
			}
		}
	}
	return nil
}

func hasClass(node *htmlparser.Node, className string) bool {
	if node == nil {
		return false
	}
	for _, attr := range node.Attr {
		if strings.EqualFold(attr.Key, "class") {
			for _, part := range strings.Fields(attr.Val) {
				if part == className {
					return true
				}
			}
		}
	}
	return false
}
