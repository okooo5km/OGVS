# OGVS Architecture

## Overview

ogvs follows the same pipeline architecture as SVGO:

```
SVG String → Parser → XAST → Plugin Pipeline → Stringifier → Optimized SVG
```

## Core Pipeline

### 1. Parser (`internal/svgast`)
Converts SVG XML string to an XAST (XML Abstract Syntax Tree).

Node types:
- `Root` — top-level container
- `Element` — SVG/XML element with name, ordered attributes, children
- `Text` — text content
- `Comment` — XML comment
- `Cdata` — CDATA section
- `Instruction` — processing instruction (e.g., `<?xml ...?>`)
- `Doctype` — DOCTYPE declaration

### 2. Visitor (`internal/svgast`)
DFS traversal with enter/exit callbacks per node type.
Supports `VisitSkip` to skip children and safe node removal during iteration.

### 3. Plugin System (`internal/plugin`)
- Plugins return a `Visitor` for transforming the AST
- Plugins are executed sequentially in a defined order
- Presets bundle multiple plugins with override support
- Default preset contains 34 plugins

### 4. Stringifier (`internal/svgast`)
Converts XAST back to SVG string with configurable formatting:
- Pretty printing with indentation
- Short tag mode (`<tag/>` vs `<tag></tag>`)
- Entity encoding
- EOL handling (LF/CRLF)

### 5. Optimize (`internal/core`)
Multipass optimization loop (1-10 iterations until convergence).

## Plugin Waves

Plugins are implemented in order of complexity:

| Wave | Type | Plugins | Complexity |
|------|------|---------|------------|
| 0 | Simple removal | 7 | Minimal |
| 1 | Attribute/structure | 7 | Low |
| 2 | Style/CSS | 5 | Medium-High |
| 3 | Geometry (medium) | 6 | Medium |
| 4 | Geometry (high risk) | 6+ | High-Expert |

## Testing Strategy

Three-level assertion system:
- **L1**: Strict byte-identical comparison
- **L2**: Canonical equivalence (normalized comparison)
- **L3**: Visual equivalence (pixel-level rendering comparison)

All validation against SVGO's 363 test fixture files.

## Quality Gates

- **Gate-1**: Wave 0+1, L1/L2 ≥ 95%, parser tests 100%
- **Gate-2**: Wave 2+3, L1/L2 ≥ 92%, L3 ≥ 99%, idempotence ≥ 98%
- **Gate-3**: All plugins ≥ 95%, no unexplained regressions
