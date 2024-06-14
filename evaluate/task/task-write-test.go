package task

import (
	"path/filepath"

	pkgerrors "github.com/pkg/errors"
	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	"github.com/symflower/eval-dev-quality/language"
	"github.com/symflower/eval-dev-quality/log"
	"github.com/symflower/eval-dev-quality/model"
	evaltask "github.com/symflower/eval-dev-quality/task"
)

// TaskWriteTests holds the write test task.
type TaskWriteTests struct {
	// ResultPath holds the directory path where results should be written to.
	ResultPath string

	// Language holds the language for which the task should be evaluated.
	Language language.Language
	// Model holds the model which the task should be evaluated.
	Model model.Model

	// Logger holds the logger for this tasks.
	Logger *log.Logger
}

var _ evaltask.Task = (*TaskWriteTests)(nil)

// NewTaskWriteTests returns a write test task.
func newTaskWriteTests(logger *log.Logger, resultPath string, model model.Model, language language.Language) (task evaltask.Task) {
	return &TaskWriteTests{
		ResultPath: resultPath,
		Language:   language,
		Model:      model,
		Logger:     logger,
	}
}

// Identifier returns the write test task identifier.
func (t *TaskWriteTests) Identifier() evaltask.Identifier {
	return IdentifierWriteTests
}

// TaskWriteTests generates test files for the given implementation file in a repository.
func (t *TaskWriteTests) Run(repository evaltask.Repository) (repositoryAssessment metrics.Assessments, problems []error, err error) {
	dataPath := repository.DataPath()

	log, logClose, err := log.WithFile(t.Logger, filepath.Join(t.ResultPath, string(t.Identifier()), model.CleanModelNameForFileSystem(t.Model.ID()), t.Language.ID(), repository.Name()+".log"))
	if err != nil {
		return nil, nil, err
	}
	defer logClose()

	log.Printf("Evaluating model %q on task %q using language %q and repository %q", t.Model.ID(), t.Identifier(), t.Language.ID(), repository.Name())
	defer func() {
		log.Printf("Evaluated model %q on task %q using language %q and repository %q: encountered %d problems: %+v", t.Model.ID(), t.Identifier(), t.Language.ID(), repository.Name(), len(problems), problems)
	}()

	filePaths, err := t.Language.Files(log, dataPath)
	if err != nil {
		return nil, problems, pkgerrors.WithStack(err)
	}

	repositoryAssessment = metrics.NewAssessments()
	for _, filePath := range filePaths {
		if err := repository.Reset(t.Logger); err != nil {
			t.Logger.Panicf("ERROR: unable to reset temporary repository path: %s", err)
		}

		ctx := evaltask.Context{
			Language: t.Language,

			RepositoryPath: dataPath,
			FilePath:       filePath,

			Logger: log,
		}
		assessments, err := t.Model.RunTask(ctx, t.Identifier())
		if err != nil {
			problems = append(problems, pkgerrors.WithMessage(err, filePath))

			continue
		}
		if assessments[metrics.AssessmentKeyProcessingTime] == 0 {
			return nil, nil, pkgerrors.Errorf("no model response time measurement present for %q at repository %q", t.Model.ID(), repository.Name())
		}
		repositoryAssessment.Add(assessments)
		repositoryAssessment.Award(metrics.AssessmentKeyResponseNoError)

		coverage, ps, err := t.Language.Execute(log, dataPath)
		problems = append(problems, ps...)
		if err != nil {
			problems = append(problems, pkgerrors.WithMessage(err, filePath))

			continue
		}
		log.Printf("Executes tests with %d coverage objects", coverage)
		repositoryAssessment.Award(metrics.AssessmentKeyFilesExecuted)
		repositoryAssessment.AwardPoints(metrics.AssessmentKeyCoverage, coverage)
	}

	return repositoryAssessment, problems, nil
}
