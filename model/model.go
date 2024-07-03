package model

import (
	"strings"

	"github.com/symflower/eval-dev-quality/language"
	"github.com/symflower/eval-dev-quality/log"
)

// Model defines a model that can be queried for generations.
type Model interface {
	// ID returns the unique ID of this model.
	ID() (id string)
	// Name returns the human-readable name of this model.
	Name() (name string)

	// Cost returns the cost of a model in US dollars.
	Cost() (cost float64)
}

// Context holds the data needed by a model for running a task.
type Context struct {
	// Language holds the language for which the task should be evaluated.
	Language language.Language

	// RepositoryPath holds the absolute path to the repository.
	RepositoryPath string
	// FilePath holds the path to the file the model should act on.
	FilePath string

	// Arguments holds extra data that can be used in a query prompt.
	Arguments any

	// Logger is used for logging during evaluation.
	Logger *log.Logger
}

// SetQueryAttempts defines a model that can set the number of query attempts when a model request errors in the process of solving a task.
type SetQueryAttempts interface {
	// SetQueryAttempts sets the number of query attempts to perform when a model request errors in the process of solving a task.
	SetQueryAttempts(attempts uint)
}

var cleanModelNameForFileSystemReplacer = strings.NewReplacer(
	"/", "_",
	"\\", "_",
	":", "_",
)

// CleanModelNameForFileSystem cleans a model name to be useable for directory and file names on the file system.
func CleanModelNameForFileSystem(modelName string) (modelNameCleaned string) {
	return cleanModelNameForFileSystemReplacer.Replace(modelName)
}
