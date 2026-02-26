package config

import (
	"path"
	"strings"
)

func NormalizePrefix(prefix string) string {
	if prefix == "" {
		return ""
	}
	prefix = strings.ReplaceAll(prefix, "\\", "/")
	for strings.Contains(prefix, "//") {
		prefix = strings.ReplaceAll(prefix, "//", "/")
	}

	prefix = strings.Trim(prefix, "/")
	return path.Clean(prefix)
}
