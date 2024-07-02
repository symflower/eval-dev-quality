package task

import (
	"context"
	"time"

	pkgerrors "github.com/pkg/errors"
	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	"github.com/symflower/eval-dev-quality/language"
	"github.com/symflower/eval-dev-quality/log"
	"github.com/symflower/eval-dev-quality/tools"
	"github.com/symflower/eval-dev-quality/util"
)

// symflowerFix runs the "symflower fix" command and returns its execution time in milliseconds.
func symflowerFix(logger *log.Logger, modelAssessment metrics.Assessments, repositoryPath string, language language.Language) (duration uint64, err error) {
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
