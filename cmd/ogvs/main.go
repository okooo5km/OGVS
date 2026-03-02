// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

// Package main is the entry point for the ogvs CLI tool.
// ogvs is a Go implementation of SVGO — an SVG optimizer.
package main

import (
	_ "github.com/okooo5km/ogvs/internal/plugins" // trigger init() registrations

	"github.com/okooo5km/ogvs/internal/cli"
)

func main() {
	cli.Execute()
}
