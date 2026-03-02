# OGVS vs SVGO Performance Benchmark

Benchmark date: 2026-03-02
Environment: macOS (Darwin 25.3.0), Apple Silicon (arm64)
Tools: [hyperfine](https://github.com/sharkdp/hyperfine) v1.20.0

- **ogvs**: v0.1.0 (Go native binary)
- **svgo**: v4.0.0 (Node.js)

## Results

| Scenario | File Size | ogvs (mean) | svgo (mean) | Speedup |
|---|---|---|---|---|
| Single file (small) | 4 KB | 7.3 ms | 169.3 ms | **23.2x** |
| Single file (medium) | 22 KB | 11.6 ms | 179.7 ms | **15.5x** |
| Single file (large) | 217 KB | 40.6 ms | 225.0 ms | **5.5x** |
| Multipass (medium) | 22 KB | 17.2 ms | 183.4 ms | **10.7x** |
| Batch folder (20 files) | 20 × 22 KB | 116.8 ms | 295.6 ms | **2.5x** |

## Output Size Comparison

Optimization quality is identical — output sizes match within 1 byte:

| Scenario | ogvs output | svgo output | Diff |
|---|---|---|---|
| Small | 3,714 B | 3,715 B | 1 B |
| Medium | 22,398 B | 22,398 B | 0 |
| Large | 70,137 B | 70,137 B | 0 |
| Multipass | 22,398 B | 22,398 B | 0 |
| Batch (total) | 447,960 B | 447,960 B | 0 |

## Detailed Results

### Single file — Small (4 KB)

```
Benchmark 1: ogvs small.svg
  Time (mean ± σ):       7.3 ms ±   1.1 ms    [User: 4.6 ms, System: 2.2 ms]
  Range (min … max):     6.0 ms …  19.2 ms    239 runs

Benchmark 2: svgo small.svg
  Time (mean ± σ):     169.3 ms ±   8.3 ms    [User: 202.2 ms, System: 31.5 ms]
  Range (min … max):   158.1 ms … 186.0 ms    17 runs

Summary: ogvs ran 23.17 ± 3.70 times faster
```

### Single file — Medium (22 KB)

```
Benchmark 1: ogvs medium.svg
  Time (mean ± σ):      11.6 ms ±   1.2 ms    [User: 8.9 ms, System: 2.9 ms]
  Range (min … max):     9.7 ms …  18.2 ms    207 runs

Benchmark 2: svgo medium.svg
  Time (mean ± σ):     179.7 ms ±   8.8 ms    [User: 225.3 ms, System: 34.1 ms]
  Range (min … max):   170.2 ms … 197.6 ms    16 runs

Summary: ogvs ran 15.48 ± 1.75 times faster
```

### Single file — Large (217 KB)

```
Benchmark 1: ogvs large.svg
  Time (mean ± σ):      40.6 ms ±   2.0 ms    [User: 46.9 ms, System: 5.4 ms]
  Range (min … max):    37.1 ms …  46.1 ms    68 runs

Benchmark 2: svgo large.svg
  Time (mean ± σ):     225.0 ms ±   9.1 ms    [User: 324.5 ms, System: 36.2 ms]
  Range (min … max):   215.6 ms … 249.4 ms    13 runs

Summary: ogvs ran 5.54 ± 0.36 times faster
```

### Multipass — Medium (22 KB)

```
Benchmark 1: ogvs medium.svg --multipass
  Time (mean ± σ):      17.2 ms ±   2.5 ms    [User: 14.8 ms, System: 3.6 ms]
  Range (min … max):    13.9 ms …  29.9 ms    126 runs

Benchmark 2: svgo medium.svg --multipass
  Time (mean ± σ):     183.4 ms ±   4.3 ms    [User: 261.3 ms, System: 33.0 ms]
  Range (min … max):   177.3 ms … 195.2 ms    15 runs

Summary: ogvs ran 10.66 ± 1.54 times faster
```

### Batch folder — 20 files (20 × 22 KB)

```
Benchmark 1: ogvs -f batch -o out-ogvs-batch
  Time (mean ± σ):     116.8 ms ±   6.9 ms    [User: 124.3 ms, System: 14.9 ms]
  Range (min … max):   109.8 ms … 141.1 ms    24 runs

Benchmark 2: svgo -f batch -o out-svgo-batch
  Time (mean ± σ):     295.6 ms ±   4.4 ms    [User: 516.4 ms, System: 41.2 ms]
  Range (min … max):   289.4 ms … 302.6 ms    10 runs

Summary: ogvs ran 2.53 ± 0.15 times faster
```

## Analysis

- **Small files (< 10 KB)**: ogvs is ~23x faster. Node.js startup overhead (~160ms) dominates; ogvs starts in ~5ms.
- **Medium files (~20 KB)**: ogvs is ~15x faster. Still startup-dominated but computation starts to matter.
- **Large files (~200 KB)**: ogvs is ~5.5x faster. Go's native performance advantage holds even as computation increases.
- **Multipass**: ogvs is ~11x faster. Multiple optimization passes amplify Go's computational advantage.
- **Batch processing**: ogvs is ~2.5x faster. svgo processes files concurrently via Node.js async, narrowing the gap. ogvs processes sequentially but each file is much faster individually.

## Test Data

| File | Size | Source |
|---|---|---|
| small.svg | 4 KB | SVGO project logo (`logo/logo-web.svg`) |
| medium.svg | 22 KB | SVGO test fixture (`test/coa/testSvg/test.svg`) |
| large.svg | 217 KB | medium.svg content repeated 10x in wrapper SVG |
| batch/ | 20 × 22 KB | 20 copies of medium.svg |

## Reproduction

```bash
# Install hyperfine
brew install hyperfine

# Build ogvs
make build

# Prepare test data
mkdir -p /tmp/ogvs-bench/batch
cp <small-svg> /tmp/ogvs-bench/small.svg
cp <medium-svg> /tmp/ogvs-bench/medium.svg
# Generate large.svg (repeat medium content 10x)
# Copy medium.svg 20 times into /tmp/ogvs-bench/batch/

# Run benchmarks
hyperfine --warmup 3 \
  'ogvs /tmp/ogvs-bench/small.svg -o /tmp/ogvs-bench/out-ogvs.svg' \
  'node svgo/bin/svgo.js /tmp/ogvs-bench/small.svg -o /tmp/ogvs-bench/out-svgo.svg'
```
