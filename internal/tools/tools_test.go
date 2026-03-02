// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

package tools

import (
	"testing"

	"github.com/okooo5km/ogvs/internal/svgast"
)

func TestRemoveLeadingZero(t *testing.T) {
	tests := []struct {
		input float64
		want  string
	}{
		{0.5, ".5"},
		{-0.5, "-.5"},
		{0.123, ".123"},
		{-0.123, "-.123"},
		{1.5, "1.5"},
		{-1.5, "-1.5"},
		{0, "0"},
		{1, "1"},
		{-1, "-1"},
		{10, "10"},
	}
	for _, tt := range tests {
		got := RemoveLeadingZero(tt.input)
		if got != tt.want {
			t.Errorf("RemoveLeadingZero(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestToFixed(t *testing.T) {
	tests := []struct {
		num       float64
		precision int
		want      float64
	}{
		{1.23456, 2, 1.23},
		{1.235, 2, 1.24},
		{0.1 + 0.2, 1, 0.3},
		{1.005, 2, 1.0}, // JS Math.round(1.005 * 100) / 100 = 1.0
		{3.14159, 4, 3.1416},
	}
	for _, tt := range tests {
		got := ToFixed(tt.num, tt.precision)
		if got != tt.want {
			t.Errorf("ToFixed(%v, %d) = %v, want %v", tt.num, tt.precision, got, tt.want)
		}
	}
}

func TestCleanupOutData(t *testing.T) {
	params := &CleanupOutDataParams{
		LeadingZero:        true,
		NegativeExtraSpace: true,
	}

	tests := []struct {
		data    []float64
		command byte
		want    string
	}{
		{[]float64{0, -1, 0.5, 0.5}, 0, "0-1 .5.5"},
		{[]float64{10, 20, 30}, 0, "10 20 30"},
		{[]float64{1.5, -2.5}, 0, "1.5-2.5"},
	}
	for _, tt := range tests {
		got := CleanupOutData(tt.data, params, tt.command)
		if got != tt.want {
			t.Errorf("CleanupOutData(%v) = %q, want %q", tt.data, got, tt.want)
		}
	}
}

func TestIncludesURLReference(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"url(#gradient001)", true},
		{"url('#gradient001')", true},
		{`url("#gradient001")`, true},
		{"none", false},
		{"#000", false},
	}
	for _, tt := range tests {
		got := IncludesURLReference(tt.input)
		if got != tt.want {
			t.Errorf("IncludesURLReference(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestFindReferences(t *testing.T) {
	tests := []struct {
		attr  string
		value string
		want  int
	}{
		{"fill", "url(#gradient)", 1},
		{"href", "#myElement", 1},
		{"begin", "foo.click", 1},
		{"fill", "#000", 0},
		{"x", "10", 0},
	}
	for _, tt := range tests {
		got := FindReferences(tt.attr, tt.value)
		if len(got) != tt.want {
			t.Errorf("FindReferences(%q, %q) returned %d results, want %d", tt.attr, tt.value, len(got), tt.want)
		}
	}
}

func TestHasScripts(t *testing.T) {
	// Script element with children
	scriptElem := &svgast.Element{
		Name:       "script",
		Attributes: svgast.NewOrderedAttrs(),
		Children:   []svgast.Node{&svgast.Text{Value: "alert(1)"}},
	}
	if !HasScripts(scriptElem) {
		t.Error("HasScripts should return true for script element with children")
	}

	// Element with onclick
	clickElem := &svgast.Element{
		Name:       "rect",
		Attributes: svgast.NewOrderedAttrs(),
	}
	clickElem.Attributes.Set("onclick", "alert(1)")
	if !HasScripts(clickElem) {
		t.Error("HasScripts should return true for element with onclick")
	}

	// Plain element
	plainElem := &svgast.Element{
		Name:       "rect",
		Attributes: svgast.NewOrderedAttrs(),
	}
	plainElem.Attributes.Set("fill", "red")
	if HasScripts(plainElem) {
		t.Error("HasScripts should return false for plain element")
	}

	// Anchor with javascript: href
	jsLink := &svgast.Element{
		Name:       "a",
		Attributes: svgast.NewOrderedAttrs(),
	}
	jsLink.Attributes.Set("href", "javascript:alert(1)")
	if !HasScripts(jsLink) {
		t.Error("HasScripts should return true for javascript: link")
	}
}
