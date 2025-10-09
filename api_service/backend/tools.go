//go:build tools

package main

import (
	_ "github.com/air-verse/air"
	_ "github.com/swaggo/swag/cmd/swag"
)

// This file is used to track tool dependencies.
// It is not meant to be compiled into the main application.
