// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

package converttransform

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

func TestConvertTransform(t *testing.T) {
	if _, err := os.Stat(svgoFixturesDir); os.IsNotExist(err) {
		t.Skipf("SVGO fixtures not available at %s", svgoFixturesDir)
	}

	cases, err := fixture.LoadPluginFixtures(svgoFixturesDir)
	if err != nil {
		t.Fatalf("loading fixtures: %v", err)
	}

	p := plugin.Get("convertTransform")
	if p == nil {
		t.Fatal("plugin convertTransform not registered")
	}

	l1Passed, l2Passed, failed := 0, 0, 0

	for _, tc := range cases {
		if tc.PluginName != "convertTransform" {
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

	t.Logf("convertTransform results: L1=%d L2=%d failed=%d", l1Passed, l2Passed, failed)
}

func TestConvertTransform_MalformedValues(t *testing.T) {
	// Transform attributes that supply too few, too many or no usable arguments
	// must be processed like any other, never crash the pipeline.
	values := []string{
		"a", "(((", "foo(1)", "unknown(1,2)", "rotate", "translate",
		"matrix(1)", "matrix(1,2,3)", "matrix(1,2,3,4,5)", "matrix(1 0 0 1 0 0 7)",
		"matrix()", "matrix( )", "matrix(,)", "rotate()", "rotate(,)", "rotate(90 ",
		"scale( ) translate(1)", "translate( ) scale(2)", "translate(10) 5 6",
		"translate(1) rotate", "skewX()", "skewY( , )", "translate(NaN)",
		"rotate(Infinity)", "matrix(1e400,0,0,1e400,0,0)", "skewX(1e400)",
		"translate(1e-400)", "matrix(1)matrix(1)", "scale(2)matrix(1,2)",
	}
	attrs := []string{"transform", "gradientTransform", "patternTransform"}

	p := plugin.Get("convertTransform")
	if p == nil {
		t.Fatal("plugin convertTransform not registered")
	}

	for _, value := range values {
		for _, attr := range attrs {
			t.Run(attr+"="+value, func(t *testing.T) {
				input := `<svg xmlns="http://www.w3.org/2000/svg"><g ` + attr + `="` +
					strings.ReplaceAll(value, `"`, "&quot;") +
					`"><path d="M1 2 3 4z"/></g></svg>`
				root, err := svgast.ParseSvg(input, "malformed.svg")
				if err != nil {
					t.Fatalf("parse error: %v", err)
				}
				visitor := p.Fn(root, map[string]any{}, &plugin.PluginInfo{Path: "malformed.svg"})
				if visitor != nil {
					svgast.Visit(root, visitor, nil)
				}
				svgast.StringifySvg(root, &svgast.StringifyOptions{UseShortTags: true, EOL: "lf"})
			})
		}
	}
}
