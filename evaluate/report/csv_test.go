package report

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zimmski/osutil"
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

func TestRecordsFromEvaluationCSVFiles(t *testing.T) {
	type testCase struct {
		Name string

		Before func(workingDirectory string)

		EvaluationCSVFilePaths []string

		ExpectedRecords [][]string
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			temporaryPath := t.TempDir()

			if tc.Before != nil {
				tc.Before(temporaryPath)
			}

			for i, evaluationCSVFilePath := range tc.EvaluationCSVFilePaths {
				tc.EvaluationCSVFilePaths[i] = filepath.Join(temporaryPath, evaluationCSVFilePath)
			}

			actualRows, actualErr := RecordsFromEvaluationCSVFiles(tc.EvaluationCSVFilePaths)
			require.NoError(t, actualErr)

			assert.Equal(t, tc.ExpectedRecords, actualRows)
		})
	}

	validate(t, &testCase{
		Name: "Only header exists",

		Before: func(workingDirectory string) {
			header := `model-id,language,repository,task,score,coverage,files-executed,files-executed-maximum-reachable,generate-tests-for-file-character-count,processing-time,response-character-count,response-no-error,response-no-excess,response-with-code`
			require.NoError(t, os.WriteFile(filepath.Join(workingDirectory, "evaluation.csv"), []byte(header), 0700))
		},

		EvaluationCSVFilePaths: []string{
			"evaluation.csv",
		},

		ExpectedRecords: nil,
	})
	validate(t, &testCase{
		Name: "Single file",

		Before: func(workingDirectory string) {
			content := bytesutil.StringTrimIndentations(`
				model-id,language,repository,task,score,coverage,files-executed,files-executed-maximum-reachable,generate-tests-for-file-character-count,processing-time,response-character-count,response-no-error,response-no-excess,response-with-code
				openrouter/anthropic/claude-2.0,golang,golang/light,write-tests,1,1,1,1,1,1,1,1,1,1
			`)
			require.NoError(t, os.WriteFile(filepath.Join(workingDirectory, "evaluation.csv"), []byte(content), 0700))
		},

		EvaluationCSVFilePaths: []string{
			"evaluation.csv",
		},

		ExpectedRecords: [][]string{
			[]string{
				"openrouter/anthropic/claude-2.0", "golang", "golang/light", "write-tests", "1", "1", "1", "1", "1", "1", "1", "1", "1", "1",
			},
		},
	})
	validate(t, &testCase{
		Name: "Multiple files",

		Before: func(workingDirectory string) {
			modelA := filepath.Join(workingDirectory, "modelA")
			modelAFileContent := bytesutil.StringTrimIndentations(`
				model-id,language,repository,task,score,coverage,files-executed,files-executed-maximum-reachable,generate-tests-for-file-character-count,processing-time,response-character-count,response-no-error,response-no-excess,response-with-code
				modelA,golang,golang/light,write-tests,1,1,1,1,1,1,1,1,1,1
				modelA,golang,golang/plain,write-tests,2,2,2,2,2,2,2,2,2,2
			`)
			require.NoError(t, osutil.MkdirAll(modelA))
			require.NoError(t, os.WriteFile(filepath.Join(modelA, "evaluation.csv"), []byte(modelAFileContent), 0700))

			modelB := filepath.Join(workingDirectory, "modelB")
			modelBFileContent := bytesutil.StringTrimIndentations(`
				model-id,language,repository,task,score,coverage,files-executed,files-executed-maximum-reachable,generate-tests-for-file-character-count,processing-time,response-character-count,response-no-error,response-no-excess,response-with-code
				modelB,java,java/light,write-tests,3,3,3,3,3,3,3,3,3,3
				modelB,java,java/plain,write-tests,4,4,4,4,4,4,4,4,4,4
			`)
			require.NoError(t, osutil.MkdirAll(modelB))
			require.NoError(t, os.WriteFile(filepath.Join(modelB, "evaluation.csv"), []byte(modelBFileContent), 0700))
		},

		EvaluationCSVFilePaths: []string{
			filepath.Join("modelA", "evaluation.csv"),
			filepath.Join("modelB", "evaluation.csv"),
		},

		ExpectedRecords: [][]string{
			[]string{"modelA", "golang", "golang/light", "write-tests", "1", "1", "1", "1", "1", "1", "1", "1", "1", "1"},
			[]string{"modelA", "golang", "golang/plain", "write-tests", "2", "2", "2", "2", "2", "2", "2", "2", "2", "2"},
			[]string{"modelB", "java", "java/light", "write-tests", "3", "3", "3", "3", "3", "3", "3", "3", "3", "3"},
			[]string{"modelB", "java", "java/plain", "write-tests", "4", "4", "4", "4", "4", "4", "4", "4", "4", "4"},
		},
	})
}

func TestEvaluationFileWriteLines(t *testing.T) {
	type testCase struct {
		Name string

		RawRecords [][]string

		ExpectedEvaluationFile string
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			var file strings.Builder
			evaluationFile, err := NewEvaluationFile(&file)
			require.NoError(t, err)

			actualErr := evaluationFile.WriteLines(tc.RawRecords)
			require.NoError(t, actualErr)

			assert.Equal(t, bytesutil.StringTrimIndentations(tc.ExpectedEvaluationFile), file.String())
		})
	}

	validate(t, &testCase{
		Name: "No records",

		ExpectedEvaluationFile: `
			model-id,language,repository,task,score,coverage,files-executed,files-executed-maximum-reachable,generate-tests-for-file-character-count,processing-time,response-character-count,response-no-error,response-no-excess,response-with-code
		`,
	})
	validate(t, &testCase{
		Name: "Single record",

		RawRecords: [][]string{
			[]string{"modelA", "golang", "golang/light", "write-tests", "1", "1", "1", "1", "1", "1", "1", "1", "1", "1"},
		},

		ExpectedEvaluationFile: `
			model-id,language,repository,task,score,coverage,files-executed,files-executed-maximum-reachable,generate-tests-for-file-character-count,processing-time,response-character-count,response-no-error,response-no-excess,response-with-code
			modelA,golang,golang/light,write-tests,1,1,1,1,1,1,1,1,1,1
		`,
	})
	validate(t, &testCase{
		Name: "Multiple records",

		RawRecords: [][]string{
			[]string{"modelA", "golang", "golang/light", "write-tests", "1", "1", "1", "1", "1", "1", "1", "1", "1", "1"},
			[]string{"modelA", "golang", "golang/plain", "write-tests", "2", "2", "2", "2", "2", "2", "2", "2", "2", "2"},
			[]string{"modelA", "java", "java/light", "write-tests", "3", "3", "3", "3", "3", "3", "3", "3", "3", "3"},
			[]string{"modelA", "java", "java/plain", "write-tests", "4", "4", "4", "4", "4", "4", "4", "4", "4", "4"},
		},

		ExpectedEvaluationFile: `
			model-id,language,repository,task,score,coverage,files-executed,files-executed-maximum-reachable,generate-tests-for-file-character-count,processing-time,response-character-count,response-no-error,response-no-excess,response-with-code
			modelA,golang,golang/light,write-tests,1,1,1,1,1,1,1,1,1,1
			modelA,golang,golang/plain,write-tests,2,2,2,2,2,2,2,2,2,2
			modelA,java,java/light,write-tests,3,3,3,3,3,3,3,3,3,3
			modelA,java,java/plain,write-tests,4,4,4,4,4,4,4,4,4,4
		`,
	})
}

func TestSortEvaluationRecords(t *testing.T) {
	type testCase struct {
		Name string

		Records [][]string

		ExpectedRecords [][]string
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			SortEvaluationRecords(tc.Records)

			assert.Equal(t, tc.ExpectedRecords, tc.Records)
		})
	}

	validate(t, &testCase{
		Name: "Single record",

		Records: [][]string{
			[]string{"modelA", "golang", "golang/light", "write-tests", "1", "1", "1", "1", "1", "1", "1", "1", "1", "1"},
		},

		ExpectedRecords: [][]string{
			[]string{"modelA", "golang", "golang/light", "write-tests", "1", "1", "1", "1", "1", "1", "1", "1", "1", "1"},
		},
	})
	validate(t, &testCase{
		Name: "Multiple records",

		Records: [][]string{
			[]string{"modelD", "languageB", "repositoryA", "taskA", "7", "7", "7", "7", "7", "7", "7", "7", "7", "7"},
			[]string{"modelD", "languageA", "repositoryA", "taskB", "6", "6", "6", "6", "6", "6", "6", "6", "6", "6"},
			[]string{"modelC", "languageA", "repositoryB", "taskB", "5", "5", "5", "5", "5", "5", "5", "5", "5", "5"},
			[]string{"modelC", "languageA", "repositoryB", "taskA", "4", "4", "4", "4", "4", "4", "4", "4", "4", "4"},
			[]string{"modelC", "languageA", "repositoryA", "taskA", "3", "3", "3", "3", "3", "3", "3", "3", "3", "3"},
			[]string{"modelB", "languageA", "repositoryA", "taskA", "2", "2", "2", "2", "2", "2", "2", "2", "2", "2"},
			[]string{"modelA", "languageA", "repositoryA", "taskA", "1", "1", "1", "1", "1", "1", "1", "1", "1", "1"},
		},

		ExpectedRecords: [][]string{
			[]string{"modelA", "languageA", "repositoryA", "taskA", "1", "1", "1", "1", "1", "1", "1", "1", "1", "1"},
			[]string{"modelB", "languageA", "repositoryA", "taskA", "2", "2", "2", "2", "2", "2", "2", "2", "2", "2"},
			[]string{"modelC", "languageA", "repositoryA", "taskA", "3", "3", "3", "3", "3", "3", "3", "3", "3", "3"},
			[]string{"modelC", "languageA", "repositoryB", "taskA", "4", "4", "4", "4", "4", "4", "4", "4", "4", "4"},
			[]string{"modelC", "languageA", "repositoryB", "taskB", "5", "5", "5", "5", "5", "5", "5", "5", "5", "5"},
			[]string{"modelD", "languageA", "repositoryA", "taskB", "6", "6", "6", "6", "6", "6", "6", "6", "6", "6"},
			[]string{"modelD", "languageB", "repositoryA", "taskA", "7", "7", "7", "7", "7", "7", "7", "7", "7", "7"},
		},
	})
}
