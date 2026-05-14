package main

import (
	"path/filepath"
)

var (
	configPath string
	format     string
	verbose    bool
)

func isSecurePath(path string) bool {
	abs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	return abs == filepath.Clean(abs)
}
