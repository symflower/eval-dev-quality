package model

import (
	"strings"
)

var cleanModelNameForFileSystemReplacer = strings.NewReplacer(
	"/", "_",
	"\\", "_",
	":", "_",
)

// CleanModelNameForFileSystem cleans a model name to be useable for directory and file names on the file system.
func CleanModelNameForFileSystem(modelName string) (modelNameCleaned string) {
	return cleanModelNameForFileSystemReplacer.Replace(modelName)
}
