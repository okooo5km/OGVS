// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

package fixture

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// filenamePattern matches SVGO fixture files like "removeComments.01.svg.txt".
var filenamePattern = regexp.MustCompile(`^(.*)\.(\d+)\.svg\.txt$`)

// separatorDesc matches the "===" separator between description and test content.
var separatorDesc = regexp.MustCompile(`\s*===\s*`)

// separatorParts matches the "@@@" separator between input/expected/params.
var separatorParts = regexp.MustCompile(`\s*@@@\s*`)

// normalize trims whitespace and converts all line endings to LF.
// This matches SVGO's normalize function: file.trim().replaceAll(EOL, '\n')
func normalize(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return s
}

// LoadPluginFixture loads and parses a single SVGO fixture file.
//
// The file name must match the pattern "{pluginName}.{index}.svg.txt".
// Returns an error if the file cannot be read or parsed.
func LoadPluginFixture(filePath string) (*PluginCase, error) {
	base := filepath.Base(filePath)
	match := filenamePattern.FindStringSubmatch(base)
	if match == nil {
		return nil, fmt.Errorf("filename %q does not match pattern {plugin}.{index}.svg.txt", base)
	}

	pluginName := match[1]
	index, err := strconv.Atoi(match[2])
	if err != nil {
		return nil, fmt.Errorf("invalid index in filename %q: %w", base, err)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading fixture %q: %w", filePath, err)
	}

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		absPath = filePath
	}

	tc, err := parseFixtureContent(string(data), pluginName, index, absPath)
	if err != nil {
		return nil, fmt.Errorf("parsing fixture %q: %w", base, err)
	}

	return tc, nil
}

// LoadPluginFixtures loads all fixture files from a directory.
//
// It scans for files matching "{pluginName}.{index}.svg.txt" and returns
// all successfully parsed test cases. Files that don't match the pattern
// are silently skipped.
func LoadPluginFixtures(dir string) ([]*PluginCase, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading directory %q: %w", dir, err)
	}

	var cases []*PluginCase
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if !filenamePattern.MatchString(entry.Name()) {
			continue
		}

		filePath := filepath.Join(dir, entry.Name())
		tc, err := LoadPluginFixture(filePath)
		if err != nil {
			return nil, fmt.Errorf("loading %s: %w", entry.Name(), err)
		}

		cases = append(cases, tc)
	}

	return cases, nil
}

// LoadPluginFixturesByName loads all fixture files for a specific plugin.
func LoadPluginFixturesByName(dir, pluginName string) ([]*PluginCase, error) {
	all, err := LoadPluginFixtures(dir)
	if err != nil {
		return nil, err
	}

	var filtered []*PluginCase
	for _, tc := range all {
		if tc.PluginName == pluginName {
			filtered = append(filtered, tc)
		}
	}

	return filtered, nil
}

// parseFixtureContent parses the content of a fixture file into a PluginCase.
//
// Format:
//
//	[Optional description]
//	===
//	[Input SVG]
//	@@@
//	[Expected output SVG]
//	@@@
//	[Optional JSON params]
//
// The "===" separator and description are optional.
// The third "@@@" section (params) is optional.
func parseFixtureContent(data, pluginName string, index int, filePath string) (*PluginCase, error) {
	normalized := normalize(data)

	// Step 1: Remove optional description (split by ===)
	descItems := separatorDesc.Split(normalized, 2)
	var description string
	var testContent string

	if len(descItems) == 2 {
		description = strings.TrimSpace(descItems[0])
		testContent = descItems[1]
	} else {
		testContent = descItems[0]
	}

	// Step 2: Split test content by @@@
	parts := separatorParts.Split(testContent, -1)
	if len(parts) < 2 {
		return nil, fmt.Errorf("expected at least 2 parts separated by @@@, got %d", len(parts))
	}

	input := parts[0]
	expected := parts[1]

	// Step 3: Parse optional params
	var params json.RawMessage
	if len(parts) >= 3 && strings.TrimSpace(parts[2]) != "" {
		raw := strings.TrimSpace(parts[2])
		if !json.Valid([]byte(raw)) {
			return nil, fmt.Errorf("invalid JSON params: %s", raw)
		}
		params = json.RawMessage(raw)
	}

	return &PluginCase{
		PluginName:  pluginName,
		Index:       index,
		Description: description,
		Input:       input,
		Expected:    expected,
		Params:      params,
		FilePath:    filePath,
	}, nil
}
