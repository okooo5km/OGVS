// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

package collections

import "testing"

func TestElemsGroups(t *testing.T) {
	// Verify group count matches SVGO
	if got := len(ElemsGroups); got != 11 {
		t.Errorf("ElemsGroups: got %d groups, want 11", got)
	}

	// Spot check key groups
	if !ShapeElems["path"] {
		t.Error("ShapeElems missing 'path'")
	}
	if !FilterPrimitiveElems["feGaussianBlur"] {
		t.Error("FilterPrimitiveElems missing 'feGaussianBlur'")
	}
	if !TextContentElems["textPath"] {
		t.Error("TextContentElems missing 'textPath'")
	}
}

func TestTextElems(t *testing.T) {
	// textElems = textContent + pre + title = 12 items
	if got := len(TextElems); got != 12 {
		t.Errorf("TextElems: got %d, want 12", got)
	}
	if !TextElems["pre"] {
		t.Error("TextElems missing 'pre'")
	}
	if !TextElems["title"] {
		t.Error("TextElems missing 'title'")
	}
}

func TestAttrsGroups(t *testing.T) {
	if got := len(AttrsGroups); got != 15 {
		t.Errorf("AttrsGroups: got %d groups, want 15", got)
	}
	if !PresentationAttrs["viewBox"] {
		// viewBox is NOT a presentation attr
		// Good, this should be false
	}
	if !PresentationAttrs["fill"] {
		t.Error("PresentationAttrs missing 'fill'")
	}
	if !ConditionalProcessingAttrs["requiredFeatures"] {
		t.Error("ConditionalProcessingAttrs missing 'requiredFeatures'")
	}
}

func TestColorsNames(t *testing.T) {
	if got := len(ColorsNames); got != 148 {
		t.Errorf("ColorsNames: got %d entries, want 148", got)
	}
	if hex, ok := ColorsNames["red"]; !ok || hex != "#f00" {
		t.Errorf("ColorsNames[red] = %q, want #f00", hex)
	}
	if hex, ok := ColorsNames["rebeccapurple"]; !ok || hex != "#639" {
		t.Errorf("ColorsNames[rebeccapurple] = %q, want #639", hex)
	}
}

func TestColorsShortNames(t *testing.T) {
	if got := len(ColorsShortNames); got != 32 {
		t.Errorf("ColorsShortNames: got %d entries, want 32", got)
	}
	if name, ok := ColorsShortNames["#f00"]; !ok || name != "red" {
		t.Errorf("ColorsShortNames[#f00] = %q, want red", name)
	}
}

func TestReferencesProps(t *testing.T) {
	if got := len(ReferencesProps); got != 10 {
		t.Errorf("ReferencesProps: got %d, want 10", got)
	}
	if !ReferencesProps["fill"] {
		t.Error("ReferencesProps missing 'fill'")
	}
}

func TestInheritableAttrs(t *testing.T) {
	if got := len(InheritableAttrs); got != 45 {
		t.Errorf("InheritableAttrs: got %d, want 45", got)
	}
}

func TestEditorNamespaces(t *testing.T) {
	if got := len(EditorNamespaces); got != 24 {
		t.Errorf("EditorNamespaces: got %d, want 24", got)
	}
}

func TestPseudoClasses(t *testing.T) {
	if got := len(PseudoClasses); got != 9 {
		t.Errorf("PseudoClasses: got %d groups, want 9", got)
	}
}
