package task

import (
	"context"
	"time"

	pkgerrors "github.com/pkg/errors"
	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	"github.com/symflower/eval-dev-quality/language"
	"github.com/symflower/eval-dev-quality/log"
	"github.com/symflower/eval-dev-quality/model"
	evaltask "github.com/symflower/eval-dev-quality/task"
	"github.com/symflower/eval-dev-quality/tools"
	"github.com/symflower/eval-dev-quality/util"
)

// symflowerFix runs the "symflower fix" command and returns its execution time in milliseconds.
func symflowerFix(logger *log.Logger, repositoryPath string, language language.Language) (duration uint64, err error) {
	start := time.Now()
	_, err = util.CommandWithResult(context.Background(), logger, &util.Command{
		Command: []string{
			tools.SymflowerPath, "fix",
			"--language", language.ID(),
			"--workspace", repositoryPath,
		},

		Directory: repositoryPath,
	})
	if err != nil {
		return 0, pkgerrors.WithStack(err)
	}

	return uint64(time.Since(start).Milliseconds()), nil
}

// symflowerTemplate runs the "symflower uts" command and returns its execution time in milliseconds.
func symflowerTemplate(logger *log.Logger, repositoryPath string, language language.Language, filePath string) (duration uint64, err error) {
	start := time.Now()

	_, err = util.CommandWithResult(context.Background(), logger, &util.Command{
		Command: []string{
			tools.SymflowerPath, "uts",
			"--language", language.ID(),
			"--workspace", repositoryPath,
			"--test-style", "basic",
			"--code-disable-fetch-dependencies",
			filePath,
		},

		Directory: repositoryPath,
	})
	if err != nil {
		return 0, pkgerrors.WithStack(err)
	}

	return uint64(time.Since(start).Milliseconds()), nil
}

// ExecuteWithSymflowerFix runs the "symflower fix" command and calculates the new assessments.
func ExecuteWithSymflowerFix(ctx evaltask.Context, logger *log.Logger, packagePath string) (testResult *language.TestResult, processingTime uint64, problems []error, err error) {
	// Run "symflower fix"  if the model response fails to execute.
	logger.Print("model response alone failed execution, attempting to fix with \"symflower fix \"")

	duration, err := symflowerFix(logger, packagePath, ctx.Language)
	if err != nil {
		return nil, 0, nil, pkgerrors.WithStack(err)
	}

	testResult, ps, err := ctx.Language.ExecuteTests(logger, packagePath)
	problems = append(problems, ps...)
	if err != nil {
		return testResult, duration, problems, pkgerrors.WithMessage(err, "symflower fix")
	}

	return testResult, duration, problems, nil
}

func runModelAndSymflowerFix(ctx evaltask.Context, modelCtx model.Context, runModel func(model.Context) (metrics.Assessments, error)) (modelAssessment metrics.Assessments, withSymflowerFixAssessment metrics.Assessments, problems []error, err error) {
	modelAssessment = metrics.NewAssessments()
	withSymflowerFixAssessment = modelAssessment // The symflower assessment tracks how the model result can be improved in case of a failure, so just link to the model assessment until we successfully applied "symflower fix".

	assessments, err := runModel(modelCtx)
	if err != nil {
		return nil, nil, append(problems, pkgerrors.WithMessage(err, modelCtx.FilePath)), nil
	}
	if assessments[metrics.AssessmentKeyProcessingTime] == 0 {
		return nil, nil, problems, pkgerrors.Errorf("no model response time measurement present for %q at repository %q", ctx.Model.ID(), ctx.Repository.Name())
	}
	modelAssessment.Add(assessments)
	modelAssessment.Award(metrics.AssessmentKeyResponseNoError)

	testResult, ps, err := ctx.Language.ExecuteTests(modelCtx.Logger, modelCtx.RepositoryPath)
	problems = append(problems, ps...)
	if err != nil {
		problems = append(problems, pkgerrors.WithMessage(err, modelCtx.FilePath))
	} else if ctx.Repository.Configuration().Validation.Execution.Validate(testResult.StdOut) {
		modelCtx.Logger.Printf("Executes tests with %d coverage objects", testResult.Coverage)
		modelAssessment.Award(metrics.AssessmentKeyFilesExecuted)
		modelAssessment.AwardPoints(metrics.AssessmentKeyCoverage, testResult.Coverage)
	}

	if ctx.Language.SupportsFix() {
		withSymflowerFixTestResult, processingTime, ps, err := ExecuteWithSymflowerFix(ctx, modelCtx.Logger, ctx.Repository.DataPath())
		problems = append(problems, ps...)
		if err != nil {
			problems = append(problems, err)
		} else if ctx.Repository.Configuration().Validation.Execution.Validate(withSymflowerFixTestResult.StdOut) {
			ctx.Logger.Printf("with symflower repair: Executes tests with %d coverage objects", withSymflowerFixTestResult.Coverage)

			// Symflower was able to fix a failure so now update the assessment with the improved results.
			withSymflowerFix := metrics.NewAssessments()
			withSymflowerFix[metrics.AssessmentKeyProcessingTime] = processingTime
			withSymflowerFix.Award(metrics.AssessmentKeyFilesExecuted)
			withSymflowerFix.AwardPoints(metrics.AssessmentKeyCoverage, withSymflowerFixTestResult.Coverage)

			withSymflowerFixAssessment = metrics.CombineWithSymflowerFixAssessments(modelAssessment, withSymflowerFix)
		}
	}

	return modelAssessment, withSymflowerFixAssessment, problems, nil
}
