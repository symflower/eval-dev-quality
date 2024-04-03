package llm

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/zimmski/osutil/bytesutil"

	"github.com/symflower/eval-symflower-codegen-testing/model"
	"github.com/symflower/eval-symflower-codegen-testing/model/prompt"
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

// llmGenerateTestForFilePrompt is the prompt used to query LLMs for test generation.
var llmGenerateTestForFilePrompt = bytesutil.StringTrimIndentations(`
	Given the following Go code file, provide a test file for this code.
	The tests should produce 100 percent code coverage and must compile.
	The response must contain only the test code and nothing else.
`)

// llmGenerateTestForFilePromptContext is the context for for the prompt template.
type llmGenerateTestForFilePromptContext struct {
	Prompt string
	Code   string
}

// llmGenerateTestForFilePromptData is the template for generating an LLM test generation prompt.
var llmGenerateTestForFilePromptData = template.Must(template.New("templateGenerateTestPrompt").Parse(bytesutil.StringTrimIndentations(`
	{{ .Prompt }}
	` + "```" + `
	{{ .Code }}
	` + "```" + `
`)))

var _ model.Model = (*llm)(nil)

// ID returns the unique ID of this model.
func (m *llm) ID() (id string) {
	return m.model
}

// GenerateTestsForFile generates test files for the given implementation file in a repository.
func (m *llm) GenerateTestsForFile(repositoryPath string, filePath string) (err error) {
	data, err := os.ReadFile(filepath.Join(repositoryPath, filePath))
	if err != nil {
		return err
	}
	fileContent := strings.TrimSpace(string(data))

	var promptBuilder strings.Builder
	if err = llmGenerateTestForFilePromptData.Execute(&promptBuilder, llmGenerateTestForFilePromptContext{
		Prompt: llmGenerateTestForFilePrompt,
		Code:   fileContent,
	}); err != nil {
		return err
	}

	response, err := m.provider.Query(context.Background(), m.model, promptBuilder.String())
	if err != nil {
		return err
	}
	log.Printf("Model %q responded to query %q with: %q", m.ID(), promptBuilder.String(), response)

	testContent := prompt.ParseResponse(response)

	// TODO Ask the model for the test file name or compute it in a more sophisticated manner.
	fileExtension := filepath.Ext(filePath)
	testFilePath := filepath.Join(repositoryPath, strings.TrimSuffix(filePath, fileExtension)+"_test"+fileExtension)

	return os.WriteFile(testFilePath, []byte(testContent), 0644)
}
