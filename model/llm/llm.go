package llm

import (
	"context"
	"errors"
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
	// id holds the full identifier, including the provider and attributes.
	id string
	// provider is the client to query the LLM model.
	provider provider.Query
	// modelID holds the identifier for the LLM model.
	modelID string

	// attributes holds query attributes.
	attributes map[string]string
	// apiRequestAttempts holds the number of allowed API requests per LLM query.
	apiRequestAttempts uint
	// apiRequestTimeout holds the timeout for API requests in seconds.
	apiRequestTimeout uint

	// metaInformation holds a model meta information.
	metaInformation *model.MetaInformation
}

// NewModel returns an LLM model corresponding to the given identifier which is queried via the given provider.
func NewModel(provider provider.Query, modelIDWithAttributes string) (llmModel *Model) {
	llmModel = &Model{
		id:       modelIDWithAttributes,
		provider: provider,

		apiRequestAttempts: 1,
		apiRequestTimeout:  0,
	}
	llmModel.modelID, llmModel.attributes = model.ParseModelID(modelIDWithAttributes)

	return llmModel
}

// NewModelWithMetaInformation returns a LLM model with meta information corresponding to the given identifier which is queried via the given provider.
func NewModelWithMetaInformation(provider provider.Query, modelIdentifier string, metaInformation *model.MetaInformation) *Model {
	return &Model{
		id:       modelIdentifier,
		provider: provider,
		modelID:  modelIdentifier,

		apiRequestAttempts: 1,
		apiRequestTimeout:  0,

		metaInformation: metaInformation,
	}
}

var _ model.Model = (*Model)(nil)

// ID returns full identifier, including the provider and attributes.
func (m *Model) ID() (id string) {
	attributeString := ""
	for key, value := range m.attributes {
		attributeString += "@" + key + "=" + value
	}

	return m.id + attributeString
}

// ModelID returns the unique identifier of this model with its provider.
func (m *Model) ModelID() (modelID string) {
	return m.modelID
}

// ModelIDWithoutProvider returns the unique identifier of this model without its provider.
func (m *Model) ModelIDWithoutProvider() (modelID string) {
	_, modelID, ok := strings.Cut(m.modelID, provider.ProviderModelSeparator)
	if !ok {
		panic(m.modelID)
	}

	return modelID
}

// Attributes returns query attributes.
func (m *Model) Attributes() (attributes map[string]string) {
	return m.attributes
}

// SetAttributes sets the given attributes.
func (m *Model) SetAttributes(attributes map[string]string) {
	m.attributes = attributes
}

// MetaInformation returns the meta information of a model.
func (m *Model) MetaInformation() (metaInformation *model.MetaInformation) {
	return m.metaInformation
}

// Clone returns a copy of the model.
func (m *Model) Clone() (clone model.Model) {
	model := *m

	return &model
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
	// HasTestsInSource determines if the tests for this repository are located within the corresponding implementation file.
	HasTestsInSource bool
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
	Given the following {{ .Language.Name }} code file "{{ .FilePath }}" {{- with .ImportPath }} with package "{{ . }}" {{- end }}, {{- if .HasTestsInSource }} provide tests for this code that can be appended to the source file{{ else }} provide a test file for this code{{ with .TestFramework }} with {{ . }} as a test framework{{ end }}{{ end -}}.
	{{- if .HasTestsInSource }}
	Add everything required for testing, but do not repeat the original file and do not try to import the code file.
	{{- end }}
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
	Given the following {{ .Language.Name }} code file "{{ .FilePath }}" {{- with .ImportPath }} with package "{{ . }}" {{- end }} and a list of compilation errors, modify the code such that the errors are resolved.
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

			Code:             fileContent,
			FilePath:         ctx.FilePath,
			ImportPath:       importPath,
			HasTestsInSource: ctx.HasTestsInSource,
		},

		Template:      arguments.Template,
		TestFramework: arguments.TestFramework,
	}).Format()
	if err != nil {
		return nil, err
	}

	queryResult, err := m.query(ctx.Logger, request)
	if err != nil {
		return nil, pkgerrors.WithStack(err)
	}

	var filePath string
	if ctx.HasTestsInSource {
		filePath = filepath.Join(ctx.RepositoryPath, ctx.FilePath)
	} else {
		filePath = filepath.Join(ctx.RepositoryPath, ctx.Language.TestFilePath(ctx.RepositoryPath, ctx.FilePath))
	}

	return handleQueryResult(queryResult, filePath, ctx.HasTestsInSource)
}

func (m *Model) query(logger *log.Logger, request string) (queryResult *provider.QueryResult, err error) {
	var duration time.Duration
	id := uuid.NewString()
	if err := retry.Do(
		func() error {
			logger.Info("querying model", "model", m.ID(), "query-id", id, "prompt", string(bytesutil.PrefixLines([]byte(request), []byte("\t"))))
			ctx := context.Background()
			if m.apiRequestTimeout > 0 {
				c, cancel := context.WithTimeoutCause(ctx, time.Second*time.Duration(m.apiRequestTimeout), pkgerrors.Errorf("API request timed out (%d seconds)", m.apiRequestTimeout))
				defer cancel()
				ctx = c
			}

			start := time.Now()
			queryResult, err = m.provider.Query(ctx, logger, m, request)
			if err != nil {
				return err
			} else if ctx.Err() != nil {
				return context.Cause(ctx)
			}
			duration = time.Since(start)
			totalCosts := float64(-1)
			if queryResult.GenerationInfo != nil {
				totalCosts = queryResult.GenerationInfo.TotalCost
			}
			logger.Info("model responded", "model", m.ID(), "query-id", id, "duration", duration.Milliseconds(), "response-id", queryResult.ResponseID, "costs-total", totalCosts, "token-input", queryResult.Usage.PromptTokens, "token-output", queryResult.Usage.CompletionTokens, "response", string(bytesutil.PrefixLines([]byte(queryResult.Message), []byte("\t"))))

			return nil
		},
		retry.Attempts(m.apiRequestAttempts),
		retry.Delay(5*time.Second),
		retry.DelayType(retry.BackOffDelay),
		retry.LastErrorOnly(true),
		retry.OnRetry(func(n uint, err error) {
			logger.Info("API request attempt failed", "count", n+1, "total", m.apiRequestAttempts, "error", err)
		}),
	); err != nil {
		return nil, err
	}

	queryResult.Duration = duration

	return queryResult, nil
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

	queryResult, err := m.query(ctx.Logger, request)
	if err != nil {
		return nil, pkgerrors.WithStack(err)
	}

	return handleQueryResult(queryResult, filepath.Join(ctx.RepositoryPath, ctx.FilePath), false)
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

	queryResult, err := m.query(ctx.Logger, request)
	if err != nil {
		return nil, pkgerrors.WithStack(err)
	}

	return handleQueryResult(queryResult, filepath.Join(ctx.RepositoryPath, ctx.FilePath), false)
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

	queryResult, err := m.query(ctx.Logger, request)
	if err != nil {
		return nil, pkgerrors.WithStack(err)
	}

	return handleQueryResult(queryResult, filepath.Join(ctx.RepositoryPath, ctx.FilePath), false)
}

func handleQueryResult(queryResult *provider.QueryResult, filePathAbsolute string, appendFile bool) (assessment metrics.Assessments, err error) {
	assessment, sourceFileContent, err := prompt.ParseResponse(queryResult.Message)
	if err != nil {
		return nil, pkgerrors.WithStack(err)
	}
	assessment[metrics.AssessmentKeyProcessingTime] = float64(queryResult.Duration.Milliseconds())
	assessment[metrics.AssessmentKeyResponseCharacterCount] = float64(len(queryResult.Message))
	assessment[metrics.AssessmentKeyGenerateTestsForFileCharacterCount] = float64(len(sourceFileContent))
	assessment[metrics.AssessmentKeyTokenInput] = float64(queryResult.Usage.PromptTokens)
	assessment[metrics.AssessmentKeyTokenOutput] = float64(queryResult.Usage.CompletionTokens)
	if queryResult.GenerationInfo != nil {
		assessment[metrics.AssessmentKeyNativeTokenInput] = float64(queryResult.GenerationInfo.NativeTokensPrompt)
		assessment[metrics.AssessmentKeyNativeTokenOutput] = float64(queryResult.GenerationInfo.NativeTokensCompletion)
		assessment[metrics.AssessmentKeyCostsTokenActual] = queryResult.GenerationInfo.TotalCost
	}

	if sourceFileContent == "" {
		return assessment, nil
	}

	if err := os.MkdirAll(filepath.Dir(filePathAbsolute), 0755); err != nil {
		return nil, pkgerrors.WithStack(err)
	}

	flags := os.O_WRONLY | os.O_CREATE
	if appendFile {
		flags = flags | os.O_APPEND
	} else {
		flags = flags | os.O_TRUNC
	}
	file, err := os.OpenFile(filePathAbsolute, flags, 0644)
	if err != nil {
		return nil, pkgerrors.WithStack(err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			err = errors.Join(err, pkgerrors.WithStack(closeErr))
		}
	}()
	if _, err := file.WriteString(sourceFileContent); err != nil {
		return nil, pkgerrors.WithStack(err)
	}

	return assessment, nil
}

var _ model.ConfigureAPIRequestHandling = (*Model)(nil)

// SetAPIRequestAttempts sets the number of allowed API requests per LLM query.
func (m *Model) SetAPIRequestAttempts(queryAttempts uint) {
	m.apiRequestAttempts = queryAttempts
}

// SetAPIRequestTimeout sets the timeout for API requests in seconds.
func (m *Model) SetAPIRequestTimeout(timeout uint) {
	m.apiRequestTimeout = timeout
}
