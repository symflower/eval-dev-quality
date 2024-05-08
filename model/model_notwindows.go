//go:build !windows

package model

import (
	"strings"
)

// CleanModelNameForFileSystem cleans a model name to be useable for directory and file names on the file system.
func CleanModelNameForFileSystem(modelName string) (modelNameCleaned string) {
	return strings.ReplaceAll(modelName, "/", "_")
}
