// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package assert provides assertion helpers for comparing SVG outputs
// against SVGO test fixture expectations.
package assert

import (
	"regexp"
	"strings"
)

// NormalizeLF converts all line endings to LF and trims the string.
// This matches SVGO's normalize: file.trim().replaceAll(EOL, '\n')
func NormalizeLF(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return s
}

// trailingWhitespace matches trailing whitespace on each line.
var trailingWhitespace = regexp.MustCompile(`[ \t]+\n`)

// multipleNewlines matches sequences of multiple blank lines.
var multipleNewlines = regexp.MustCompile(`\n{3,}`)

// NormalizeCanonical applies the full L2 normalization pipeline:
//  1. Unify line endings (LF)
//  2. Remove trailing whitespace per line
//  3. Remove excess blank lines (max 1 consecutive blank line)
//  4. Trim leading/trailing whitespace
//
// Note: Attribute sorting and numeric normalization are deferred to
// when we have the XAST parser available (Phase 2+). For now, L2
// provides text-level normalization only.
func NormalizeCanonical(s string) string {
	// Step 1: Unify line endings
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")

	// Step 2: Remove trailing whitespace per line
	s = trailingWhitespace.ReplaceAllString(s, "\n")

	// Step 3: Collapse multiple blank lines
	s = multipleNewlines.ReplaceAllString(s, "\n\n")

	// Step 4: Trim
	s = strings.TrimSpace(s)

	return s
}
