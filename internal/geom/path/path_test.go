// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

package path

import (
	"math"
	"testing"
)

func TestParsePathData_Basic(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		want     int // expected number of items
		firstCmd byte
	}{
		{"simple moveto lineto", "M0 0L10 10", 2, 'M'},
		{"relative moveto", "m0 0l10 10", 2, 'm'},
		{"closepath", "M0 0L10 10Z", 3, 'M'},
		{"empty string", "", 0, 0},
		{"only whitespace", "  \t\n", 0, 0},
		{"no leading moveto", "L10 10", 0, 0},
		{"horizontal lineto", "M0 0H10", 2, 'M'},
		{"vertical lineto", "M0 0V10", 2, 'M'},
		{"cubic bezier", "M0 0C10 10 20 20 30 30", 2, 'M'},
		{"quadratic bezier", "M0 0Q10 10 20 20", 2, 'M'},
		{"arc command", "M0 0A25 25 -30 0 1 50 25", 2, 'M'},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParsePathData(tt.input)
			if len(got) != tt.want {
				t.Errorf("ParsePathData(%q): got %d items, want %d", tt.input, len(got), tt.want)
				return
			}
			if tt.want > 0 && got[0].Command != tt.firstCmd {
				t.Errorf("ParsePathData(%q): first command = %c, want %c", tt.input, got[0].Command, tt.firstCmd)
			}
		})
	}
}

func TestParsePathData_ImplicitLineto(t *testing.T) {
	// Subsequent moveto coordinates should be treated as implicit lineto commands
	got := ParsePathData("M0 0 10 10 20 20")
	if len(got) != 3 {
		t.Fatalf("expected 3 items, got %d", len(got))
	}
	if got[0].Command != 'M' {
		t.Errorf("item 0 command = %c, want M", got[0].Command)
	}
	if got[1].Command != 'L' {
		t.Errorf("item 1 command = %c, want L (implicit lineto)", got[1].Command)
	}
	if got[2].Command != 'L' {
		t.Errorf("item 2 command = %c, want L (implicit lineto)", got[2].Command)
	}
}

func TestParsePathData_ArcFlags(t *testing.T) {
	// Arc flags can be 0 or 1 without separators
	got := ParsePathData("M0 0A25 25 -30 0150 25")
	if len(got) != 2 {
		t.Fatalf("expected 2 items, got %d", len(got))
	}
	arcArgs := got[1].Args
	if len(arcArgs) != 7 {
		t.Fatalf("arc args: got %d, want 7", len(arcArgs))
	}
	if arcArgs[3] != 0 {
		t.Errorf("large-arc-flag = %v, want 0", arcArgs[3])
	}
	if arcArgs[4] != 1 {
		t.Errorf("sweep-flag = %v, want 1", arcArgs[4])
	}
	if arcArgs[5] != 50 {
		t.Errorf("x = %v, want 50", arcArgs[5])
	}
}

func TestParsePathData_Commas(t *testing.T) {
	got := ParsePathData("M0,0 L10,10")
	if len(got) != 2 {
		t.Errorf("expected 2 items, got %d", len(got))
	}
}

func TestParsePathData_ScientificNotation(t *testing.T) {
	got := ParsePathData("M1e2 2E3")
	if len(got) != 1 {
		t.Fatalf("expected 1 item, got %d", len(got))
	}
	if got[0].Args[0] != 100 {
		t.Errorf("args[0] = %v, want 100", got[0].Args[0])
	}
	if got[0].Args[1] != 2000 {
		t.Errorf("args[1] = %v, want 2000", got[0].Args[1])
	}
}

func TestParsePathData_NegativeNumbers(t *testing.T) {
	got := ParsePathData("M-10-20L-30-40")
	if len(got) != 2 {
		t.Fatalf("expected 2 items, got %d", len(got))
	}
	if got[0].Args[0] != -10 || got[0].Args[1] != -20 {
		t.Errorf("M args = %v, want [-10, -20]", got[0].Args)
	}
	if got[1].Args[0] != -30 || got[1].Args[1] != -40 {
		t.Errorf("L args = %v, want [-30, -40]", got[1].Args)
	}
}

func TestParsePathData_DecimalWithoutWhole(t *testing.T) {
	got := ParsePathData("M.5.5L.1.2")
	if len(got) != 2 {
		t.Fatalf("expected 2 items, got %d", len(got))
	}
	if got[0].Args[0] != 0.5 || got[0].Args[1] != 0.5 {
		t.Errorf("M args = %v, want [0.5, 0.5]", got[0].Args)
	}
}

func TestParsePathData_DoubleCommaBreaks(t *testing.T) {
	// Double comma should break parsing
	got := ParsePathData("M0 0,,L10 10")
	if len(got) != 1 {
		t.Errorf("expected 1 item (stopped at double comma), got %d", len(got))
	}
}

func TestParsePathData_ArcNoSignOnRadii(t *testing.T) {
	// Arc radii should not accept sign characters
	got := ParsePathData("M0 0A-25 25 0 0 1 50 25")
	if len(got) != 1 {
		t.Errorf("expected 1 item (arc with negative radius should fail), got %d", len(got))
	}
}

func TestStringifyPathData_Basic(t *testing.T) {
	tests := []struct {
		name string
		data []PathDataItem
		prec int
		want string
	}{
		{
			"simple M L",
			[]PathDataItem{
				{Command: 'M', Args: []float64{0, 0}},
				{Command: 'L', Args: []float64{10, 10}},
			},
			-1,
			"M0 0 10 10",
		},
		{
			"M L different",
			[]PathDataItem{
				{Command: 'M', Args: []float64{0, 0}},
				{Command: 'L', Args: []float64{10, 10}},
				{Command: 'H', Args: []float64{20}},
			},
			-1,
			"M0 0 10 10H20",
		},
		{
			"closepath",
			[]PathDataItem{
				{Command: 'M', Args: []float64{0, 0}},
				{Command: 'L', Args: []float64{10, 10}},
				{Command: 'Z'},
			},
			-1,
			"M0 0 10 10Z",
		},
		{
			"negative numbers no space",
			[]PathDataItem{
				{Command: 'M', Args: []float64{0, 0}},
				{Command: 'L', Args: []float64{-10, -20}},
			},
			-1,
			"M0 0-10-20",
		},
		{
			"with precision",
			[]PathDataItem{
				{Command: 'M', Args: []float64{1.23456, 7.89012}},
			},
			2,
			"M1.23 7.89",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StringifyPathData(&StringifyPathDataOptions{
				PathData:  tt.data,
				Precision: tt.prec,
			})
			if got != tt.want {
				t.Errorf("StringifyPathData() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStringifyPathData_CombineCommands(t *testing.T) {
	// Multiple L commands should be combined
	data := []PathDataItem{
		{Command: 'M', Args: []float64{0, 0}},
		{Command: 'L', Args: []float64{10, 10}},
		{Command: 'L', Args: []float64{20, 20}},
		{Command: 'L', Args: []float64{30, 30}},
	}
	got := StringifyPathData(&StringifyPathDataOptions{
		PathData:  data,
		Precision: -1,
	})
	want := "M0 0 10 10 20 20 30 30"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestStringifyPathData_DecimalCompaction(t *testing.T) {
	// Decimal numbers after decimal should omit space
	data := []PathDataItem{
		{Command: 'M', Args: []float64{0.5, 0.5}},
		{Command: 'L', Args: []float64{0.3, 0.4}},
	}
	got := StringifyPathData(&StringifyPathDataOptions{
		PathData:  data,
		Precision: -1,
	})
	want := "M.5.5.3.4"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRoundtrip(t *testing.T) {
	// Parse then stringify should produce valid output
	inputs := []string{
		"M0 0L10 10Z",
		"M100 200C100 100 250 100 250 200S400 300 400 200",
		"M0 0Q50 50 100 0T200 0",
		"M10 80A25 25 -30 0 1 50 25",
	}
	for _, input := range inputs {
		parsed := ParsePathData(input)
		if len(parsed) == 0 {
			t.Errorf("ParsePathData(%q) returned empty", input)
			continue
		}
		output := StringifyPathData(&StringifyPathDataOptions{
			PathData:  parsed,
			Precision: -1,
		})
		if output == "" {
			t.Errorf("StringifyPathData returned empty for input %q", input)
		}
		// Re-parse the output should give same structure
		reparsed := ParsePathData(output)
		if len(reparsed) != len(parsed) {
			t.Errorf("roundtrip: parsed %d items, reparsed %d items for input %q", len(parsed), len(reparsed), input)
			continue
		}
		for j := range parsed {
			if reparsed[j].Command != parsed[j].Command {
				t.Errorf("roundtrip: item %d command mismatch: %c != %c", j, reparsed[j].Command, parsed[j].Command)
			}
			if len(reparsed[j].Args) != len(parsed[j].Args) {
				t.Errorf("roundtrip: item %d args length mismatch: %d != %d", j, len(reparsed[j].Args), len(parsed[j].Args))
				continue
			}
			for k := range parsed[j].Args {
				if math.Abs(reparsed[j].Args[k]-parsed[j].Args[k]) > 1e-10 {
					t.Errorf("roundtrip: item %d arg %d: %v != %v", j, k, reparsed[j].Args[k], parsed[j].Args[k])
				}
			}
		}
	}
}

func TestReadNumber(t *testing.T) {
	tests := []struct {
		input  string
		cursor int
		wantI  int
		wantN  float64
	}{
		{"123", 0, 2, 123},
		{"-45", 0, 2, -45},
		{"0.5", 0, 2, 0.5},
		{".5", 0, 1, 0.5},
		{"1e2", 0, 2, 100},
		{"1.5E-3", 0, 5, 0.0015},
		{"+10", 0, 2, 10},
	}

	for _, tt := range tests {
		gotI, gotN := readNumber(tt.input, tt.cursor)
		if gotI != tt.wantI {
			t.Errorf("readNumber(%q, %d): cursor = %d, want %d", tt.input, tt.cursor, gotI, tt.wantI)
		}
		if math.Abs(gotN-tt.wantN) > 1e-15 {
			t.Errorf("readNumber(%q, %d): number = %v, want %v", tt.input, tt.cursor, gotN, tt.wantN)
		}
	}
}
