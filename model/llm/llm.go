package llm

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/avast/retry-go"
	pkgerrors "github.com/pkg/errors"
	"github.com/zimmski/osutil/bytesutil"

	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	"github.com/symflower/eval-dev-quality/language"
	"github.com/symflower/eval-dev-quality/log"
	"github.com/symflower/eval-dev-quality/model"
	"github.com/symflower/eval-dev-quality/model/llm/prompt"
	"github.com/symflower/eval-dev-quality/provider"
)

// Model represents a LLM model accessed via a provider.
type Model struct {
	// provider is the client to query the LLM model.
	provider provider.Query
	// model holds the identifier for the LLM model.
	model string

	// queryAttempts holds the number of query attempts to perform when a model request errors in the process of solving a task.
	queryAttempts uint
}

// NewModel returns an LLM model corresponding to the given identifier which is queried via the given provider.
func NewModel(provider provider.Query, modelIdentifier string) model.Model {
	return &Model{
		provider: provider,
		model:    modelIdentifier,

		queryAttempts: 1,
	}
}

// llmGenerateTestForFilePromptContext is the context for template for generating an LLM test generation prompt.
type llmGenerateTestForFilePromptContext struct {
	// Language holds the programming language name.
	Language language.Language

	// Code holds the source code of the file.
	Code string
	// FilePath holds the file path of the file.
	FilePath string
	// ImportPath holds the import path of the file.
	ImportPath string
}

// llmGenerateTestForFilePromptTemplate is the template for generating an LLM test generation prompt.
var llmGenerateTestForFilePromptTemplate = template.Must(template.New("model-llm-generate-test-for-file-prompt").Parse(bytesutil.StringTrimIndentations(`
	Given the following {{ .Language.Name }} code file "{{ .FilePath }}" with package "{{ .ImportPath }}", provide a test file for this code{{ with $testFramework := .Language.TestFramework }} with {{ $testFramework }} as a test framework{{ end }}.
	The tests should produce 100 percent code coverage and must compile.
	The response must contain only the test code and nothing else.

	` + "```" + `{{ .Language.ID }}
	{{ .Code }}
	` + "```" + `
`)))

// llmGenerateTestForFilePrompt returns the prompt for generating an LLM test generation.
func llmGenerateTestForFilePrompt(data *llmGenerateTestForFilePromptContext) (message string, err error) {
	data.Code = strings.TrimSpace(data.Code)

	var b strings.Builder
	if err := llmGenerateTestForFilePromptTemplate.Execute(&b, data); err != nil {
		return "", pkgerrors.WithStack(err)
	}

	return b.String(), nil
}

var _ model.Model = (*Model)(nil)

// ID returns the unique ID of this model.
func (m *Model) ID() (id string) {
	return m.model
}

// GenerateTestsForFile generates test files for the given implementation file in a repository.
func (m *Model) GenerateTestsForFile(logger *log.Logger, language language.Language, repositoryPath string, filePath string) (assessment metrics.Assessments, err error) {
	data, err := os.ReadFile(filepath.Join(repositoryPath, filePath))
	if err != nil {
		return nil, pkgerrors.WithStack(err)
	}
	fileContent := strings.TrimSpace(string(data))

	importPath := language.ImportPath(repositoryPath, filePath)

	request, err := llmGenerateTestForFilePrompt(&llmGenerateTestForFilePromptContext{
		Language: language,

		Code:       fileContent,
		FilePath:   filePath,
		ImportPath: importPath,
	})
	if err != nil {
		return nil, err
	}

	var response string
	var duration time.Duration
	if err := retry.Do(
		func() error {
			logger.Printf("Querying model %q with:\n%s", m.ID(), string(bytesutil.PrefixLines([]byte(request), []byte("\t"))))
			start := time.Now()
			response, err = m.provider.Query(context.Background(), m.model, request)
			if err != nil {
				return err
			}
			duration = time.Since(start)
			logger.Printf("Model %q responded (%d ms) with:\n%s", m.ID(), duration.Milliseconds(), string(bytesutil.PrefixLines([]byte(response), []byte("\t"))))

			return nil
		},
		retry.Attempts(m.queryAttempts),
		retry.Delay(2*time.Second),
		retry.LastErrorOnly(true),
		retry.OnRetry(func(n uint, err error) {
			logger.Printf("Attempt %d/%d: %s", n+1, m.queryAttempts, err)
		}),
		retry.RetryIf(func(err error) bool {
			return true
		}),
	); err != nil {
		return nil, err
	}

	assessment, testContent, err := prompt.ParseResponse(response)
	if err != nil {
		return nil, err
	}
	assessment[metrics.AssessmentKeyProcessingTime] = uint64(duration.Milliseconds())

	testFilePath := language.TestFilePath(repositoryPath, filePath)
	if err := os.MkdirAll(filepath.Join(repositoryPath, filepath.Dir(testFilePath)), 0755); err != nil {
		return nil, pkgerrors.WithStack(err)
	}
	if err := os.WriteFile(filepath.Join(repositoryPath, testFilePath), []byte(testContent), 0644); err != nil {
		return nil, pkgerrors.WithStack(err)
	}

	return assessment, nil
}

// SetQueryAttempts sets the number of query attempts to perform when a model request errors in the process of solving a task.
func (m *Model) SetQueryAttempts(queryAttempts uint) {
	m.queryAttempts = queryAttempts
}
