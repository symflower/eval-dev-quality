package symflower

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"time"

	pkgerrors "github.com/pkg/errors"

	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	"github.com/symflower/eval-dev-quality/model"
	"github.com/symflower/eval-dev-quality/provider"
	"github.com/symflower/eval-dev-quality/tools"
	"github.com/symflower/eval-dev-quality/util"
)

// defaultSymflowerTimeout defines the default timeout for Symflower models.
var defaultSymflowerTimeout = time.Duration(10 * time.Minute)

// Model holds a Symflower model using the locally installed CLI.
type Model struct {
	// symbolicExecutionTimeout defines the symbolic execution timeout.
	symbolicExecutionTimeout time.Duration

	// command is the sub-command to use in the Symflower binary.
	command string
	// id is the ID of the model.
	id string
}

// NewModelSymbolicExecution returns a symbolic execution Symflower model.
func NewModelSymbolicExecution() (model *Model) {
	return NewModelSymbolicExecutionWithTimeout(defaultSymflowerTimeout)
}

// NewModelSymbolicExecutionWithTimeout returns a symbolic execution Symflower model with a given timeout.
func NewModelSymbolicExecutionWithTimeout(timeout time.Duration) (model *Model) {
	return &Model{
		symbolicExecutionTimeout: timeout,

		command: "unit-tests",
		id:      "symbolic-execution",
	}
}

// NewModelSmartTemplate returns a smart template Symflower model.
func NewModelSmartTemplate() (model *Model) {
	return NewModelSmartTemplateWithTimeout(defaultSymflowerTimeout)
}

// NewModelSmartTemplateWithTimeout returns a smart template Symflower model with a given timeout.
func NewModelSmartTemplateWithTimeout(timeout time.Duration) (model *Model) {
	return &Model{
		symbolicExecutionTimeout: timeout,

		command: "unit-test-skeletons",
		id:      "smart-template",
	}
}

var _ model.Model = (*Model)(nil)

// ID returns full identifier, including the provider and attributes.
func (m *Model) ID() (id string) {
	return "symflower" + provider.ProviderModelSeparator + m.id
}

// ModelID returns the unique identifier of this model with its provider.
func (m *Model) ModelID() (modelID string) {
	return "symflower" + provider.ProviderModelSeparator + m.id
}

// ModelIDWithoutProvider returns the unique identifier of this model without its provider.
func (m *Model) ModelIDWithoutProvider() (modelID string) {
	return m.id
}

// Attributes returns query attributes.
func (m *Model) Attributes() (attributes map[string]string) {
	return nil
}

// SetAttributes sets the given attributes.
func (m *Model) SetAttributes(attributes map[string]string) {
}

// MetaInformation returns the meta information of a model.
func (m *Model) MetaInformation() (metaInformation *model.MetaInformation) {
	return nil
}

// Clone returns a copy of the model.
func (m *Model) Clone() (clone model.Model) {
	model := *m

	return &model
}

var _ model.CapabilityWriteTests = (*Model)(nil)

// WriteTests generates test files for the given implementation file in a repository.
func (m *Model) WriteTests(ctx model.Context) (assessment metrics.Assessments, err error) {
	ctxWithTimeout, cancel := context.WithTimeout(context.Background(), m.symbolicExecutionTimeout)
	defer cancel()

	start := time.Now()

	output, err := util.CommandWithResult(ctxWithTimeout, ctx.Logger, &util.Command{
		Command: []string{
			tools.SymflowerPath, m.command,
			"--code-disable-fetch-dependencies",
			"--workspace", ctx.RepositoryPath,
			ctx.FilePath,
		},

		Directory: ctx.RepositoryPath,
	})
	if err != nil {
		return nil, pkgerrors.WithStack(err)
	}

	processingTime := float64(time.Since(start).Milliseconds())

	characterCount, err := countCharactersOfGeneratedFiles(ctx.RepositoryPath, extractGeneratedFilePaths(output))
	if err != nil {
		return nil, err
	}

	return metrics.Assessments{ // Symflower always generates just source code when it does not fail, so no need to check the assessment properties.
		metrics.AssessmentKeyProcessingTime:                     processingTime,
		metrics.AssessmentKeyGenerateTestsForFileCharacterCount: float64(characterCount),
		metrics.AssessmentKeyResponseCharacterCount:             float64(characterCount),
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
