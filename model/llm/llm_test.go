package llm

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/zimmski/osutil/bytesutil"

	"github.com/symflower/eval-symflower-codegen-testing/language"
	providertesting "github.com/symflower/eval-symflower-codegen-testing/provider/testing"
)

func TestModelLLMGenerateTestsForFile(t *testing.T) {
	type testCase struct {
		Name string

		SetupMock func(mockedProvider *providertesting.MockQueryProvider)

		Language          language.Language
		ModelID           string
		SourceFileContent string
		SourceFilePath    string

		ExpectedTestFileContent string
		ExpectedTestFilePath    string
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			temporaryPath := t.TempDir()
			temporaryPath = filepath.Join(temporaryPath, "native")
			require.NoError(t, os.Mkdir(temporaryPath, 0755))

			require.NoError(t, os.WriteFile(filepath.Join(temporaryPath, tc.SourceFilePath), []byte(bytesutil.StringTrimIndentations(tc.SourceFileContent)), 0644))

			mock := &providertesting.MockQueryProvider{}
			tc.SetupMock(mock)
			llm := NewLLMModel(mock, tc.ModelID)

			assert.NoError(t, llm.GenerateTestsForFile(tc.Language, temporaryPath, tc.SourceFilePath))

			actualTestFileContent, err := os.ReadFile(filepath.Join(temporaryPath, tc.ExpectedTestFilePath))
			assert.NoError(t, err)

			assert.Equal(t, strings.TrimSpace(bytesutil.StringTrimIndentations(tc.ExpectedTestFileContent)), string(actualTestFileContent))
		})
	}

	sourceFileContent := `
		package native

		func main() {}
	`
	sourceFilePath := "simple.go"
	promptMessage, err := llmGenerateTestForFilePrompt(&llmGenerateTestForFilePromptContext{
		Language: &language.LanguageGolang{},

		Code:       bytesutil.StringTrimIndentations(sourceFileContent),
		FilePath:   sourceFilePath,
		ImportPath: "native",
	})
	require.NoError(t, err)
	validate(t, &testCase{
		Name: "Simple",

		SetupMock: func(mockedProvider *providertesting.MockQueryProvider) {
			mockedProvider.On("Query", mock.Anything, "model-id", promptMessage).Return(bytesutil.StringTrimIndentations(`
					`+"```"+`
					package native

					func TestMain() {}
					`+"```"+`
				`), nil)
		},

		Language:          &language.LanguageGolang{},
		ModelID:           "model-id",
		SourceFileContent: sourceFileContent,
		SourceFilePath:    sourceFilePath,

		ExpectedTestFileContent: `
			package native

			func TestMain() {}
		`,
		ExpectedTestFilePath: "simple_test.go",
	})
}
