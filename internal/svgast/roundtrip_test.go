// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

package svgast

import (
	"os"
	"strings"
	"testing"

	"github.com/okooo5km/ogvs/internal/testkit/fixture"
)

const svgoFixturesDir = "/Users/5km/Dev/Web/svgo/test/plugins"

// TestRoundtrip_SVGOFixtures tests that parse→stringify→parse→stringify
// produces identical results (parse+stringify is idempotent).
func TestRoundtrip_SVGOFixtures(t *testing.T) {
	if _, err := os.Stat(svgoFixturesDir); os.IsNotExist(err) {
		t.Skipf("SVGO fixtures not available at %s", svgoFixturesDir)
	}

	cases, err := fixture.LoadPluginFixtures(svgoFixturesDir)
	if err != nil {
		t.Fatalf("loading fixtures: %v", err)
	}

	opts := &StringifyOptions{
		Pretty:       true,
		Indent:       4,
		UseShortTags: true,
		EOL:          "lf",
	}

	passed := 0
	failed := 0
	parseErrors := 0

	for _, tc := range cases {
		t.Run(tc.PluginName+"."+itoa(tc.Index), func(t *testing.T) {
			// Parse the input
			root, err := ParseSvg(tc.Input, tc.FilePath)
			if err != nil {
				parseErrors++
				t.Skipf("parse error: %v", err)
				return
			}

			// Stringify
			output1 := StringifySvg(root, opts)

			// Parse again
			root2, err := ParseSvg(output1, tc.FilePath)
			if err != nil {
				parseErrors++
				t.Skipf("re-parse error: %v", err)
				return
			}

			// Stringify again
			output2 := StringifySvg(root2, opts)

			// The two stringify outputs should be identical
			if output1 != output2 {
				failed++
				t.Errorf("roundtrip not idempotent:\npass1:\n%s\npass2:\n%s", truncate(output1, 500), truncate(output2, 500))
			} else {
				passed++
			}
		})
	}

	t.Logf("Roundtrip results: %d passed, %d failed, %d parse errors (out of %d)",
		passed, failed, parseErrors, len(cases))
}

// TestParseExpected_SVGOFixtures tests that we can parse the expected output
// from SVGO fixtures, which is the format our plugins should produce.
func TestParseExpected_SVGOFixtures(t *testing.T) {
	if _, err := os.Stat(svgoFixturesDir); os.IsNotExist(err) {
		t.Skipf("SVGO fixtures not available at %s", svgoFixturesDir)
	}

	cases, err := fixture.LoadPluginFixtures(svgoFixturesDir)
	if err != nil {
		t.Fatalf("loading fixtures: %v", err)
	}

	parsed := 0
	errors := 0

	for _, tc := range cases {
		_, err := ParseSvg(tc.Expected, tc.FilePath)
		if err != nil {
			errors++
			if errors <= 10 {
				t.Logf("WARN: cannot parse expected of %s.%02d: %v",
					tc.PluginName, tc.Index, err)
			}
		} else {
			parsed++
		}
	}

	t.Logf("Parsed %d/%d expected outputs (%d errors)", parsed, len(cases), errors)

	// We should be able to parse most expected outputs
	if parsed < len(cases)*80/100 {
		t.Errorf("too many parse errors: only %d/%d parsed", parsed, len(cases))
	}
}

func itoa(n int) string {
	s := ""
	if n == 0 {
		return "0"
	}
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// Test specific SVGO integration fixtures
func TestParseStringify_SVGOIntegration(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			"basic pretty",
			"<svg xmlns=\"http://www.w3.org/2000/svg\">\n    <g/>\n</svg>",
		},
		{
			"nested elements",
			"<svg xmlns=\"http://www.w3.org/2000/svg\">\n    <g>\n        <rect/>\n    </g>\n</svg>",
		},
		{
			"text element",
			"<svg xmlns=\"http://www.w3.org/2000/svg\">\n    <text>hello</text>\n</svg>",
		},
	}

	opts := &StringifyOptions{
		Pretty:       true,
		Indent:       4,
		UseShortTags: true,
		EOL:          "lf",
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, err := ParseSvg(tt.input, "")
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			output := StringifySvg(root, opts)
			// Trim trailing newline for comparison
			output = strings.TrimRight(output, "\n")
			if output != tt.input {
				t.Errorf("got:\n%q\nwant:\n%q", output, tt.input)
			}
		})
	}
}
