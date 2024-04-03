package llm

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	pkgerrors "github.com/pkg/errors"
	"github.com/zimmski/osutil/bytesutil"

	"github.com/symflower/eval-symflower-codegen-testing/language"
	"github.com/symflower/eval-symflower-codegen-testing/model"
	"github.com/symflower/eval-symflower-codegen-testing/model/llm/prompt"
	"github.com/symflower/eval-symflower-codegen-testing/provider"
)

// llm represents a LLM model accessed via a provider.
type llm struct {
	// provider is the client to query the LLM model.
	provider provider.QueryProvider
	// model holds the identifier for the LLM model.
	model string
}

// NewLLMModel returns an LLM model corresponding to the given identifier which is queried via the given provider.
func NewLLMModel(provider provider.QueryProvider, modelIdentifier string) model.Model {
	return &llm{
		provider: provider,
		model:    modelIdentifier,
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
	Given the following {{ .Language.Name }} code file "{{ .FilePath }}" with package "{{ .ImportPath }}", provide a test file for this code.
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

var _ model.Model = (*llm)(nil)

// ID returns the unique ID of this model.
func (m *llm) ID() (id string) {
	return m.model
}

// GenerateTestsForFile generates test files for the given implementation file in a repository.
func (m *llm) GenerateTestsForFile(language language.Language, repositoryPath string, filePath string) (err error) {
	data, err := os.ReadFile(filepath.Join(repositoryPath, filePath))
	if err != nil {
		return err
	}
	fileContent := strings.TrimSpace(string(data))

	importPath := filepath.Join(filepath.Base(repositoryPath), filepath.Dir(filePath))

	request, err := llmGenerateTestForFilePrompt(&llmGenerateTestForFilePromptContext{
		Language: language,

		Code:       fileContent,
		FilePath:   filePath,
		ImportPath: importPath,
	})
	if err != nil {
		return err
	}

	response, err := m.provider.Query(context.Background(), m.model, request)
	if err != nil {
		return err
	}
	log.Printf("Model %q responded to query %s with: %s", m.ID(), string(bytesutil.PrefixLines([]byte(request), []byte("\t"))), string(bytesutil.PrefixLines([]byte(response), []byte("\t"))))

	testContent := prompt.ParseResponse(response)

	// TODO Ask the model for the test file name or compute it in a more sophisticated manner.
	fileExtension := filepath.Ext(filePath)
	testFilePath := filepath.Join(repositoryPath, strings.TrimSuffix(filePath, fileExtension)+"_test"+fileExtension)

	return os.WriteFile(testFilePath, []byte(testContent), 0644)
}
