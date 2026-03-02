// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package tools provides numeric and string utility functions
// ported from SVGO's lib/svgo/tools.js.
package tools

import (
	"encoding/base64"
	"fmt"
	"math"
	"net/url"
	"regexp"
	"strings"

	"github.com/okooo5km/ogvs/internal/collections"
	"github.com/okooo5km/ogvs/internal/svgast"
)

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

// ToFixed rounds a number to the specified precision.
// Unlike fmt.Sprintf, this returns a float64.
func ToFixed(num float64, precision int) float64 {
	pow := math.Pow(10, float64(precision))
	return math.Round(num*pow) / pow
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

// formatFloat formats a float64 to a string, removing trailing zeros.
// Negative zero is normalized to positive zero to match JS behavior.
func formatFloat(f float64) string {
	// Normalize negative zero to positive zero
	if f == 0 {
		f = 0
	}
	s := fmt.Sprintf("%g", f)
	// Go's %g uses at least two digits for the exponent (e.g., "1e-07"),
	// but JS uses minimal digits (e.g., "1e-7"). Strip leading zeros
	// from the exponent to match JS behavior.
	return stripExponentLeadingZeros(s)
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
