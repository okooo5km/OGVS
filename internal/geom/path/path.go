// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package path provides SVG path data parsing and stringification,
// ported from SVGO's lib/path.js.
// Based on https://www.w3.org/TR/SVG11/paths.html#PathDataBNF.
package path

import (
	"errors"
	"math"
	"strconv"
	"strings"

	"github.com/okooo5km/ogvs/internal/tools"
)

// PathDataItem represents a single path command with its arguments.
type PathDataItem struct {
	Command byte
	Args    []float64
}

// argsCountPerCommand maps each SVG path command to its expected argument count.
var argsCountPerCommand = map[byte]int{
	'M': 2, 'm': 2,
	'Z': 0, 'z': 0,
	'L': 2, 'l': 2,
	'H': 1, 'h': 1,
	'V': 1, 'v': 1,
	'C': 6, 'c': 6,
	'S': 4, 's': 4,
	'Q': 4, 'q': 4,
	'T': 2, 't': 2,
	'A': 7, 'a': 7,
}

func isCommand(c byte) bool {
	_, ok := argsCountPerCommand[c]
	return ok
}

func isWhiteSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\r' || c == '\n'
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

// readNumberState represents the state machine states for number parsing.
type readNumberState int

const (
	stateNone readNumberState = iota
	stateSign
	stateWhole
	stateDecimalPoint
	stateDecimal
	stateE
	stateExponentSign
	stateExponent
)

// readNumber reads a number from the string starting at cursor position.
// Returns the new cursor position and the parsed number.
// If no valid number is found, returns the original cursor and NaN.
func readNumber(s string, cursor int) (int, float64) {
	i := cursor
	start := cursor
	end := cursor
	state := stateNone

	for ; i < len(s); i++ {
		c := s[i]

		if c == '+' || c == '-' {
			if state == stateNone {
				state = stateSign
				end = i + 1
				continue
			}
			if state == stateE {
				state = stateExponentSign
				end = i + 1
				continue
			}
		}

		if isDigit(c) {
			if state == stateNone || state == stateSign || state == stateWhole {
				state = stateWhole
				end = i + 1
				continue
			}
			if state == stateDecimalPoint || state == stateDecimal {
				state = stateDecimal
				end = i + 1
				continue
			}
			if state == stateE || state == stateExponentSign || state == stateExponent {
				state = stateExponent
				end = i + 1
				continue
			}
		}

		if c == '.' {
			if state == stateNone || state == stateSign || state == stateWhole {
				state = stateDecimalPoint
				end = i + 1
				continue
			}
		}

		if c == 'E' || c == 'e' {
			if state == stateWhole || state == stateDecimalPoint || state == stateDecimal {
				state = stateE
				end = i + 1
				continue
			}
		}

		break
	}

	numStr := s[start:end]
	if numStr == "" || numStr == "+" || numStr == "-" || numStr == "." ||
		numStr == "+." || numStr == "-." {
		return cursor, math.NaN()
	}

	// Parse the number manually for exactness matching JS behavior
	number := parseFloat(numStr)
	if math.IsNaN(number) {
		return cursor, math.NaN()
	}

	// step back to delegate iteration to parent loop (i-1 because parent loop will i++)
	return i - 1, number
}

// parseFloat parses a floating point number string, matching JS
// Number.parseFloat behavior: the longest leading substring that forms a
// decimal literal is converted, and anything else yields NaN.
func parseFloat(s string) float64 {
	end := decimalPrefixEnd(s)
	if end == 0 {
		return math.NaN()
	}
	value, err := strconv.ParseFloat(s[:end], 64)
	if err != nil {
		// Out-of-range magnitudes saturate to ±Inf / 0, which is what
		// JS produces as well; anything else is not a number.
		var numErr *strconv.NumError
		if errors.As(err, &numErr) && errors.Is(numErr.Err, strconv.ErrRange) {
			return value
		}
		return math.NaN()
	}
	return value
}

// decimalPrefixEnd returns the length of the longest prefix of s that is a
// decimal literal. Trailing fragments JS tolerates but strconv rejects — a
// dangling exponent marker such as "1e" or "1e+" — are excluded, as are the
// Go-only spellings (hex floats, digit separators, "Infinity", "NaN") that
// JS parseFloat would not accept here.
func decimalPrefixEnd(s string) int {
	i := 0
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		i++
	}
	intDigits := 0
	for i < len(s) && isDigit(s[i]) {
		i++
		intDigits++
	}
	fracDigits := 0
	if i < len(s) && s[i] == '.' {
		j := i + 1
		for j < len(s) && isDigit(s[j]) {
			j++
			fracDigits++
		}
		if intDigits != 0 || fracDigits != 0 {
			i = j
		}
	}
	if intDigits == 0 && fracDigits == 0 {
		return 0
	}
	end := i
	if i < len(s) && (s[i] == 'e' || s[i] == 'E') {
		j := i + 1
		if j < len(s) && (s[j] == '+' || s[j] == '-') {
			j++
		}
		expDigits := 0
		for j < len(s) && isDigit(s[j]) {
			j++
			expDigits++
		}
		if expDigits != 0 {
			end = j
		}
	}
	return end
}

// ParsePathData parses an SVG path data string into a slice of PathDataItems.
func ParsePathData(s string) []PathDataItem {
	var pathData []PathDataItem
	var command byte
	var args []float64
	argsCount := 0
	canHaveComma := false
	hadComma := false
	commandSet := false

	for i := 0; i < len(s); i++ {
		c := s[i]

		if isWhiteSpace(c) {
			continue
		}

		// allow comma only between arguments
		if canHaveComma && c == ',' {
			if hadComma {
				break
			}
			hadComma = true
			continue
		}

		if isCommand(c) {
			if hadComma {
				return pathData
			}
			if !commandSet {
				// moveto should be leading command
				if c != 'M' && c != 'm' {
					return pathData
				}
			} else if len(args) != 0 {
				// stop if previous command arguments are not flushed
				return pathData
			}
			command = c
			commandSet = true
			args = nil
			argsCount = argsCountPerCommand[command]
			canHaveComma = false
			// flush command without arguments
			if argsCount == 0 {
				pathData = append(pathData, PathDataItem{Command: command, Args: nil})
			}
			continue
		}

		// avoid parsing arguments if no command detected
		if !commandSet {
			return pathData
		}

		// read next argument
		newCursor := i
		number := math.NaN()

		if command == 'A' || command == 'a' {
			position := len(args)
			if position == 0 || position == 1 {
				// allow only positive number without sign as first two arguments
				if c != '+' && c != '-' {
					newCursor, number = readNumber(s, i)
				}
			}
			if position == 2 || position == 5 || position == 6 {
				newCursor, number = readNumber(s, i)
			}
			if position == 3 || position == 4 {
				// read flags
				if c == '0' {
					number = 0
				}
				if c == '1' {
					number = 1
				}
			}
		} else {
			newCursor, number = readNumber(s, i)
		}

		if math.IsNaN(number) {
			return pathData
		}

		args = append(args, number)
		canHaveComma = true
		hadComma = false
		i = newCursor

		// flush arguments when necessary count is reached
		if len(args) == argsCount {
			pathData = append(pathData, PathDataItem{
				Command: command,
				Args:    append([]float64(nil), args...),
			})
			// subsequent moveto coordinates are treated as implicit lineto commands
			if command == 'M' {
				command = 'L'
			}
			if command == 'm' {
				command = 'l'
			}
			args = nil
		}
	}

	return pathData
}

// StringifyPathDataOptions controls path data stringification.
type StringifyPathDataOptions struct {
	PathData               []PathDataItem
	Precision              int // -1 means no rounding
	DisableSpaceAfterFlags bool
}

// roundAndStringify rounds a number to precision and returns the string.
func roundAndStringify(number float64, precision int) (string, float64) {
	if precision >= 0 {
		number = tools.ToFixed(number, precision)
	}
	return tools.RemoveLeadingZero(number), number
}

// stringifyArgs converts command arguments to an optimized string.
func stringifyArgs(command byte, args []float64, precision int, disableSpaceAfterFlags bool) string {
	var sb strings.Builder
	var previous float64
	previousSet := false

	for i, arg := range args {
		roundedStr, rounded := roundAndStringify(arg, precision)

		if disableSpaceAfterFlags &&
			(command == 'A' || command == 'a') &&
			(i%7 == 4 || i%7 == 5) {
			sb.WriteString(roundedStr)
		} else if i == 0 || rounded < 0 {
			// avoid space before first and negative numbers
			sb.WriteString(roundedStr)
		} else if previousSet && !isInteger(previous) && len(roundedStr) > 0 && !isDigit(roundedStr[0]) {
			// remove space before decimal with zero whole
			// only when previous number is also decimal
			sb.WriteString(roundedStr)
		} else {
			sb.WriteByte(' ')
			sb.WriteString(roundedStr)
		}

		previous = rounded
		previousSet = true
	}

	return sb.String()
}

// isInteger checks if a float64 is an integer value (matching JS Number.isInteger).
func isInteger(n float64) bool {
	return n == math.Trunc(n) && !math.IsInf(n, 0) && !math.IsNaN(n)
}

// StringifyPathData converts parsed path data back to an optimized SVG path string.
func StringifyPathData(opts *StringifyPathDataOptions) string {
	pathData := opts.PathData
	precision := opts.Precision
	disableSpaceAfterFlags := opts.DisableSpaceAfterFlags

	if len(pathData) == 0 {
		return ""
	}

	if len(pathData) == 1 {
		item := pathData[0]
		return string(item.Command) + stringifyArgs(item.Command, item.Args, precision, disableSpaceAfterFlags)
	}

	var result strings.Builder

	// Start with a copy of the first item
	prevCommand := pathData[0].Command
	prevArgs := append([]float64(nil), pathData[0].Args...)

	// match leading moveto with following lineto
	if pathData[1].Command == 'L' {
		prevCommand = 'M'
	} else if pathData[1].Command == 'l' {
		prevCommand = 'm'
	}

	for i := 1; i < len(pathData); i++ {
		command := pathData[i].Command
		args := pathData[i].Args

		if (prevCommand == command && prevCommand != 'M' && prevCommand != 'm') ||
			// combine matching moveto and lineto sequences
			(prevCommand == 'M' && command == 'L') ||
			(prevCommand == 'm' && command == 'l') {
			prevArgs = append(prevArgs, args...)
			if i == len(pathData)-1 {
				result.WriteByte(prevCommand)
				result.WriteString(stringifyArgs(prevCommand, prevArgs, precision, disableSpaceAfterFlags))
			}
		} else {
			result.WriteByte(prevCommand)
			result.WriteString(stringifyArgs(prevCommand, prevArgs, precision, disableSpaceAfterFlags))

			if i == len(pathData)-1 {
				result.WriteByte(command)
				result.WriteString(stringifyArgs(command, args, precision, disableSpaceAfterFlags))
			} else {
				prevCommand = command
				prevArgs = append([]float64(nil), args...)
			}
		}
	}

	return result.String()
}
