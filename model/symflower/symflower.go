package symflower

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"time"

	pkgerrors "github.com/pkg/errors"

	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	evaluatetask "github.com/symflower/eval-dev-quality/evaluate/task"
	"github.com/symflower/eval-dev-quality/model"
	"github.com/symflower/eval-dev-quality/provider"
	"github.com/symflower/eval-dev-quality/task"
	"github.com/symflower/eval-dev-quality/tools"
	"github.com/symflower/eval-dev-quality/util"
)

// Model holds a Symflower model using the locally installed CLI.
type Model struct{}

// NewModel returns a Symflower model.
func NewModel() (model model.Model) {
	return &Model{}
}

var _ model.Model = (*Model)(nil)

// ID returns the unique ID of this model.
func (m *Model) ID() (id string) {
	return "symflower" + provider.ProviderModelSeparator + "symbolic-execution"
}

// IsTaskSupported returns whether the model supports the given task or not.
func (m *Model) IsTaskSupported(taskIdentifier task.Identifier) (isSupported bool) {
	switch taskIdentifier {
	case evaluatetask.IdentifierWriteTests:
		return true
	case evaluatetask.IdentifierCodeRepair:
		return false
	default:
		return false
	}
}

// RunTask runs the given task.
func (m *Model) RunTask(ctx task.Context, taskIdentifier task.Identifier) (assessments metrics.Assessments, err error) {
	switch taskIdentifier {
	case evaluatetask.IdentifierWriteTests:
		return m.generateTestsForFile(ctx)
	default:
		return nil, pkgerrors.Wrap(task.ErrTaskUnsupported, string(taskIdentifier))
	}
}

// generateTestsForFile generates test files for the given implementation file in a repository.
func (m *Model) generateTestsForFile(ctx task.Context) (assessment metrics.Assessments, err error) {
	start := time.Now()

	output, err := util.CommandWithResult(context.Background(), ctx.Logger, &util.Command{
		Command: []string{
			tools.SymflowerPath, "unit-tests",
			"--code-disable-fetch-dependencies",
			"--workspace", ctx.RepositoryPath,
			ctx.FilePath,
		},

		Directory: ctx.RepositoryPath,
	})
	if err != nil {
		return nil, pkgerrors.WithStack(err)
	}

	processingTime := uint64(time.Since(start).Milliseconds())

	characterCount, err := countCharactersOfGeneratedFiles(ctx.RepositoryPath, extractGeneratedFilePaths(output))
	if err != nil {
		return nil, err
	}

	return metrics.Assessments{ // Symflower always generates just source code when it does not fail, so no need to check the assessment properties.
		metrics.AssessmentKeyProcessingTime:                     processingTime,
		metrics.AssessmentKeyGenerateTestsForFileCharacterCount: characterCount,
		metrics.AssessmentKeyResponseCharacterCount:             characterCount,
		metrics.AssessmentKeyResponseNoExcess:                   1,
		metrics.AssessmentKeyResponseWithCode:                   1,
	}, nil
}

// unitTestFilePathsRe defines the regex to search for generated unit test file paths in a log output.
var unitTestFilePathsRe = regexp.MustCompile(`(?m): generated unit test file (.+)$`)

func extractGeneratedFilePaths(output string) (filePaths []string) {
	matches := unitTestFilePathsRe.FindAllStringSubmatch(output, -1)

	for _, match := range matches {
		filePaths = append(filePaths, match[1])
	}

	return filePaths
}

func countCharactersOfGeneratedFiles(repositoryPath string, filePaths []string) (count uint64, err error) {
	for _, filePath := range filePaths {
		fileContent, err := os.ReadFile(filepath.Join(repositoryPath, filePath))
		if err != nil {
			return 0, pkgerrors.WithStack(err)
		}

		count += uint64(len(string(fileContent)))
	}

	return count, nil
}

// Cost returns the cost of the model.
func (m *Model) Cost() (cost float64) {
	return 0
}
