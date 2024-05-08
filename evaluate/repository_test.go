package evaluate

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zimmski/osutil"

	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	metricstesting "github.com/symflower/eval-dev-quality/evaluate/metrics/testing"
	"github.com/symflower/eval-dev-quality/language"
	"github.com/symflower/eval-dev-quality/language/golang"
	"github.com/symflower/eval-dev-quality/log"
	"github.com/symflower/eval-dev-quality/model"
	"github.com/symflower/eval-dev-quality/model/symflower"
)

func TestRepository(t *testing.T) {
	type testCase struct {
		Name string

		Model          model.Model
		Language       language.Language
		TestDataPath   string
		RepositoryPath string

		ExpectedRepositoryAssessment metrics.Assessments
		ExpectedResultFiles          map[string]func(t *testing.T, filePath string, data string)
		ExpectedProblems             []error
		ExpectedError                error
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			temporaryPath := t.TempDir()

			_, logger := log.Buffer()
			actualRepositoryAssessment, actualProblems, actualErr := Repository(logger, temporaryPath, tc.Model, tc.Language, tc.TestDataPath, tc.RepositoryPath)

			metricstesting.AssertAssessmentsEqual(t, tc.ExpectedRepositoryAssessment, actualRepositoryAssessment)
			assert.Equal(t, tc.ExpectedProblems, actualProblems)
			assert.Equal(t, tc.ExpectedError, actualErr)

			actualResultFiles, err := osutil.FilesRecursive(temporaryPath)
			require.NoError(t, err)
			for i, p := range actualResultFiles {
				actualResultFiles[i], err = filepath.Rel(temporaryPath, p)
				require.NoError(t, err)
			}
			sort.Strings(actualResultFiles)
			expectedResultFiles := make([]string, 0, len(tc.ExpectedResultFiles))
			for filePath, validate := range tc.ExpectedResultFiles {
				expectedResultFiles = append(expectedResultFiles, filePath)

				if validate != nil {
					data, err := os.ReadFile(filepath.Join(temporaryPath, filePath))
					if assert.NoError(t, err) {
						validate(t, filePath, string(data))
					}
				}
			}
			sort.Strings(expectedResultFiles)
			assert.Equal(t, expectedResultFiles, actualResultFiles)
		})
	}

	validate(t, &testCase{
		Name: "Plain",

		Model:          symflower.NewModel(),
		Language:       &golang.Language{},
		TestDataPath:   filepath.Join("..", "testdata"),
		RepositoryPath: filepath.Join("golang", "plain"),

		ExpectedRepositoryAssessment: metrics.Assessments{
			metrics.AssessmentKeyCoverageStatement: 10,
			metrics.AssessmentKeyFilesExecuted:     1,
			metrics.AssessmentKeyResponseNoError:   1,
			metrics.AssessmentKeyResponseNoExcess:  1,
			metrics.AssessmentKeyResponseWithCode:  1,
		},
		ExpectedResultFiles: map[string]func(t *testing.T, filePath string, data string){
			filepath.Join("symflower_symbolic-execution", "golang", "golang", "plain.log"): func(t *testing.T, filePath, data string) {
				assert.Contains(t, data, "Evaluating model \"symflower/symbolic-execution\"")
				assert.Contains(t, data, "Generated 1 test")
				assert.Contains(t, data, "PASS: TestSymflowerPlain")
				assert.Contains(t, data, "Evaluated model \"symflower/symbolic-execution\"")
			},
		},
	})
}
