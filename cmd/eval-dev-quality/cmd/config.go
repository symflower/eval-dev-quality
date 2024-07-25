package cmd

import (
	"encoding/json"
	"io"

	pkgerrors "github.com/pkg/errors"
)

// EvaluationConfiguration holds data of how an evaluation was configured.
type EvaluationConfiguration struct {
	// Models holds model configuration data.
	Models ModelsConfiguration
}

// ModelsConfiguration holds model data of how an evaluation was configured.
type ModelsConfiguration struct {
	// Selected holds the models selected for an evaluation.
	Selected []string
	// Available holds the models that were available at the time of an evaluation.
	Available []string
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
	return &EvaluationConfiguration{}
}
