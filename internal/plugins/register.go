// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package plugins imports all plugin sub-packages to trigger their init()
// registrations. Import this package to make all builtin plugins available.
package plugins

import (
	"github.com/okooo5km/ogvs/internal/plugin"

	// Wave 0: simple removal plugins
	_ "github.com/okooo5km/ogvs/internal/plugins/removecomments"
	_ "github.com/okooo5km/ogvs/internal/plugins/removedesc"
	_ "github.com/okooo5km/ogvs/internal/plugins/removedoctype"
	_ "github.com/okooo5km/ogvs/internal/plugins/removeeditorsnsdata"
	_ "github.com/okooo5km/ogvs/internal/plugins/removeemptyattrs"
	_ "github.com/okooo5km/ogvs/internal/plugins/removemetadata"
	_ "github.com/okooo5km/ogvs/internal/plugins/removetitle"
	_ "github.com/okooo5km/ogvs/internal/plugins/removexmlns"
	_ "github.com/okooo5km/ogvs/internal/plugins/removexmlprocinst"

	// Wave 1: attribute/structure plugins
	_ "github.com/okooo5km/ogvs/internal/plugins/cleanupattrs"
	_ "github.com/okooo5km/ogvs/internal/plugins/removedimensions"
	_ "github.com/okooo5km/ogvs/internal/plugins/removeunusedns"
	_ "github.com/okooo5km/ogvs/internal/plugins/sortattrs"
	_ "github.com/okooo5km/ogvs/internal/plugins/sortdefschildren"

	// Wave 2: CSS/style plugins
	_ "github.com/okooo5km/ogvs/internal/plugins/convertstylettoattrs"
	_ "github.com/okooo5km/ogvs/internal/plugins/inlinestyles"
	_ "github.com/okooo5km/ogvs/internal/plugins/mergestyles"
	_ "github.com/okooo5km/ogvs/internal/plugins/minifystyles"
	_ "github.com/okooo5km/ogvs/internal/plugins/removeattrsbyselector"

	// Wave 3: geometry/numeric plugins
	_ "github.com/okooo5km/ogvs/internal/plugins/cleanuplistofvalues"
	_ "github.com/okooo5km/ogvs/internal/plugins/cleanupnumericvalues"
	_ "github.com/okooo5km/ogvs/internal/plugins/convertcolors"
	_ "github.com/okooo5km/ogvs/internal/plugins/convertellipsetocircle"
	_ "github.com/okooo5km/ogvs/internal/plugins/convertshapetopath"
	_ "github.com/okooo5km/ogvs/internal/plugins/removeviewbox"

	// Wave 4: removal/structure plugins
	_ "github.com/okooo5km/ogvs/internal/plugins/addattributestosvgelement"
	_ "github.com/okooo5km/ogvs/internal/plugins/addclassestosvgelement"
	_ "github.com/okooo5km/ogvs/internal/plugins/movegroupattrstoelems"
	_ "github.com/okooo5km/ogvs/internal/plugins/removeattrs"
	_ "github.com/okooo5km/ogvs/internal/plugins/removeelementsbyattr"
	_ "github.com/okooo5km/ogvs/internal/plugins/removeemptycontainers"
	_ "github.com/okooo5km/ogvs/internal/plugins/removeemptytext"
	_ "github.com/okooo5km/ogvs/internal/plugins/removenoninheritablegroupattrs"
	_ "github.com/okooo5km/ogvs/internal/plugins/removerasterimages"
	_ "github.com/okooo5km/ogvs/internal/plugins/removescripts"
	_ "github.com/okooo5km/ogvs/internal/plugins/removestyleelement"
	_ "github.com/okooo5km/ogvs/internal/plugins/removeuselessdefs"

	// Wave 5: remaining preset-default plugins
	_ "github.com/okooo5km/ogvs/internal/plugins/cleanupenablebackground"
	_ "github.com/okooo5km/ogvs/internal/plugins/cleanupids"
	_ "github.com/okooo5km/ogvs/internal/plugins/collapsegroups"
	_ "github.com/okooo5km/ogvs/internal/plugins/convertpathdata"
	_ "github.com/okooo5km/ogvs/internal/plugins/converttransform"
	_ "github.com/okooo5km/ogvs/internal/plugins/mergepaths"
	_ "github.com/okooo5km/ogvs/internal/plugins/moveelemsattrstogroup"
	_ "github.com/okooo5km/ogvs/internal/plugins/removedeprecatedattrs"
	_ "github.com/okooo5km/ogvs/internal/plugins/removehiddenelems"
	_ "github.com/okooo5km/ogvs/internal/plugins/removeunknownsanddefaults"
	_ "github.com/okooo5km/ogvs/internal/plugins/removeuselessstrokeandfill"

	// Phase 10: remaining optional plugins
	_ "github.com/okooo5km/ogvs/internal/plugins/convertonestopgradients"
	_ "github.com/okooo5km/ogvs/internal/plugins/prefixids"
	_ "github.com/okooo5km/ogvs/internal/plugins/removeoffcanvaspaths"
	_ "github.com/okooo5km/ogvs/internal/plugins/removexlink"
	_ "github.com/okooo5km/ogvs/internal/plugins/reusepaths"
)

func init() {
	// Register preset-default after all individual plugins are registered.
	// Go guarantees imported packages' init() functions run before ours.
	plugin.RegisterPresetDefault()
}
