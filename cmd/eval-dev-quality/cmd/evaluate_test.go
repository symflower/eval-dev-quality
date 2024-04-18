package cmd

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zimmski/osutil"
)

func TestEvaluateExecute(t *testing.T) {
	type testCase struct {
		Name string

		Arguments []string

		ExpectedOutputContains string
		ExpectedError          error
		ExpectedResultFiles    []string
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			temporaryPath := t.TempDir()

			actualOutput, actualError := osutil.Capture(func() {
				Execute(append([]string{
					"evaluate",
					"--result-path", temporaryPath,
					"--testdata", "../../../testdata",
				}, tc.Arguments...))
			})

			assert.Contains(t, string(actualOutput), tc.ExpectedOutputContains)
			assert.Equal(t, tc.ExpectedError, actualError)

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

		Arguments: []string{
			"--language", "golang",
			"--model", "symflower/symbolic-execution",
			"--repository", "golang/plain",
		},

		ExpectedOutputContains: `Evaluation score for "symflower/symbolic-execution": score=6, coverage-statement=1, files-executed=1, response-no-error=1, response-no-excess=1, response-not-empty=1, response-with-code=1`,
		ExpectedResultFiles: []string{
			"evaluation.csv",
			"evaluation.log",
			"symflower_symbolic-execution/golang/golang/plain.log",
		},
	})
}
