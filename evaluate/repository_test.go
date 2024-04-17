package evaluate

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zimmski/osutil"

	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	"github.com/symflower/eval-dev-quality/language"
	"github.com/symflower/eval-dev-quality/model"
	"github.com/symflower/eval-dev-quality/model/symflower"
)

func TestEvaluateRepository(t *testing.T) {
	type testCase struct {
		Name string

		Model          model.Model
		Language       language.Language
		TestDataPath   string
		RepositoryPath string

		ExpectedRepositoryAssessment metrics.Assessments
		ExpectedResultFiles          []string
		ExpectedProblems             []error
		ExpectedError                error
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			temporaryPath := t.TempDir()

			actualRepositoryAssessment, actualProblems, actualErr := EvaluateRepository(temporaryPath, tc.Model, tc.Language, tc.TestDataPath, tc.RepositoryPath)

			assert.Equal(t, tc.ExpectedRepositoryAssessment, actualRepositoryAssessment)
			assert.Equal(t, tc.ExpectedProblems, actualProblems)
			assert.Equal(t, tc.ExpectedError, actualErr)

			actualResultFiles, err := osutil.FilesRecursive(temporaryPath)
			require.NoError(t, err)
			for i, p := range actualResultFiles {
				actualResultFiles[i], err = filepath.Rel(temporaryPath, p)
				require.NoError(t, err)
			}
			assert.Equal(t, tc.ExpectedResultFiles, actualResultFiles)
		})
	}

	validate(t, &testCase{
		Name: "Plain",

		Model:          &symflower.ModelSymflower{},
		Language:       &language.LanguageGolang{},
		TestDataPath:   "../testdata",
		RepositoryPath: "golang/plain",

		ExpectedRepositoryAssessment: metrics.Assessments{
			metrics.AssessmentKeyCoverageStatement: 1,
			metrics.AssessmentKeyFilesExecuted:     1,
			metrics.AssessmentKeyNoExcessResponse:  1,
		},
		ExpectedResultFiles: []string{
			"symflower_symbolic-execution/golang/golang/plain.log",
		},
	})
}
