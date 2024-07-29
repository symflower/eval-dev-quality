package cmd

import (
	"encoding/json"
	"io"

	pkgerrors "github.com/pkg/errors"
	"github.com/symflower/eval-dev-quality/task"
)

// EvaluationConfiguration holds data of how an evaluation was configured.
type EvaluationConfiguration struct {
	// Models holds model configuration data.
	Models ModelsConfiguration
	// Repositories holds repository configuration data.
	Repositories RepositoryConfiguration
}

// ModelsConfiguration holds model data of how an evaluation was configured.
type ModelsConfiguration struct {
	// Selected holds the models selected for an evaluation.
	Selected []string
	// Available holds the models that were available at the time of an evaluation.
	Available []string
}

// RepositoryConfiguration holds repository data of how an evaluation was configured.
type RepositoryConfiguration struct {
	// Selected holds the repositories selected for an evaluation.
	Selected []string
	// Available holds the repositories that were available at the time of an evaluation including their tasks.
	Available map[string][]task.Identifier
}

// Write stores the configuration in JSON format.
func (c *EvaluationConfiguration) Write(writer io.Writer) error {
	encoder := json.NewEncoder(writer)
	if err := encoder.Encode(c); err != nil {
		return pkgerrors.Wrap(err, "writing configuration")
	}

	return nil
}

// NewEvaluationConfiguration creates an empty configuration.
func NewEvaluationConfiguration() *EvaluationConfiguration {
	return &EvaluationConfiguration{
		Repositories: RepositoryConfiguration{
			Available: map[string][]task.Identifier{},
		},
	}
}
