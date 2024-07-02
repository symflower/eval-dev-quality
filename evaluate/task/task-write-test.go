package task

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	pkgerrors "github.com/pkg/errors"
	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	"github.com/symflower/eval-dev-quality/log"
	"github.com/symflower/eval-dev-quality/model"
	evaltask "github.com/symflower/eval-dev-quality/task"
)

// TaskWriteTests holds the write test task.
type TaskWriteTests struct {
}

var _ evaltask.Task = (*TaskWriteTests)(nil)

// Identifier returns the write test task identifier.
func (t *TaskWriteTests) Identifier() evaltask.Identifier {
	return IdentifierWriteTests
}

// TaskWriteTests generates test files for the given implementation file in a repository.
func (t *TaskWriteTests) Run(ctx evaltask.Context) (repositoryAssessment map[evaltask.Identifier]metrics.Assessments, problems []error, err error) {
	modelCapability, ok := ctx.Model.(model.CapabilityWriteTests)
	if !ok {
		pkgerrors.Wrap(evaltask.ErrTaskUnsupportedByModel, fmt.Sprintf("%q does not support %q", ctx.Model.ID(), string(t.Identifier())))
	}

	dataPath := ctx.Repository.DataPath()

	log, logClose, err := log.WithFile(ctx.Logger, filepath.Join(ctx.ResultPath, string(t.Identifier()), model.CleanModelNameForFileSystem(ctx.Model.ID()), ctx.Language.ID(), ctx.Repository.Name()+".log"))
	if err != nil {
		return nil, nil, err
	}
	defer logClose()

	log.Printf("Evaluating model %q on task %q using language %q and repository %q", ctx.Model.ID(), t.Identifier(), ctx.Language.ID(), ctx.Repository.Name())
	defer func() {
		log.Printf("Evaluated model %q on task %q using language %q and repository %q: encountered %d problems: %+v", ctx.Model.ID(), t.Identifier(), ctx.Language.ID(), ctx.Repository.Name(), len(problems), problems)
	}()

	filePaths, err := ctx.Language.Files(log, dataPath)
	if err != nil {
		return nil, problems, pkgerrors.WithStack(err)
	}

	modelAssessment := metrics.NewAssessments()
	withSymflowerAssessment := metrics.NewAssessments()
	for _, filePath := range filePaths {
		modelAssessmentForFile := metrics.NewAssessments()
		withSymflowerAssessmentForFile := modelAssessmentForFile // The symflower assessment tracks how the model result can be improved in case of a failure, so just link to the model assessment until a failure actually happens.

		if err := ctx.Repository.Reset(ctx.Logger); err != nil {
			ctx.Logger.Panicf("ERROR: unable to reset temporary repository path: %s", err)
		}

		modelContext := model.Context{
			Language: ctx.Language,

			RepositoryPath: dataPath,
			FilePath:       filePath,

			Logger: log,
		}
		assessments, err := modelCapability.WriteTests(modelContext)
		if err != nil {
			problems = append(problems, pkgerrors.WithMessage(err, filePath))

			continue
		}
		if assessments[metrics.AssessmentKeyProcessingTime] == 0 {
			return nil, nil, pkgerrors.Errorf("no model response time measurement present for %q at repository %q", ctx.Model.ID(), ctx.Repository.Name())
		}
		modelAssessmentForFile.Add(assessments)
		modelAssessmentForFile.Award(metrics.AssessmentKeyResponseNoError)

		coverage, ps, err := ctx.Language.Execute(log, dataPath)
		problems = append(problems, ps...)
		if err != nil {
			problems = append(problems, pkgerrors.WithMessage(err, filePath))

			// If there is an execution timeout do not run "symflower fix" because the code itself is correct.
			if errors.Is(err, context.DeadlineExceeded) {
				modelAssessment.Add(modelAssessmentForFile)
				withSymflowerAssessment.Add(withSymflowerAssessmentForFile)

				continue
			}

			// Run "symflower fix"  if the model response fails to execute.
			if ctx.Language.ID() == "golang" { // Currently we only support Go for "symflower fix".
				log.Print("model response alone failed execution, attempting to fix with \"symflower fix \"")

				duration, err := symflowerFix(log, modelAssessment, dataPath, ctx.Language)
				if err != nil {
					problems = append(problems, err)

					modelAssessment.Add(modelAssessmentForFile)
					withSymflowerAssessment.Add(withSymflowerAssessmentForFile)

					continue
				}

				coverage, ps, err := ctx.Language.Execute(log, dataPath)
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
