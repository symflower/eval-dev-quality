package testintegration

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	evaluatetask "github.com/symflower/eval-dev-quality/evaluate/task"
	tasktesting "github.com/symflower/eval-dev-quality/evaluate/task/testing"
	"github.com/symflower/eval-dev-quality/language/golang"
	"github.com/symflower/eval-dev-quality/log"
	"github.com/symflower/eval-dev-quality/model/symflower"
	evaltask "github.com/symflower/eval-dev-quality/task"
	"github.com/symflower/eval-dev-quality/tools"
	toolstesting "github.com/symflower/eval-dev-quality/tools/testing"
)

func TestWriteTestsRun(t *testing.T) {
	toolstesting.RequiresTool(t, tools.NewSymflower())

	validate := func(t *testing.T, tc *tasktesting.TestCaseTask) {
		t.Run(tc.Name, func(t *testing.T) {
			task, err := evaluatetask.ForIdentifier(evaluatetask.IdentifierWriteTests)
			require.NoError(t, err)
			tc.Task = task

			tc.Validate(t,
				func(logger *log.Logger, testDataPath string, repositoryPathRelative string) (repository evaltask.Repository, cleanup func(), err error) {
					return evaluatetask.TemporaryRepository(logger, testDataPath, repositoryPathRelative)
				},
			)
		})
	}

	validate(t, &tasktesting.TestCaseTask{
		Name: "Plain",

		Model:          symflower.NewModelSymbolicExecution(),
		Language:       &golang.Language{},
		TestDataPath:   filepath.Join("..", "..", "..", "testdata"),
		RepositoryPath: filepath.Join("golang", "plain"),

		ExpectedRepositoryAssessment: map[string]map[evaltask.Identifier]metrics.Assessments{
			"plain.go": {
				evaluatetask.IdentifierWriteTests: metrics.Assessments{
					metrics.AssessmentKeyGenerateTestsForFileCharacterCount: 254,
					metrics.AssessmentKeyResponseCharacterCount:             254,
					metrics.AssessmentKeyCoverage:                           1,
					metrics.AssessmentKeyFilesExecuted:                      1,
					metrics.AssessmentKeyFilesExecutedMaximumReachable:      1,
					metrics.AssessmentKeyResponseNoError:                    1,
					metrics.AssessmentKeyResponseNoExcess:                   1,
					metrics.AssessmentKeyResponseWithCode:                   1,
				},
				evaluatetask.IdentifierWriteTestsSymflowerFix: metrics.Assessments{
					metrics.AssessmentKeyGenerateTestsForFileCharacterCount: 254,
					metrics.AssessmentKeyResponseCharacterCount:             254,
					metrics.AssessmentKeyCoverage:                           1,
					metrics.AssessmentKeyFilesExecuted:                      1,
					metrics.AssessmentKeyFilesExecutedMaximumReachable:      1,
					metrics.AssessmentKeyResponseNoError:                    1,
					metrics.AssessmentKeyResponseNoExcess:                   1,
					metrics.AssessmentKeyResponseWithCode:                   1,
				},
				evaluatetask.IdentifierWriteTestsSymflowerTemplate: metrics.Assessments{
					metrics.AssessmentKeyGenerateTestsForFileCharacterCount: 254,
					metrics.AssessmentKeyResponseCharacterCount:             254,
					metrics.AssessmentKeyCoverage:                           1,
					metrics.AssessmentKeyFilesExecuted:                      1,
					metrics.AssessmentKeyFilesExecutedMaximumReachable:      1,
					metrics.AssessmentKeyResponseNoError:                    1,
					metrics.AssessmentKeyResponseNoExcess:                   1,
					metrics.AssessmentKeyResponseWithCode:                   1,
				},
				evaluatetask.IdentifierWriteTestsSymflowerTemplateSymflowerFix: metrics.Assessments{
					metrics.AssessmentKeyGenerateTestsForFileCharacterCount: 254,
					metrics.AssessmentKeyResponseCharacterCount:             254,
					metrics.AssessmentKeyCoverage:                           1,
					metrics.AssessmentKeyFilesExecuted:                      1,
					metrics.AssessmentKeyFilesExecutedMaximumReachable:      1,
					metrics.AssessmentKeyResponseNoError:                    1,
					metrics.AssessmentKeyResponseNoExcess:                   1,
					metrics.AssessmentKeyResponseWithCode:                   1,
				},
			},
		},
		ValidateLog: func(t *testing.T, data string) {
			assert.Contains(t, data, "msg=\"evaluating model\" model=symflower/symbolic-execution")
			assert.Contains(t, data, "Generated 1 test")
			assert.Contains(t, data, "PASS: TestSymflowerPlain")
			assert.Contains(t, data, "msg=\"evaluated model\" model=symflower/symbolic-execution")
		},
	})
}
