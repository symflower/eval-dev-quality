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
