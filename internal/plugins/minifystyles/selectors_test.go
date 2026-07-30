// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

package minifystyles

import "testing"

func newIndex(tags, classes, ids []string, cfg usageConfig) *usageIndex {
	u := &usageIndex{
		tags:    make(map[string]bool),
		classes: make(map[string]bool),
		ids:     make(map[string]bool),
		cfg:     cfg,
	}
	for _, t := range tags {
		u.tags[t] = true
	}
	for _, c := range classes {
		u.classes[c] = true
	}
	for _, i := range ids {
		u.ids[i] = true
	}
	return u
}

var allUsage = usageConfig{ids: true, classes: true, tags: true}

// rectIndex mirrors a document containing <svg><style/><rect class="used" id="used"/></svg>.
func rectIndex() *usageIndex {
	return newIndex([]string{"svg", "style", "rect"}, []string{"used"}, []string{"used"}, allUsage)
}

func TestRemoveUnusedRulesAttributeSelectors(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		// Tokens inside a quoted attribute value must never be read as
		// selectors of their own.
		{"space in value", `rect[data-x="a b"]{fill:red}`, `rect[data-x="a b"]{fill:red}`},
		{"dot in value", `rect[data-x=".foo"]{fill:red}`, `rect[data-x=".foo"]{fill:red}`},
		{"hash in value", `rect[data-x="#bar"]{fill:red}`, `rect[data-x="#bar"]{fill:red}`},
		{"color in value", `rect[fill="#ff0000"]{stroke:blue}`, `rect[fill="#ff0000"]{stroke:blue}`},
		{"comma in value", `rect[data-x="a,b"]{fill:red}`, `rect[data-x="a,b"]{fill:red}`},
		{"braces in value", `rect[title="a{b}c"]{fill:red}`, `rect[title="a{b}c"]{fill:red}`},
		{"single quotes", `rect[data-x='.foo']{fill:red}`, `rect[data-x='.foo']{fill:red}`},
		{"escaped quote in value", `rect[a="\"]"]{fill:red}`, `rect[a="\"]"]{fill:red}`},
		{"escaped ident value", `rect[a=\.foo]{fill:red}`, `rect[a=\.foo]{fill:red}`},
		{"unquoted value", `rect[data-x=foo]{fill:red}`, `rect[data-x=foo]{fill:red}`},
		{"bare attribute", `rect[data-x]{fill:red}`, `rect[data-x]{fill:red}`},
		{"operators", `rect[a~="x"][b^="y"][c$="z"][d*="w"][e|="v"]{fill:red}`,
			`rect[a~="x"][b^="y"][c$="z"][d*="w"][e|="v"]{fill:red}`},
		// The tag itself is still checked.
		{"unused tag with attribute", `circle[data-x="rect"]{fill:red}`, ``},
		// Braces inside a declaration value must not terminate the block.
		{"braces in url", `rect{fill:url("a{b}")}`, `rect{fill:url("a{b}")}`},
		{"comma in declaration", `rect{font-family:a,b,c}`, `rect{font-family:a,b,c}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := removeUnusedRules(tt.in, rectIndex()); got != tt.want {
				t.Errorf("removeUnusedRules(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestRemoveUnusedRulesSelectorList(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"prune unused member", `rect,circle{fill:red}`, `rect{fill:red}`},
		{"prune leading member", `circle,rect{fill:red}`, `rect{fill:red}`},
		{"all unused", `circle,ellipse{fill:red}`, ``},
		{"all used", `rect,style{fill:red}`, `rect,style{fill:red}`},
		{"comma inside attribute is not a separator", `rect[a="a,b"],circle{fill:red}`,
			`rect[a="a,b"]{fill:red}`},
		{"whitespace around comma", `rect , circle{fill:red}`, `rect{fill:red}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := removeUnusedRules(tt.in, rectIndex()); got != tt.want {
				t.Errorf("removeUnusedRules(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestRemoveUnusedRulesPseudos(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		// csso prunes the argument of :is/:where/:has/:matches/:-*-any.
		{"is pruned", `:is(rect,circle){fill:red}`, `:is(rect){fill:red}`},
		{"is emptied", `:is(circle,ellipse){fill:red}`, ``},
		{"where pruned", `:where(circle,rect){fill:red}`, `:where(rect){fill:red}`},
		{"matches pruned", `:matches(rect,circle){fill:red}`, `:matches(rect){fill:red}`},
		{"has kept", `:has(rect){fill:red}`, `:has(rect){fill:red}`},
		{"has removed", `:has(circle){fill:red}`, ``},
		// csso leaves the argument of :not alone.
		{"not opaque", `:not(.nope){fill:red}`, `:not(.nope){fill:red}`},
		{"not nested in is", `:is(:not(.unused),circle){fill:red}`, `:is(:not(.unused)){fill:red}`},
		// Unknown functional pseudos are opaque in csso's vendored css-tree.
		{"host opaque", `:host(.unused){fill:red}`, `:host(.unused){fill:red}`},
		{"unknown opaque", `:foo(.unused){fill:red}`, `:foo(.unused){fill:red}`},
		{"nth opaque", `rect:nth-child(2n+1){fill:red}`, `rect:nth-child(2n+1){fill:red}`},
		{"lang opaque", `rect:lang(en){fill:red}`, `rect:lang(en){fill:red}`},
		// ::slotted takes a single selector whose parts are checked inline.
		{"slotted checked", `::slotted(.unused){fill:red}`, ``},
		{"slotted kept", `::slotted(.used){fill:red}`, `::slotted(.used){fill:red}`},
		{"pseudo element", `rect::before{fill:red}`, `rect::before{fill:red}`},
		{"pseudo class", `rect:hover{fill:red}`, `rect:hover{fill:red}`},
		{"pseudo on unused tag", `circle:hover{fill:red}`, ``},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := removeUnusedRules(tt.in, rectIndex()); got != tt.want {
				t.Errorf("removeUnusedRules(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestRemoveUnusedRulesTypeSelectors(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		// csso's TypeSelector name includes the namespace prefix verbatim.
		{"namespaced tag", `svg|rect{fill:red}`, ``},
		{"universal", `*{fill:red}`, `*{fill:red}`},
		{"namespaced universal", `svg|*{fill:red}`, `svg|*{fill:red}`},
		{"any namespace universal", `*|*{fill:red}`, `*|*{fill:red}`},
		{"no namespace", `|rect{fill:red}`, ``},
		// Type selectors match case-insensitively.
		{"uppercase tag", `RECT{fill:red}`, `RECT{fill:red}`},
		// A tag name only counts at the start of a compound selector.
		{"class then ident is not a tag", `.used{fill:red}`, `.used{fill:red}`},
		{"descendant combinator", `rect circle{fill:red}`, ``},
		{"child combinator", `rect>style{fill:red}`, `rect>style{fill:red}`},
		{"sibling combinators", `rect+style{fill:red}`, `rect+style{fill:red}`},
		{"general sibling", `rect~style{fill:red}`, `rect~style{fill:red}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := removeUnusedRules(tt.in, rectIndex()); got != tt.want {
				t.Errorf("removeUnusedRules(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestRemoveUnusedRulesEscapes(t *testing.T) {
	// csso compares the raw selector text against the document usage data and
	// never decodes CSS escapes, so an escaped name only matches when the
	// document attribute carries the very same escape sequence.
	u := newIndex([]string{"svg", "style", "rect"}, []string{"used", "123", `a\.b`}, []string{"used"}, allUsage)
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"hex escaped class never matches decoded", `.\31 23{fill:red}`, ``},
		{"char escaped class matches raw", `.a\.b{fill:red}`, `.a\.b{fill:red}`},
		{"char escaped class unused", `.a\.c{fill:red}`, ``},
		{"escaped id", `#\75 sed{fill:red}`, ``},
		{"escaped tag", `\72 ect{fill:red}`, ``}, //nolint:misspell // CSS-escaped spelling of "rect"
		{"escape inside tag", `r\65 ct{fill:red}`, ``},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := removeUnusedRules(tt.in, u); got != tt.want {
				t.Errorf("removeUnusedRules(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestRemoveUnusedRulesAtRules(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		// csso never applies usage filtering inside @keyframes, so the
		// "from"/"to" keyframe selectors must survive.
		{"keyframes", `@keyframes a{from{fill:red}to{fill:red}}`,
			`@keyframes a{from{fill:red}to{fill:red}}`},
		{"prefixed keyframes", `@-webkit-keyframes a{from{fill:red}to{fill:red}}`,
			`@-webkit-keyframes a{from{fill:red}to{fill:red}}`},
		{"percent keyframes", `@keyframes a{0%,to{fill:red}4%,96%{fill:red}}`,
			`@keyframes a{0%,to{fill:red}4%,96%{fill:red}}`},
		{"media descends", `@media screen{circle{fill:red}rect{fill:blue}}`,
			`@media screen{rect{fill:blue}}`},
		{"supports descends", `@supports(fill:red){circle{fill:red}rect{fill:red}}`,
			`@supports(fill:red){rect{fill:red}}`},
		{"nested media", `@media screen{@media(min-width:1px){circle{fill:red}rect{fill:red}}}`,
			`@media screen{@media(min-width:1px){rect{fill:red}}}`},
		{"font-face untouched", `@font-face{font-family:x;src:url(y)}`,
			`@font-face{font-family:x;src:url(y)}`},
		{"at statement", `@import "x";circle{fill:red}rect{fill:red}`,
			`@import "x";rect{fill:red}`},
		{"empty media dropped", `@media screen{circle{fill:red}}`, ``},
		{"already empty media dropped", `@media screen{}rect{fill:red}`, `rect{fill:red}`},
		{"nested empty media dropped", `@media a{@media b{circle{fill:red}}}rect{fill:red}`,
			`rect{fill:red}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := removeUnusedRules(tt.in, rectIndex()); got != tt.want {
				t.Errorf("removeUnusedRules(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestRemoveUnusedRulesUsageConfig(t *testing.T) {
	const in = `.used{p:1}.unused{p:2}#used{p:3}#unused{p:4}g{p:5}unused{p:6}`

	// minifyStyles.04: all three usage kinds enabled.
	all := newIndex([]string{"svg", "style", "g"}, []string{"used"}, []string{"used"}, allUsage)
	if got, want := removeUnusedRules(in, all), `.used{p:1}#used{p:3}g{p:5}`; got != want {
		t.Errorf("all usage = %q, want %q", got, want)
	}

	// minifyStyles.05: {"usage":{"ids":false,"tags":false}}.
	classesOnly := newIndex([]string{"svg", "style", "g"}, []string{"used"}, []string{"used"},
		usageConfig{classes: true})
	if got, want := removeUnusedRules(in, classesOnly),
		`.used{p:1}#used{p:3}#unused{p:4}g{p:5}unused{p:6}`; got != want {
		t.Errorf("classes only = %q, want %q", got, want)
	}

	// minifyStyles.11: nothing used, the whole stylesheet disappears.
	none := newIndex([]string{"svg", "style"}, nil, nil, allUsage)
	if got := removeUnusedRules(`.st1{p:1}.st2{p:2}.st3{p:3}`, none); got != "" {
		t.Errorf("no usage = %q, want empty", got)
	}

	// minifyStyles.01/.03: an @media block around a used class is untouched.
	st0 := newIndex([]string{"svg", "style", "path"}, []string{"st0"}, nil, allUsage)
	const media = `.st0{fill:red;padding:1em}@media screen and (max-width:200px){.st0{display:none}}`
	if got := removeUnusedRules(media, st0); got != media {
		t.Errorf("media passthrough = %q, want %q", got, media)
	}
}

func TestRemoveUnusedRulesMalformed(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", ``, ``},
		{"unterminated block", `rect{fill:red`, `rect{fill:red`},
		{"unterminated at-rule", `@media screen{rect{fill:red}`, `@media screen{rect{fill:red}`},
		{"stray brace", `}rect{fill:red}`, `}rect{fill:red}`},
		{"unterminated string", `rect[a="b]{fill:red}`, `rect[a="b]{fill:red}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := removeUnusedRules(tt.in, rectIndex()); got != tt.want {
				t.Errorf("removeUnusedRules(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestAtRuleBasename(t *testing.T) {
	tests := []struct{ in, want string }{
		{"media", "media"},
		{"KEYFRAMES", "keyframes"},
		{"-webkit-keyframes", "keyframes"},
		{"-moz-keyframes", "keyframes"},
		{"--custom", "--custom"},
		{"-x-", "-x-"},
	}
	for _, tt := range tests {
		if got := atRuleBasename(tt.in); got != tt.want {
			t.Errorf("atRuleBasename(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
