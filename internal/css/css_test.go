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

func TestParseNth(t *testing.T) {
	tests := []struct {
		in   string
		a, b int
		ok   bool
	}{
		{"odd", 2, 1, true},
		{"even", 2, 0, true},
		{"EVEN", 2, 0, true},
		{"2n+1", 2, 1, true},
		{"-n+3", -1, 3, true},
		{"3", 0, 3, true},
		{"+3", 0, 3, true},
		{"-1", 0, -1, true},
		{"n", 1, 0, true},
		{"2n", 2, 0, true},
		{" -2n + 3 ", -2, 3, true},
		{"+3n-2", 3, -2, true},
		{"0", 0, 0, true},
		{"bogus", 0, 0, false},
		{"", 0, 0, false},
		{"2n+", 0, 0, false},
		{"n+2x", 0, 0, false},
	}

	for _, tt := range tests {
		got, ok := parseNth(tt.in)
		if ok != tt.ok || (ok && (got.a != tt.a || got.b != tt.b)) {
			t.Errorf("parseNth(%q) = {%d,%d},%v, want {%d,%d},%v",
				tt.in, got.a, got.b, ok, tt.a, tt.b, tt.ok)
		}
	}
}

func TestNthExprMatches(t *testing.T) {
	// Positions selected out of the 1-based index range 1..4.
	tests := []struct {
		formula string
		want    []int
	}{
		{"n", []int{1, 2, 3, 4}},
		{"-n+2", []int{1, 2}},
		{"2n+1", []int{1, 3}},
		{"0", nil},
		{"-1", nil},
		{"1", []int{1}},
		{"+3", []int{3}},
		{"even", []int{2, 4}},
		{"odd", []int{1, 3}},
	}

	for _, tt := range tests {
		expr, ok := parseNth(tt.formula)
		if !ok {
			t.Errorf("parseNth(%q) failed", tt.formula)
			continue
		}
		var got []int
		for i := 1; i <= 4; i++ {
			// matchesNone/matchesAll are the css-select shortcuts; both
			// degenerate correctly for a parent that is an element.
			if expr.matchesNone() {
				break
			}
			if expr.matchesAll() || expr.matches(i) {
				got = append(got, i)
			}
		}
		if len(got) != len(tt.want) {
			t.Errorf("%q selected %v, want %v", tt.formula, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("%q selected %v, want %v", tt.formula, got, tt.want)
				break
			}
		}
	}
}

func TestParseAttrSelector(t *testing.T) {
	tests := []struct {
		in    string
		name  string
		op    string
		value string
		ic    bool
		bad   bool
	}{
		{"[fill]", "fill", "", "", false, false},
		{"[fill=red]", "fill", "=", "red", false, false},
		{`[fill='#00ff00']`, "fill", "=", "#00ff00", false, false},
		{`[size=""]`, "size", "=", "", false, false},
		{"[class~=foo]", "class", "~=", "foo", false, false},
		{"[lang|=en]", "lang", "|=", "en", false, false},
		{"[a^=b]", "a", "^=", "b", false, false},
		{"[a$=b]", "a", "$=", "b", false, false},
		{"[a*=b]", "a", "*=", "b", false, false},
		{"[a!=b]", "a", "!=", "b", false, false},
		{"[a=b i]", "a", "=", "b", true, false},
		{"[a=b s]", "a", "=", "b", false, false},
		{`[a=a\]b]`, "a", "=", "a]b", false, false},
		{"[|a=b]", "a", "=", "b", false, false},
		{"[ns|a]", "a", "", "", false, true},
		{"[a=b c]", "a", "=", "b", false, true},
	}

	for _, tt := range tests {
		i := 1 // skip '['
		as := parseAttrSelector(tt.in, &i)
		if as.name != tt.name || as.op != tt.op || as.value != tt.value ||
			as.ignoreCase != tt.ic || as.invalid != tt.bad {
			t.Errorf("parseAttrSelector(%q) = %+v", tt.in, as)
		}
		if i != len(tt.in) {
			t.Errorf("parseAttrSelector(%q) consumed %d of %d bytes", tt.in, i, len(tt.in))
		}
	}
}

func TestMatches_AttrOperators(t *testing.T) {
	parents := make(map[svgast.Node]svgast.Parent)
	elem := &svgast.Element{Name: "rect", Attributes: svgast.NewOrderedAttrs()}
	elem.Attributes.Set("class", "foo bar")
	elem.Attributes.Set("lang", "en-US")
	elem.Attributes.Set("fill", "RED")
	elem.Attributes.Set("empty", "")

	tests := []struct {
		selector string
		want     bool
	}{
		{"[class~=foo]", true},
		{"[class~=bar]", true},
		{"[class~=fo]", false},
		{`[class~="foo bar"]`, false}, // whitespace in value never matches
		{`[empty~=""]`, true},         // css-select quirk: empty value matches empty attr
		{`[class~=""]`, false},
		{"[lang|=en]", true},
		{"[lang|=en-US]", true},
		{"[lang|=e]", false},
		{`[empty|=""]`, true},
		{"[class^=foo]", true},
		{"[class^=oo]", false},
		{`[class^=""]`, false}, // empty prefix never matches
		{"[class$=bar]", true},
		{`[class$=""]`, false},
		{"[class*=o b]", false},
		{"[class*=oo]", true},
		{`[class*=""]`, false},
		{"[fill=RED]", true},
		{"[fill=red]", false},
		{"[fill=red i]", true},
		{"[fill=red s]", false},
		{"[fill=RED I]", true},
		{"[class!=foo]", true},
		{`[class!="foo bar"]`, false},
		{"[missing!=x]", true},
		{`[empty!=""]`, false},
		{"[ class ~= foo ]", true},
		{"[missing~=foo]", false},
		{"[missing|=foo]", false},
		{"[xlink|href]", false}, // namespaced: css-select rejects
	}

	for _, tt := range tests {
		if got := Matches(elem, tt.selector, parents); got != tt.want {
			t.Errorf("Matches(%q) = %v, want %v", tt.selector, got, tt.want)
		}
	}
}

// structuralFixture builds <svg><g id="g1"><!--c--><rect id="a"/>text<circle
// id="b"/><rect id="c"/></g><g id="g2"/></svg> and returns the parent map plus
// a lookup from id to element.
func structuralFixture() (map[svgast.Node]svgast.Parent, map[string]*svgast.Element) {
	newElem := func(name, id string) *svgast.Element {
		e := &svgast.Element{Name: name, Attributes: svgast.NewOrderedAttrs()}
		if id != "" {
			e.Attributes.Set("id", id)
		}
		return e
	}

	rectA := newElem("rect", "a")
	circleB := newElem("circle", "b")
	rectC := newElem("rect", "c")
	g1 := newElem("g", "g1")
	g1.Children = []svgast.Node{
		&svgast.Comment{Value: "c"},
		rectA,
		&svgast.Text{Value: "text"},
		circleB,
		rectC,
	}
	g2 := newElem("g", "g2")
	svg := newElem("svg", "")
	svg.Children = []svgast.Node{g1, g2}
	root := &svgast.Root{Children: []svgast.Node{svg}}

	parents := map[svgast.Node]svgast.Parent{
		svg: root, g1: svg, g2: svg,
		rectA: g1, circleB: g1, rectC: g1,
	}
	for _, c := range g1.Children {
		parents[c] = g1
	}

	return parents, map[string]*svgast.Element{
		"svg": svg, "g1": g1, "g2": g2,
		"a": rectA, "b": circleB, "c": rectC,
	}
}

func TestMatches_StructuralPseudoClasses(t *testing.T) {
	parents, byID := structuralFixture()

	tests := []struct {
		selector string
		want     []string // ids of the elements that must match
	}{
		{"rect:first-child", []string{"a"}}, // the leading comment does not count
		{"rect:last-child", []string{"c"}},
		{"rect:first-of-type", []string{"a"}},
		{"rect:last-of-type", []string{"c"}},
		{"circle:only-of-type", []string{"b"}},
		{"rect:only-of-type", nil},
		{"g:only-child", nil},
		{"rect:nth-child(1)", []string{"a"}},
		{"rect:nth-child(3)", []string{"c"}},
		{"rect:nth-child(odd)", []string{"a", "c"}},
		{"rect:nth-child(-n+2)", []string{"a"}},
		{"rect:nth-of-type(2)", []string{"c"}},
		{"rect:nth-last-of-type(1)", []string{"c"}},
		{"rect:nth-last-child(1)", []string{"c"}},
		{"rect:nth-child(0)", nil},
		{":root", []string{"svg"}},
		{"g:empty", []string{"g2"}},
		{":is(circle, g)", []string{"g1", "g2", "b"}},
		{":where(rect)", []string{"a", "c"}},
		{"g:has(circle)", []string{"g1"}},
		{"g:has(> rect)", []string{"g1"}},
		{"rect:not([id=a])", []string{"c"}},
	}

	ids := []string{"svg", "g1", "g2", "a", "b", "c"}
	for _, tt := range tests {
		want := make(map[string]bool, len(tt.want))
		for _, id := range tt.want {
			want[id] = true
		}
		for _, id := range ids {
			got := Matches(byID[id], tt.selector, parents)
			if got != want[id] {
				t.Errorf("Matches(%s, %q) = %v, want %v", id, tt.selector, got, want[id])
			}
		}
	}
}

func TestCalculateSpecificity_PseudoClasses(t *testing.T) {
	tests := []struct {
		selector string
		want     Specificity
	}{
		{"rect:first-child", Specificity{0, 0, 1, 1}},
		{"path:not([fill=blue])", Specificity{0, 0, 1, 1}},
		{":not(.a)", Specificity{0, 0, 1, 0}},
		{":not(#x)", Specificity{0, 1, 0, 0}},
		{":not(.a, #b)", Specificity{0, 1, 0, 0}}, // max of the list, not the sum
		{":not(.a.b)", Specificity{0, 0, 2, 0}},
		{":is(.a, #b)", Specificity{0, 1, 0, 0}},
		{":where(#a.b)", Specificity{0, 0, 0, 0}},
		{":where(.a):where(.b)", Specificity{0, 0, 0, 0}},
		{":has(.x)", Specificity{0, 0, 1, 0}},
		{":has(> .x)", Specificity{0, 0, 1, 0}},
		{":has(a b)", Specificity{0, 0, 0, 2}},
		{":hover", Specificity{0, 0, 1, 0}},
		{"rect::before", Specificity{0, 0, 0, 2}},
		{"rect:nth-child(2n+1)", Specificity{0, 0, 1, 1}},
		{"a:not(.b):not(.c)", Specificity{0, 0, 2, 1}},
		{"a.b#c[d]:hover::before", Specificity{0, 1, 3, 2}},
		{":not(:not(.a))", Specificity{0, 0, 1, 0}},
		{"g:not(.a) rect:first-of-type", Specificity{0, 0, 2, 2}},
		{"svg .hidden:target", Specificity{0, 0, 2, 1}},
	}

	for _, tt := range tests {
		if got := CalculateSpecificity(tt.selector); got != tt.want {
			t.Errorf("CalculateSpecificity(%q) = %v, want %v", tt.selector, got, tt.want)
		}
	}
}

func TestStripAllPseudoClasses(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"rect:first-child", "rect"},
		{"path:not([fill=blue])", "path"},
		{"svg .hidden:target", "svg .hidden"},
		{".st0:hover", ".st0"},
		{"rect:nth-child(2n+1)", "rect"},
		{"rect::before", "rect::before"},
		{":root", ""},
	}

	for _, tt := range tests {
		if got := StripAllPseudoClasses(tt.input); got != tt.want {
			t.Errorf("StripAllPseudoClasses(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestContainsAnyPseudoClass(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"rect:first-child", true},
		{"path:not([fill=blue])", true},
		{"svg .hidden:target", true},
		{"rect::before", false},
		{".cls", false},
		{`[href="a:b"]`, false}, // a colon inside an attribute value is not a pseudo
	}

	for _, tt := range tests {
		if got := ContainsAnyPseudoClass(tt.input); got != tt.want {
			t.Errorf("ContainsAnyPseudoClass(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
