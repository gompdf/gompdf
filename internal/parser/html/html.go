package html

import (
	"bytes"
	"io"
	"strings"

	"golang.org/x/net/html"
)

// Parser represents an HTML parser
type Parser struct {
	// Configuration options could be added here
}

// Node represents an HTML node in the document tree
type Node struct {
	Type        html.NodeType
	Data        string
	Attr        []html.Attribute
	Parent      *Node
	FirstChild  *Node
	LastChild   *Node
	PrevSibling *Node
	NextSibling *Node
}

// Document represents a parsed HTML document
type Document struct {
	Root *Node
}

// NewParser creates a new HTML parser
func NewParser() *Parser {
	return &Parser{}
}

// ParseString parses HTML from a string
func (p *Parser) ParseString(content string) (*Document, error) {
	return p.Parse(strings.NewReader(content))
}

// Parse parses HTML from an io.Reader
func (p *Parser) Parse(r io.Reader) (*Document, error) {
	node, err := html.Parse(r)
	if err != nil {
		return nil, err
	}

	root := convertNode(node, nil)
	return &Document{Root: root}, nil
}

// convertNode converts an html.Node to our Node structure
func convertNode(n *html.Node, parent *Node) *Node {
	if n == nil {
		return nil
	}

	node := &Node{
		Type:   n.Type,
		Data:   n.Data,
		Attr:   n.Attr,
		Parent: parent,
	}

	var lastChild *Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		child := convertNode(c, node)
		if node.FirstChild == nil {
			node.FirstChild = child
		}
		if lastChild != nil {
			lastChild.NextSibling = child
			child.PrevSibling = lastChild
		}
		lastChild = child
	}
	node.LastChild = lastChild

	return node
}

// Render renders the document back to HTML
func (d *Document) Render() (string, error) {
	var buf bytes.Buffer
	err := renderNode(&buf, d.Root)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// renderNode renders a node and its children to HTML
func renderNode(w io.Writer, n *Node) error {
	if n == nil {
		return nil
	}

	node := &html.Node{
		Type: n.Type,
		Data: n.Data,
		Attr: n.Attr,
	}

	var firstChild, lastChild *html.Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		child := &html.Node{
			Type: c.Type,
			Data: c.Data,
			Attr: c.Attr,
		}
		if firstChild == nil {
			firstChild = child
		}
		if lastChild != nil {
			lastChild.NextSibling = child
			child.PrevSibling = lastChild
		}
		lastChild = child
	}
	node.FirstChild = firstChild
	node.LastChild = lastChild

	return html.Render(w, node)
}
