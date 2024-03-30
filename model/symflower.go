package model

import (
	pkgerrors "github.com/pkg/errors"

	"github.com/symflower/eval-symflower-codegen-testing/util"
)

// ModelSymflower holds a Symflower model using the locally installed CLI.
type ModelSymflower struct{}

func init() {
	Register(&ModelSymflower{})
}

var _ Model = (*ModelSymflower)(nil)

// ID returns the unique ID of this model.
func (language *ModelSymflower) ID() (id string) {
	return "symflower"
}

// GenerateTestsForFile generates test files for the given implementation file in a repository.
func (model *ModelSymflower) GenerateTestsForFile(repositoryPath string, filePath string) (err error) {
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
