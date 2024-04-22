package cmd

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zimmski/osutil"
	"github.com/zimmski/osutil/bytesutil"

	"github.com/symflower/eval-dev-quality/log"
)

func TestEvaluateExecute(t *testing.T) {
	type testCase struct {
		Name string

		Arguments []string

		ExpectedOutputValidate func(t *testing.T, output string, resultPath string)
		ExpectedResultFiles    map[string]func(t *testing.T, resultPath string, filePath string, data string)
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			temporaryPath := t.TempDir()

			logOutput, logger := log.Buffer()
			Execute(logger, append([]string{
				"evaluate",
				"--result-path", temporaryPath,
				"--testdata", "../../../testdata",
			}, tc.Arguments...))

			if tc.ExpectedOutputValidate != nil {
				tc.ExpectedOutputValidate(t, string(logOutput.String()), temporaryPath)
			}

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
						validate(t, temporaryPath, filePath, string(data))
					}
				}
			}
			sort.Strings(expectedResultFiles)
			assert.Equal(t, expectedResultFiles, actualResultFiles)
		})
	}

	validate(t, &testCase{
		Name: "Plain",

		Arguments: []string{
			"--language", "golang",
			"--model", "symflower/symbolic-execution",
			"--repository", "golang/plain",
		},

		ExpectedOutputValidate: func(t *testing.T, output string, resultPath string) {
			assert.Contains(t, output, `Evaluation score for "symflower/symbolic-execution" ("No Excess Response"): score=6, coverage-statement=1, files-executed=1, response-no-error=1, response-no-excess=1, response-not-empty=1, response-with-code=1`)
			if !assert.Equal(t, 1, strings.Count(output, "Evaluation score for")) {
				t.Logf("Output: %s", output)
			}
		},
		ExpectedResultFiles: map[string]func(t *testing.T, resultPath string, filePath string, data string){
			"evaluation.csv": func(t *testing.T, resultPath string, filePath, data string) {
				assert.Equal(t, bytesutil.StringTrimIndentations(`
					model,language,repository,score,coverage-statement,files-executed,response-no-error,response-no-excess,response-not-empty,response-with-code
					symflower/symbolic-execution,golang,golang/plain,6,1,1,1,1,1,1
				`), data)
			},
			"evaluation.log": nil,
			"report.md": func(t *testing.T, resultPath string, filePath, data string) {
				// Ensure the report links to the CSV file and logs.
				assert.Contains(t, data, filepath.Join(resultPath, "evaluation.csv"))
				assert.Contains(t, data, filepath.Join(resultPath, "evaluation.log"))
			},
			"symflower_symbolic-execution/golang/golang/plain.log": nil,
		},
	})
}
