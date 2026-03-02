// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

package assert

// L1Result holds the result of an L1 strict comparison.
type L1Result struct {
	Pass bool
	Diff string // human-readable diff description, empty if pass
}

// L1StrictEqual performs L1 (strict snapshot) comparison.
//
// It normalizes both strings (trim + LF) and then compares them byte-for-byte.
// This matches SVGO's test assertion: expect(normalize(result.data)).toStrictEqual(should)
func L1StrictEqual(got, expected string) L1Result {
	gotNorm := NormalizeLF(got)
	expNorm := NormalizeLF(expected)

	if gotNorm == expNorm {
		return L1Result{Pass: true}
	}

	return L1Result{
		Pass: false,
		Diff: buildDiff(gotNorm, expNorm),
	}
}

// buildDiff creates a simple diff description showing first difference.
func buildDiff(got, expected string) string {
	gotLines := splitLines(got)
	expLines := splitLines(expected)

	maxLines := max(len(gotLines), len(expLines))

	for i := range maxLines {
		var gotLine, expLine string
		if i < len(gotLines) {
			gotLine = gotLines[i]
		}
		if i < len(expLines) {
			expLine = expLines[i]
		}
		if gotLine != expLine {
			return lineContext(i+1, gotLine, expLine)
		}
	}

	return "strings differ but no line-level difference found"
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	lines := make([]string, 0)
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start <= len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func lineContext(lineNum int, got, expected string) string {
	return "line " + itoa(lineNum) + ":\n" +
		"  got:      " + quote(got) + "\n" +
		"  expected: " + quote(expected)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

func quote(s string) string {
	if len(s) > 120 {
		return "\"" + s[:120] + "\"..."
	}
	return "\"" + s + "\""
}
