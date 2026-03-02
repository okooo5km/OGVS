// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

package removeunknownsanddefaults

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/okooo5km/ogvs/internal/plugin"
	"github.com/okooo5km/ogvs/internal/svgast"
	"github.com/okooo5km/ogvs/internal/testkit/assert"
	"github.com/okooo5km/ogvs/internal/testkit/fixture"
)

const svgoFixturesDir = "/Users/5km/Dev/Web/svgo/test/plugins"

func TestRemoveUnknownsAndDefaults(t *testing.T) {
	if _, err := os.Stat(svgoFixturesDir); os.IsNotExist(err) {
		t.Skipf("SVGO fixtures not available at %s", svgoFixturesDir)
	}

	cases, err := fixture.LoadPluginFixtures(svgoFixturesDir)
	if err != nil {
		t.Fatalf("loading fixtures: %v", err)
	}

	p := plugin.Get("removeUnknownsAndDefaults")
	if p == nil {
		t.Fatal("plugin removeUnknownsAndDefaults not registered")
	}

	l1Passed, l2Passed, failed := 0, 0, 0

	for _, tc := range cases {
		if tc.PluginName != "removeUnknownsAndDefaults" {
			continue
		}

		name := filepath.Base(tc.FilePath)
		t.Run(name, func(t *testing.T) {
			root, err := svgast.ParseSvg(tc.Input, tc.FilePath)
			if err != nil {
				t.Skipf("parse error: %v", err)
				return
			}

			var params map[string]any
			if len(tc.Params) > 0 {
				json.Unmarshal(tc.Params, &params)
			}
			if params == nil {
				params = make(map[string]any)
			}

			info := &plugin.PluginInfo{Path: tc.FilePath}
			visitor := p.Fn(root, params, info)
			if visitor != nil {
				svgast.Visit(root, visitor, nil)
			}

			opts := &svgast.StringifyOptions{
				Pretty:       true,
				Indent:       4,
				UseShortTags: true,
				EOL:          "lf",
				FinalNewline: true,
			}
			output := strings.TrimRight(svgast.StringifySvg(root, opts), "\n")
			expected := strings.TrimRight(tc.Expected, "\n")

			r1 := assert.L1StrictEqual(output, expected)
			if r1.Pass {
				l1Passed++
				return
			}

			r2 := assert.L2CanonicalEqual(output, expected)
			if r2.Pass {
				l2Passed++
				return
			}

			failed++
			t.Errorf("L1: %s\nL2: %s\n\nGot:\n%s\n\nExpected:\n%s", r1.Diff, r2.Diff, output, expected)
		})
	}

	t.Logf("removeUnknownsAndDefaults results: L1=%d L2=%d failed=%d", l1Passed, l2Passed, failed)
}
