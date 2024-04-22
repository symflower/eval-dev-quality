package evaluate

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	pkgerrors "github.com/pkg/errors"
	"github.com/zimmski/osutil"

	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	"github.com/symflower/eval-dev-quality/language"
	"github.com/symflower/eval-dev-quality/log"
	"github.com/symflower/eval-dev-quality/model"
)

// Repository evaluate a repository with the given model and language.
func Repository(logger *log.Logger, resultPath string, model model.Model, language language.Language, testDataPath string, repositoryPath string) (repositoryAssessment metrics.Assessments, problems []error, err error) {
	log, logClose, err := log.WithFile(logger, filepath.Join(resultPath, strings.ReplaceAll(model.ID(), "/", "_"), language.ID(), repositoryPath+".log"))
	if err != nil {
		return nil, nil, err
	}
	defer logClose()

	log.Printf("Evaluating model %q using language %q and repository %q", model.ID(), language.ID(), repositoryPath)
	defer func() {
		log.Printf("Evaluated model %q using language %q and repository %q: encountered %d problems: %+v", model.ID(), language.ID(), repositoryPath, len(problems), problems)
	}()

	dataPath := filepath.Join(testDataPath, repositoryPath)

	temporaryPath, err := os.MkdirTemp("", "eval-dev-quality")
	if err != nil {
		return nil, problems, pkgerrors.WithStack(err)
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
	if err := osutil.CopyTree(dataPath, temporaryRepositoryPath); err != nil {
		return nil, problems, pkgerrors.WithStack(err)
	}

	filePaths, err := language.Files(log, dataPath)
	if err != nil {
		return nil, problems, pkgerrors.WithStack(err)
	}

	repositoryAssessment = metrics.NewAssessments()
	for _, filePath := range filePaths {
		assessments, err := model.GenerateTestsForFile(log, language, temporaryRepositoryPath, filePath)
		if err != nil {
			problems = append(problems, pkgerrors.WithMessage(err, filePath))

			continue
		}
		repositoryAssessment.Add(assessments)
		repositoryAssessment[metrics.AssessmentKeyResponseNoError]++

		coverage, err := language.Execute(log, temporaryRepositoryPath)
		if err != nil {
			problems = append(problems, pkgerrors.WithMessage(err, filePath))

			continue
		}
		repositoryAssessment[metrics.AssessmentKeyFilesExecuted]++
		if coverage == 100 {
			repositoryAssessment[metrics.AssessmentKeyCoverageStatement]++
		}
	}

	return repositoryAssessment, problems, nil
}
