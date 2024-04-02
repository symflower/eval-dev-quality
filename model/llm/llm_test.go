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

	providertesting "github.com/symflower/eval-symflower-codegen-testing/provider/testing"
)

func TestModelLLMGenerateTestsForFile(t *testing.T) {
	type testCase struct {
		Name string

		SetupMock func(mockedProvider *providertesting.MockQueryProvider)

		SourceFileContent string
		SourceFilePath    string
		ModelID           string

		ExpectedTestFileContent string
		ExpectedTestFilePath    string
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			tempDir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(tempDir, tc.SourceFilePath), []byte(bytesutil.StringTrimIndentations(tc.SourceFileContent)), 0644))

			mock := &providertesting.MockQueryProvider{}
			tc.SetupMock(mock)
			llm := NewLLMModel(mock, tc.ModelID)

			assert.NoError(t, llm.GenerateTestsForFile(tempDir, tc.SourceFilePath))

			actualTestFileContent, err := os.ReadFile(filepath.Join(tempDir, tc.ExpectedTestFilePath))
			assert.NoError(t, err)

			assert.Equal(t, strings.TrimSpace(bytesutil.StringTrimIndentations(tc.ExpectedTestFileContent)), string(actualTestFileContent))
		})
	}

	validate(t, &testCase{
		Name: "Simple",

		SetupMock: func(mockedProvider *providertesting.MockQueryProvider) {
			mockedProvider.On("Query", mock.Anything, "model-id",
				bytesutil.StringTrimIndentations(`
					Given the following Go code file, provide a test file for this code.
					The tests should produce 100 percent code coverage and must compile.
					The response must contain only the test code and nothing else.

					`+"```"+`
					func main() {}
					`+"```"+`
				`)).Return(bytesutil.StringTrimIndentations(`
					`+"```"+`
					func TestMain() {}
					`+"```"+`
				`), nil)
		},

		SourceFileContent: `
			func main() {}
		`,
		SourceFilePath: "simple.go",
		ModelID:        "model-id",

		ExpectedTestFileContent: `
			func TestMain() {}
		`,
		ExpectedTestFilePath: "simple_test.go",
	})
}
