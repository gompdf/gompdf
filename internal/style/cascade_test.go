package style

import (
	"testing"

	"github.com/gompdf/gompdf/internal/parser/css"
	htmlparser "github.com/gompdf/gompdf/internal/parser/html"
	xhtml "golang.org/x/net/html"
)

func TestStyleCascadeRespectsSpecificityAndInlineStyles(t *testing.T) {
	doc, err := htmlparser.NewParser().ParseString(`<!doctype html>
<html>
  <body>
    <p id="hero">First</p>
    <p class="hero" style="color: green;">Second</p>
  </body>
</html>`)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	engine := NewStyleEngine()

	authorCSS := css.NewParser()
	sheet, err := authorCSS.ParseString(`
#hero { color: blue; }
p { color: red; }
.hero { font-weight: bold; }`)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}
	engine.AddStylesheet(sheet)

	computed := engine.ComputeStyles(doc)

	hero := findElementByAttr(doc.Root, "id", "hero")
	if hero == nil {
		t.Fatal("expected to find the id=hero paragraph")
	}

	if got := computed[hero]["color"].Value; got != "blue" {
		t.Fatalf("expected id selector to win over tag selector, got %q", got)
	}

	secondary := findElementByText(doc.Root, "Second")
	if secondary == nil {
		t.Fatal("expected to find the inline-styled paragraph")
	}

	if got := computed[secondary]["color"].Value; got != "green" {
		t.Fatalf("expected inline style to win, got %q", got)
	}
	if got := computed[secondary]["font-weight"].Value; got != "bold" {
		t.Fatalf("expected class selector to apply font-weight, got %q", got)
	}
}

func findElementByAttr(n *htmlparser.Node, key, value string) *htmlparser.Node {
	if n == nil {
		return nil
	}
	if n.Type == xhtml.ElementNode {
		for _, attr := range n.Attr {
			if attr.Key == key && attr.Val == value {
				return n
			}
		}
	}
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if found := findElementByAttr(child, key, value); found != nil {
			return found
		}
	}
	return nil
}

func findElementByText(n *htmlparser.Node, text string) *htmlparser.Node {
	if n == nil {
		return nil
	}
	if n.Type == xhtml.TextNode && n.Data == text {
		return n.Parent
	}
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if found := findElementByText(child, text); found != nil {
			return found
		}
	}
	return nil
}
