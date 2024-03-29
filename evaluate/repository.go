package evaluate

import (
	"errors"
	"os"
	"path/filepath"

	pkgerrors "github.com/pkg/errors"
	"github.com/zimmski/osutil"

	"github.com/symflower/eval-symflower-codegen-testing/language"
	"github.com/symflower/eval-symflower-codegen-testing/model"
)

// EvaluateRepository evaluate a repository with the given model and language.
func EvaluateRepository(repositoryPath string, model model.Model, language language.Language) (err error) {
	temporaryPath, err := os.MkdirTemp("", "eval-symflower-codegen-testing")
	if err != nil {
		return pkgerrors.WithStack(err)
	}
	defer func() {
		if e := os.RemoveAll(temporaryPath); e != nil {
			if err != nil {
				err = errors.Join(err, e)
			} else {
				err = e
			}
		}
	}()
	temporaryRepositoryPath := filepath.Join(temporaryPath, filepath.Base(repositoryPath))
	if err := osutil.CopyTree(repositoryPath, temporaryRepositoryPath); err != nil {
		return pkgerrors.WithStack(err)
	}

	filePaths, err := language.Files(repositoryPath)
	if err != nil {
		return err
	}

	for _, filePath := range filePaths {
		if err := model.GenerateTestsForFile(temporaryRepositoryPath, filePath); err != nil {
			return err
		}

		if err := language.Execute(temporaryRepositoryPath); err != nil {
			return err
		}
	}

	return
}
