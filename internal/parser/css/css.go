package css

import (
	"errors"
	"io"
	"strings"
)

// Parser represents a CSS parser
type Parser struct {
	// Configuration options could be added here
}

// Rule represents a CSS rule
type Rule struct {
	Selectors    []string
	Declarations []*Declaration
}

// Declaration represents a CSS declaration (property-value pair)
type Declaration struct {
	Property  string
	Value     string
	Important bool
}

// Stylesheet represents a parsed CSS stylesheet
type Stylesheet struct {
	Rules []*Rule
}

// NewParser creates a new CSS parser
func NewParser() *Parser {
	return &Parser{}
}

// ParseString parses CSS from a string
func (p *Parser) ParseString(content string) (*Stylesheet, error) {
	return p.Parse(strings.NewReader(content))
}

// Parse parses CSS from an io.Reader
func (p *Parser) Parse(r io.Reader) (*Stylesheet, error) {
	content, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return p.parseCSS(string(content))
}

// parseCSS parses CSS content
func (p *Parser) parseCSS(content string) (*Stylesheet, error) {
	stylesheet := &Stylesheet{
		Rules: []*Rule{},
	}

	content = removeComments(content)
	ruleStrings := splitRules(content)

	for _, ruleStr := range ruleStrings {
		rule, err := p.parseRule(ruleStr)
		if err != nil {
			continue // Skip invalid rules
		}
		stylesheet.Rules = append(stylesheet.Rules, rule)
	}

	return stylesheet, nil
}

// parseRule parses a single CSS rule
func (p *Parser) parseRule(ruleStr string) (*Rule, error) {
	parts := strings.SplitN(ruleStr, "{", 2)
	if len(parts) != 2 {
		return nil, errors.New("invalid rule format")
	}

	selectorStr := strings.TrimSpace(parts[0])
	declarationsStr := strings.TrimSpace(parts[1])

	declarationsStr = strings.TrimSuffix(declarationsStr, "}")

	selectors := parseSelectors(selectorStr)
	if len(selectors) == 0 {
		return nil, errors.New("no selectors found")
	}

	declarations := parseDeclarations(declarationsStr)

	return &Rule{
		Selectors:    selectors,
		Declarations: declarations,
	}, nil
}

// parseSelectors parses CSS selectors
func parseSelectors(selectorStr string) []string {
	selectors := strings.Split(selectorStr, ",")
	result := make([]string, 0, len(selectors))

	for _, selector := range selectors {
		selector = strings.TrimSpace(selector)
		if selector != "" {
			result = append(result, selector)
		}
	}

	return result
}

// parseDeclarations parses CSS declarations
func parseDeclarations(declarationsStr string) []*Declaration {
	declarationStrings := strings.Split(declarationsStr, ";")
	result := make([]*Declaration, 0, len(declarationStrings))

	for _, declStr := range declarationStrings {
		declStr = strings.TrimSpace(declStr)
		if declStr == "" {
			continue
		}

		parts := strings.SplitN(declStr, ":", 2)
		if len(parts) != 2 {
			continue
		}

		property := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		important := false
		if strings.HasSuffix(value, "!important") {
			important = true
			value = strings.TrimSuffix(value, "!important")
			value = strings.TrimSpace(value)
		}

		result = append(result, &Declaration{
			Property:  property,
			Value:     value,
			Important: important,
		})
	}

	return result
}

// removeComments removes CSS comments
func removeComments(content string) string {
	var result strings.Builder
	i := 0

	for i < len(content) {
		if i+1 < len(content) && content[i] == '/' && content[i+1] == '*' {
			commentEnd := strings.Index(content[i+2:], "*/")
			if commentEnd == -1 {
				break
			}
			i += commentEnd + 4
		} else {
			result.WriteByte(content[i])
			i++
		}
	}

	return result.String()
}

// splitRules splits CSS content into individual rules
func splitRules(content string) []string {
	var rules []string
	var currentRule strings.Builder
	braceCount := 0

	for i := 0; i < len(content); i++ {
		char := content[i]

		if char == '{' {
			braceCount++
		} else if char == '}' {
			braceCount--

			if braceCount == 0 {
				currentRule.WriteByte(char)
				rules = append(rules, currentRule.String())
				currentRule.Reset()
				continue
			}
		}

		if braceCount > 0 || !isWhitespace(char) {
			currentRule.WriteByte(char)
		}
	}

	return rules
}

// isWhitespace checks if a character is whitespace
func isWhitespace(char byte) bool {
	return char == ' ' || char == '\t' || char == '\n' || char == '\r'
}
