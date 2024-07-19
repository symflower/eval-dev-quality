package languagetesting

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	language "github.com/symflower/eval-dev-quality/language"
	log "github.com/symflower/eval-dev-quality/log"
	"github.com/zimmski/osutil"
)

type TestCaseExecute struct {
	Name string

	Language language.Language

	RepositoryPath   string
	RepositoryChange func(t *testing.T, repositoryPath string)

	ExpectedCoverage     uint64
	ExpectedProblemTexts []string
	ExpectedError        error
	ExpectedErrorText    string
}

func (tc *TestCaseExecute) Validate(t *testing.T) {
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

		actualCoverage, actualProblems, actualError := tc.Language.Execute(logger, repositoryPath)

		require.Equal(t, len(tc.ExpectedProblemTexts), len(actualProblems), "the number of expected problems need to match the number of actual problems")
		for i, expectedProblemText := range tc.ExpectedProblemTexts {
			assert.ErrorContains(t, actualProblems[i], expectedProblemText)
		}

		if tc.ExpectedError != nil {
			assert.ErrorIs(t, actualError, tc.ExpectedError)
		} else if actualError != nil && tc.ExpectedErrorText != "" {
			assert.ErrorContains(t, actualError, tc.ExpectedErrorText)
		} else {
			assert.NoError(t, actualError)
			assert.Equal(t, tc.ExpectedCoverage, actualCoverage)
		}
	})
}
