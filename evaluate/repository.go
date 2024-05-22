package evaluate

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	pkgerrors "github.com/pkg/errors"
	"github.com/zimmski/osutil"

	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	"github.com/symflower/eval-dev-quality/language"
	"github.com/symflower/eval-dev-quality/log"
	evalmodel "github.com/symflower/eval-dev-quality/model"
	"github.com/symflower/eval-dev-quality/util"
)

// Repository evaluate a repository with the given model and language.
func Repository(logger *log.Logger, resultPath string, model evalmodel.Model, language language.Language, testDataPath string, repositoryName string) (repositoryAssessment metrics.Assessments, problems []error, err error) {
	log, logClose, err := log.WithFile(logger, filepath.Join(resultPath, evalmodel.CleanModelNameForFileSystem(model.ID()), language.ID(), repositoryName+".log"))
	if err != nil {
		return nil, nil, err
	}
	defer logClose()

	log.Printf("Evaluating model %q using language %q and repository %q", model.ID(), language.ID(), repositoryName)
	defer func() {
		log.Printf("Evaluated model %q using language %q and repository %q: encountered %d problems: %+v", model.ID(), language.ID(), repositoryName, len(problems), problems)
	}()

	filePaths, err := language.Files(log, testDataPath)
	if err != nil {
		return nil, problems, pkgerrors.WithStack(err)
	}

	repositoryAssessment = metrics.NewAssessments()
	for _, filePath := range filePaths {
		assessments, err := model.GenerateTestsForFile(log, language, testDataPath, filePath)
		if err != nil {
			problems = append(problems, pkgerrors.WithMessage(err, filePath))

			continue
		}
		if assessments[metrics.AssessmentKeyProcessingTime] == 0 {
			return nil, nil, pkgerrors.Errorf("no model response time measurement present for %q at repository %q", model.ID(), repositoryName)
		}
		repositoryAssessment.Add(assessments)
		repositoryAssessment.Award(metrics.AssessmentKeyResponseNoError)

		coverage, err := language.Execute(log, testDataPath)
		if err != nil {
			problems = append(problems, pkgerrors.WithMessage(err, filePath))

			continue
		}
		repositoryAssessment.Award(metrics.AssessmentKeyFilesExecuted)
		if coverage == 100 {
			repositoryAssessment.Award(metrics.AssessmentKeyCoverageStatement)
		}
	}

	return repositoryAssessment, problems, nil
}

// TemporaryRepository creates a temporary repository and initializes a git repo in it.
func TemporaryRepository(logger *log.Logger, dataPath string) (temporaryRepositoryPath string, cleanup func(), err error) {
	temporaryPath, err := os.MkdirTemp("", "eval-dev-quality")
	if err != nil {
		return "", cleanup, pkgerrors.WithStack(err)
	}

	cleanup = func() {
		if e := os.RemoveAll(temporaryPath); e != nil {
			if err != nil {
				err = errors.Join(err, pkgerrors.WithStack(e))
			} else {
				err = pkgerrors.WithStack(e)
			}
		}
	}

	temporaryRepositoryPath = filepath.Join(temporaryPath, filepath.Base(dataPath))
	if err := osutil.CopyTree(dataPath, temporaryRepositoryPath); err != nil {
		return "", cleanup, pkgerrors.WithStack(err)
	}

	// Add git repository and commit changes.
	out, err := util.CommandWithResult(context.Background(), logger, &util.Command{
		Command: []string{
			"git",
			"init",
		},

		Directory: temporaryRepositoryPath,
	})
	if err != nil {
		return "", cleanup, pkgerrors.WithStack(pkgerrors.Wrap(err, fmt.Sprintf("%s - %s", "unable to initialize git repository", out)))
	}

	out, err = util.CommandWithResult(context.Background(), logger, &util.Command{
		Command: []string{
			"git",
			"add",
			".",
		},

		Directory: temporaryRepositoryPath,
	})
	if err != nil {
		return "", cleanup, pkgerrors.WithStack(pkgerrors.Wrap(err, fmt.Sprintf("%s - %s", "unable to add files", out)))
	}

	out, err = util.CommandWithResult(context.Background(), logger, &util.Command{
		Command: []string{
			"git",
			"commit",
			"-m",
			"initial",
		},

		Directory: temporaryRepositoryPath,
	})
	if err != nil {
		return "", cleanup, pkgerrors.WithStack(pkgerrors.Wrap(err, fmt.Sprintf("%s - %s", "unable to commit", out)))
	}

	return temporaryRepositoryPath, cleanup, nil
}

// ResetTemporaryRepository resets a temporary repository back to its "initial" commit.
func ResetTemporaryRepository(logger *log.Logger, path string) (err error) {
	out, err := util.CommandWithResult(context.Background(), logger, &util.Command{
		Command: []string{
			"git",
			"clean",
			"-df",
		},

		Directory: path,
	})
	if err != nil {
		return pkgerrors.WithStack(pkgerrors.Wrap(err, fmt.Sprintf("%s - %s", "unable to clean git repository", out)))
	}

	return nil
}
