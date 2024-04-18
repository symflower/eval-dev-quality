package symflower

import (
	pkgerrors "github.com/pkg/errors"

	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	"github.com/symflower/eval-dev-quality/language"
	"github.com/symflower/eval-dev-quality/model"
	"github.com/symflower/eval-dev-quality/provider"
	"github.com/symflower/eval-dev-quality/util"
)

// ModelSymflower holds a Symflower model using the locally installed CLI.
type ModelSymflower struct{}

var _ model.Model = (*ModelSymflower)(nil)

// ID returns the unique ID of this model.
func (m *ModelSymflower) ID() (id string) {
	return "symflower" + provider.ProviderModelSeparator + "symbolic-execution"
}

// GenerateTestsForFile generates test files for the given implementation file in a repository.
func (m *ModelSymflower) GenerateTestsForFile(language language.Language, repositoryPath string, filePath string) (assessment metrics.Assessments, err error) {
	_, _, err = util.CommandWithResult(&util.Command{
		Command: []string{
			"symflower", "unit-tests",
			"--workspace", repositoryPath,
			filePath,
		},

		Directory: repositoryPath,
	})
	if err != nil {
		return nil, pkgerrors.WithStack(err)
	}

	return metrics.Assessments{ // Symflower always generates just source code when it does not fail, so no need to check the assessment properties.
		metrics.AssessmentKeyResponseNoExcess: 1,
		metrics.AssessmentKeyResponseNotEmpty: 1,
		metrics.AssessmentKeyResponseWithCode: 1,
	}, nil
}
