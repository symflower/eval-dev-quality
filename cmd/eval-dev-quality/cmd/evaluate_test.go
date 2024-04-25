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
		ExpectedResultFiles    map[string]func(t *testing.T, filePath string, data string)
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			temporaryPath := t.TempDir()

			logOutput, logger := log.Buffer()
			defer func() {
				if t.Failed() {
					t.Logf("Logging output: %s", logOutput.String())
				}
			}()
			Execute(logger, append([]string{
				"evaluate",
				"--result-path", temporaryPath,
				"--testdata", "../../../testdata",
			}, tc.Arguments...))

			if tc.ExpectedOutputValidate != nil {
				tc.ExpectedOutputValidate(t, logOutput.String(), temporaryPath)
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
						validate(t, filePath, string(data))
					}
				}
			}
			sort.Strings(expectedResultFiles)
			assert.Equal(t, expectedResultFiles, actualResultFiles)
		})
	}

	t.Run("Language filter", func(t *testing.T) {
		validate(t, &testCase{
			Name: "Single",

			Arguments: []string{
				"--language", "golang",
				"--model", "symflower/symbolic-execution",
			},

			ExpectedOutputValidate: func(t *testing.T, output string, resultPath string) {
				assert.Contains(t, output, `Evaluation score for "symflower/symbolic-execution" ("code-no-excess"): score=6, coverage-statement=1, files-executed=1, response-no-error=1, response-no-excess=1, response-not-empty=1, response-with-code=1`)
				if !assert.Equal(t, 1, strings.Count(output, "Evaluation score for")) {
					t.Logf("Output: %s", output)
				}
			},
			ExpectedResultFiles: map[string]func(t *testing.T, filePath string, data string){
				"evaluation.csv": func(t *testing.T, filePath, data string) {
					assert.Equal(t, bytesutil.StringTrimIndentations(`
						model,language,repository,score,coverage-statement,files-executed,response-no-error,response-no-excess,response-not-empty,response-with-code
						symflower/symbolic-execution,golang,golang/plain,6,1,1,1,1,1,1
					`), data)
				},
				"evaluation.log": nil,
				"symflower_symbolic-execution/golang/golang/plain.log": nil,
			},
		})
		validate(t, &testCase{
			Name: "Multiple",

			Arguments: []string{
				"--model", "symflower/symbolic-execution",
			},

			ExpectedOutputValidate: func(t *testing.T, output string, resultPath string) {
				assert.Contains(t, output, `Evaluation score for "symflower/symbolic-execution" ("code-no-excess"): score=12, coverage-statement=2, files-executed=2, response-no-error=2, response-no-excess=2, response-not-empty=2, response-with-code=2`)
				if !assert.Equal(t, 1, strings.Count(output, "Evaluation score for")) {
					t.Logf("Output: %s", output)
				}
			},
			ExpectedResultFiles: map[string]func(t *testing.T, filePath string, data string){
				"evaluation.csv": func(t *testing.T, filePath, data string) {
					assert.Equal(t, bytesutil.StringTrimIndentations(`
						model,language,repository,score,coverage-statement,files-executed,response-no-error,response-no-excess,response-not-empty,response-with-code
						symflower/symbolic-execution,golang,golang/plain,6,1,1,1,1,1,1
						symflower/symbolic-execution,java,java/plain,6,1,1,1,1,1,1
					`), data)
				},
				"evaluation.log": nil,
				"symflower_symbolic-execution/golang/golang/plain.log": nil,
				"symflower_symbolic-execution/java/java/plain.log":     nil,
			},
		})
	})

	t.Run("Repository filter", func(t *testing.T) {
		t.Run("Single", func(t *testing.T) {
			validate(t, &testCase{
				Name: "Single Language",

				Arguments: []string{
					"--language", "golang",
					"--model", "symflower/symbolic-execution",
					"--repository", "golang/plain",
				},

				ExpectedOutputValidate: func(t *testing.T, output string, resultPath string) {
					assert.Contains(t, output, `Evaluation score for "symflower/symbolic-execution" ("code-no-excess"): score=6, coverage-statement=1, files-executed=1, response-no-error=1, response-no-excess=1, response-not-empty=1, response-with-code=1`)
					if !assert.Equal(t, 1, strings.Count(output, "Evaluation score for")) {
						t.Logf("Output: %s", output)
					}
				},
				ExpectedResultFiles: map[string]func(t *testing.T, filePath string, data string){
					"evaluation.csv": func(t *testing.T, filePath, data string) {
						assert.Equal(t, bytesutil.StringTrimIndentations(`
							model,language,repository,score,coverage-statement,files-executed,response-no-error,response-no-excess,response-not-empty,response-with-code
							symflower/symbolic-execution,golang,golang/plain,6,1,1,1,1,1,1
						`), data)
					},
					"evaluation.log": nil,
					"symflower_symbolic-execution/golang/golang/plain.log": nil,
				},
			})
			validate(t, &testCase{
				Name: "Multiple Languages",

				Arguments: []string{
					"--model", "symflower/symbolic-execution",
					"--repository", "golang/plain",
				},

				ExpectedOutputValidate: func(t *testing.T, output string, resultPath string) {
					assert.Contains(t, output, `Evaluation score for "symflower/symbolic-execution" ("code-no-excess"): score=6, coverage-statement=1, files-executed=1, response-no-error=1, response-no-excess=1, response-not-empty=1, response-with-code=1`)
					if !assert.Equal(t, 1, strings.Count(output, "Evaluation score for")) {
						t.Logf("Output: %s", output)
					}
				},
				ExpectedResultFiles: map[string]func(t *testing.T, filePath string, data string){
					"evaluation.csv": func(t *testing.T, filePath, data string) {
						assert.Equal(t, bytesutil.StringTrimIndentations(`
							model,language,repository,score,coverage-statement,files-executed,response-no-error,response-no-excess,response-not-empty,response-with-code
							symflower/symbolic-execution,golang,golang/plain,6,1,1,1,1,1,1
						`), data)
					},
					"evaluation.log": nil,
					"symflower_symbolic-execution/golang/golang/plain.log": nil,
				},
			})
		})
	})
}
