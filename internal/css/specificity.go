// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

package css

import (
	"strings"
)

// CompareSpecificity compares two specificity tuples.
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
func CompareSpecificity(a, b Specificity) int {
	for i := 0; i < 4; i++ {
		if a[i] < b[i] {
			return -1
		} else if a[i] > b[i] {
			return 1
		}
	}
	return 0
}

// CalculateSpecificity calculates the CSS specificity of a selector string.
// Returns [inline, ids, classes, elements].
// This is a simplified implementation for SVG selectors.
func CalculateSpecificity(selector string) Specificity {
	var spec Specificity
	// spec[0] = inline style (always 0 for stylesheet rules)

	// Remove content inside brackets to avoid false matches
	clean := removeBracketContent(selector)

	// Count IDs (#)
	spec[1] = strings.Count(clean, "#")

	// Count classes (.), attribute selectors ([), and pseudo-classes (:)
	spec[2] = strings.Count(clean, ".")
	spec[2] += countBracketSelectors(selector)
	spec[2] += countPseudoClasses(clean)

	// Count element selectors and pseudo-elements (::)
	spec[3] = countElementSelectors(clean)
	spec[3] += countPseudoElements(clean)

	return spec
}

// removeBracketContent removes content inside [] brackets.
func removeBracketContent(s string) string {
	var result strings.Builder
	depth := 0
	for _, ch := range s {
		if ch == '[' {
			depth++
			continue
		}
		if ch == ']' {
			depth--
			continue
		}
		if depth == 0 {
			result.WriteRune(ch)
		}
	}
	return result.String()
}

// countBracketSelectors counts the number of attribute selectors [attr].
func countBracketSelectors(s string) int {
	count := 0
	for _, ch := range s {
		if ch == '[' {
			count++
		}
	}
	return count
}

// countPseudoClasses counts single-colon pseudo-classes (not pseudo-elements).
func countPseudoClasses(s string) int {
	count := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ':' {
			// Check it's not a pseudo-element (::)
			if i+1 < len(s) && s[i+1] == ':' {
				i++ // skip the second colon
				continue
			}
			count++
		}
	}
	return count
}

// countPseudoElements counts :: pseudo-elements.
func countPseudoElements(s string) int {
	return strings.Count(s, "::")
}

// countElementSelectors counts element type selectors.
func countElementSelectors(s string) int {
	// Remove IDs, classes, pseudo-classes, pseudo-elements from selector
	// Then count remaining identifiers
	count := 0

	// Split by combinators and count element names
	parts := splitByCombinators(s)
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || part == "*" {
			continue
		}

		// Remove IDs (#id), classes (.class), pseudo-classes (:hover), pseudo-elements (::before)
		clean := removeIDClassPseudo(part)
		clean = strings.TrimSpace(clean)

		if clean != "" && clean != "*" {
			count++
		}
	}

	return count
}

// splitByCombinators splits selector by combinator characters.
func splitByCombinators(s string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ' ' || s[i] == '>' || s[i] == '+' || s[i] == '~' {
			if start < i {
				parts = append(parts, s[start:i])
			}
			start = i + 1
		}
	}
	if start < len(s) {
		parts = append(parts, s[start:])
	}
	return parts
}

// removeIDClassPseudo removes #id, .class, :pseudo from a simple selector.
func removeIDClassPseudo(s string) string {
	var result strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '#' || s[i] == '.' || s[i] == ':' {
			// Skip until next # . : or end
			i++
			for i < len(s) && s[i] != '#' && s[i] != '.' && s[i] != ':' &&
				s[i] != ' ' && s[i] != '>' && s[i] != '+' && s[i] != '~' {
				i++
			}
		} else {
			result.WriteByte(s[i])
			i++
		}
	}
	return result.String()
}

// evaluatablePseudoClasses are pseudo-classes that can be evaluated at optimization
// time (they don't depend on runtime state like :hover or :focus).
var evaluatablePseudoClasses = map[string]bool{
	"not":              true,
	"is":               true,
	"where":            true,
	"has":              true,
	"matches":          true,
	"nth-child":        true,
	"nth-last-child":   true,
	"nth-of-type":      true,
	"nth-last-of-type": true,
	"first-child":      true,
	"last-child":       true,
	"first-of-type":    true,
	"last-of-type":     true,
	"only-child":       true,
	"only-of-type":     true,
	"empty":            true,
	"root":             true,
}

// containsPseudoClass checks if a selector contains any non-evaluatable pseudo-class
// (single colon, not ::, not evaluatable like :not/:is/:where/:has).
func containsPseudoClass(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == ':' {
			if i+1 < len(s) && s[i+1] == ':' {
				i++ // skip pseudo-element ::
				continue
			}
			// Extract pseudo-class name
			nameStart := i + 1
			nameEnd := nameStart
			for nameEnd < len(s) && s[nameEnd] != '(' && s[nameEnd] != ' ' &&
				s[nameEnd] != ':' && s[nameEnd] != '.' && s[nameEnd] != '#' &&
				s[nameEnd] != '[' && s[nameEnd] != ')' && s[nameEnd] != ',' {
				nameEnd++
			}
			name := s[nameStart:nameEnd]
			if evaluatablePseudoClasses[name] {
				// Skip past the parenthesized content if present
				if nameEnd < len(s) && s[nameEnd] == '(' {
					depth := 1
					nameEnd++
					for nameEnd < len(s) && depth > 0 {
						if s[nameEnd] == '(' {
							depth++
						}
						if s[nameEnd] == ')' {
							depth--
						}
						nameEnd++
					}
				}
				i = nameEnd - 1
				continue
			}
			return true
		}
	}
	return false
}

// StripPseudoClasses removes non-evaluatable pseudo-classes from a selector string,
// preserving pseudo-elements (::) and evaluatable pseudo-classes like :not(), :is(), etc.
func StripPseudoClasses(s string) string {
	var result strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == ':' {
			// Check if it's a pseudo-element (::)
			if i+1 < len(s) && s[i+1] == ':' {
				// Pseudo-element — keep it
				result.WriteString("::")
				i += 2
				// Copy the pseudo-element name
				for i < len(s) && s[i] != ' ' && s[i] != '>' && s[i] != '+' &&
					s[i] != '~' && s[i] != ':' && s[i] != '.' && s[i] != '#' &&
					s[i] != '[' && s[i] != '(' && s[i] != ')' && s[i] != ',' {
					result.WriteByte(s[i])
					i++
				}
				continue
			}

			// Extract pseudo-class name to check if evaluatable
			nameStart := i + 1
			nameEnd := nameStart
			for nameEnd < len(s) && s[nameEnd] != '(' && s[nameEnd] != ' ' &&
				s[nameEnd] != ':' && s[nameEnd] != '.' && s[nameEnd] != '#' &&
				s[nameEnd] != '[' && s[nameEnd] != ')' && s[nameEnd] != ',' &&
				s[nameEnd] != '>' && s[nameEnd] != '+' && s[nameEnd] != '~' {
				nameEnd++
			}
			name := s[nameStart:nameEnd]

			if evaluatablePseudoClasses[name] {
				// Evaluatable pseudo-class — keep it (including arguments)
				result.WriteByte(':')
				i++
				// Copy the name
				for i < len(s) && s[i] != '(' && s[i] != ' ' && s[i] != ':' &&
					s[i] != '.' && s[i] != '#' && s[i] != '[' && s[i] != ')' &&
					s[i] != ',' && s[i] != '>' && s[i] != '+' && s[i] != '~' {
					result.WriteByte(s[i])
					i++
				}
				// Copy parenthesized arguments
				if i < len(s) && s[i] == '(' {
					depth := 1
					result.WriteByte('(')
					i++
					for i < len(s) && depth > 0 {
						if s[i] == '(' {
							depth++
						}
						if s[i] == ')' {
							depth--
						}
						result.WriteByte(s[i])
						i++
					}
				}
				continue
			}

			// Non-evaluatable pseudo-class — skip it
			i++ // skip the colon
			// Skip the name
			for i < len(s) && s[i] != ' ' && s[i] != '>' && s[i] != '+' &&
				s[i] != '~' && s[i] != ':' && s[i] != '.' && s[i] != '#' &&
				s[i] != '[' && s[i] != '(' && s[i] != ')' && s[i] != ',' {
				i++
			}
			// Skip parenthesized arguments like :nth-child(2n+1)
			if i < len(s) && s[i] == '(' {
				depth := 1
				i++
				for i < len(s) && depth > 0 {
					if s[i] == '(' {
						depth++
					}
					if s[i] == ')' {
						depth--
					}
					i++
				}
			}
		} else {
			result.WriteByte(s[i])
			i++
		}
	}
	return result.String()
}
