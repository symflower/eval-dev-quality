package report

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zimmski/osutil/bytesutil"

	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	evaluatetask "github.com/symflower/eval-dev-quality/evaluate/task"
	languagetesting "github.com/symflower/eval-dev-quality/language/testing"
	modeltesting "github.com/symflower/eval-dev-quality/model/testing"
	"github.com/symflower/eval-dev-quality/task"
)

func TestGenerateCSVForAssessmentPerModel(t *testing.T) {
	type testCase struct {
		Name string

		Assessments AssessmentPerModel

		ExpectedString string
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			actualString, err := GenerateCSV(tc.Assessments)
			assert.NoError(t, err)

			assert.Equal(t, bytesutil.StringTrimIndentations(tc.ExpectedString), actualString)
		})
	}

	validate(t, &testCase{
		Name: "Single Empty Model",

		Assessments: AssessmentPerModel{
			modeltesting.NewMockModelNamedWithCosts(t, "some-model", "Some Model", 0): {},
		},

		ExpectedString: `
			model-id,model-name,cost,score,coverage,files-executed,generate-tests-for-file-character-count,processing-time,response-character-count,response-no-error,response-no-excess,response-with-code
			some-model,Some Model,0,0,0,0,0,0,0,0,0,0
		`,
	})
	validate(t, &testCase{
		Name: "Multiple Models",

		Assessments: AssessmentPerModel{
			modeltesting.NewMockModelNamedWithCosts(t, "some-model-a", "Some Model A", 0.0001): {
				metrics.AssessmentKeyGenerateTestsForFileCharacterCount: 50,
				metrics.AssessmentKeyResponseCharacterCount:             100,
				metrics.AssessmentKeyCoverage:                           1,
				metrics.AssessmentKeyFilesExecuted:                      2,
				metrics.AssessmentKeyResponseNoError:                    3,
				metrics.AssessmentKeyResponseNoExcess:                   4,
				metrics.AssessmentKeyResponseWithCode:                   5,
				metrics.AssessmentKeyProcessingTime:                     200,
			},
			modeltesting.NewMockModelNamedWithCosts(t, "some-model-b", "Some Model B", 0.0005): {
				metrics.AssessmentKeyGenerateTestsForFileCharacterCount: 100,
				metrics.AssessmentKeyResponseCharacterCount:             200,
				metrics.AssessmentKeyCoverage:                           1,
				metrics.AssessmentKeyFilesExecuted:                      2,
				metrics.AssessmentKeyResponseNoError:                    3,
				metrics.AssessmentKeyResponseNoExcess:                   4,
				metrics.AssessmentKeyResponseWithCode:                   5,
				metrics.AssessmentKeyProcessingTime:                     300,
			},
		},

		ExpectedString: `
			model-id,model-name,cost,score,coverage,files-executed,generate-tests-for-file-character-count,processing-time,response-character-count,response-no-error,response-no-excess,response-with-code
			some-model-a,Some Model A,0.0001,15,1,2,50,200,100,3,4,5
			some-model-b,Some Model B,0.0005,15,1,2,100,300,200,3,4,5
		`,
	})
}

func TestWriteEvaluationHeader(t *testing.T) {
	resultPath := t.TempDir()

	WriteEvaluationHeader(resultPath)

	_, err := os.Stat(filepath.Join(resultPath, "evaluation.csv"))
	require.NoError(t, err)

	actualHeader, err := os.ReadFile(filepath.Join(resultPath, "evaluation.csv"))
	require.NoError(t, err)

	expectedHeader := bytesutil.StringTrimIndentations(`
		model-id,model-name,cost,language,repository,task,score,coverage,files-executed,generate-tests-for-file-character-count,processing-time,response-character-count,response-no-error,response-no-excess,response-with-code
	`)

	assert.Equal(t, expectedHeader, string(actualHeader))
}

func TestWriteEvaluationRecord(t *testing.T) {
	type testCase struct {
		Name string

		Before func(resultPath string)

		Assessments map[task.Identifier]metrics.Assessments

		ExpectedCSV string
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			resultPath := t.TempDir()
			evaluationCSVFilePath := filepath.Join(resultPath, "evaluation.csv")

			modelMock := modeltesting.NewMockModelNamedWithCosts(t, "mocked-model", "Mocked Model", 0.0001)
			languageMock := languagetesting.NewMockLanguageNamed(t, "golang")

			if tc.Before != nil {
				tc.Before(resultPath)
			}

			err := WriteEvaluationRecord(resultPath, modelMock, languageMock, "golang/plain", tc.Assessments)
			require.NoError(t, err)

			_, err = os.Stat(evaluationCSVFilePath)
			require.NoError(t, err)

			actualCSV, err := os.ReadFile(evaluationCSVFilePath)
			require.NoError(t, err)

			assert.Equal(t, bytesutil.StringTrimIndentations(tc.ExpectedCSV), string(actualCSV))
		})
	}

	validate(t, &testCase{
		Name: "Evaluation file does not exist",

		Assessments: map[task.Identifier]metrics.Assessments{
			evaluatetask.IdentifierWriteTests: metrics.Assessments{
				metrics.AssessmentKeyFilesExecuted:   1,
				metrics.AssessmentKeyResponseNoError: 1,
				metrics.AssessmentKeyCoverage:        10,
			},
			evaluatetask.IdentifierWriteTestsSymflowerFix: metrics.Assessments{
				metrics.AssessmentKeyFilesExecuted:   1,
				metrics.AssessmentKeyResponseNoError: 1,
				metrics.AssessmentKeyCoverage:        10,
			},
		},

		ExpectedCSV: `
			mocked-model,Mocked Model,0.0001,golang,golang/plain,write-tests,12,10,1,0,0,0,1,0,0
			mocked-model,Mocked Model,0.0001,golang,golang/plain,write-tests-symflower-fix,12,10,1,0,0,0,1,0,0
		`,
	})
	validate(t, &testCase{
		Name: "Single task with empty assessments",

		Before: func(resultPath string) {
			WriteEvaluationHeader(resultPath)
		},

		Assessments: map[task.Identifier]metrics.Assessments{
			evaluatetask.IdentifierWriteTests: metrics.NewAssessments(),
		},

		ExpectedCSV: `
			model-id,model-name,cost,language,repository,task,score,coverage,files-executed,generate-tests-for-file-character-count,processing-time,response-character-count,response-no-error,response-no-excess,response-with-code
			mocked-model,Mocked Model,0.0001,golang,golang/plain,write-tests,0,0,0,0,0,0,0,0,0
		`,
	})
	validate(t, &testCase{
		Name: "Multiple tasks with assessments",

		Before: func(resultPath string) {
			WriteEvaluationHeader(resultPath)
		},

		Assessments: map[task.Identifier]metrics.Assessments{
			evaluatetask.IdentifierWriteTests: metrics.Assessments{
				metrics.AssessmentKeyFilesExecuted:   1,
				metrics.AssessmentKeyResponseNoError: 1,
				metrics.AssessmentKeyCoverage:        0,
			},
			evaluatetask.IdentifierWriteTestsSymflowerFix: metrics.Assessments{
				metrics.AssessmentKeyFilesExecuted:   1,
				metrics.AssessmentKeyResponseNoError: 1,
				metrics.AssessmentKeyCoverage:        10,
			},
		},

		ExpectedCSV: `
			model-id,model-name,cost,language,repository,task,score,coverage,files-executed,generate-tests-for-file-character-count,processing-time,response-character-count,response-no-error,response-no-excess,response-with-code
			mocked-model,Mocked Model,0.0001,golang,golang/plain,write-tests,2,0,1,0,0,0,1,0,0
			mocked-model,Mocked Model,0.0001,golang,golang/plain,write-tests-symflower-fix,12,10,1,0,0,0,1,0,0
		`,
	})
}
