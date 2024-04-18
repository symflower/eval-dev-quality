package model

import (
	"log"

	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	"github.com/symflower/eval-dev-quality/language"
)

// Model defines a model that can be queried for generations.
type Model interface {
	// ID returns the unique ID of this model.
	ID() (id string)

	// GenerateTestsForFile generates test files for the given implementation file in a repository.
	GenerateTestsForFile(log *log.Logger, language language.Language, repositoryPath string, filePath string) (assessments metrics.Assessments, err error)
}
