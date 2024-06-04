package symflower

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"time"

	pkgerrors "github.com/pkg/errors"

	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	"github.com/symflower/eval-dev-quality/language"
	"github.com/symflower/eval-dev-quality/log"
	"github.com/symflower/eval-dev-quality/model"
	"github.com/symflower/eval-dev-quality/provider"
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

// GenerateTestsForFile generates test files for the given implementation file in a repository.
func (m *Model) GenerateTestsForFile(logger *log.Logger, language language.Language, repositoryPath string, filePath string) (assessment metrics.Assessments, err error) {
	start := time.Now()

	output, err := util.CommandWithResult(context.Background(), logger, &util.Command{
		Command: []string{
			tools.SymflowerPath, "unit-tests",
			"--code-disable-fetch-dependencies",
			"--workspace", repositoryPath,
			filePath,
		},

		Directory: repositoryPath,
	})
	if err != nil {
		return nil, pkgerrors.WithStack(err)
	}

	processingTime := uint64(time.Since(start).Milliseconds())

	characterCount, err := countCharactersOfGeneratedFiles(repositoryPath, extractGeneratedFilePaths(output))
	if err != nil {
		return nil, err
	}

	if language.ID() == "golang" {
		_, err = util.CommandWithResult(context.Background(), logger, &util.Command{
			Command: []string{
				"go",
				"mod",
				"tidy",
			},

			Directory: repositoryPath,
		})
		if err != nil {
			return nil, pkgerrors.WithStack(err)
		}
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
