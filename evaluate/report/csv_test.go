package report

import (
	"strings"
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

func TestNewEvaluationFile(t *testing.T) {
	var file strings.Builder
	_, err := NewEvaluationFile(&file)
	require.NoError(t, err)

	actualEvaluationFileContent := file.String()
	require.NoError(t, err)

	expectedEvaluationFileContent := bytesutil.StringTrimIndentations(`
		model-id,language,repository,task,score,coverage,files-executed,files-executed-maximum-reachable,generate-tests-for-file-character-count,processing-time,response-character-count,response-no-error,response-no-excess,response-with-code
	`)

	assert.Equal(t, expectedEvaluationFileContent, string(actualEvaluationFileContent))
}

func TestWriteEvaluationRecord(t *testing.T) {
	type testCase struct {
		Name string

		Assessments map[task.Identifier]metrics.Assessments

		ExpectedCSV string
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			var file strings.Builder
			evaluationFile, err := NewEvaluationFile(&file)
			require.NoError(t, err)

			modelMock := modeltesting.NewMockModelNamed(t, "mocked-model")
			languageMock := languagetesting.NewMockLanguageNamed(t, "golang")

			err = evaluationFile.WriteEvaluationRecord(modelMock, languageMock, "golang/plain", tc.Assessments)
			require.NoError(t, err)

			assert.Equal(t, bytesutil.StringTrimIndentations(tc.ExpectedCSV), file.String())
		})
	}

	validate(t, &testCase{
		Name: "Single task with empty assessments",

		Assessments: map[task.Identifier]metrics.Assessments{
			evaluatetask.IdentifierWriteTests: metrics.NewAssessments(),
		},

		ExpectedCSV: `
			model-id,language,repository,task,score,coverage,files-executed,files-executed-maximum-reachable,generate-tests-for-file-character-count,processing-time,response-character-count,response-no-error,response-no-excess,response-with-code
			mocked-model,golang,golang/plain,write-tests,0,0,0,0,0,0,0,0,0,0
		`,
	})
	validate(t, &testCase{
		Name: "Multiple tasks with assessments",

		Assessments: map[task.Identifier]metrics.Assessments{
			evaluatetask.IdentifierWriteTests: metrics.Assessments{
				metrics.AssessmentKeyFilesExecuted:                 1,
				metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
				metrics.AssessmentKeyResponseNoError:               1,
				metrics.AssessmentKeyCoverage:                      0,
			},
			evaluatetask.IdentifierWriteTestsSymflowerFix: metrics.Assessments{
				metrics.AssessmentKeyFilesExecuted:                 1,
				metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
				metrics.AssessmentKeyResponseNoError:               1,
				metrics.AssessmentKeyCoverage:                      10,
			},
		},

		ExpectedCSV: `
			model-id,language,repository,task,score,coverage,files-executed,files-executed-maximum-reachable,generate-tests-for-file-character-count,processing-time,response-character-count,response-no-error,response-no-excess,response-with-code
			mocked-model,golang,golang/plain,write-tests,2,0,1,1,0,0,0,1,0,0
			mocked-model,golang,golang/plain,write-tests-symflower-fix,12,10,1,1,0,0,0,1,0,0
		`,
	})
}
