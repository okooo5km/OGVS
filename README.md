<p align="center">
  <img src="https://raw.githubusercontent.com/svg/svgo/main/logo/logo.svg" width="180" alt="ogvs logo">
</p>

<h1 align="center">ogvs</h1>

<p align="center">
  A Go implementation of <a href="https://github.com/svg/svgo">SVGO</a> — the SVG Optimizer
</p>

<p align="center">
  <a href="https://goreportcard.com/report/github.com/okooo5km/ogvs"><img src="https://goreportcard.com/badge/github.com/okooo5km/ogvs" alt="Go Report Card"></a>
  <a href="https://pkg.go.dev/github.com/okooo5km/ogvs"><img src="https://pkg.go.dev/badge/github.com/okooo5km/ogvs.svg" alt="Go Reference"></a>
  <a href="https://github.com/okooo5km/ogvs/releases"><img src="https://img.shields.io/github/v/release/okooo5km/ogvs" alt="Release"></a>
  <a href="LICENSE"><img src="https://img.shields.io/github/license/okooo5km/ogvs" alt="License"></a>
</p>

---

## Why?

[SVGO](https://github.com/svg/svgo) is the industry-standard SVG optimizer, written in JavaScript. **ogvs** is a ground-up Go rewrite with behavioral compatibility — validated against all 363 SVGO test fixtures.

- **Single binary** — no Node.js or npm required
- **Zero runtime dependencies** — just download and run
- **Fast** — 5x to 23x faster than SVGO ([benchmarks](doc/benchmark.md))
- **Drop-in replacement** — same plugins, same defaults, same output

## Installation

### Homebrew (macOS/Linux)

```bash
brew install okooo5km/tap/ogvs
```

### Go install

```bash
go install github.com/okooo5km/ogvs/cmd/ogvs@latest
```

### GitHub Releases

Download prebuilt binaries from the [Releases](https://github.com/okooo5km/ogvs/releases) page.

## CLI Usage

```bash
# Optimize a file (in-place)
ogvs input.svg

# Optimize a file to a different output
ogvs input.svg -o output.svg

# Optimize from stdin
cat input.svg | ogvs -

# Optimize from string
ogvs -s '<svg><title>test</title></svg>'

# Optimize all SVGs in a folder (recursive)
ogvs -f ./icons -o ./optimized --recursive

# Multi-pass optimization (run until stable)
ogvs input.svg --multipass

# Pretty-print output
ogvs input.svg --pretty --indent 2

# Custom precision for numeric values
ogvs input.svg --precision 2

# Use a custom config file
ogvs input.svg --config ogvs.config.yaml

# List all available plugins
ogvs --show-plugins
```

## Configuration

ogvs automatically discovers `ogvs.config.yaml` or `ogvs.config.yml` in the current directory (searching upward). You can also specify a config file with `--config`.

```yaml
# ogvs.config.yaml
multipass: true
floatPrecision: 3

js2svg:
  pretty: true
  indent: 2

plugins:
  # Use preset-default with overrides
  - name: preset-default
    params:
      overrides:
        removeViewBox: false
        cleanupIds:
          minify: false

  # Enable optional plugins
  - name: removeXlink
  - name: prefixIds
    params:
      prefix: "icon-"
  - name: removeDimensions
```

## Built-in Plugins

ogvs includes all **53 plugins** from SVGO.

### Preset Default (34 plugins)

These plugins run by default (in this order):

| Plugin | Description |
|--------|-------------|
| `removeDoctype` | Remove `<!DOCTYPE>` declaration |
| `removeXMLProcInst` | Remove XML processing instructions |
| `removeComments` | Remove comments |
| `removeDeprecatedAttrs` | Remove deprecated attributes |
| `removeMetadata` | Remove `<metadata>` |
| `removeEditorsNSData` | Remove editors' namespace data |
| `cleanupAttrs` | Clean up attribute whitespace |
| `mergeStyles` | Merge multiple `<style>` elements |
| `inlineStyles` | Move CSS rules to inline `style` attributes |
| `minifyStyles` | Minify `<style>` content |
| `cleanupIds` | Minify and remove unused IDs |
| `removeUselessDefs` | Remove empty or useless `<defs>` |
| `cleanupNumericValues` | Round numeric values, remove defaults |
| `convertColors` | Convert colors to shorter forms |
| `removeUnknownsAndDefaults` | Remove unknown elements/attributes and defaults |
| `removeNonInheritableGroupAttrs` | Remove non-inheritable presentational attributes from groups |
| `removeUselessStrokeAndFill` | Remove useless `stroke` and `fill` |
| `cleanupEnableBackground` | Remove or clean up `enable-background` |
| `removeHiddenElems` | Remove hidden or zero-sized elements |
| `removeEmptyText` | Remove empty `<text>` elements |
| `convertShapeToPath` | Convert basic shapes to `<path>` |
| `convertEllipseToCircle` | Convert `<ellipse>` with equal radii to `<circle>` |
| `moveElemsAttrsToGroup` | Move common attributes to parent `<g>` |
| `moveGroupAttrsToElems` | Move `<g>` attributes to child elements (if single child) |
| `collapseGroups` | Collapse useless `<g>` elements |
| `convertPathData` | Optimize path data: shorten commands, remove redundancies |
| `convertTransform` | Collapse and optimize `transform` attributes |
| `removeEmptyAttrs` | Remove empty attributes |
| `removeEmptyContainers` | Remove empty container elements |
| `mergePaths` | Merge adjacent `<path>` elements |
| `removeUnusedNS` | Remove unused namespace declarations |
| `sortAttrs` | Sort element attributes for consistency |
| `sortDefsChildren` | Sort `<defs>` children for consistency |
| `removeDesc` | Remove `<desc>` |

### Optional Plugins (19 more)

These plugins are not included in the default preset and must be enabled explicitly:

| Plugin | Description |
|--------|-------------|
| `removeXlink` | Replace `xlink:href` with `href` |
| `removeDimensions` | Remove `width`/`height` and add `viewBox` if missing |
| `removeStyleElement` | Remove `<style>` elements |
| `removeScripts` | Remove `<script>` elements |
| `removeRasterImages` | Remove raster image elements |
| `removeViewBox` | Remove `viewBox` when possible |
| `removeAttrs` | Remove attributes by pattern |
| `removeElementsByAttr` | Remove elements by attribute |
| `removeAttributesBySelector` | Remove attributes by CSS selector |
| `addAttributesToSVGElement` | Add attributes to `<svg>` |
| `addClassesToSVGElement` | Add classes to `<svg>` |
| `convertStyleToAttrs` | Convert `style` to presentational attributes |
| `cleanupListOfValues` | Round values in list attributes |
| `convertOneStopGradients` | Convert single-stop gradients to plain colors |
| `removeOffCanvasPaths` | Remove `<path>` elements outside the viewBox |
| `reusePaths` | Replace duplicated `<path>` elements with `<use>` |
| `prefixIds` | Prefix IDs and classes to avoid conflicts |
| `removeEmptyText` | Remove empty text elements |
| `removeNonInheritableGroupAttrs` | Remove non-inheritable group attributes |

## API Usage (Go)

```go
package main

import (
	"fmt"
	"log"

	"github.com/okooo5km/ogvs/internal/core"
)

func main() {
	input := `<svg xmlns="http://www.w3.org/2000/svg">
		<title>Example</title>
		<circle cx="50" cy="50" r="40"/>
	</svg>`

	result, err := core.Optimize(input, &core.Config{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result.Data)
}
```

## Differences from SVGO

ogvs aims for behavioral compatibility with SVGO. Known minor differences:

- **Entity handling**: When the SVG contains a DOCTYPE with entity definitions, ogvs outputs the DOCTYPE before the XML processing instruction (SVGO outputs them in the original order).
- **Inline style colors**: ogvs does not apply `convertColors` to inline `style` attribute values (`fill:blue` stays as-is, while SVGO may convert to `fill:#00f`).

## Benchmark

Measured with [hyperfine](https://github.com/sharkdp/hyperfine) on macOS (Apple Silicon). Both tools produce identical output.

| Scenario | ogvs | svgo | Speedup |
|---|---|---|---|
| Single file (4 KB) | 7.3 ms | 169 ms | **23x** |
| Single file (22 KB) | 11.6 ms | 180 ms | **15x** |
| Single file (217 KB) | 40.6 ms | 225 ms | **5.5x** |
| Multipass (22 KB) | 17.2 ms | 183 ms | **11x** |
| Batch (20 × 22 KB) | 117 ms | 296 ms | **2.5x** |

See [doc/benchmark.md](doc/benchmark.md) for full details and reproduction steps.

## Development

```bash
make build        # Build binary to bin/ogvs
make test         # Run tests with race detection
make test-cover   # Run tests with coverage report
make lint         # Run golangci-lint
make fmt          # Format code
make vet          # Run go vet
make clean        # Remove build artifacts
```

## License

MIT &copy; [okooo5km(十里)](https://github.com/okooo5km)
