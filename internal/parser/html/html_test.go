package html

import (
	"strings"
	"testing"

	xhtml "golang.org/x/net/html"
)

func TestParseAndRenderRoundTrip(t *testing.T) {
	parser := NewParser()

	doc, err := parser.ParseString(`<!doctype html>
<html>
  <body>
    <p>Hello <strong>world</strong></p>
  </body>
</html>`)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}
	if doc == nil || doc.Root == nil {
		t.Fatal("ParseString() returned a nil document tree")
	}

	p := findElement(doc.Root, "p")
	if p == nil {
		t.Fatal("expected to find a <p> element in the parsed tree")
	}
	if p.FirstChild == nil || p.FirstChild.Type != xhtml.TextNode {
		t.Fatal("expected the paragraph to contain a text node")
	}

	rendered, err := doc.Render()
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if !strings.Contains(rendered, "<p>Hello <strong>world</strong></p>") {
		t.Fatalf("rendered HTML does not contain the expected paragraph: %s", rendered)
	}
}

func findElement(n *Node, tag string) *Node {
	if n == nil {
		return nil
	}
	if n.Type == xhtml.ElementNode && strings.EqualFold(n.Data, tag) {
		return n
	}
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if found := findElement(child, tag); found != nil {
			return found
		}
	}
	return nil
}
