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
func (t *TaskWriteTests) Run(repository evaltask.Repository) (repositoryAssessment map[evaltask.Identifier]metrics.Assessments, problems []error, err error) {
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

	modelAssessment := metrics.NewAssessments()
	withSymflowerAssessment := metrics.NewAssessments()
	for _, filePath := range filePaths {
		modelAssessmentForFile := metrics.NewAssessments()
		withSymflowerAssessmentForFile := modelAssessmentForFile // The symflower assessment tracks how the model result can be improved in case of a failure, so just link to the model assessment until a failure actually happens.

		if err := repository.Reset(t.Logger); err != nil {
			t.Logger.Panicf("ERROR: unable to reset temporary repository path: %s", err)
		}

		modelContext := model.Context{
			Language: t.Language,

			RepositoryPath: dataPath,
			FilePath:       filePath,

			Logger: log,
		}
		assessments, err := t.Model.RunTask(modelContext, t.Identifier())
		if err != nil {
			problems = append(problems, pkgerrors.WithMessage(err, filePath))

			continue
		}
		if assessments[metrics.AssessmentKeyProcessingTime] == 0 {
			return nil, nil, pkgerrors.Errorf("no model response time measurement present for %q at repository %q", t.Model.ID(), repository.Name())
		}
		modelAssessmentForFile.Add(assessments)
		modelAssessmentForFile.Award(metrics.AssessmentKeyResponseNoError)

		coverage, ps, err := t.Language.Execute(log, dataPath)
		problems = append(problems, ps...)
		if err != nil {
			problems = append(problems, pkgerrors.WithMessage(err, filePath))

			// Run "symflower fix"  if the model response fails to execute.
			if t.Language.ID() == "golang" { // Currently we only support Go for "symflower fix".
				log.Print("model response alone failed execution, attempting to fix with \"symflower fix \"")

				duration, err := symflowerFix(log, modelAssessment, dataPath, t.Language)
				if err != nil {
					problems = append(problems, err)

					modelAssessment.Add(modelAssessmentForFile)
					withSymflowerAssessment.Add(withSymflowerAssessmentForFile)

					continue
				}

				coverage, ps, err := t.Language.Execute(log, dataPath)
				problems = append(problems, ps...)
				if err != nil {
					problems = append(problems, pkgerrors.WithMessage(err, "symflower fix"))

					modelAssessment.Add(modelAssessmentForFile)
					withSymflowerAssessment.Add(withSymflowerAssessmentForFile)

					continue
				}
				log.Printf("with symflower repair: Executes tests with %d coverage objects", coverage)

				// Symflower was able to fix a failure so now update the assessment with the improved results.
				withSymflowerAssessmentForFile = metrics.NewAssessments()
				withSymflowerAssessmentForFile[metrics.AssessmentKeyProcessingTime] = duration
				withSymflowerAssessmentForFile.Award(metrics.AssessmentKeyFilesExecuted)
				withSymflowerAssessmentForFile.AwardPoints(metrics.AssessmentKeyCoverage, coverage)

				withSymflowerAssessmentForFile = metrics.CombineWithSymflowerFixAssessments(modelAssessmentForFile, withSymflowerAssessmentForFile)
			}
		} else {
			log.Printf("Executes tests with %d coverage objects", coverage)
			modelAssessmentForFile.Award(metrics.AssessmentKeyFilesExecuted)
			modelAssessmentForFile.AwardPoints(metrics.AssessmentKeyCoverage, coverage)
		}

		modelAssessment.Add(modelAssessmentForFile)
		withSymflowerAssessment.Add(withSymflowerAssessmentForFile)
	}

	repositoryAssessment = map[evaltask.Identifier]metrics.Assessments{
		IdentifierWriteTests:             modelAssessment,
		IdentifierWriteTestsSymflowerFix: withSymflowerAssessment,
	}

	return repositoryAssessment, problems, nil
}
