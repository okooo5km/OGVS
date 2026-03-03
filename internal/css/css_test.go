// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

package css

import (
	"testing"

	"github.com/okooo5km/ogvs/internal/svgast"
)

func TestParseStyleDeclarations(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"fill:red; stroke:blue", 2},
		{"fill: red !important", 1},
		{"font-size: 12px; font-family: Arial", 2},
		{"", 0},
	}

	for _, tt := range tests {
		got := ParseStyleDeclarations(tt.input)
		if len(got) != tt.want {
			t.Errorf("ParseStyleDeclarations(%q): got %d declarations, want %d", tt.input, len(got), tt.want)
		}
	}
}

func TestParseStyleDeclarations_Values(t *testing.T) {
	got := ParseStyleDeclarations("fill:red; stroke:blue !important")
	if len(got) != 2 {
		t.Fatalf("expected 2 declarations, got %d", len(got))
	}

	if got[0].Name != "fill" || got[0].Value != "red" || got[0].Important {
		t.Errorf("decl 0: got {%q, %q, %v}, want {fill, red, false}", got[0].Name, got[0].Value, got[0].Important)
	}
	if got[1].Name != "stroke" || got[1].Value != "blue" || !got[1].Important {
		t.Errorf("decl 1: got {%q, %q, %v}, want {stroke, blue, true}", got[1].Name, got[1].Value, got[1].Important)
	}
}

func TestCompareSpecificity(t *testing.T) {
	tests := []struct {
		a, b Specificity
		want int
	}{
		{Specificity{0, 1, 0, 0}, Specificity{0, 0, 1, 0}, 1},
		{Specificity{0, 0, 1, 0}, Specificity{0, 1, 0, 0}, -1},
		{Specificity{0, 0, 1, 0}, Specificity{0, 0, 1, 0}, 0},
		{Specificity{0, 0, 0, 1}, Specificity{0, 0, 0, 2}, -1},
		{Specificity{1, 0, 0, 0}, Specificity{0, 10, 10, 10}, 1},
	}

	for _, tt := range tests {
		got := CompareSpecificity(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("CompareSpecificity(%v, %v) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestCalculateSpecificity(t *testing.T) {
	tests := []struct {
		selector string
		want     Specificity
	}{
		{"div", Specificity{0, 0, 0, 1}},
		{".class", Specificity{0, 0, 1, 0}},
		{"#id", Specificity{0, 1, 0, 0}},
		{"div.class", Specificity{0, 0, 1, 1}},
		{"div#id.class", Specificity{0, 1, 1, 1}},
		{"*", Specificity{0, 0, 0, 0}},
	}

	for _, tt := range tests {
		got := CalculateSpecificity(tt.selector)
		if got != tt.want {
			t.Errorf("CalculateSpecificity(%q) = %v, want %v", tt.selector, got, tt.want)
		}
	}
}

func TestMatches_Element(t *testing.T) {
	elem := &svgast.Element{
		Name:       "rect",
		Attributes: svgast.NewOrderedAttrs(),
	}
	parents := make(map[svgast.Node]svgast.Parent)

	if !Matches(elem, "rect", parents) {
		t.Error("should match element name")
	}
	if Matches(elem, "circle", parents) {
		t.Error("should not match wrong element name")
	}
	if !Matches(elem, "*", parents) {
		t.Error("should match universal selector")
	}
}

func TestMatches_Class(t *testing.T) {
	elem := &svgast.Element{
		Name:       "rect",
		Attributes: svgast.NewOrderedAttrs(),
	}
	elem.Attributes.Set("class", "foo bar")
	parents := make(map[svgast.Node]svgast.Parent)

	if !Matches(elem, ".foo", parents) {
		t.Error("should match class foo")
	}
	if !Matches(elem, ".bar", parents) {
		t.Error("should match class bar")
	}
	if Matches(elem, ".baz", parents) {
		t.Error("should not match class baz")
	}
}

func TestMatches_ID(t *testing.T) {
	elem := &svgast.Element{
		Name:       "rect",
		Attributes: svgast.NewOrderedAttrs(),
	}
	elem.Attributes.Set("id", "myRect")
	parents := make(map[svgast.Node]svgast.Parent)

	if !Matches(elem, "#myRect", parents) {
		t.Error("should match ID")
	}
	if Matches(elem, "#other", parents) {
		t.Error("should not match wrong ID")
	}
}

func TestMatches_Compound(t *testing.T) {
	elem := &svgast.Element{
		Name:       "rect",
		Attributes: svgast.NewOrderedAttrs(),
	}
	elem.Attributes.Set("class", "active")
	elem.Attributes.Set("id", "main")
	parents := make(map[svgast.Node]svgast.Parent)

	if !Matches(elem, "rect.active", parents) {
		t.Error("should match element.class")
	}
	if !Matches(elem, "rect#main", parents) {
		t.Error("should match element#id")
	}
	if !Matches(elem, "rect.active#main", parents) {
		t.Error("should match element.class#id")
	}
}

func TestMatches_Descendant(t *testing.T) {
	svg := &svgast.Element{
		Name:       "svg",
		Attributes: svgast.NewOrderedAttrs(),
	}
	g := &svgast.Element{
		Name:       "g",
		Attributes: svgast.NewOrderedAttrs(),
	}
	rect := &svgast.Element{
		Name:       "rect",
		Attributes: svgast.NewOrderedAttrs(),
	}
	svg.Children = []svgast.Node{g}
	g.Children = []svgast.Node{rect}

	parents := map[svgast.Node]svgast.Parent{
		g:    svg,
		rect: g,
	}

	if !Matches(rect, "svg rect", parents) {
		t.Error("should match descendant selector")
	}
	if !Matches(rect, "g rect", parents) {
		t.Error("should match direct parent selector")
	}
	if Matches(rect, "circle rect", parents) {
		t.Error("should not match wrong ancestor")
	}
}

func TestMatches_ChildCombinator(t *testing.T) {
	g := &svgast.Element{
		Name:       "g",
		Attributes: svgast.NewOrderedAttrs(),
	}
	rect := &svgast.Element{
		Name:       "rect",
		Attributes: svgast.NewOrderedAttrs(),
	}
	g.Children = []svgast.Node{rect}

	parents := map[svgast.Node]svgast.Parent{
		rect: g,
	}

	if !Matches(rect, "g > rect", parents) {
		t.Error("should match child combinator")
	}
}

func TestMatches_CommaList(t *testing.T) {
	elem := &svgast.Element{
		Name:       "rect",
		Attributes: svgast.NewOrderedAttrs(),
	}
	parents := make(map[svgast.Node]svgast.Parent)

	if !Matches(elem, "circle, rect, path", parents) {
		t.Error("should match in comma-separated list")
	}
}

func TestCollectStylesheet(t *testing.T) {
	styleText := &svgast.Text{Value: "rect { fill: red; } .cls { stroke: blue; }"}
	styleElem := &svgast.Element{
		Name:       "style",
		Attributes: svgast.NewOrderedAttrs(),
		Children:   []svgast.Node{styleText},
	}
	root := &svgast.Root{
		Children: []svgast.Node{styleElem},
	}

	ss := CollectStylesheet(root)
	if len(ss.Rules) == 0 {
		t.Error("expected rules from stylesheet")
	}
}

func TestComputeOwnStyle(t *testing.T) {
	// Create a simple SVG with a style element and a rect
	styleText := &svgast.Text{Value: "rect { fill: blue; }"}
	styleElem := &svgast.Element{
		Name:       "style",
		Attributes: svgast.NewOrderedAttrs(),
		Children:   []svgast.Node{styleText},
	}
	rect := &svgast.Element{
		Name:       "rect",
		Attributes: svgast.NewOrderedAttrs(),
	}
	rect.Attributes.Set("stroke", "green") // presentation attribute

	svgElem := &svgast.Element{
		Name:       "svg",
		Attributes: svgast.NewOrderedAttrs(),
		Children:   []svgast.Node{styleElem, rect},
	}
	root := &svgast.Root{
		Children: []svgast.Node{svgElem},
	}

	ss := CollectStylesheet(root)
	styles := ComputeOwnStyle(ss, rect)

	if fill, ok := styles["fill"]; !ok {
		t.Error("expected fill style from CSS rule")
	} else if fill.Value != "blue" {
		t.Errorf("fill = %q, want blue", fill.Value)
	}

	if stroke, ok := styles["stroke"]; !ok {
		t.Error("expected stroke style from presentation attribute")
	} else if stroke.Value != "green" {
		t.Errorf("stroke = %q, want green", stroke.Value)
	}
}

func TestParseStyleDeclarations_CustomProperties(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []StylesheetDeclaration
	}{
		{
			name:  "single custom property",
			input: "--bg: #fff",
			want: []StylesheetDeclaration{
				{Name: "--bg", Value: "#fff", Important: false},
			},
		},
		{
			name:  "custom property with var()",
			input: "--_text: var(--fg)",
			want: []StylesheetDeclaration{
				{Name: "--_text", Value: "var(--fg)", Important: false},
			},
		},
		{
			name:  "custom property with !important",
			input: "--color: red !important",
			want: []StylesheetDeclaration{
				{Name: "--color", Value: "red", Important: true},
			},
		},
		{
			name:  "mixed standard and custom properties",
			input: "fill: red; --bg: #000; stroke: blue; --fg: #fff",
			want: []StylesheetDeclaration{
				{Name: "fill", Value: "red", Important: false},
				{Name: "--bg", Value: "#000", Important: false},
				{Name: "stroke", Value: "blue", Important: false},
				{Name: "--fg", Value: "#fff", Important: false},
			},
		},
		{
			name:  "custom property with complex value",
			input: "--gradient: linear-gradient(to right, #000, #fff)",
			want: []StylesheetDeclaration{
				{Name: "--gradient", Value: "linear-gradient(to right, #000, #fff)", Important: false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseStyleDeclarations(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d declarations, want %d\ngot: %+v", len(got), len(tt.want), got)
			}
			for i, d := range got {
				w := tt.want[i]
				if d.Name != w.Name || d.Value != w.Value || d.Important != w.Important {
					t.Errorf("decl[%d]: got {%q, %q, %v}, want {%q, %q, %v}",
						i, d.Name, d.Value, d.Important, w.Name, w.Value, w.Important)
				}
			}
		})
	}
}

func TestParseStylesheet_CustomProperties(t *testing.T) {
	cssText := `svg { --bg: #fff; --fg: #000; fill: var(--fg); }`
	rules := ParseStylesheet(cssText, false)

	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}

	rule := rules[0]
	if len(rule.Declarations) != 3 {
		t.Fatalf("expected 3 declarations, got %d: %+v", len(rule.Declarations), rule.Declarations)
	}

	// --bg
	if rule.Declarations[0].Name != "--bg" || rule.Declarations[0].Value != "#fff" {
		t.Errorf("decl[0]: got {%q, %q}, want {--bg, #fff}", rule.Declarations[0].Name, rule.Declarations[0].Value)
	}
	// --fg
	if rule.Declarations[1].Name != "--fg" || rule.Declarations[1].Value != "#000" {
		t.Errorf("decl[1]: got {%q, %q}, want {--fg, #000}", rule.Declarations[1].Name, rule.Declarations[1].Value)
	}
	// fill
	if rule.Declarations[2].Name != "fill" || rule.Declarations[2].Value != "var(--fg)" {
		t.Errorf("decl[2]: got {%q, %q}, want {fill, var(--fg)}", rule.Declarations[2].Name, rule.Declarations[2].Value)
	}
}

func TestStripPseudoClasses(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"a:hover", "a"},
		{"div:nth-child(2n+1)", "div:nth-child(2n+1)"}, // evaluatable, kept
		{"p::before", "p::before"},
		{".cls:focus", ".cls"},
		{"path:not([fill=blue])", "path:not([fill=blue])"}, // evaluatable, kept
		{"a:not(.cls):hover", "a:not(.cls)"},               // :not kept, :hover stripped
	}

	for _, tt := range tests {
		got := StripPseudoClasses(tt.input)
		if got != tt.want {
			t.Errorf("StripPseudoClasses(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestContainsPseudoClass(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"a:hover", true},
		{"p::before", false},
		{".cls", false},
		{"div:nth-child(2)", false},      // evaluatable
		{"path:not([fill=blue])", false}, // evaluatable
		{"a:not(.cls):hover", true},      // :not is evaluatable but :hover is not
		{".cls:first-child", false},      // evaluatable
	}

	for _, tt := range tests {
		got := containsPseudoClass(tt.input)
		if got != tt.want {
			t.Errorf("containsPseudoClass(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
