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
	"github.com/symflower/eval-dev-quality/task"
	"github.com/symflower/eval-dev-quality/tools"
	toolstesting "github.com/symflower/eval-dev-quality/tools/testing"
)

func TestTaskWriteTestsRun(t *testing.T) {
	toolstesting.RequiresTool(t, tools.NewSymflower())

	validate := func(t *testing.T, tc *tasktesting.TestCaseTask) {
		t.Run(tc.Name, func(t *testing.T) {
			resultPath := t.TempDir()

			logOutput, logger := log.Buffer()
			defer func() {
				if t.Failed() {
					t.Logf("Logging output: %s", logOutput.String())
				}
			}()
			repository, cleanup, err := evaluatetask.TemporaryRepository(logger, tc.TestDataPath, tc.RepositoryPath)
			assert.NoError(t, err)
			defer cleanup()

			taskWriteTests, err := evaluatetask.TaskForIdentifier(evaluatetask.IdentifierWriteTests, logger, resultPath, tc.Model, tc.Language)
			require.NoError(t, err)

			tc.Validate(t, taskWriteTests, repository, resultPath)
		})
	}

	validate(t, &tasktesting.TestCaseTask{
		Name: "Plain",

		Model:          symflower.NewModel(),
		Language:       &golang.Language{},
		TestDataPath:   filepath.Join("..", "..", "..", "testdata"),
		RepositoryPath: filepath.Join("golang", "plain"),

		ExpectedRepositoryAssessment: map[task.Identifier]metrics.Assessments{
			evaluatetask.IdentifierWriteTests: metrics.Assessments{
				metrics.AssessmentKeyGenerateTestsForFileCharacterCount: 254,
				metrics.AssessmentKeyResponseCharacterCount:             254,
				metrics.AssessmentKeyCoverage:                           10,
				metrics.AssessmentKeyFilesExecuted:                      1,
				metrics.AssessmentKeyResponseNoError:                    1,
				metrics.AssessmentKeyResponseNoExcess:                   1,
				metrics.AssessmentKeyResponseWithCode:                   1,
			},
		},
		ExpectedResultFiles: map[string]func(t *testing.T, filePath string, data string){
			filepath.Join(string(evaluatetask.IdentifierWriteTests), "symflower_symbolic-execution", "golang", "golang", "plain.log"): func(t *testing.T, filePath, data string) {
				assert.Contains(t, data, "Evaluating model \"symflower/symbolic-execution\"")
				assert.Contains(t, data, "Generated 1 test")
				assert.Contains(t, data, "PASS: TestSymflowerPlain")
				assert.Contains(t, data, "Evaluated model \"symflower/symbolic-execution\"")
			},
		},
	})
}
