package css

import "testing"

func TestParseStylesheet(t *testing.T) {
	parser := NewParser()

	stylesheet, err := parser.ParseString(`
/* comment */
h1, .title {
  color: #111827;
  font-weight: bold !important;
}

#main {
  margin: 12px 24px;
}`)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	if stylesheet == nil {
		t.Fatal("ParseString() returned a nil stylesheet")
	}
	if got, want := len(stylesheet.Rules), 2; got != want {
		t.Fatalf("expected %d rules, got %d", want, got)
	}

	first := stylesheet.Rules[0]
	if got, want := len(first.Selectors), 2; got != want {
		t.Fatalf("expected %d selectors in first rule, got %d", want, got)
	}
	if first.Selectors[0] != "h1" || first.Selectors[1] != ".title" {
		t.Fatalf("unexpected selectors in first rule: %#v", first.Selectors)
	}
	if got, want := len(first.Declarations), 2; got != want {
		t.Fatalf("expected %d declarations in first rule, got %d", want, got)
	}
	if first.Declarations[1].Property != "font-weight" {
		t.Fatalf("unexpected declaration property: %s", first.Declarations[1].Property)
	}
	if !first.Declarations[1].Important {
		t.Fatal("expected !important to be preserved")
	}

	second := stylesheet.Rules[1]
	if got, want := len(second.Declarations), 1; got != want {
		t.Fatalf("expected %d declaration in second rule, got %d", want, got)
	}
	if second.Declarations[0].Property != "margin" || second.Declarations[0].Value != "12px 24px" {
		t.Fatalf("unexpected second rule declaration: %#v", second.Declarations[0])
	}
}
