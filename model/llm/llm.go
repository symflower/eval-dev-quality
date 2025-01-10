package llm

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/avast/retry-go"
	"github.com/google/uuid"
	pkgerrors "github.com/pkg/errors"
	"github.com/zimmski/osutil/bytesutil"

	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	evaluatetask "github.com/symflower/eval-dev-quality/evaluate/task"
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

	// metaInformation holds a model meta information.
	metaInformation *model.MetaInformation
}

// NewModel returns an LLM model corresponding to the given identifier which is queried via the given provider.
func NewModel(provider provider.Query, modelIdentifier string) *Model {
	return &Model{
		provider: provider,
		model:    modelIdentifier,

		queryAttempts: 1,
	}
}

// NewModelWithMetaInformation returns a LLM model with meta information corresponding to the given identifier which is queried via the given provider.
func NewModelWithMetaInformation(provider provider.Query, modelIdentifier string, metaInformation *model.MetaInformation) *Model {
	return &Model{
		provider: provider,
		model:    modelIdentifier,

		queryAttempts: 1,

		metaInformation: metaInformation,
	}
}

// MetaInformation returns the meta information of a model.
func (m *Model) MetaInformation() (metaInformation *model.MetaInformation) {
	return m.metaInformation
}

// llmSourceFilePromptContext is the base template context for an LLM generation prompt.
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

// llmWriteTestSourceFilePromptContext is the template context for a write test LLM prompt.
type llmWriteTestSourceFilePromptContext struct {
	// llmSourceFilePromptContext holds the context for a source file prompt.
	llmSourceFilePromptContext

	// Template holds the template data to base the tests onto.
	Template string
	// TestFramework holds the test framework to use.
	TestFramework string
}

// llmWriteTestForFilePromptTemplate is the template for generating an LLM test generation prompt.
var llmWriteTestForFilePromptTemplate = template.Must(template.New("model-llm-write-test-for-file-prompt").Parse(bytesutil.StringTrimIndentations(`
	Given the following {{ .Language.Name }} code file "{{ .FilePath }}" with package "{{ .ImportPath }}", provide a test file for this code{{ with .TestFramework }} with {{ . }} as a test framework{{ end }}.
	The tests should produce 100 percent code coverage and must compile.
	The response must contain only the test code in a fenced code block and nothing else.

	` + "```" + `{{ .Language.ID }}
	{{ .Code }}
	` + "```" + `
	{{- if .Template}}

	The tests should be based on this template:

	` + "```" + `{{ .Language.ID }}
	{{ .Template -}}
	` + "```" + `
	{{- end}}
`)))

// Format returns the prompt for generating an LLM test generation.
func (ctx *llmWriteTestSourceFilePromptContext) Format() (message string, err error) {
	// Use Linux paths even when running the evaluation on Windows to ensure consistency in prompting.
	ctx.FilePath = filepath.ToSlash(ctx.FilePath)
	ctx.Code = strings.TrimSpace(ctx.Code)

	var b strings.Builder
	if err := llmWriteTestForFilePromptTemplate.Execute(&b, ctx); err != nil {
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
	The response must contain only the source code in a fenced code block and nothing else.

	` + "```" + `{{ .Language.ID }}
	{{ .Code }}
	` + "```" + `

	The list of compilation errors is the following:{{ range .Mistakes }}
	- {{.}}{{ end }}
`)))

// Format returns the prompt to code repair a source file.
func (ctx *llmCodeRepairSourceFilePromptContext) Format() (message string, err error) {
	// Use Linux paths even when running the evaluation on Windows to ensure consistency in prompting.
	ctx.FilePath = filepath.ToSlash(ctx.FilePath)
	ctx.Code = strings.TrimSpace(ctx.Code)

	var b strings.Builder
	if err := llmCodeRepairSourceFilePromptTemplate.Execute(&b, ctx); err != nil {
		return "", pkgerrors.WithStack(err)
	}

	return b.String(), nil
}

// llmTranspileSourceFilePromptContext is the template context for a transpilation LLM prompt.
type llmTranspileSourceFilePromptContext struct {
	// llmSourceFilePromptContext holds the context for a source file prompt.
	llmSourceFilePromptContext

	// OriginLanguage holds the language we are transpiling from.
	OriginLanguage language.Language
	// OriginFileContent holds the code we want to transpile.
	OriginFileContent string
}

// llmTranspileSourceFilePromptTemplate is the template for generating an LLM transpilation prompt.
var llmTranspileSourceFilePromptTemplate = template.Must(template.New("model-llm-transpile-source-file-prompt").Parse(bytesutil.StringTrimIndentations(`
	Given the following {{ .OriginLanguage.Name }} code file, transpile it into a {{ .Language.Name }} code file.
	The response must contain only the transpiled {{ .Language.Name }} source code in a fenced code block and nothing else.

	` + "```" + `{{ .OriginLanguage.ID }}
	{{ .OriginFileContent }}
	` + "```" + `

	The transpiled {{ .Language.Name }} code file must have the package "{{ .ImportPath }}" and the following signature:

	` + "```" + `{{ .Language.ID }}
	{{ .Code }}
	` + "```" + `
`)))

// Format returns the prompt to transpile a source file.
func (ctx *llmTranspileSourceFilePromptContext) Format() (message string, err error) {
	// Use Linux paths even when running the evaluation on Windows to ensure consistency in prompting.
	ctx.FilePath = filepath.ToSlash(ctx.FilePath)
	ctx.Code = strings.TrimSpace(ctx.Code)
	ctx.OriginFileContent = strings.TrimSpace(ctx.OriginFileContent)

	var b strings.Builder
	if err := llmTranspileSourceFilePromptTemplate.Execute(&b, ctx); err != nil {
		return "", pkgerrors.WithStack(err)
	}

	return b.String(), nil
}

// llmMigrateSourceFilePromptContext is the template context for a migration LLM prompt.
type llmMigrateSourceFilePromptContext struct {
	// llmSourceFilePromptContext holds the context for a source file prompt.
	llmSourceFilePromptContext

	// TestFramework defines the target test framework for migration.
	TestFramework string
}

// llmMigrateSourceFilePromptTemplate is the template for generating an LLM migration prompt.
var llmMigrateSourceFilePromptTemplate = template.Must(template.New("model-llm-migration-source-file-prompt").Parse(bytesutil.StringTrimIndentations(`
	Given the following {{ .Language.Name }} test file "{{ .FilePath }}" with package "{{ .ImportPath }}", migrate the test file to {{ .TestFramework }} as the test framework.
	The tests should produce 100 percent code coverage and must compile.
	The response must contain only the test code in a fenced code block and nothing else.

	` + "```" + `{{ .Language.ID }}
	{{ .Code }}
	` + "```" + `
`)))

// Format returns the prompt to migrate a source file.
func (ctx *llmMigrateSourceFilePromptContext) Format() (message string, err error) {
	// Use Linux paths even when running the evaluation on Windows to ensure consistency in prompting.
	ctx.FilePath = filepath.ToSlash(ctx.FilePath)
	ctx.Code = strings.TrimSpace(ctx.Code)

	var b strings.Builder
	if err := llmMigrateSourceFilePromptTemplate.Execute(&b, ctx); err != nil {
		return "", pkgerrors.WithStack(err)
	}

	return b.String(), nil
}

var _ model.Model = (*Model)(nil)

// ID returns the unique ID of this model.
func (m *Model) ID() (id string) {
	return m.model
}

var _ model.CapabilityWriteTests = (*Model)(nil)

// WriteTests generates test files for the given implementation file in a repository.
func (m *Model) WriteTests(ctx model.Context) (assessment metrics.Assessments, err error) {
	arguments, ok := ctx.Arguments.(*evaluatetask.ArgumentsWriteTest)
	if !ok {
		return nil, pkgerrors.Errorf("unexpected type %T", ctx.Arguments)
	}

	data, err := os.ReadFile(filepath.Join(ctx.RepositoryPath, ctx.FilePath))
	if err != nil {
		return nil, pkgerrors.WithStack(err)
	}
	fileContent := strings.TrimSpace(string(data))

	importPath := ctx.Language.ImportPath(ctx.RepositoryPath, ctx.FilePath)

	request, err := (&llmWriteTestSourceFilePromptContext{
		llmSourceFilePromptContext: llmSourceFilePromptContext{
			Language: ctx.Language,

			Code:       fileContent,
			FilePath:   ctx.FilePath,
			ImportPath: importPath,
		},

		Template:      arguments.Template,
		TestFramework: arguments.TestFramework,
	}).Format()
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

func (m *Model) query(logger *log.Logger, request string) (response string, duration time.Duration, err error) {
	if err := retry.Do(
		func() error {
			id := uuid.NewString
			logger.Info("querying model", "model", m.ID(), "id", id, "prompt", string(bytesutil.PrefixLines([]byte(request), []byte("\t"))))
			start := time.Now()
			response, err = m.provider.Query(context.Background(), m.model, request)
			if err != nil {
				return err
			}
			duration = time.Since(start)
			logger.Info("model responded", "model", m.ID(), "id", id, "duration", duration.Milliseconds(), "response", string(bytesutil.PrefixLines([]byte(response), []byte("\t"))))

			return nil
		},
		retry.Attempts(m.queryAttempts),
		retry.Delay(5*time.Second),
		retry.DelayType(retry.BackOffDelay),
		retry.LastErrorOnly(true),
		retry.OnRetry(func(n uint, err error) {
			logger.Info("query retry", "count", n+1, "total", m.queryAttempts, "error", err)
		}),
	); err != nil {
		return "", 0, err
	}

	return response, duration, nil
}

var _ model.CapabilityRepairCode = (*Model)(nil)

// RepairCode queries the model to repair a source code with compilation error.
func (m *Model) RepairCode(ctx model.Context) (assessment metrics.Assessments, err error) {
	codeRepairArguments, ok := ctx.Arguments.(*evaluatetask.ArgumentsCodeRepair)
	if !ok {
		return nil, pkgerrors.Errorf("unexpected type %#v", ctx.Arguments)
	}

	data, err := os.ReadFile(filepath.Join(ctx.RepositoryPath, ctx.FilePath))
	if err != nil {
		return nil, pkgerrors.WithStack(err)
	}
	fileContent := strings.TrimSpace(string(data))

	importPath := ctx.Language.ImportPath(ctx.RepositoryPath, ctx.FilePath)

	request, err := (&llmCodeRepairSourceFilePromptContext{
		llmSourceFilePromptContext: llmSourceFilePromptContext{
			Language: ctx.Language,

			Code:       fileContent,
			FilePath:   ctx.FilePath,
			ImportPath: importPath,
		},

		Mistakes: codeRepairArguments.Mistakes,
	}).Format()
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

var _ model.CapabilityTranspile = (*Model)(nil)

// Transpile queries the model to transpile source code to another language.
func (m *Model) Transpile(ctx model.Context) (assessment metrics.Assessments, err error) {
	transpileArguments, ok := ctx.Arguments.(*evaluatetask.ArgumentsTranspile)
	if !ok {
		return nil, pkgerrors.Errorf("unexpected type %#v", ctx.Arguments)
	}

	data, err := os.ReadFile(filepath.Join(ctx.RepositoryPath, ctx.FilePath))
	if err != nil {
		return nil, pkgerrors.WithStack(err)
	}
	stubFileContent := strings.TrimSpace(string(data))

	data, err = os.ReadFile(filepath.Join(ctx.RepositoryPath, transpileArguments.OriginFilePath))
	if err != nil {
		return nil, pkgerrors.WithStack(err)
	}
	originFileContent := strings.TrimSpace(string(data))

	importPath := ctx.Language.ImportPath(ctx.RepositoryPath, ctx.FilePath)

	request, err := (&llmTranspileSourceFilePromptContext{
		llmSourceFilePromptContext: llmSourceFilePromptContext{
			Language: ctx.Language,

			Code:       stubFileContent,
			FilePath:   ctx.FilePath,
			ImportPath: importPath,
		},

		OriginLanguage:    transpileArguments.OriginLanguage,
		OriginFileContent: originFileContent,
	}).Format()
	if err != nil {
		return nil, err
	}

	response, duration, err := m.query(ctx.Logger, request)
	if err != nil {
		return nil, pkgerrors.WithStack(err)
	}

	assessment, originFileContent, err = prompt.ParseResponse(response)
	if err != nil {
		return nil, pkgerrors.WithStack(err)
	}
	assessment[metrics.AssessmentKeyProcessingTime] = uint64(duration.Milliseconds())
	assessment[metrics.AssessmentKeyResponseCharacterCount] = uint64(len(response))
	assessment[metrics.AssessmentKeyGenerateTestsForFileCharacterCount] = uint64(len(originFileContent))

	err = os.WriteFile(filepath.Join(ctx.RepositoryPath, ctx.FilePath), []byte(originFileContent), 0644)
	if err != nil {
		return nil, pkgerrors.WithStack(err)
	}

	return assessment, nil
}

var _ model.CapabilityMigrate = (*Model)(nil)

// Migrate queries the model to migrate source code.
func (m *Model) Migrate(ctx model.Context) (assessment metrics.Assessments, err error) {
	arguments, ok := ctx.Arguments.(*evaluatetask.ArgumentsMigrate)
	if !ok {
		return nil, pkgerrors.Errorf("unexpected type %T", ctx.Arguments)
	}

	data, err := os.ReadFile(filepath.Join(ctx.RepositoryPath, ctx.FilePath))
	if err != nil {
		return nil, pkgerrors.WithStack(err)
	}
	fileContent := strings.TrimSpace(string(data))

	importPath := ctx.Language.ImportPath(ctx.RepositoryPath, ctx.FilePath)

	request, err := (&llmMigrateSourceFilePromptContext{
		llmSourceFilePromptContext: llmSourceFilePromptContext{
			Language: ctx.Language,

			Code:       fileContent,
			FilePath:   ctx.FilePath,
			ImportPath: importPath,
		},

		TestFramework: arguments.TestFramework,
	}).Format()
	if err != nil {
		return nil, err
	}

	response, duration, err := m.query(ctx.Logger, request)
	if err != nil {
		return nil, pkgerrors.WithStack(err)
	}

	assessment, migrationFileContent, err := prompt.ParseResponse(response)
	if err != nil {
		return nil, pkgerrors.WithStack(err)
	}
	assessment[metrics.AssessmentKeyProcessingTime] = uint64(duration.Milliseconds())
	assessment[metrics.AssessmentKeyResponseCharacterCount] = uint64(len(response))
	assessment[metrics.AssessmentKeyGenerateTestsForFileCharacterCount] = uint64(len(migrationFileContent))

	err = os.WriteFile(filepath.Join(ctx.RepositoryPath, ctx.FilePath), []byte(migrationFileContent), 0644)
	if err != nil {
		return nil, pkgerrors.WithStack(err)
	}

	return assessment, nil
}

var _ model.SetQueryAttempts = (*Model)(nil)

// SetQueryAttempts sets the number of query attempts to perform when a model request errors in the process of solving a task.
func (m *Model) SetQueryAttempts(queryAttempts uint) {
	m.queryAttempts = queryAttempts
}
