// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

package assert

import "testing"

func TestL1StrictEqual_Pass(t *testing.T) {
	result := L1StrictEqual(
		"<svg xmlns=\"http://www.w3.org/2000/svg\">\n    <g/>\n</svg>",
		"<svg xmlns=\"http://www.w3.org/2000/svg\">\n    <g/>\n</svg>",
	)
	if !result.Pass {
		t.Errorf("expected pass, got fail: %s", result.Diff)
	}
}

func TestL1StrictEqual_PassWithTrailingNewline(t *testing.T) {
	// SVGO normalize trims whitespace, so trailing newlines should not matter
	result := L1StrictEqual(
		"<svg/>\n",
		"<svg/>",
	)
	if !result.Pass {
		t.Errorf("expected pass (trim), got fail: %s", result.Diff)
	}
}

func TestL1StrictEqual_PassWithCRLF(t *testing.T) {
	result := L1StrictEqual(
		"<svg>\r\n    <g/>\r\n</svg>",
		"<svg>\n    <g/>\n</svg>",
	)
	if !result.Pass {
		t.Errorf("expected pass (CRLF→LF), got fail: %s", result.Diff)
	}
}

func TestL1StrictEqual_Fail(t *testing.T) {
	result := L1StrictEqual(
		"<svg><g/></svg>",
		"<svg><rect/></svg>",
	)
	if result.Pass {
		t.Error("expected fail, got pass")
	}
	if result.Diff == "" {
		t.Error("expected non-empty diff")
	}
}

func TestL1StrictEqual_FailMultiline(t *testing.T) {
	result := L1StrictEqual(
		"<svg>\n    <g/>\n</svg>",
		"<svg>\n    <rect/>\n</svg>",
	)
	if result.Pass {
		t.Error("expected fail, got pass")
	}
	// Should mention line 2
	if result.Diff == "" {
		t.Error("expected non-empty diff")
	}
}

func TestL2CanonicalEqual_Pass(t *testing.T) {
	result := L2CanonicalEqual(
		"<svg>\n    <g/>\n</svg>",
		"<svg>\n    <g/>\n</svg>",
	)
	if !result.Pass {
		t.Errorf("expected pass, got fail: %s", result.Diff)
	}
}

func TestL2CanonicalEqual_PassTrailingSpaces(t *testing.T) {
	// L2 removes trailing whitespace per line
	result := L2CanonicalEqual(
		"<svg>   \n    <g/>   \n</svg>   ",
		"<svg>\n    <g/>\n</svg>",
	)
	if !result.Pass {
		t.Errorf("expected pass (trailing spaces), got fail: %s", result.Diff)
	}
}

func TestL2CanonicalEqual_PassExtraBlankLines(t *testing.T) {
	result := L2CanonicalEqual(
		"<svg>\n\n\n    <g/>\n\n\n</svg>",
		"<svg>\n\n    <g/>\n\n</svg>",
	)
	if !result.Pass {
		t.Errorf("expected pass (extra blank lines), got fail: %s", result.Diff)
	}
}

func TestL2CanonicalEqual_Fail(t *testing.T) {
	result := L2CanonicalEqual(
		"<svg><g/></svg>",
		"<svg><rect/></svg>",
	)
	if result.Pass {
		t.Error("expected fail, got pass")
	}
}

func TestNormalizeLF(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"  hello  ", "hello"},
		{"hello\r\nworld", "hello\nworld"},
		{"hello\rworld", "hello\nworld"},
		{"\n\nhello\n\n", "hello"},
	}

	for _, tt := range tests {
		got := NormalizeLF(tt.input)
		if got != tt.expected {
			t.Errorf("NormalizeLF(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestNormalizeCanonical(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Trailing spaces removed
		{"hello   \nworld   ", "hello\nworld"},
		// Multiple blank lines collapsed
		{"a\n\n\n\nb", "a\n\nb"},
		// CRLF converted
		{"a\r\nb\r\nc", "a\nb\nc"},
		// Combined: leading spaces on lines are preserved, only trailing removed
		{"  a  \r\n\r\n\r\n  b  \r\n", "a\n\n  b"},
	}

	for _, tt := range tests {
		got := NormalizeCanonical(tt.input)
		if got != tt.expected {
			t.Errorf("NormalizeCanonical(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
