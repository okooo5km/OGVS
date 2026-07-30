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
// Returns [inline, ids, classes, elements]; index 0 is always 0 for stylesheet
// rules. Mirrors csso's syntax.specificity().
func CalculateSpecificity(selector string) Specificity {
	var spec Specificity
	accumulateSpecificity(selector, &spec)
	return spec
}

// accumulateSpecificity walks a selector and adds its contribution to spec.
func accumulateSpecificity(s string, spec *Specificity) {
	i := 0
	for i < len(s) {
		switch s[i] {
		case '#':
			i++
			readIdent(s, &i)
			spec[1]++

		case '.':
			i++
			readIdent(s, &i)
			spec[2]++

		case '[':
			i++
			parseAttrSelector(s, &i)
			spec[2]++

		case ':':
			i++
			if i < len(s) && s[i] == ':' {
				// Pseudo-element: counts as an element.
				i++
				readIdent(s, &i)
				if i < len(s) && s[i] == '(' {
					readParenGroup(s, &i)
				}
				spec[3]++
				continue
			}
			name := strings.ToLower(readIdent(s, &i))
			var arg string
			if i < len(s) && s[i] == '(' {
				arg = readParenGroup(s, &i)
			}
			switch name {
			case "where":
				// :where() always contributes zero specificity.
			case "is", "matches", "not", "has":
				// Contribute the specificity of the most specific argument.
				var best Specificity
				for _, part := range splitSelectors(arg) {
					var cur Specificity
					accumulateSpecificity(part, &cur)
					if CompareSpecificity(cur, best) > 0 {
						best = cur
					}
				}
				spec[1] += best[1]
				spec[2] += best[2]
				spec[3] += best[3]
			default:
				// Every other pseudo-class counts as a class.
				spec[2]++
			}

		case '*', ' ', '\t', '\n', '\r', '\f', '>', '+', '~', ',':
			i++

		default:
			before := i
			readIdent(s, &i)
			if i == before {
				i++
				continue
			}
			spec[3]++
		}
	}
}

// evaluatablePseudoClasses are the pseudo-classes SVGO's inlineStyles keeps in
// the selector and lets css-select evaluate. It is exactly
// pseudoClasses.functional + pseudoClasses.treeStructural (SVGO's
// `preservedPseudos` in plugins/inlineStyles.js), plus :matches, the historical
// alias of :is that css-select also understands.
var evaluatablePseudoClasses = func() map[string]bool {
	m := map[string]bool{"matches": true}
	for k := range treeStructuralPseudoClasses {
		m[k] = true
	}
	for k := range functionalPseudoClasses {
		m[k] = true
	}
	return m
}()

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
				s[nameEnd] != '[' && s[nameEnd] != ')' && s[nameEnd] != ',' &&
				s[nameEnd] != '>' && s[nameEnd] != '+' && s[nameEnd] != '~' {
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

// ContainsAnyPseudoClass reports whether a selector contains any pseudo-class,
// mirroring `hasPseudoClasses` in SVGO's lib/style.js parseRule.
// Pseudo-elements (::) do not count.
func ContainsAnyPseudoClass(s string) bool {
	i := 0
	for i < len(s) {
		switch s[i] {
		case '[':
			i++
			parseAttrSelector(s, &i)
		case ':':
			i++
			if i < len(s) && s[i] == ':' {
				i++
				readIdent(s, &i)
				continue
			}
			return true
		default:
			before := i
			readIdent(s, &i)
			if i == before {
				i++
			}
		}
	}
	return false
}

// StripAllPseudoClasses removes every pseudo-class, including its functional
// argument, from a selector. This mirrors SVGO's lib/style.js parseRule, which
// walks the selector and removes every PseudoClassSelector node before using
// the result for matching. Pseudo-elements (::) are preserved, as SVGO
// preserves them too.
func StripAllPseudoClasses(s string) string {
	var out strings.Builder
	i := 0
	for i < len(s) {
		switch s[i] {
		case '[':
			start := i
			i++
			parseAttrSelector(s, &i)
			out.WriteString(s[start:i])
		case ':':
			if i+1 < len(s) && s[i+1] == ':' {
				start := i
				i += 2
				readIdent(s, &i)
				out.WriteString(s[start:i])
				continue
			}
			i++
			readIdent(s, &i)
			if i < len(s) && s[i] == '(' {
				readParenGroup(s, &i)
			}
		default:
			before := i
			readIdent(s, &i)
			if i == before {
				out.WriteByte(s[i])
				i++
				continue
			}
			out.WriteString(s[before:i])
		}
	}
	return strings.TrimSpace(out.String())
}
