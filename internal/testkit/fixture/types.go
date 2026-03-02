// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package fixture provides loading and parsing of SVGO test fixture files.
package fixture

import "encoding/json"

// PluginCase represents a single plugin test case loaded from a *.svg.txt file.
//
// The fixture format (from SVGO) is:
//
//	[Optional description]
//	===
//	[Input SVG]
//	@@@
//	[Expected output SVG]
//	@@@
//	[Optional JSON params]
type PluginCase struct {
	// PluginName is the plugin identifier, e.g. "removeComments".
	PluginName string

	// Index is the numeric index from the filename, e.g. 1 for "removeComments.01.svg.txt".
	Index int

	// Description is the optional text before the "===" separator.
	Description string

	// Input is the original SVG content (before optimization).
	Input string

	// Expected is the expected SVG output (after optimization).
	Expected string

	// Params is the optional JSON parameters for the plugin.
	// nil means no parameters (use plugin defaults).
	Params json.RawMessage

	// FilePath is the absolute path to the source fixture file.
	FilePath string
}

// HasParams returns true if this test case has plugin parameters.
func (tc *PluginCase) HasParams() bool {
	return len(tc.Params) > 0
}

// IsIdempotenceExcluded returns true if this plugin is excluded from
// idempotence testing (should only run 1 pass instead of 2).
//
// From SVGO: const exclude = ['addAttributesToSVGElement', 'convertTransform'];
func (tc *PluginCase) IsIdempotenceExcluded() bool {
	return tc.PluginName == "addAttributesToSVGElement" ||
		tc.PluginName == "convertTransform"
}

// MultipassCount returns how many passes to run for idempotence testing.
// Returns 1 for excluded plugins, 2 for all others.
func (tc *PluginCase) MultipassCount() int {
	if tc.IsIdempotenceExcluded() {
		return 1
	}
	return 2
}
