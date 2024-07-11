package llm

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/avast/retry-go"
	pkgerrors "github.com/pkg/errors"
	"github.com/zimmski/osutil/bytesutil"

	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	evaluatetask "github.com/symflower/eval-dev-quality/evaluate/task"
	"github.com/symflower/eval-dev-quality/language"
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
	// name holds the name for the LLM model.
	name string

	// queryAttempts holds the number of query attempts to perform when a model request errors in the process of solving a task.
	queryAttempts uint

	// cost holds the cost of a model
	cost float64
}

// NewModel returns an LLM model corresponding to the given identifier which is queried via the given provider.
func NewModel(provider provider.Query, modelIdentifier string) *Model {
	return &Model{
		provider: provider,
		model:    modelIdentifier,

		queryAttempts: 1,
	}
}

// NewNamedModelWithCost returns an LLM model corresponding to the given identifier which is queried via the given provider, and with name and pricing information.
func NewNamedModelWithCost(provider provider.Query, modelIdentifier string, name string, cost float64) *Model {
	return &Model{
		provider: provider,
		model:    modelIdentifier,
		name:     name,

		queryAttempts: 1,

		cost: cost,
	}
}

// llmSourceFilePromptContext is the context for template for generating an LLM test generation prompt.
type llmSourceFilePromptContext struct {
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
	The response must contain only the test code in a fenced code block and nothing else.

	` + "```" + `{{ .Language.ID }}
	{{ .Code }}
	` + "```" + `
`)))

// llmGenerateTestForFilePrompt returns the prompt for generating an LLM test generation.
func llmGenerateTestForFilePrompt(data *llmSourceFilePromptContext) (message string, err error) {
	data.Code = strings.TrimSpace(data.Code)

	var b strings.Builder
	if err := llmGenerateTestForFilePromptTemplate.Execute(&b, data); err != nil {
		return "", pkgerrors.WithStack(err)
	}

	return b.String(), nil
}

// llmCodeRepairSourceFilePromptContext is the template context for a code repair LLM prompt.
type llmCodeRepairSourceFilePromptContext struct {
	// llmSourceFilePromptContext holds the context for a source file prompt.
	llmSourceFilePromptContext

	// Mistakes holds the list of compilation errors of a package.
	Mistakes []string
}

// llmCodeRepairSourceFilePromptTemplate is the template for generating an LLM code repair prompt.
var llmCodeRepairSourceFilePromptTemplate = template.Must(template.New("model-llm-code-repair-source-file-prompt").Parse(bytesutil.StringTrimIndentations(`
	Given the following {{ .Language.Name }} code file "{{ .FilePath }}" with package "{{ .ImportPath }}" and a list of compilation errors, modify the code such that the errors are resolved.
	The response must contain only the source code and nothing else.

	` + "```" + `{{ .Language.ID }}
	{{ .Code }}
	` + "```" + `

	The list of compilation errors is the following:{{ range .Mistakes }}
	- {{.}}{{ end }}
`)))

// llmCodeRepairSourceFilePrompt returns the prompt to code repair a source file.
func llmCodeRepairSourceFilePrompt(data *llmCodeRepairSourceFilePromptContext) (message string, err error) {
	data.Code = strings.TrimSpace(data.Code)

	var b strings.Builder
	if err := llmCodeRepairSourceFilePromptTemplate.Execute(&b, data); err != nil {
		return "", pkgerrors.WithStack(err)
	}

	return b.String(), nil
}

var _ model.Model = (*Model)(nil)

// ID returns the unique ID of this model.
func (m *Model) ID() (id string) {
	return m.model
}

// Name returns the name of this model.
func (m *Model) Name() (name string) {
	return m.name
}

var _ model.CapabilityWriteTests = (*Model)(nil)

// WriteTests generates test files for the given implementation file in a repository.
func (m *Model) WriteTests(ctx model.Context) (assessment metrics.Assessments, err error) {
	data, err := os.ReadFile(filepath.Join(ctx.RepositoryPath, ctx.FilePath))
	if err != nil {
		return nil, pkgerrors.WithStack(err)
	}
	fileContent := strings.TrimSpace(string(data))

	importPath := ctx.Language.ImportPath(ctx.RepositoryPath, ctx.FilePath)

	request, err := llmGenerateTestForFilePrompt(&llmSourceFilePromptContext{
		Language: ctx.Language,

		Code:       fileContent,
		FilePath:   ctx.FilePath,
		ImportPath: importPath,
	})
	if err != nil {
		return nil, err
	}

	response, duration, err := m.query(ctx.Logger, request)
	if err != nil {
		return nil, pkgerrors.WithStack(err)
	}

	assessment, testContent, err := prompt.ParseResponse(response)
	if err != nil {
		return nil, pkgerrors.WithStack(err)
	}
	assessment[metrics.AssessmentKeyProcessingTime] = uint64(duration.Milliseconds())
	assessment[metrics.AssessmentKeyResponseCharacterCount] = uint64(len(response))
	assessment[metrics.AssessmentKeyGenerateTestsForFileCharacterCount] = uint64(len(testContent))

	testFilePath := ctx.Language.TestFilePath(ctx.RepositoryPath, ctx.FilePath)
	if err := os.MkdirAll(filepath.Join(ctx.RepositoryPath, filepath.Dir(testFilePath)), 0755); err != nil {
		return nil, pkgerrors.WithStack(err)
	}
	if err := os.WriteFile(filepath.Join(ctx.RepositoryPath, testFilePath), []byte(testContent), 0644); err != nil {
		return nil, pkgerrors.WithStack(err)
	}

	return assessment, nil
}

func (m *Model) query(log *log.Logger, request string) (response string, duration time.Duration, err error) {
	if err := retry.Do(
		func() error {
			log.Printf("Querying model %q with:\n%s", m.ID(), string(bytesutil.PrefixLines([]byte(request), []byte("\t"))))
			start := time.Now()
			response, err = m.provider.Query(context.Background(), m.model, request)
			if err != nil {
				return err
			}
			duration = time.Since(start)
			log.Printf("Model %q responded (%d ms) with:\n%s", m.ID(), duration.Milliseconds(), string(bytesutil.PrefixLines([]byte(response), []byte("\t"))))

			return nil
		},
		retry.Attempts(m.queryAttempts),
		retry.Delay(5*time.Second),
		retry.DelayType(retry.BackOffDelay),
		retry.LastErrorOnly(true),
		retry.OnRetry(func(n uint, err error) {
			log.Printf("Attempt %d/%d: %s", n+1, m.queryAttempts, err)
		}),
	); err != nil {
		return "", 0, err
	}

	return response, duration, nil
}

var _ model.CapabilityRepairCode = (*Model)(nil)

// RepairCode queries the model to repair a source code with compilation error.
func (m *Model) RepairCode(ctx model.Context) (assessment metrics.Assessments, err error) {
	codeRepairArguments, ok := ctx.Arguments.(*evaluatetask.TaskArgumentsCodeRepair)
	if !ok {
		return nil, pkgerrors.Errorf("unexpected type %#v", ctx.Arguments)
	}

	assessment = map[metrics.AssessmentKey]uint64{}

	data, err := os.ReadFile(filepath.Join(ctx.RepositoryPath, ctx.FilePath))
	if err != nil {
		return nil, pkgerrors.WithStack(err)
	}
	fileContent := strings.TrimSpace(string(data))

	importPath := ctx.Language.ImportPath(ctx.RepositoryPath, ctx.FilePath)

	request, err := llmCodeRepairSourceFilePrompt(&llmCodeRepairSourceFilePromptContext{
		llmSourceFilePromptContext: llmSourceFilePromptContext{
			Language: ctx.Language,

			Code:       fileContent,
			FilePath:   ctx.FilePath,
			ImportPath: importPath,
		},

		Mistakes: codeRepairArguments.Mistakes,
	})
	if err != nil {
		return nil, err
	}

	response, duration, err := m.query(ctx.Logger, request)
	if err != nil {
		return nil, pkgerrors.WithStack(err)
	}

	assessment, sourceFileContent, err := prompt.ParseResponse(response)
	if err != nil {
		return nil, pkgerrors.WithStack(err)
	}
	assessment[metrics.AssessmentKeyProcessingTime] = uint64(duration.Milliseconds())
	assessment[metrics.AssessmentKeyResponseCharacterCount] = uint64(len(response))
	assessment[metrics.AssessmentKeyGenerateTestsForFileCharacterCount] = uint64(len(sourceFileContent))

	err = os.WriteFile(filepath.Join(ctx.RepositoryPath, ctx.FilePath), []byte(sourceFileContent), 0644)
	if err != nil {
		return nil, pkgerrors.WithStack(err)
	}

	return assessment, nil
}

// Cost returns the cost of the model.
func (m *Model) Cost() (cost float64) {
	return m.cost
}

// SetCost sets the cost of a model.
func (m *Model) SetCost(cost float64) {
	m.cost = cost
}

var _ model.SetQueryAttempts = (*Model)(nil)

// SetQueryAttempts sets the number of query attempts to perform when a model request errors in the process of solving a task.
func (m *Model) SetQueryAttempts(queryAttempts uint) {
	m.queryAttempts = queryAttempts
}
