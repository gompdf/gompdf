package style

import (
	"strings"

	"github.com/gompdf/gompdf/internal/parser/css"
	"github.com/gompdf/gompdf/internal/parser/html"
	xhtml "golang.org/x/net/html"
)

// Specificity represents the specificity of a CSS selector
type Specificity struct {
	ID      int
	Class   int
	Element int
}

// StyleProperty represents a computed style property
type StyleProperty struct {
	Name      string
	Value     string
	Important bool
	Source    Source
}

// Source represents the source of a style property
type Source int

const (
	SourceUserAgent Source = iota
	SourceAuthor
	SourceInline
)

// ComputedStyle represents the computed style for an element
type ComputedStyle map[string]StyleProperty

// StyleEngine handles the CSS cascade and style computation
type StyleEngine struct {
	userAgentStyles *css.Stylesheet
	authorStyles    []*css.Stylesheet
}

// NewStyleEngine creates a new style engine
func NewStyleEngine() *StyleEngine {
	return &StyleEngine{
		userAgentStyles: defaultUserAgentStyles(),
		authorStyles:    []*css.Stylesheet{},
	}
}

// AddStylesheet adds an author stylesheet to the style engine
func (e *StyleEngine) AddStylesheet(stylesheet *css.Stylesheet) {
	e.authorStyles = append(e.authorStyles, stylesheet)
}

// ComputeStyles computes styles for all elements in the document
func (e *StyleEngine) ComputeStyles(doc *html.Document) map[*html.Node]ComputedStyle {
	result := make(map[*html.Node]ComputedStyle)
	e.computeStylesRecursive(doc.Root, result)
	return result
}

// computeStylesRecursive computes styles for an element and its children
func (e *StyleEngine) computeStylesRecursive(node *html.Node, result map[*html.Node]ComputedStyle) {
	if node == nil {
		return
	}

	if node.Type == xhtml.ElementNode {
		result[node] = e.computeStyleForElement(node)
	}

	for child := node.FirstChild; child != nil; child = child.NextSibling {
		e.computeStylesRecursive(child, result)
	}
}

// computeStyleForElement computes the style for a single element
func (e *StyleEngine) computeStyleForElement(node *html.Node) ComputedStyle {
	style := make(ComputedStyle)

	e.applyStylesheet(style, node, e.userAgentStyles, SourceUserAgent)

	for _, stylesheet := range e.authorStyles {
		e.applyStylesheet(style, node, stylesheet, SourceAuthor)
	}

	e.applyInlineStyles(style, node)

	return style
}

// applyStylesheet applies styles from a stylesheet to an element
func (e *StyleEngine) applyStylesheet(style ComputedStyle, node *html.Node, stylesheet *css.Stylesheet, source Source) {
	for _, rule := range stylesheet.Rules {
		for _, selector := range rule.Selectors {
			if e.selectorMatches(node, selector) {
				specificity := calculateSpecificity(selector)
				e.applyDeclarations(style, rule.Declarations, specificity, source)
			}
		}
	}
}

// applyInlineStyles applies inline styles to an element
func (e *StyleEngine) applyInlineStyles(style ComputedStyle, node *html.Node) {
	for _, attr := range node.Attr {
		if attr.Key == "style" {
			parser := css.NewParser()
			inlineStyles, err := parser.ParseString("dummy { " + attr.Val + " }")
			if err != nil || len(inlineStyles.Rules) == 0 {
				continue
			}

			specificity := Specificity{1, 0, 0}
			e.applyDeclarations(style, inlineStyles.Rules[0].Declarations, specificity, SourceInline)
		}
	}
}

// applyDeclarations applies CSS declarations to a style
func (e *StyleEngine) applyDeclarations(style ComputedStyle, declarations []*css.Declaration, specificity Specificity, source Source) {
	for _, decl := range declarations {
		property := decl.Property
		existing, exists := style[property]

		// Apply the new declaration if:
		// 1. The property doesn't exist yet, or
		// 2. The new declaration is !important and the existing one is not, or
		// 3. Both have the same importance but the new one has higher specificity, or
		// 4. Both have the same importance and specificity but the new one comes from a higher priority source
		if !exists ||
			(decl.Important && !existing.Important) ||
			(decl.Important == existing.Important && compareSpecificity(specificity, Specificity{}) > 0) ||
			(decl.Important == existing.Important && compareSpecificity(specificity, Specificity{}) == 0 && source > existing.Source) {

			style[property] = StyleProperty{
				Name:      property,
				Value:     decl.Value,
				Important: decl.Important,
				Source:    source,
			}
		}
	}
}

// selectorMatches checks if an element matches a CSS selector
func (e *StyleEngine) selectorMatches(node *html.Node, selector string) bool {
	parts := strings.Fields(selector)
	if len(parts) == 0 || node == nil {
		return false
	}
	if !matchCompoundSelector(node, parts[len(parts)-1]) {
		return false
	}

	current := node.Parent
	for i := len(parts) - 2; i >= 0; i-- {
		found := false
		for anc := current; anc != nil; anc = anc.Parent {
			if anc.Type == xhtml.ElementNode && matchCompoundSelector(anc, parts[i]) {
				found = true
				current = anc.Parent
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// matchCompoundSelector matches a single compound selector against a node.
// Compound selectors can be forms like:
//   - tag
//   - .class
//   - #id
//   - tag.class
//   - tag#id.class1.class2
//   - .class1.class2
//
// It does not support attributes, pseudo-classes, or combinators.
func matchCompoundSelector(node *html.Node, sel string) bool {
	if node == nil || node.Type != xhtml.ElementNode || sel == "" {
		return false
	}

	var wantTag string
	var wantID string
	var wantClasses []string

	// Parse the compound selector
	// Scan sel once, extracting optional tag, optional id, and any number of classes
	i := 0
	// Extract tag if first character is a letter or '*'
	if i < len(sel) && sel[i] != '.' && sel[i] != '#' {
		// read until '#' or '.'
		j := i
		for j < len(sel) && sel[j] != '#' && sel[j] != '.' {
			j++
		}
		wantTag = sel[i:j]
		i = j
	}
	// Extract sequences of (#id | .class)
	for i < len(sel) {
		if sel[i] == '#' {
			// id
			j := i + 1
			for j < len(sel) && sel[j] != '.' && sel[j] != '#' {
				j++
			}
			wantID = sel[i+1 : j]
			i = j
			continue
		}
		if sel[i] == '.' {
			j := i + 1
			for j < len(sel) && sel[j] != '.' && sel[j] != '#' {
				j++
			}
			wantClasses = append(wantClasses, sel[i+1:j])
			i = j
			continue
		}
		// Unexpected character; fail safe
		return false
	}

	if wantTag != "" && wantTag != node.Data && wantTag != "*" {
		return false
	}

	if wantID != "" {
		matched := false
		for _, attr := range node.Attr {
			if attr.Key == "id" && attr.Val == wantID {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	if len(wantClasses) > 0 {
		var classAttr string
		for _, attr := range node.Attr {
			if attr.Key == "class" {
				classAttr = attr.Val
				break
			}
		}
		if classAttr == "" {
			return false
		}
		have := strings.Fields(classAttr)
		set := make(map[string]struct{}, len(have))
		for _, c := range have {
			set[c] = struct{}{}
		}
		for _, need := range wantClasses {
			if _, ok := set[need]; !ok {
				return false
			}
		}
	}

	return true
}

// calculateSpecificity calculates the specificity of a CSS selector
func calculateSpecificity(selector string) Specificity {
	specificity := Specificity{}

	specificity.ID = strings.Count(selector, "#")

	specificity.Class = strings.Count(selector, ".") +
		strings.Count(selector, "[") +
		strings.Count(selector, ":")
	specificity.Element = strings.Count(selector, "::") +
		len(strings.Fields(strings.NewReplacer(
			"#", " ",
			".", " ",
			"[", " ",
			":", " ",
		).Replace(selector)))

	return specificity
}

// compareSpecificity compares two specificities
func compareSpecificity(a, b Specificity) int {
	if a.ID != b.ID {
		return a.ID - b.ID
	}
	if a.Class != b.Class {
		return a.Class - b.Class
	}
	return a.Element - b.Element
}

// defaultUserAgentStyles returns the default user agent stylesheet
func defaultUserAgentStyles() *css.Stylesheet {
	parser := css.NewParser()
	stylesheet, _ := parser.ParseString(`
		body { margin: 8px; }
		h1 { font-size: 2em; margin: 0.67em 0; }
		h2 { font-size: 1.5em; margin: 0.75em 0; }
		h3 { font-size: 1.17em; margin: 0.83em 0; }
		h4 { margin: 1.12em 0; }
		h5 { font-size: 0.83em; margin: 1.5em 0; }
		h6 { font-size: 0.75em; margin: 1.67em 0; }
		p { margin: 1em 0; }
		a { color: #0000EE; text-decoration: underline; }
		a:visited { color: #551A8B; }
		b, strong { font-weight: bold; }
		i, em { font-style: italic; }
		pre { white-space: pre; }
		table { border-collapse: separate; border-spacing: 2px; }
		th, td { border: 1px solid #ddd; padding: 4px; }
		th { background-color: #f2f2f2; }
	`)
	return stylesheet
}
