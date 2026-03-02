// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

package fixture

import (
	"os"
	"path/filepath"
	"testing"
)

// svgoFixturesDir is the path to SVGO's plugin test fixtures.
const svgoFixturesDir = "/Users/5km/Dev/Web/svgo/test/plugins"

func TestParseFixtureContent_NoDescription_NoParams(t *testing.T) {
	// Format: input @@@\n expected (like removeComments.01)
	data := `<svg xmlns="http://www.w3.org/2000/svg">
    <!--- test -->
    <g>
        <!--- test -->
    </g>
</svg>

@@@

<svg xmlns="http://www.w3.org/2000/svg">
    <g/>
</svg>`

	tc, err := parseFixtureContent(data, "removeComments", 1, "/test/removeComments.01.svg.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tc.PluginName != "removeComments" {
		t.Errorf("PluginName = %q, want %q", tc.PluginName, "removeComments")
	}
	if tc.Index != 1 {
		t.Errorf("Index = %d, want %d", tc.Index, 1)
	}
	if tc.Description != "" {
		t.Errorf("Description = %q, want empty", tc.Description)
	}
	if tc.HasParams() {
		t.Error("HasParams() = true, want false")
	}
	if tc.Input == "" {
		t.Error("Input is empty")
	}
	if tc.Expected == "" {
		t.Error("Expected is empty")
	}
}

func TestParseFixtureContent_WithDescription_WithParams(t *testing.T) {
	// Format: description === input @@@ expected @@@ params
	data := `Add multiple attributes without value

===

<svg xmlns="http://www.w3.org/2000/svg">
    test
</svg>

@@@

<svg xmlns="http://www.w3.org/2000/svg" data-icon className={classes}>
    test
</svg>

@@@

{"attributes":["data-icon","className={classes}"]}`

	tc, err := parseFixtureContent(data, "addAttributesToSVGElement", 1, "/test/addAttributesToSVGElement.01.svg.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tc.Description != "Add multiple attributes without value" {
		t.Errorf("Description = %q", tc.Description)
	}
	if !tc.HasParams() {
		t.Error("HasParams() = false, want true")
	}
	if !tc.IsIdempotenceExcluded() {
		t.Error("addAttributesToSVGElement should be idempotence-excluded")
	}
	if tc.MultipassCount() != 1 {
		t.Errorf("MultipassCount() = %d, want 1", tc.MultipassCount())
	}
}

func TestParseFixtureContent_Idempotence(t *testing.T) {
	tc := &PluginCase{PluginName: "removeComments"}
	if tc.IsIdempotenceExcluded() {
		t.Error("removeComments should not be idempotence-excluded")
	}
	if tc.MultipassCount() != 2 {
		t.Errorf("MultipassCount() = %d, want 2", tc.MultipassCount())
	}

	tc2 := &PluginCase{PluginName: "convertTransform"}
	if !tc2.IsIdempotenceExcluded() {
		t.Error("convertTransform should be idempotence-excluded")
	}
	if tc2.MultipassCount() != 1 {
		t.Errorf("MultipassCount() = %d, want 1", tc2.MultipassCount())
	}
}

func TestLoadPluginFixture_RealFile(t *testing.T) {
	path := filepath.Join(svgoFixturesDir, "removeComments.01.svg.txt")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skipf("SVGO fixtures not available at %s", svgoFixturesDir)
	}

	tc, err := LoadPluginFixture(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tc.PluginName != "removeComments" {
		t.Errorf("PluginName = %q, want %q", tc.PluginName, "removeComments")
	}
	if tc.Index != 1 {
		t.Errorf("Index = %d, want %d", tc.Index, 1)
	}
	if tc.Input == "" {
		t.Error("Input is empty")
	}
	if tc.Expected == "" {
		t.Error("Expected is empty")
	}
}

func TestLoadPluginFixtures_AllSVGO(t *testing.T) {
	if _, err := os.Stat(svgoFixturesDir); os.IsNotExist(err) {
		t.Skipf("SVGO fixtures not available at %s", svgoFixturesDir)
	}

	cases, err := LoadPluginFixtures(svgoFixturesDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// SVGO has 363 fixture files
	if len(cases) < 350 {
		t.Errorf("loaded %d fixtures, expected at least 350", len(cases))
	}

	t.Logf("Successfully loaded %d fixtures", len(cases))

	// Verify all fixtures have non-empty input and expected
	for _, tc := range cases {
		if tc.Input == "" {
			t.Errorf("%s.%02d: Input is empty", tc.PluginName, tc.Index)
		}
		if tc.Expected == "" {
			t.Errorf("%s.%02d: Expected is empty", tc.PluginName, tc.Index)
		}
		if tc.PluginName == "" {
			t.Errorf("fixture at %s has empty PluginName", tc.FilePath)
		}
	}

	// Count unique plugins
	plugins := make(map[string]int)
	for _, tc := range cases {
		plugins[tc.PluginName]++
	}
	t.Logf("Fixtures cover %d unique plugins", len(plugins))

	// Verify some known plugins are present
	knownPlugins := []string{
		"removeComments",
		"convertPathData",
		"inlineStyles",
		"cleanupIds",
		"sortAttrs",
	}
	for _, name := range knownPlugins {
		if count, ok := plugins[name]; !ok || count == 0 {
			t.Errorf("expected fixtures for plugin %q, found %d", name, count)
		}
	}
}

func TestLoadPluginFixturesByName(t *testing.T) {
	if _, err := os.Stat(svgoFixturesDir); os.IsNotExist(err) {
		t.Skipf("SVGO fixtures not available at %s", svgoFixturesDir)
	}

	cases, err := LoadPluginFixturesByName(svgoFixturesDir, "convertPathData")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// convertPathData should have many fixtures (37 in SVGO)
	if len(cases) < 30 {
		t.Errorf("loaded %d fixtures for convertPathData, expected at least 30", len(cases))
	}

	for _, tc := range cases {
		if tc.PluginName != "convertPathData" {
			t.Errorf("unexpected plugin %q in filtered results", tc.PluginName)
		}
	}
}

func TestLoadPluginFixture_InvalidFilename(t *testing.T) {
	_, err := LoadPluginFixture("/tmp/notavalidname.txt")
	if err == nil {
		t.Error("expected error for invalid filename")
	}
}

func TestParseFixtureContent_InvalidFormat(t *testing.T) {
	// Missing @@@ separator
	_, err := parseFixtureContent("just some text without separator", "test", 1, "/test.svg.txt")
	if err == nil {
		t.Error("expected error for missing @@@ separator")
	}
}
