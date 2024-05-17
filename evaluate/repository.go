package evaluate

import (
	"errors"
	"os"
	"path/filepath"

	pkgerrors "github.com/pkg/errors"
	"github.com/zimmski/osutil"

	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	"github.com/symflower/eval-dev-quality/language"
	"github.com/symflower/eval-dev-quality/log"
	evalmodel "github.com/symflower/eval-dev-quality/model"
)

// Repository evaluate a repository with the given model and language.
func Repository(logger *log.Logger, resultPath string, model evalmodel.Model, language language.Language, testDataPath string, repositoryPath string) (repositoryAssessment metrics.Assessments, problems []error, err error) {
	log, logClose, err := log.WithFile(logger, filepath.Join(resultPath, evalmodel.CleanModelNameForFileSystem(model.ID()), language.ID(), repositoryPath+".log"))
	if err != nil {
		return nil, nil, err
	}
	defer logClose()

	log.Printf("Evaluating model %q using language %q and repository %q", model.ID(), language.ID(), repositoryPath)
	defer func() {
		log.Printf("Evaluated model %q using language %q and repository %q: encountered %d problems: %+v", model.ID(), language.ID(), repositoryPath, len(problems), problems)
	}()

	dataPath := filepath.Join(testDataPath, repositoryPath)

	filePaths, err := language.Files(log, dataPath)
	if err != nil {
		return nil, problems, pkgerrors.WithStack(err)
	}

	repositoryAssessment = metrics.NewAssessments()
	for _, filePath := range filePaths {
		temporaryPath, err := os.MkdirTemp("", "eval-dev-quality")
		if err != nil {
			return nil, problems, pkgerrors.WithStack(err)
		}
		defer func() {
			if e := os.RemoveAll(temporaryPath); e != nil {
				if err != nil {
					err = errors.Join(err, pkgerrors.WithStack(e))
				} else {
					err = pkgerrors.WithStack(e)
				}
			}
		}()
		temporaryRepositoryPath := filepath.Join(temporaryPath, filepath.Base(repositoryPath))
		if err := osutil.CopyTree(dataPath, temporaryRepositoryPath); err != nil {
			return nil, problems, pkgerrors.WithStack(err)
		}

		assessments, err := model.GenerateTestsForFile(log, language, temporaryRepositoryPath, filePath)
		if err != nil {
			problems = append(problems, pkgerrors.WithMessage(err, filePath))

			continue
		}
		if assessments[metrics.AssessmentKeyProcessingTime] == 0 {
			return nil, nil, pkgerrors.Errorf("no model response time measurement present for %q at repository %q", model.ID(), repositoryPath)
		}
		repositoryAssessment.Add(assessments)
		repositoryAssessment.Award(metrics.AssessmentKeyResponseNoError)

		coverage, err := language.Execute(log, temporaryRepositoryPath)
		if err != nil {
			problems = append(problems, pkgerrors.WithMessage(err, filePath))

			continue
		}
		repositoryAssessment.Award(metrics.AssessmentKeyFilesExecuted)
		repositoryAssessment.AwardPoints(metrics.AssessmentKeyCoverage, coverage)
	}

	return repositoryAssessment, problems, nil
}
