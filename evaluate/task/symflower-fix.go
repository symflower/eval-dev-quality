package task

import (
	"context"
	"time"

	pkgerrors "github.com/pkg/errors"
	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	"github.com/symflower/eval-dev-quality/language"
	"github.com/symflower/eval-dev-quality/log"
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

// ExecuteWithSymflowerFix runs the "symflower fix" command and calculates the new assessments.
func ExecuteWithSymflowerFix(ctx evaltask.Context, logger *log.Logger, packagePath string) (assessments metrics.Assessments, problems []error, err error) {
	// Run "symflower fix"  if the model response fails to execute.
	logger.Print("model response alone failed execution, attempting to fix with \"symflower fix \"")

	duration, err := symflowerFix(logger, packagePath, ctx.Language)
	if err != nil {
		return nil, problems, pkgerrors.WithStack(err)
	}

	coverage, ps, err := ctx.Language.Execute(logger, packagePath)
	problems = append(problems, ps...)
	if err != nil {
		return nil, problems, pkgerrors.WithMessage(err, "symflower fix")
	}
	logger.Printf("with symflower repair: Executes tests with %d coverage objects", coverage)

	// Symflower was able to fix a failure so now update the assessment with the improved results.
	assessments = metrics.NewAssessments()
	assessments[metrics.AssessmentKeyProcessingTime] = duration
	assessments.Award(metrics.AssessmentKeyFilesExecuted)
	assessments.AwardPoints(metrics.AssessmentKeyCoverage, coverage)

	return assessments, problems, nil
}
