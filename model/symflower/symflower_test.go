package symflower

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zimmski/osutil"

	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	"github.com/symflower/eval-dev-quality/language"
)

func TestModelSymflowerGenerateTestsForFile(t *testing.T) {
	type testCase struct {
		Name string

		ModelSymflower *ModelSymflower
		Language       language.Language

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
			temporaryPath := t.TempDir()
			repositoryPath := filepath.Join(temporaryPath, filepath.Base(tc.RepositoryPath))
			require.NoError(t, osutil.CopyTree(tc.RepositoryPath, repositoryPath))

			if tc.RepositoryChange != nil {
				tc.RepositoryChange(t, repositoryPath)
			}

			if tc.ModelSymflower == nil {
				tc.ModelSymflower = &ModelSymflower{}
			}
			actualAssessment, actualError := tc.ModelSymflower.GenerateTestsForFile(tc.Language, repositoryPath, tc.FilePath)

			if tc.ExpectedError != nil {
				assert.ErrorIs(t, tc.ExpectedError, actualError)
			} else if actualError != nil || tc.ExpectedErrorText != "" {
				assert.ErrorContains(t, actualError, tc.ExpectedErrorText)
			}
			assert.Equal(t, tc.ExpectedAssessment, actualAssessment)

			actualCoverage, err := tc.Language.Execute(repositoryPath)
			require.NoError(t, err)
			assert.Equal(t, tc.ExpectedCoverage, actualCoverage)
		})
	}

	validate(t, &testCase{
		Name: "Go",

		Language: &language.LanguageGolang{},

		RepositoryPath: "../../testdata/golang/plain/",
		FilePath:       "plain.go",

		ExpectedAssessment: metrics.Assessments{
			metrics.AssessmentKeyNoExcessResponse: 1,
		},
		ExpectedCoverage: 100,
	})
}
