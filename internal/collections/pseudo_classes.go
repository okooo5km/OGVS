// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

package collections

// PseudoClasses maps CSS pseudo-class categories to their members.
var PseudoClasses = map[string]StringSet{
	"displayState": {
		"fullscreen": true, "modal": true, "picture-in-picture": true,
	},
	"input": {
		"autofill": true, "blank": true, "checked": true, "default": true,
		"disabled": true, "enabled": true, "in-range": true,
		"indeterminate": true, "invalid": true, "optional": true,
		"out-of-range": true, "placeholder-shown": true, "read-only": true,
		"read-write": true, "required": true, "user-invalid": true, "valid": true,
	},
	"linguistic": {
		"dir": true, "lang": true,
	},
	"location": {
		"any-link": true, "link": true, "local-link": true,
		"scope": true, "target-within": true, "target": true, "visited": true,
	},
	"resourceState": {
		"playing": true, "paused": true,
	},
	"timeDimensional": {
		"current": true, "past": true, "future": true,
	},
	"treeStructural": {
		"empty": true, "first-child": true, "first-of-type": true,
		"last-child": true, "last-of-type": true, "nth-child": true,
		"nth-last-child": true, "nth-last-of-type": true, "nth-of-type": true,
		"only-child": true, "only-of-type": true, "root": true,
	},
	"userAction": {
		"active": true, "focus-visible": true, "focus-within": true,
		"focus": true, "hover": true,
	},
	"functional": {
		"is": true, "not": true, "where": true, "has": true,
	},
}
