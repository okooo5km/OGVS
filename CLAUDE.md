# OGVS - Go Implementation of SVGO

## Project Overview

OGVS is a Go rewrite of [SVGO](https://github.com/nicerobot/svgo) — an SVG optimizer.
The goal is behavioral compatibility with SVGO, validated against SVGO's 363 test fixtures.

## Quick Commands

```bash
make build        # Build binary to bin/ogvs
make test         # Run tests with race detection
make test-cover   # Run tests with coverage report
make lint         # Run golangci-lint
make fmt          # Format code
make vet          # Run go vet
make clean        # Remove build artifacts
```

## Module Layout

```
cmd/ogvs/          CLI entry point
internal/
  version/         Version constant
  svgast/          XAST data model + parser + visitor + stringifier
  core/            Optimize pipeline + multipass + config
  plugin/          Plugin interface + registry + preset
  plugins/         All plugin implementations (one sub-package per plugin)
  collections/     SVG spec constants (element groups, attribute groups, colors)
  geom/            Path parse/stringify + transform algebra + arc conversion
  css/             CSS parsing + cascade + selector matching
  tools/           Numeric utilities + encoding + reference detection
  testkit/         Test framework (fixture loader + L1/L2 assertions + reports)
  cli/             CLI argument handling + config loading + file I/O
```

## Development Guidelines

- **Language**: Code and comments in English; communication in Chinese
- **Author**: okooo5km(十里)
- **License**: MIT
- **Go version**: 1.24+
- **TDD**: All features must be validated against SVGO test fixtures
- **Golden standard**: SVGO fixture files at `/Users/5km/Dev/Web/svgo/test/plugins/`

## SVGO Source Reference

- **SVGO project path**: `/Users/5km/Dev/Web/svgo`
- **Core code**: `/Users/5km/Dev/Web/svgo/lib`
- **Plugins**: `/Users/5km/Dev/Web/svgo/plugins`
- **Test fixtures**: `/Users/5km/Dev/Web/svgo/test/plugins` (363 files)

## Testing

Tests use SVGO fixture files as the golden standard:
- **L1 (Strict)**: Byte-identical output comparison
- **L2 (Canonical)**: Normalized comparison (LF, trim, attribute order, numeric normalization)

Run a specific plugin test:
```bash
go test -v -run TestRemoveComments ./internal/plugins/removecomments/
```

## Architecture Decisions

- XML parsing: `encoding/xml` Decoder with RawToken (preserves original namespace prefixes)
- CSS parsing: `tdewolff/parse/v2/css`
- CLI framework: `spf13/cobra`
- Path data parsing: Custom implementation (SVG Path BNF state machine)
- Attribute ordering: Custom `OrderedAttrs` type (Go maps are unordered)

## Phase Progress

- **Phase 0**: Project scaffolding — go.mod, Makefile, CI, CLAUDE.md ✅
- **Phase 1**: Test infrastructure — fixture loader (363 cases), L1/L2 assertions ✅
- **Phase 2**: Core engine — XAST model, parser, visitor, stringifier, optimize pipeline ✅
  - Roundtrip test: 363/363 passed, 0 failed, 0 errors
  - Expected output parse: 358/363 parsed (5 errors: HTML boolean attrs in expected output)
- **Phase 3**: Plugin framework — PluginFunc interface, registry, preset system, invokePlugins engine ✅
  - 21 tests: registry CRUD, param merging, sequential execution, preset overrides, float precision
- **Phase 4**: Wave 0 — 7 simple removal plugins + fixture test framework ✅
  - Plugins: removeDoctype, removeXMLProcInst, removeComments, removeMetadata, removeTitle, removeDesc, removeXMLNS
  - Fixture tests: 10/10 L1 passed, 0 failed, idempotence all passed
- **Phase 5**: Wave 1 — 7 attribute/structure plugins ✅
  - Plugins: cleanupAttrs, removeEmptyAttrs, removeDimensions, removeUnusedNS, sortAttrs, sortDefsChildren, removeEditorsNSData
  - Parser fix: switched to RawToken() to preserve original namespace prefixes (fixes same-URI-different-prefix issue)
  - Fixture tests: 33/33 L1 passed (Wave 0 + Wave 1), 0 failed, idempotence all passed
- **Phase 6**: Tool infrastructure ✅
  - collections/ — 5 files: element groups (11), attribute groups (15), colors (148+32), references, pseudo-classes (9)
  - tools/ — CleanupOutData, RemoveLeadingZero, ToFixed, HasScripts, FindReferences, EncodeSVGDataURI
  - geom/path/ — ParsePathData, StringifyPathData (SVG path d="" BNF state machine)
  - geom/transform/ — Transform2JS, TransformsMultiply, MatrixToTransform, TransformArc, JS2Transform
  - css/ — ParseStyleDeclarations, ParseStylesheet, CollectStylesheet, ComputeStyle, CSS selector matching
  - All tests passing, zero regressions
- **Phase 7**: Wave 2 — 5 CSS plugins ✅
  - Plugins: mergeStyles, inlineStyles, minifyStyles, convertStyleToAttrs, removeAttributesBySelector
  - New css/compact.go: CompactStylesheet — compact CSS generator preserving at-rules while removing specified selectors
  - CSS selector matching: added :not() support, evaluatable pseudo-class distinction
  - Fixture tests: 92/92 L1 passed — ALL PASS
  - Fixes applied: CDATA detection via InputOffset(), CSS escape unescaping, CSS error recovery, shorthand collapse (padding/margin)
- **Phase 8**: Wave 3 — 6 geometry/numeric plugins ✅
  - Plugins: convertEllipseToCircle, removeViewBox, cleanupNumericValues, cleanupListOfValues, convertColors, convertShapeToPath
  - Fixture tests: 115/115 L1 passed (Wave 0 + Wave 1 + Wave 2 + Wave 3) — ALL PASS
  - Idempotence: all passed
- **Phase 9**: Wave 4 — 12 removal/structure plugins ✅
  - Plugins: removeStyleElement, removeRasterImages, removeNonInheritableGroupAttrs, removeEmptyText, removeUselessDefs, removeScripts, moveGroupAttrsToElems, removeElementsByAttr, addAttributesToSVGElement, addClassesToSVGElement, removeAttrs, removeEmptyContainers
  - Fixes: charsetReader ignores encoding declaration (matches SVGO), UndefinedAttrValue for HTML-style boolean attrs, regexp2 fallback for PCRE lookaheads
  - Fixture tests: 168/168 L1 passed (Wave 0-4) — ALL PASS
  - Idempotence: all passed
- **Phase 9.5**: Wave 5 — 11 remaining preset-default plugins ✅
  - Infrastructure: collections/elems_config.go (82 SVG element definitions), css/includes_attr_selector.go, geom/path/intersects.go (GJK algorithm), geom/path/apply_transforms.go, geom/path/optimize.go (~1000 lines of path optimization)
  - Plugins (11): removeDeprecatedAttrs, cleanupEnableBackground, removeUselessStrokeAndFill, moveElemsAttrsToGroup, collapseGroups, removeHiddenElems, cleanupIds, removeUnknownsAndDefaults, convertTransform, convertPathData, mergePaths
  - Fixture tests: 331/331 L1 passed (Wave 0-5) — ALL PASS, 0 failed, 32 skipped (non-preset-default)
  - Idempotence: all passed
  - Key fixes: negative zero normalization, Go value vs JS reference semantics in makeArcs (output[0] = arc sync), applyTransforms must write back to d attr (no pathJS cache in Go), textElems case sensitivity (textPath not textpath), Go RE2 backreference workaround
  - Note: plugins compile and are registered via init() but not yet imported in register.go
- **Phase 10**: Wave 6 — 5 remaining optional plugins ✅
  - Plugins (5): removeOffCanvasPaths, removeXlink, convertOneStopGradients, reusePaths, prefixIds
  - Fixture tests: 363/363 L1 passed (ALL fixtures) — FULL COVERAGE, 0 failed, 0 skipped
  - Idempotence: all passed
  - Total plugins: 53 (48 preset-default + 5 optional)
  - Key fixes: Go RE2 backreference workaround for url() pattern matching (3-group pattern + quote validation)
  - prefixIds: tdewolff CSS tokenizer-based CSS rewriting (ID/class selectors, url() references, @keyframes)
  - All plugins registered in register.go via init() imports
- **Phase 11**: CLI ✅
  - Framework: spf13/cobra for CLI, gopkg.in/yaml.v3 for config
  - Files: `internal/cli/cli.go` (root command + flags), `internal/cli/run.go` (core execution), `internal/cli/config.go` (YAML config loading)
  - Input sources: positional args, `-i`, `-s`/`--string`, `-f`/`--folder`, stdin (pipe or `-`)
  - Output: `-o` file/folder, stdout (`-`), in-place overwrite (default for files)
  - Options: `--precision`, `--multipass`, `--pretty`, `--indent`, `--eol`, `--final-newline`, `--datauri`, `--config`, `--recursive`, `--exclude`, `--quiet`, `--show-plugins`
  - Config: YAML discovery (ogvs.config.yaml/.yml upward search), `--config` override, CLI flags override config file
  - preset-default auto-registered via `register.go` init()
  - All 363/363 fixture tests still passing — zero regressions
- **Phase 12**: Integration regression ✅
  - **Layer 1: Core integration tests** — `internal/core/optimize_integration_test.go` (20 tests)
    - Testdata: `internal/core/testdata/` (8 fixture files ported from SVGO `test/svgo/`)
    - Fixture-based tests (9): PrettyIndent2, PluginOrder, EmptySVG, StyleSpecificity, Whitespaces, KeyframeSelectors, Entities, PreElement, PreElementPretty
    - Programmatic tests (11): MultipassConvergence, EmptyPluginsList, FloatPrecisionOverride, PresetOverrideDisable, PresetOverrideParams, InvalidSVG, NilConfig, EmptyStringInput, CustomPluginFn, EOLOption, FinalNewline
    - Known differences from SVGO (documented in test comments):
      - entities.svg.txt: DOCTYPE output before ProcInst (OGVS parser pre-scans DOCTYPE for entity extraction)
      - style-specificity.svg.txt: `fill:blue` instead of `fill:#00f` (OGVS does not apply convertColors to inline style values)
  - **Layer 2: CLI end-to-end tests** — `internal/cli/cli_test.go` (39 tests)
    - TestMain builds binary once, all subtests reuse it via `os/exec`
    - Coverage: --version, --show-plugins, --string, stdin pipe, file I/O, folder recursive, --pretty/--indent, --multipass, --eol/--final-newline, --datauri (base64/enc/unenc), --config, --quiet, --precision, error handling, stats output
    - Data URI tests write to file (never stdout) per project guidelines
  - **Layer 3: Config unit tests** — `internal/cli/config_test.go` (28 tests)
    - discoverConfig (6): yaml/yml discovery, priority, nested search, not found
    - loadConfig (8): full YAML, empty, invalid, partial, floatPrecision:0, js2svg only
    - resolvePluginConfigs (14): string, object, mixed, invalid types, missing name, empty/nil
  - **Total: 87 new integration/CLI/config tests, all passing**
  - All 363/363 fixture tests still passing — zero regressions
- **Phase 13**: Open-source release preparation ✅
  - LICENSE: MIT license file
  - README.md: Full rewrite with badges, installation (Homebrew/go install/releases), CLI usage, config examples, all 53 plugins listed, API usage, known differences
  - .goreleaser.yaml: Multi-platform builds (linux/darwin/windows × amd64/arm64), Homebrew tap (okooo5km/homebrew-tap), checksums, changelog
  - Version: 0.1.0-dev → 0.1.0
  - Cleanup: removed .temp_convert.log, GO_REWRITE_BLUEPRINT.md, hardcoded SVGO_FIXTURES path from Makefile
