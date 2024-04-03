package symflower

import (
	pkgerrors "github.com/pkg/errors"

	"github.com/symflower/eval-symflower-codegen-testing/language"
	"github.com/symflower/eval-symflower-codegen-testing/model"
	"github.com/symflower/eval-symflower-codegen-testing/provider"
	"github.com/symflower/eval-symflower-codegen-testing/util"
)

// ModelSymflower holds a Symflower model using the locally installed CLI.
type ModelSymflower struct{}

var _ model.Model = (*ModelSymflower)(nil)

// ID returns the unique ID of this model.
func (m *ModelSymflower) ID() (id string) {
	return "symflower" + provider.ProviderModelSeparator + "symbolic-execution"
}

// GenerateTestsForFile generates test files for the given implementation file in a repository.
func (m *ModelSymflower) GenerateTestsForFile(language language.Language, repositoryPath string, filePath string) (err error) {
	_, _, err = util.CommandWithResult(&util.Command{
		Command: []string{
			"symflower", "unit-tests",
			"--workspace", repositoryPath,
			filePath,
		},

		Directory: repositoryPath,
	})
	if err != nil {
		return pkgerrors.WithStack(err)
	}

	return nil
}
