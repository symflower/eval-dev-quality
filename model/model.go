package model

import (
	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	"github.com/symflower/eval-dev-quality/language"
	"github.com/symflower/eval-dev-quality/log"
)

// Model defines a model that can be queried for generations.
type Model interface {
	// ID returns the unique ID of this model.
	ID() (id string)

	// GenerateTestsForFile generates test files for the given implementation file in a repository.
	GenerateTestsForFile(logger *log.Logger, language language.Language, repositoryPath string, filePath string) (assessments metrics.Assessments, err error)
}

// SetQueryAttempts defines a model that can set the number of query attempts when a model request errors in the process of solving a task.
type SetQueryAttempts interface {
	// SetQueryAttempts sets the number of query attempts to perform when a model request errors in the process of solving a task.
	SetQueryAttempts(attempts uint)
}
