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

	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	metricstesting "github.com/symflower/eval-dev-quality/evaluate/metrics/testing"
	"github.com/symflower/eval-dev-quality/language"
	"github.com/symflower/eval-dev-quality/language/golang"
	"github.com/symflower/eval-dev-quality/log"
	providertesting "github.com/symflower/eval-dev-quality/provider/testing"
)

func TestModelGenerateTestsForFile(t *testing.T) {
	type testCase struct {
		Name string

		SetupMock func(mockedProvider *providertesting.MockQueryProvider)

		Language          language.Language
		ModelID           string
		SourceFileContent string
		SourceFilePath    string

		ExpectedAssessment      metrics.Assessments
		ExpectedTestFileContent string
		ExpectedTestFilePath    string
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			log, logger := log.Buffer()
			defer func() {
				if t.Failed() {
					t.Log(log.String())
				}
			}()

			temporaryPath := t.TempDir()
			temporaryPath = filepath.Join(temporaryPath, "native")
			require.NoError(t, os.Mkdir(temporaryPath, 0755))

			require.NoError(t, os.WriteFile(filepath.Join(temporaryPath, tc.SourceFilePath), []byte(bytesutil.StringTrimIndentations(tc.SourceFileContent)), 0644))

			mock := &providertesting.MockQueryProvider{}
			tc.SetupMock(mock)
			llm := NewModel(mock, tc.ModelID)

			actualAssessment, actualError := llm.GenerateTestsForFile(logger, tc.Language, temporaryPath, tc.SourceFilePath)
			assert.NoError(t, actualError)
			metricstesting.AssertAssessmentsEqual(t, tc.ExpectedAssessment, actualAssessment)

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
		Language: &golang.Language{},

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

		Language:          &golang.Language{},
		ModelID:           "model-id",
		SourceFileContent: sourceFileContent,
		SourceFilePath:    sourceFilePath,

		ExpectedAssessment: metrics.Assessments{
			metrics.AssessmentKeyResponseNoExcess: 1,
			metrics.AssessmentKeyResponseNotEmpty: 1,
			metrics.AssessmentKeyResponseWithCode: 1,
		},
		ExpectedTestFileContent: `
			package native

			func TestMain() {}
		`,
		ExpectedTestFilePath: "simple_test.go",
	})
}
