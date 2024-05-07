package symflower

import (
	pkgerrors "github.com/pkg/errors"

	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	"github.com/symflower/eval-dev-quality/language"
	"github.com/symflower/eval-dev-quality/log"
	"github.com/symflower/eval-dev-quality/model"
	"github.com/symflower/eval-dev-quality/provider"
	"github.com/symflower/eval-dev-quality/tools"
	"github.com/symflower/eval-dev-quality/util"
)

// Model holds a Symflower model using the locally installed CLI.
type Model struct{}

// NewModel returns a Symflower model.
func NewModel() (model model.Model) {
	return &Model{}
}

var _ model.Model = (*Model)(nil)

// ID returns the unique ID of this model.
func (m *Model) ID() (id string) {
	return "symflower" + provider.ProviderModelSeparator + "symbolic-execution"
}

// GenerateTestsForFile generates test files for the given implementation file in a repository.
func (m *Model) GenerateTestsForFile(logger *log.Logger, language language.Language, repositoryPath string, filePath string) (assessment metrics.Assessments, err error) {
	_, err = util.CommandWithResult(logger, &util.Command{
		Command: []string{
			tools.SymflowerPath, "unit-tests",
			"--code-disable-fetch-dependencies",
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
		metrics.AssessmentKeyResponseWithCode: 1,
	}, nil
}
