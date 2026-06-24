package assets

import (
	_ "embed"
)

var (
	//go:embed logo.png
	Logo []byte
	//go:embed MonaSans-Medium.ttf
	Font []byte
)
