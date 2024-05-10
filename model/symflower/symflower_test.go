package symflower

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zimmski/osutil"

	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	metricstesting "github.com/symflower/eval-dev-quality/evaluate/metrics/testing"
	"github.com/symflower/eval-dev-quality/language"
	"github.com/symflower/eval-dev-quality/language/golang"
	"github.com/symflower/eval-dev-quality/log"
)

func TestModelGenerateTestsForFile(t *testing.T) {
	type testCase struct {
		Name string

		Model    *Model
		Language language.Language

		RepositoryPath   string
		RepositoryChange func(t *testing.T, repositoryPath string)
		FilePath         string

		ExpectedAssessment metrics.Assessments
		ExpectedCoverage   float64
		ExpectedError      error
		ExpectedErrorText  string
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			logOutput, logger := log.Buffer()
			defer func() {
				if t.Failed() {
					t.Log(logOutput.String())
				}
			}()

			temporaryPath := t.TempDir()
			repositoryPath := filepath.Join(temporaryPath, filepath.Base(tc.RepositoryPath))
			require.NoError(t, osutil.CopyTree(tc.RepositoryPath, repositoryPath))

			if tc.RepositoryChange != nil {
				tc.RepositoryChange(t, repositoryPath)
			}

			if tc.Model == nil {
				tc.Model = &Model{}
			}
			actualAssessment, actualError := tc.Model.GenerateTestsForFile(logger, tc.Language, repositoryPath, tc.FilePath)

			if tc.ExpectedError != nil {
				assert.ErrorIs(t, tc.ExpectedError, actualError)
			} else if actualError != nil || tc.ExpectedErrorText != "" {
				assert.ErrorContains(t, actualError, tc.ExpectedErrorText)
			}
			metricstesting.AssertAssessmentsEqual(t, tc.ExpectedAssessment, actualAssessment)

			actualCoverage, err := tc.Language.Execute(logger, repositoryPath)
			require.NoError(t, err)
			assert.Equal(t, tc.ExpectedCoverage, actualCoverage)
		})
	}

	validate(t, &testCase{
		Name: "Go",

		Language: &golang.Language{},

		RepositoryPath: filepath.Join("..", "..", "testdata", "golang", "plain"),
		FilePath:       "plain.go",

		ExpectedAssessment: metrics.Assessments{
			metrics.AssessmentKeyResponseNoExcess: 1,
			metrics.AssessmentKeyResponseWithCode: 1,
		},
		ExpectedCoverage: 100,
	})
}
