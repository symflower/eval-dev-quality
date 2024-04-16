package evaluate

import (
	"errors"
	"log"
	"os"
	"path/filepath"

	pkgerrors "github.com/pkg/errors"
	"github.com/zimmski/osutil"

	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	"github.com/symflower/eval-dev-quality/language"
	"github.com/symflower/eval-dev-quality/model"
)

// EvaluateRepository evaluate a repository with the given model and language.
func EvaluateRepository(model model.Model, language language.Language, repositoryPath string) (metrics metrics.Metrics, problems []error, err error) {
	log.Printf("Evaluating model %q using language %q and repository %q", model.ID(), language.ID(), repositoryPath)
	defer func() {
		log.Printf("Evaluated model %q using language %q and repository %q: encountered %d problems", model.ID(), language.ID(), repositoryPath, len(problems))
	}()

	temporaryPath, err := os.MkdirTemp("", "eval-dev-quality")
	if err != nil {
		return metrics, problems, pkgerrors.WithStack(err)
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
		return metrics, problems, pkgerrors.WithStack(err)
	}

	filePaths, err := language.Files(repositoryPath)
	if err != nil {
		return metrics, problems, pkgerrors.WithStack(err)
	}

	for _, filePath := range filePaths {
		metrics.Total++
		if err := model.GenerateTestsForFile(language, temporaryRepositoryPath, filePath); err != nil {
			problems = append(problems, pkgerrors.WithMessage(err, filePath))
			metrics.Problems++

			continue
		}

		coverage, err := language.Execute(temporaryRepositoryPath)
		if err != nil {
			problems = append(problems, pkgerrors.WithMessage(err, filePath))
			metrics.Problems++

			continue
		}
		metrics.Executed++
		metrics.Coverage = append(metrics.Coverage, coverage)
	}

	return metrics, problems, nil
}
