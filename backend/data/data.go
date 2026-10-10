// Package data embeds the curated seed recipe dataset shipped in the binary.
package data

import _ "embed"

//go:embed recipes.seed.json
var RecipesJSON []byte
