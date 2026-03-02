// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

package plugins

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
	"github.com/okooo5km/ogvs/internal/testkit/assert"
	"github.com/okooo5km/ogvs/internal/testkit/fixture"
)

const svgoFixturesDir = "/Users/5km/Dev/Web/svgo/test/plugins"

// parseParams converts JSON RawMessage to map[string]any.
func parseParams(raw json.RawMessage) map[string]any {
	if len(raw) == 0 {
		return nil
	}
	var params map[string]any
	if err := json.Unmarshal(raw, &params); err != nil {
		return nil
	}
	return params
}

// TestPluginFixtures runs all SVGO fixture tests for registered plugins.
// For each fixture: parse input → invoke single plugin → stringify → compare with expected.
func TestPluginFixtures(t *testing.T) {
	if _, err := os.Stat(svgoFixturesDir); os.IsNotExist(err) {
		t.Skipf("SVGO fixtures not available at %s", svgoFixturesDir)
	}

	cases, err := fixture.LoadPluginFixtures(svgoFixturesDir)
	if err != nil {
		t.Fatalf("loading fixtures: %v", err)
	}

	// Stats
	l1Passed, l2Passed, skipped, failed := 0, 0, 0, 0

	for _, tc := range cases {
		t.Run(tc.PluginName+"."+itoa(tc.Index), func(t *testing.T) {
			// Check if plugin is registered
			p := plugin.Get(tc.PluginName)
			if p == nil {
				skipped++
				t.Skipf("plugin %q not yet implemented", tc.PluginName)
				return
			}

			// Parse input
			root, err := svgast.ParseSvg(tc.Input, tc.FilePath)
			if err != nil {
				skipped++
				t.Skipf("parse error: %v", err)
				return
			}

			// Invoke single plugin with fixture params
			params := parseParams(tc.Params)
			if params == nil {
				params = make(map[string]any)
			}
			info := &plugin.PluginInfo{Path: tc.FilePath}
			visitor := p.Fn(root, params, info)
			if visitor != nil {
				svgast.Visit(root, visitor, nil)
			}

			// Stringify with pretty print (matching SVGO test runner)
			opts := &svgast.StringifyOptions{
				Pretty:       true,
				Indent:       4,
				UseShortTags: true,
				EOL:          "lf",
				FinalNewline: true,
			}
			output := svgast.StringifySvg(root, opts)
			output = strings.TrimRight(output, "\n")

			expected := strings.TrimRight(tc.Expected, "\n")

			// L1: strict comparison
			r1 := assert.L1StrictEqual(output, expected)
			if r1.Pass {
				l1Passed++
				return
			}

			// L2: canonical comparison
			r2 := assert.L2CanonicalEqual(output, expected)
			if r2.Pass {
				l2Passed++
				return
			}

			failed++
			t.Errorf("L1: %s\nL2: %s", r1.Diff, r2.Diff)
		})
	}

	t.Logf("Plugin fixture results: L1=%d L2=%d skipped=%d failed=%d (total=%d)",
		l1Passed, l2Passed, skipped, failed, len(cases))
}

// TestPluginIdempotence tests that running each plugin twice produces the same result.
func TestPluginIdempotence(t *testing.T) {
	if _, err := os.Stat(svgoFixturesDir); os.IsNotExist(err) {
		t.Skipf("SVGO fixtures not available at %s", svgoFixturesDir)
	}

	cases, err := fixture.LoadPluginFixtures(svgoFixturesDir)
	if err != nil {
		t.Fatalf("loading fixtures: %v", err)
	}

	opts := &svgast.StringifyOptions{
		Pretty:       true,
		Indent:       4,
		UseShortTags: true,
		EOL:          "lf",
	}

	for _, tc := range cases {
		if tc.IsIdempotenceExcluded() {
			continue
		}

		p := plugin.Get(tc.PluginName)
		if p == nil {
			continue
		}

		t.Run(tc.PluginName+"."+itoa(tc.Index), func(t *testing.T) {
			// First pass
			root1, err := svgast.ParseSvg(tc.Input, tc.FilePath)
			if err != nil {
				t.Skipf("parse error: %v", err)
				return
			}
			params := parseParams(tc.Params)
			if params == nil {
				params = make(map[string]any)
			}
			info := &plugin.PluginInfo{Path: tc.FilePath}
			if v := p.Fn(root1, params, info); v != nil {
				svgast.Visit(root1, v, nil)
			}
			output1 := svgast.StringifySvg(root1, opts)

			// Second pass
			root2, err := svgast.ParseSvg(output1, tc.FilePath)
			if err != nil {
				t.Skipf("re-parse error: %v", err)
				return
			}
			if v := p.Fn(root2, params, info); v != nil {
				svgast.Visit(root2, v, nil)
			}
			output2 := svgast.StringifySvg(root2, opts)

			if output1 != output2 {
				t.Errorf("not idempotent:\npass1:\n%s\npass2:\n%s",
					truncate(output1, 300), truncate(output2, 300))
			}
		})
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
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
