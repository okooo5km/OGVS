// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

package assert

// L2Result holds the result of an L2 canonical comparison.
type L2Result struct {
	Pass bool
	Diff string // human-readable diff description, empty if pass
}

// L2CanonicalEqual performs L2 (canonical equivalence) comparison.
//
// It applies the full normalization pipeline to both strings and compares.
// This is more lenient than L1: it tolerates trailing whitespace differences,
// extra blank lines, etc.
//
// Note: Full L2 normalization (attribute sorting, numeric normalization)
// will be enhanced once the XAST parser is available. Currently provides
// text-level normalization.
func L2CanonicalEqual(got, expected string) L2Result {
	gotNorm := NormalizeCanonical(got)
	expNorm := NormalizeCanonical(expected)

	if gotNorm == expNorm {
		return L2Result{Pass: true}
	}

	return L2Result{
		Pass: false,
		Diff: buildDiff(gotNorm, expNorm),
	}
}
