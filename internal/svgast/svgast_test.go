// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

package svgast

import (
	"strings"
	"testing"
)

// --- OrderedAttrs Tests ---

func TestOrderedAttrs_BasicOps(t *testing.T) {
	oa := NewOrderedAttrs()
	oa.Set("xmlns", "http://www.w3.org/2000/svg")
	oa.Set("width", "100")
	oa.Set("height", "200")

	if oa.Len() != 3 {
		t.Errorf("Len() = %d, want 3", oa.Len())
	}

	v, ok := oa.Get("width")
	if !ok || v != "100" {
		t.Errorf("Get(width) = %q, %v", v, ok)
	}

	oa.Set("width", "150")
	v, _ = oa.Get("width")
	if v != "150" {
		t.Errorf("after Set, Get(width) = %q, want 150", v)
	}

	// Order preserved
	entries := oa.Entries()
	if entries[0].Name != "xmlns" || entries[1].Name != "width" || entries[2].Name != "height" {
		t.Errorf("order not preserved: %v", entries)
	}
}

func TestOrderedAttrs_Delete(t *testing.T) {
	oa := NewOrderedAttrs()
	oa.Set("a", "1")
	oa.Set("b", "2")
	oa.Set("c", "3")

	oa.Delete("b")

	if oa.Len() != 2 {
		t.Errorf("Len() = %d after delete, want 2", oa.Len())
	}
	if oa.Has("b") {
		t.Error("Has(b) = true after delete")
	}

	entries := oa.Entries()
	if entries[0].Name != "a" || entries[1].Name != "c" {
		t.Errorf("order after delete: %v", entries)
	}
}

// --- Parser Tests ---

func TestParseSvg_BasicElement(t *testing.T) {
	input := `<svg xmlns="http://www.w3.org/2000/svg"><g/></svg>`
	root, err := ParseSvg(input, "")
	if err != nil {
		t.Fatalf("ParseSvg error: %v", err)
	}

	if len(root.Children) != 1 {
		t.Fatalf("root children = %d, want 1", len(root.Children))
	}

	svg, ok := root.Children[0].(*Element)
	if !ok {
		t.Fatalf("root child type = %T, want *Element", root.Children[0])
	}
	if svg.Name != "svg" {
		t.Errorf("svg name = %q", svg.Name)
	}

	xmlns, _ := svg.Attributes.Get("xmlns")
	if xmlns != "http://www.w3.org/2000/svg" {
		t.Errorf("xmlns = %q", xmlns)
	}

	if len(svg.Children) != 1 {
		t.Fatalf("svg children = %d, want 1", len(svg.Children))
	}
	g, ok := svg.Children[0].(*Element)
	if !ok || g.Name != "g" {
		t.Errorf("child = %T %v", svg.Children[0], svg.Children[0])
	}
}

func TestParseSvg_Comment(t *testing.T) {
	input := `<svg xmlns="http://www.w3.org/2000/svg"><!-- test comment --></svg>`
	root, err := ParseSvg(input, "")
	if err != nil {
		t.Fatalf("ParseSvg error: %v", err)
	}

	svg := root.Children[0].(*Element)
	if len(svg.Children) != 1 {
		t.Fatalf("svg children = %d, want 1", len(svg.Children))
	}

	comment, ok := svg.Children[0].(*Comment)
	if !ok {
		t.Fatalf("child type = %T, want *Comment", svg.Children[0])
	}
	if comment.Value != "test comment" {
		t.Errorf("comment value = %q, want %q", comment.Value, "test comment")
	}
}

func TestParseSvg_TextTrimming(t *testing.T) {
	input := `<svg xmlns="http://www.w3.org/2000/svg"><g>   hello   </g></svg>`
	root, err := ParseSvg(input, "")
	if err != nil {
		t.Fatalf("ParseSvg error: %v", err)
	}

	svg := root.Children[0].(*Element)
	g := svg.Children[0].(*Element)
	text := g.Children[0].(*Text)

	// Non-textElem: text should be trimmed
	if text.Value != "hello" {
		t.Errorf("text value = %q, want %q", text.Value, "hello")
	}
}

func TestParseSvg_TextPreservation(t *testing.T) {
	input := `<svg xmlns="http://www.w3.org/2000/svg"><text>  hello  </text></svg>`
	root, err := ParseSvg(input, "")
	if err != nil {
		t.Fatalf("ParseSvg error: %v", err)
	}

	svg := root.Children[0].(*Element)
	textElem := svg.Children[0].(*Element)
	text := textElem.Children[0].(*Text)

	// textElem: whitespace preserved
	if text.Value != "  hello  " {
		t.Errorf("text value = %q, want %q", text.Value, "  hello  ")
	}
}

func TestParseSvg_ProcessingInstruction(t *testing.T) {
	input := `<?xml version="1.0" encoding="utf-8"?><svg xmlns="http://www.w3.org/2000/svg"/>`
	root, err := ParseSvg(input, "")
	if err != nil {
		t.Fatalf("ParseSvg error: %v", err)
	}

	// Should have PI and SVG element
	if len(root.Children) < 2 {
		t.Fatalf("root children = %d, want >= 2", len(root.Children))
	}

	pi, ok := root.Children[0].(*Instruction)
	if !ok {
		t.Fatalf("first child type = %T, want *Instruction", root.Children[0])
	}
	if pi.Name != "xml" {
		t.Errorf("PI name = %q", pi.Name)
	}
}

func TestParseSvg_EmptyTextSkipped(t *testing.T) {
	input := `<svg xmlns="http://www.w3.org/2000/svg">
    <g/>
</svg>`
	root, err := ParseSvg(input, "")
	if err != nil {
		t.Fatalf("ParseSvg error: %v", err)
	}

	svg := root.Children[0].(*Element)
	// Whitespace-only text nodes outside textElems should be skipped
	for _, child := range svg.Children {
		if _, ok := child.(*Text); ok {
			t.Error("whitespace-only text node should be skipped")
		}
	}
}

// --- Visitor Tests ---

func TestVisit_EnterExit(t *testing.T) {
	root := &Root{
		Children: []Node{
			&Element{
				Name:       "svg",
				Attributes: NewOrderedAttrs(),
				Children: []Node{
					&Element{
						Name:       "g",
						Attributes: NewOrderedAttrs(),
					},
				},
			},
		},
	}

	var entered, exited []string
	visitor := &Visitor{
		Element: &VisitorCallbacks{
			Enter: func(node Node, parent Parent) error {
				entered = append(entered, node.(*Element).Name)
				return nil
			},
			Exit: func(node Node, parent Parent) {
				exited = append(exited, node.(*Element).Name)
			},
		},
	}

	Visit(root, visitor, nil)

	if len(entered) != 2 || entered[0] != "svg" || entered[1] != "g" {
		t.Errorf("entered = %v, want [svg g]", entered)
	}
	if len(exited) != 2 || exited[0] != "g" || exited[1] != "svg" {
		t.Errorf("exited = %v, want [g svg]", exited)
	}
}

func TestVisit_SkipChildren(t *testing.T) {
	root := &Root{
		Children: []Node{
			&Element{
				Name:       "svg",
				Attributes: NewOrderedAttrs(),
				Children: []Node{
					&Element{
						Name:       "g",
						Attributes: NewOrderedAttrs(),
					},
				},
			},
		},
	}

	var entered []string
	visitor := &Visitor{
		Element: &VisitorCallbacks{
			Enter: func(node Node, parent Parent) error {
				entered = append(entered, node.(*Element).Name)
				if node.(*Element).Name == "svg" {
					return ErrVisitSkip
				}
				return nil
			},
		},
	}

	Visit(root, visitor, nil)

	// Should only enter svg, skip g
	if len(entered) != 1 || entered[0] != "svg" {
		t.Errorf("entered = %v, want [svg]", entered)
	}
}

func TestVisit_DetachDuringTraversal(t *testing.T) {
	g := &Element{Name: "g", Attributes: NewOrderedAttrs()}
	rect := &Element{Name: "rect", Attributes: NewOrderedAttrs()}
	svg := &Element{
		Name:       "svg",
		Attributes: NewOrderedAttrs(),
		Children:   []Node{g, rect},
	}
	root := &Root{Children: []Node{svg}}

	var entered []string
	visitor := &Visitor{
		Element: &VisitorCallbacks{
			Enter: func(node Node, parent Parent) error {
				elem := node.(*Element)
				entered = append(entered, elem.Name)
				if elem.Name == "g" {
					// Detach rect during traversal
					DetachNodeFromParent(rect, parent)
				}
				return nil
			},
		},
	}

	Visit(root, visitor, nil)

	// rect should still be entered because we iterate over a copy
	if len(entered) != 3 {
		t.Errorf("entered = %v, want [svg g rect]", entered)
	}
}

// --- Stringifier Tests ---

func TestStringifySvg_Basic(t *testing.T) {
	root := &Root{
		Children: []Node{
			&Element{
				Name:       "svg",
				Attributes: NewOrderedAttrsFromEntries([]AttrEntry{{Name: "xmlns", Value: "http://www.w3.org/2000/svg"}}),
				Children: []Node{
					&Element{
						Name:       "g",
						Attributes: NewOrderedAttrs(),
					},
				},
			},
		},
	}

	result := StringifySvg(root, nil)
	expected := `<svg xmlns="http://www.w3.org/2000/svg"><g/></svg>`
	if result != expected {
		t.Errorf("got:\n%s\nwant:\n%s", result, expected)
	}
}

func TestStringifySvg_Pretty(t *testing.T) {
	root := &Root{
		Children: []Node{
			&Element{
				Name:       "svg",
				Attributes: NewOrderedAttrsFromEntries([]AttrEntry{{Name: "xmlns", Value: "http://www.w3.org/2000/svg"}}),
				Children: []Node{
					&Element{
						Name:       "g",
						Attributes: NewOrderedAttrs(),
					},
				},
			},
		},
	}

	result := StringifySvg(root, &StringifyOptions{
		Pretty:       true,
		Indent:       4,
		UseShortTags: true,
		EOL:          "lf",
	})
	expected := "<svg xmlns=\"http://www.w3.org/2000/svg\">\n    <g/>\n</svg>\n"
	if result != expected {
		t.Errorf("got:\n%q\nwant:\n%q", result, expected)
	}
}

func TestStringifySvg_EntityEncoding(t *testing.T) {
	root := &Root{
		Children: []Node{
			&Element{
				Name:       "svg",
				Attributes: NewOrderedAttrsFromEntries([]AttrEntry{{Name: "xmlns", Value: "http://www.w3.org/2000/svg"}}),
				Children: []Node{
					&Text{Value: "a < b & c > d"},
				},
			},
		},
	}

	result := StringifySvg(root, nil)
	if !strings.Contains(result, "&lt;") || !strings.Contains(result, "&amp;") || !strings.Contains(result, "&gt;") {
		t.Errorf("entity encoding missing: %s", result)
	}
}

func TestStringifySvg_Comment(t *testing.T) {
	root := &Root{
		Children: []Node{
			&Comment{Value: " test "},
		},
	}

	result := StringifySvg(root, nil)
	if result != "<!-- test -->" {
		t.Errorf("got %q, want %q", result, "<!-- test -->")
	}
}

func TestStringifySvg_ShortTagDisabled(t *testing.T) {
	root := &Root{
		Children: []Node{
			&Element{
				Name:       "g",
				Attributes: NewOrderedAttrs(),
			},
		},
	}

	result := StringifySvg(root, &StringifyOptions{UseShortTags: false, Indent: 4, EOL: "lf"})
	if result != "<g></g>" {
		t.Errorf("got %q, want %q", result, "<g></g>")
	}
}

// --- Roundtrip Tests ---

func TestRoundtrip_Simple(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			"empty svg",
			`<svg xmlns="http://www.w3.org/2000/svg"/>`,
		},
		{
			"svg with child",
			`<svg xmlns="http://www.w3.org/2000/svg"><g/></svg>`,
		},
		{
			"comment",
			`<svg xmlns="http://www.w3.org/2000/svg"><!--test--></svg>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, err := ParseSvg(tt.input, "")
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			output := StringifySvg(root, nil)
			if output != tt.input {
				t.Errorf("roundtrip:\ngot:  %q\nwant: %q", output, tt.input)
			}
		})
	}
}

// --- DetachNodeFromParent Test ---

func TestDetachNodeFromParent(t *testing.T) {
	child1 := &Element{Name: "a", Attributes: NewOrderedAttrs()}
	child2 := &Element{Name: "b", Attributes: NewOrderedAttrs()}
	child3 := &Element{Name: "c", Attributes: NewOrderedAttrs()}
	parent := &Element{
		Name:       "g",
		Attributes: NewOrderedAttrs(),
		Children:   []Node{child1, child2, child3},
	}

	DetachNodeFromParent(child2, parent)

	if len(parent.Children) != 2 {
		t.Errorf("children count = %d, want 2", len(parent.Children))
	}
	if parent.Children[0] != child1 || parent.Children[1] != child3 {
		t.Error("wrong children after detach")
	}
}
