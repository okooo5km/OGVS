// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package tools provides numeric and string utility functions
// ported from SVGO's lib/svgo/tools.js.
package tools

import (
	"encoding/base64"
	"math"
	"math/big"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/okooo5km/ogvs/internal/collections"
	"github.com/okooo5km/ogvs/internal/svgast"
)

// JSRound rounds like JavaScript's Math.round: halves round toward +Infinity.
// Go's math.Round rounds halves away from zero, which disagrees with JS for
// negative X.5 values (e.g. -9124.5 → JS -9124 vs math.Round -9125) and can
// cascade into structurally different path output.
func JSRound(x float64) float64 {
	if math.IsNaN(x) || math.IsInf(x, 0) {
		return x
	}
	return math.Floor(x + 0.5)
}

// JSIndex reads data[i] with JavaScript array semantics: an index past the end
// yields undefined, which behaves as NaN in every arithmetic context SVGO uses.
// Ports of SVGO code that destructures or indexes number arrays of unvalidated
// length (transform data, in particular) must go through this so short input
// produces NaN output instead of a panic.
func JSIndex(data []float64, i int) float64 {
	if i < 0 || i >= len(data) {
		return math.NaN()
	}
	return data[i]
}

// JSFalsy reports whether v is falsy as a JavaScript number: 0, -0 and NaN are,
// everything else is not. Pairs with JSIndex, since a missing element reads as
// NaN and JavaScript treats undefined as falsy too.
func JSFalsy(v float64) bool {
	return v == 0 || math.IsNaN(v)
}

// JSOr mirrors the JavaScript `v || alt` idiom for numbers, substituting alt
// whenever v is falsy.
func JSOr(v, alt float64) float64 {
	if JSFalsy(v) {
		return alt
	}
	return v
}

var (
	regReferencesURL   = regexp.MustCompile(`\burl\(["']?#(.+?)["']?\)`)
	regReferencesHref  = regexp.MustCompile(`^#(.+?)$`)
	regReferencesBegin = regexp.MustCompile(`(\w+)\.[a-zA-Z]`)
)

// CleanupOutDataParams controls numeric output formatting.
type CleanupOutDataParams struct {
	NoSpaceAfterFlags  bool
	LeadingZero        bool
	NegativeExtraSpace bool
}

// CleanupOutData converts a row of numbers to an optimized string view.
//
// Example: [0, -1, .5, .5] → "0-1 .5.5"
func CleanupOutData(data []float64, params *CleanupOutDataParams, command byte) string {
	var sb strings.Builder
	var prev float64

	for i, item := range data {
		delimiter := " "

		// No extra space in front of first number
		if i == 0 {
			delimiter = ""
		}

		// No extra space after arc command flags (large-arc and sweep flags)
		if params.NoSpaceAfterFlags && (command == 'A' || command == 'a') {
			pos := i % 7
			if pos == 4 || pos == 5 {
				delimiter = ""
			}
		}

		// Remove leading zeros if enabled
		var itemStr string
		if params.LeadingZero {
			itemStr = RemoveLeadingZero(item)
		} else {
			itemStr = formatFloat(item)
		}

		// No extra space in front of negative number or
		// in front of a floating number if previous was also floating
		if params.NegativeExtraSpace && delimiter != "" {
			if item < 0 || (len(itemStr) > 0 && itemStr[0] == '.' && math.Mod(prev, 1) != 0) {
				delimiter = ""
			}
		}

		prev = item
		sb.WriteString(delimiter)
		sb.WriteString(itemStr)
	}

	return sb.String()
}

// RemoveLeadingZero removes the leading zero from floating-point numbers.
//
// Examples: 0.5 → ".5", -0.5 → "-.5"
func RemoveLeadingZero(value float64) string {
	str := formatFloat(value)

	if value > 0 && value < 1 && strings.HasPrefix(str, "0") {
		return str[1:]
	}

	if value > -1 && value < 0 && len(str) > 1 && str[1] == '0' {
		return string(str[0]) + str[2:]
	}

	return str
}

// ToFixed rounds a number to the specified precision, mirroring SVGO's
// tools.toFixed (`Math.round(num * 10 ** precision) / 10 ** precision`).
// Halves therefore round toward +Infinity: ToFixed(-2.5, 0) == -2.
//
// This is NOT the same function as JavaScript's Number.prototype.toFixed —
// use NativeToFixed for ports of `x.toFixed(p)` call sites.
func ToFixed(num float64, precision int) float64 {
	pow := math.Pow(10, float64(precision))
	return JSRound(num*pow) / pow
}

// NativeToFixed rounds num to precision decimals exactly as JavaScript's
// Number.prototype.toFixed followed by a numeric conversion does, i.e.
// `Number(num.toFixed(precision))`. The decision is made on the exact binary
// value of num and halves round away from zero:
//
//	NativeToFixed(-2.5, 0)    == -3   (ToFixed gives -2)
//	NativeToFixed(-0.0025, 3) == -0.003
//	NativeToFixed(1.005, 2)   == 1    (1.005 is below 1.005 as a double)
//
// Neither ToFixed (halves toward +Infinity) nor strconv.FormatFloat (halves to
// even) can stand in for it. SVGO mixes both helpers, sometimes within one
// function — smartRound in plugins/_transforms.js guards with tools.toFixed but
// produces its values with the native method, and cleanupNumericValues /
// cleanupListOfValues use the native method throughout. Each ported call site
// must use the helper matching the JavaScript it mirrors.
func NativeToFixed(num float64, precision int) float64 {
	if num == 0 {
		// toFixed drops the sign of -0: (-0).toFixed(2) is "0.00".
		return 0
	}
	if math.IsNaN(num) || math.IsInf(num, 0) {
		// Stringified as "NaN"/"Infinity", which convert straight back.
		return num
	}
	if precision < 0 {
		precision = 0
	}
	// From 1e21 up, toFixed falls back to Number.prototype.toString, so the
	// round trip is lossless.
	if math.Abs(num) >= 1e21 {
		return num
	}
	if decimalPlaces(num) <= precision {
		return num
	}

	pow := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(precision)), nil)
	scaled := new(big.Rat).SetFloat64(math.Abs(num))
	scaled.Mul(scaled, new(big.Rat).SetInt(pow))
	// floor(scaled + 1/2) on a non-negative value rounds halves up in
	// magnitude, which is the tie-break toFixed specifies.
	scaled.Add(scaled, big.NewRat(1, 2))
	n := new(big.Int).Quo(scaled.Num(), scaled.Denom())
	result, _ := new(big.Rat).SetFrac(n, pow).Float64()
	if num < 0 {
		result = -result
	}
	return result
}

// decimalPlaces returns the number of digits after the decimal point in the
// shortest decimal representation that round-trips to f.
func decimalPlaces(f float64) int {
	str := strconv.FormatFloat(f, 'f', -1, 64)
	dot := strings.IndexByte(str, '.')
	if dot < 0 {
		return 0
	}
	return len(str) - dot - 1
}

// HasScripts checks if a node contains any scripts.
// This checks the node's own properties, not parents or children.
func HasScripts(node *svgast.Element) bool {
	if node.Name == "script" && len(node.Children) > 0 {
		return true
	}

	if node.Name == "a" {
		for _, entry := range node.Attributes.Entries() {
			if entry.Name == "href" || strings.HasSuffix(entry.Name, ":href") {
				if strings.HasPrefix(strings.TrimLeft(entry.Value, " \t\n\r"), "javascript:") {
					return true
				}
			}
		}
	}

	// Check for event handler attributes
	for _, entry := range node.Attributes.Entries() {
		if isScriptEventAttr(entry.Name) {
			return true
		}
	}

	return false
}

// IncludesURLReference checks if a string contains a url(#ref) reference.
func IncludesURLReference(body string) bool {
	return regReferencesURL.MatchString(body)
}

// FindReferences extracts all URL/href references from an attribute value.
func FindReferences(attribute, value string) []string {
	var results []string

	if collections.ReferencesProps[attribute] {
		matches := regReferencesURL.FindAllStringSubmatch(value, -1)
		for _, m := range matches {
			results = append(results, m[1])
		}
	}

	if attribute == "href" || strings.HasSuffix(attribute, ":href") {
		if m := regReferencesHref.FindStringSubmatch(value); m != nil {
			results = append(results, m[1])
		}
	}

	if attribute == "begin" {
		if m := regReferencesBegin.FindStringSubmatch(value); m != nil {
			results = append(results, m[1])
		}
	}

	// Decode URI-encoded references
	for i, ref := range results {
		if decoded, err := url.PathUnescape(ref); err == nil {
			results[i] = decoded
		}
	}

	return results
}

// EncodeSVGDataURI encodes an SVG string as a data URI.
// Type can be "base64", "enc" (URI encoded), or "unenc" (unencoded).
func EncodeSVGDataURI(str string, dataType string) string {
	prefix := "data:image/svg+xml"
	switch dataType {
	case "base64", "":
		return prefix + ";base64," + base64.StdEncoding.EncodeToString([]byte(str))
	case "enc":
		return prefix + "," + url.PathEscape(str)
	case "unenc":
		return prefix + "," + str
	default:
		return prefix + ";base64," + base64.StdEncoding.EncodeToString([]byte(str))
	}
}

// DecodeSVGDataURI decodes a data URI back to an SVG string.
func DecodeSVGDataURI(str string) string {
	re := regexp.MustCompile(`data:image/svg\+xml(;charset=[^;,]*)?(;base64)?,(.*)`)
	match := re.FindStringSubmatch(str)
	if match == nil {
		return str
	}

	data := match[3]
	if match[2] != "" {
		// base64
		decoded, err := base64.StdEncoding.DecodeString(data)
		if err == nil {
			return string(decoded)
		}
		return str
	} else if len(data) > 0 && data[0] == '%' {
		// URI encoded
		decoded, err := url.PathUnescape(data)
		if err == nil {
			return decoded
		}
		return str
	} else if len(data) > 0 && data[0] == '<' {
		return data
	}

	return str
}

// FormatNumber formats a float64 the way JavaScript's Number.prototype.toString()
// does, which is what SVGO relies on for all numeric output.
func FormatNumber(f float64) string {
	return formatFloat(f)
}

// formatFloat formats a float64 to a string, removing trailing zeros.
// Negative zero is normalized to positive zero to match JS behavior.
func formatFloat(f float64) string {
	// Normalize negative zero to positive zero
	if f == 0 {
		return "0"
	}
	// JS spells the infinities out; Go's strconv would emit "+Inf"/"-Inf".
	if math.IsInf(f, 1) {
		return "Infinity"
	}
	if math.IsInf(f, -1) {
		return "-Infinity"
	}
	// Match JS Number.prototype.toString(): decimal notation for the range
	// [1e-6, 1e21), exponential outside it. Go's %g switches to exponential
	// at a much lower magnitude (|v| >= 1e6), which corrupts large
	// coordinates/viewBoxes and breaks SVGO byte-compatibility.
	abs := math.Abs(f)
	if abs < 1e-6 || abs >= 1e21 {
		// Exponential; Go pads the exponent to two digits ("1e-07"),
		// JS uses minimal digits ("1e-7") — strip the leading zeros.
		return stripExponentLeadingZeros(strconv.FormatFloat(f, 'e', -1, 64))
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// stripExponentLeadingZeros removes leading zeros from the exponent part
// of a number string. "1e-07" → "1e-7", "1e+08" → "1e+8", "1e-07" → "1e-7".
func stripExponentLeadingZeros(s string) string {
	eIdx := strings.IndexByte(s, 'e')
	if eIdx < 0 {
		return s
	}
	prefix := s[:eIdx+1] // "1e" or "-3.5e"
	exp := s[eIdx+1:]    // "-07" or "+8" or "7"
	if len(exp) == 0 {
		return s
	}

	sign := ""
	digits := exp
	if exp[0] == '+' || exp[0] == '-' {
		sign = string(exp[0])
		digits = exp[1:]
	}

	// Remove leading zeros from digits
	i := 0
	for i < len(digits)-1 && digits[i] == '0' {
		i++
	}
	return prefix + sign + digits[i:]
}

// isScriptEventAttr checks if an attribute name is a script event handler.
func isScriptEventAttr(name string) bool {
	return collections.AnimationEventAttrs[name] ||
		collections.DocumentEventAttrs[name] ||
		collections.DocumentElementEventAttrs[name] ||
		collections.GlobalEventAttrs[name] ||
		collections.GraphicalEventAttrs[name]
}
