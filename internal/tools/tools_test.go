// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

package tools

import (
	"math"
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

func TestNativeToFixed(t *testing.T) {
	// Expectations produced with node: Number(v.toFixed(p)).
	tests := []struct {
		num       float64
		precision int
		want      float64
	}{
		{-2.5, 0, -3},
		{2.5, 0, 3},
		{-1.5, 0, -2},
		{-0.5, 0, -1},
		{-0.0025, 3, -0.003},
		{0.0025, 3, 0.003},
		{1.005, 2, 1},
		{-1.005, 2, -1},
		{2.675, 2, 2.67},
		{-2.675, 2, -2.67},
		{8.835, 2, 8.84},
		{-8.835, 2, -8.84},
		{-0.0625, 3, -0.063},
		{-1.0005, 3, -1},
		{65.425, 2, 65.42},
		{-79.015, 2, -79.02},
		{12.754997, 3, 12.755},
		{0.125, 2, 0.13},
		{-0.125, 2, -0.13},
		{9.995, 2, 9.99},
		{1e-7, 3, 0},
		{-1e-7, 3, 0},
		{1e20, 0, 1e20},
		{1e21, 0, 1e21},
		{-1e21, 3, -1e21},
		{0, 3, 0},
		{-0, 3, 0},
		{1, 5, 1},
	}

	for _, tt := range tests {
		if got := NativeToFixed(tt.num, tt.precision); got != tt.want {
			t.Errorf("NativeToFixed(%v, %d) = %v, want %v", tt.num, tt.precision, got, tt.want)
		}
	}

	if got := NativeToFixed(math.NaN(), 3); !math.IsNaN(got) {
		t.Errorf("NativeToFixed(NaN, 3) = %v, want NaN", got)
	}
	if got := NativeToFixed(math.Inf(-1), 3); !math.IsInf(got, -1) {
		t.Errorf("NativeToFixed(-Inf, 3) = %v, want -Inf", got)
	}

	// ToFixed keeps Math.round semantics and must not follow NativeToFixed.
	if got := ToFixed(-2.5, 0); got != -2 {
		t.Errorf("ToFixed(-2.5, 0) = %v, want -2", got)
	}
	if got := ToFixed(-0.0025, 3); got != -0.002 {
		t.Errorf("ToFixed(-0.0025, 3) = %v, want -0.002", got)
	}
}

func TestJSIndexAndFalsy(t *testing.T) {
	data := []float64{1, 0, math.NaN()}

	if got := JSIndex(data, 0); got != 1 {
		t.Errorf("JSIndex(data, 0) = %v, want 1", got)
	}
	if got := JSIndex(data, 3); !math.IsNaN(got) {
		t.Errorf("JSIndex(data, 3) = %v, want NaN", got)
	}
	if got := JSIndex(nil, 0); !math.IsNaN(got) {
		t.Errorf("JSIndex(nil, 0) = %v, want NaN", got)
	}

	for _, falsy := range []float64{0, -0, math.NaN()} {
		if !JSFalsy(falsy) {
			t.Errorf("JSFalsy(%v) = false, want true", falsy)
		}
	}
	for _, truthy := range []float64{1, -1, 1e-9, math.Inf(1)} {
		if JSFalsy(truthy) {
			t.Errorf("JSFalsy(%v) = true, want false", truthy)
		}
	}

	if got := JSOr(math.NaN(), 7); got != 7 {
		t.Errorf("JSOr(NaN, 7) = %v, want 7", got)
	}
	if got := JSOr(3, 7); got != 3 {
		t.Errorf("JSOr(3, 7) = %v, want 3", got)
	}
}

func TestFormatNumberInfinity(t *testing.T) {
	tests := []struct {
		num  float64
		want string
	}{
		{math.Inf(1), "Infinity"},
		{math.Inf(-1), "-Infinity"},
		{math.NaN(), "NaN"},
		{-0, "0"},
	}

	for _, tt := range tests {
		if got := FormatNumber(tt.num); got != tt.want {
			t.Errorf("FormatNumber(%v) = %q, want %q", tt.num, got, tt.want)
		}
	}
}
