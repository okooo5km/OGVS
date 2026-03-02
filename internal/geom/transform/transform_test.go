// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

package transform

import (
	"math"
	"testing"
)

func TestTransform2JS(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int // number of transforms
		first string
	}{
		{"translate", "translate(10, 20)", 1, "translate"},
		{"rotate", "rotate(45)", 1, "rotate"},
		{"scale", "scale(2)", 1, "scale"},
		{"multiple", "translate(10,20) rotate(45) scale(2)", 3, "translate"},
		{"matrix", "matrix(1 0 0 1 10 20)", 1, "matrix"},
		{"skewX", "skewX(30)", 1, "skewX"},
		{"skewY", "skewY(45)", 1, "skewY"},
		{"empty", "", 0, ""},
		{"malformed", "foo(1,2)", 0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Transform2JS(tt.input)
			if len(got) != tt.want {
				t.Errorf("Transform2JS(%q): got %d transforms, want %d", tt.input, len(got), tt.want)
				return
			}
			if tt.want > 0 && got[0].Name != tt.first {
				t.Errorf("Transform2JS(%q): first name = %q, want %q", tt.input, got[0].Name, tt.first)
			}
		})
	}
}

func TestTransform2JS_Data(t *testing.T) {
	got := Transform2JS("translate(10.5, -20.3)")
	if len(got) != 1 {
		t.Fatalf("expected 1 transform, got %d", len(got))
	}
	if got[0].Data[0] != 10.5 {
		t.Errorf("tx = %v, want 10.5", got[0].Data[0])
	}
	if got[0].Data[1] != -20.3 {
		t.Errorf("ty = %v, want -20.3", got[0].Data[1])
	}
}

func TestTransformsMultiply(t *testing.T) {
	// translate(10, 20) should give matrix [1, 0, 0, 1, 10, 20]
	transforms := []TransformItem{
		{Name: "translate", Data: []float64{10, 20}},
	}
	got := TransformsMultiply(transforms)
	if got.Name != "matrix" {
		t.Errorf("name = %q, want matrix", got.Name)
	}
	expected := []float64{1, 0, 0, 1, 10, 20}
	for i, v := range expected {
		if math.Abs(got.Data[i]-v) > 1e-10 {
			t.Errorf("data[%d] = %v, want %v", i, got.Data[i], v)
		}
	}
}

func TestTransformsMultiply_TwoTransforms(t *testing.T) {
	// translate(10, 0) then scale(2)
	// reduce(multiply): multiply(translate, scale)
	// translate = [1,0,0,1,10,0], scale = [2,0,0,2,0,0]
	// Result: [2, 0, 0, 2, 10, 0]
	transforms := []TransformItem{
		{Name: "translate", Data: []float64{10, 0}},
		{Name: "scale", Data: []float64{2}},
	}
	got := TransformsMultiply(transforms)
	expected := []float64{2, 0, 0, 2, 10, 0}
	for i, v := range expected {
		if math.Abs(got.Data[i]-v) > 1e-10 {
			t.Errorf("data[%d] = %v, want %v", i, got.Data[i], v)
		}
	}
}

func TestMultiplyTransformMatrices(t *testing.T) {
	// Identity * identity = identity
	identity := []float64{1, 0, 0, 1, 0, 0}
	got := MultiplyTransformMatrices(identity, identity)
	for i, v := range identity {
		if got[i] != v {
			t.Errorf("identity*identity: data[%d] = %v, want %v", i, got[i], v)
		}
	}
}

func TestJS2Transform(t *testing.T) {
	params := &TransformParams{
		FloatPrecision:     3,
		TransformPrecision: 5,
		LeadingZero:        true,
		NegativeExtraSpace: true,
	}

	transforms := []TransformItem{
		{Name: "translate", Data: []float64{10, 20}},
	}
	got := JS2Transform(transforms, params)
	if got != "translate(10 20)" {
		t.Errorf("got %q, want %q", got, "translate(10 20)")
	}
}

func TestJS2Transform_Multiple(t *testing.T) {
	params := &TransformParams{
		FloatPrecision:     3,
		TransformPrecision: 5,
		LeadingZero:        true,
		NegativeExtraSpace: true,
	}

	transforms := []TransformItem{
		{Name: "translate", Data: []float64{10, 20}},
		{Name: "rotate", Data: []float64{45}},
	}
	got := JS2Transform(transforms, params)
	want := "translate(10 20)rotate(45)"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTransformArc_Identity(t *testing.T) {
	// Use a small arc that doesn't need radius scaling.
	// Arc from (0,0) to (10,0) with rx=ry=10: within reach.
	cursor := [2]float64{0, 0}
	arc := []float64{10, 10, 0, 0, 1, 10, 0}
	identity := []float64{1, 0, 0, 1, 0, 0}

	result := TransformArc(cursor, arc, identity)
	if math.Abs(result[0]-10) > 1e-10 {
		t.Errorf("rx = %v, want 10", result[0])
	}
	if math.Abs(result[1]-10) > 1e-10 {
		t.Errorf("ry = %v, want 10", result[1])
	}
}

func TestRoundTransform(t *testing.T) {
	params := &TransformParams{
		FloatPrecision:     2,
		TransformPrecision: 3,
	}

	tr := TransformItem{Name: "translate", Data: []float64{10.1234, 20.5678}}
	RoundTransform(&tr, params)
	if tr.Data[0] != 10.12 {
		t.Errorf("tx = %v, want 10.12", tr.Data[0])
	}
	if tr.Data[1] != 20.57 {
		t.Errorf("ty = %v, want 20.57", tr.Data[1])
	}
}

func TestSmartRound(t *testing.T) {
	// 2.349 with precision 2 should round to 2.35
	data := []float64{2.349}
	got := smartRound(2, data)
	if got[0] != 2.35 {
		t.Errorf("smartRound(2, [2.349]) = %v, want 2.35", got[0])
	}
}

func TestIsIdentityTransform(t *testing.T) {
	tests := []struct {
		item TransformItem
		want bool
	}{
		{TransformItem{Name: "translate", Data: []float64{0, 0}}, true},
		{TransformItem{Name: "translate", Data: []float64{10, 0}}, false},
		{TransformItem{Name: "scale", Data: []float64{1, 1}}, true},
		{TransformItem{Name: "scale", Data: []float64{2, 1}}, false},
		{TransformItem{Name: "rotate", Data: []float64{0}}, true},
		{TransformItem{Name: "rotate", Data: []float64{45}}, false},
		{TransformItem{Name: "skewX", Data: []float64{0}}, true},
	}

	for _, tt := range tests {
		got := isIdentityTransform(&tt.item)
		if got != tt.want {
			t.Errorf("isIdentityTransform(%v) = %v, want %v", tt.item, got, tt.want)
		}
	}
}
