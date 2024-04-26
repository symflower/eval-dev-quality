package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zimmski/osutil"
	"github.com/zimmski/osutil/bytesutil"

	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	"github.com/symflower/eval-dev-quality/log"
)

// validateReportLinks checks if the Markdown report data contains all the links to other relevant report files.
func validateReportLinks(t *testing.T, data string, modelLogNames []string) {
	assert.Contains(t, data, "](./categories.svg)")
	assert.Contains(t, data, "](./evaluation.csv)")
	assert.Contains(t, data, "](./evaluation.log)")
	for _, m := range modelLogNames {
		assert.Contains(t, data, fmt.Sprintf("](./%s/)", m))
	}
}

// validateSVGContent checks if the SVG data contains all given categories and an axis label for the maximal model count.
func validateSVGContent(t *testing.T, data string, categories []*metrics.AssessmentCategory, maxModelCount uint) {
	for _, category := range categories {
		assert.Contains(t, data, fmt.Sprintf("%s</text>", category.Name))
	}
	assert.Contains(t, data, fmt.Sprintf("%d</text>", maxModelCount))
}

func TestEvaluateExecute(t *testing.T) {
	type testCase struct {
		Name string

		Before func(t *testing.T, resultPath string)
		After  func(t *testing.T, resultPath string)

		Arguments []string

		ExpectedOutputValidate func(t *testing.T, output string, resultPath string)
		ExpectedResultFiles    map[string]func(t *testing.T, filePath string, data string)
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			temporaryPath := t.TempDir()

			if tc.Before != nil {
				tc.Before(t, temporaryPath)
			}
			if tc.After != nil {
				defer tc.After(t, temporaryPath)
			}

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
				"categories.svg": func(t *testing.T, filePath, data string) {
					validateSVGContent(t, data, []*metrics.AssessmentCategory{metrics.AssessmentCategoryCodeNoExcess}, 1)
				},
				"evaluation.csv": func(t *testing.T, filePath, data string) {
					assert.Equal(t, bytesutil.StringTrimIndentations(`
						model,language,repository,score,coverage-statement,files-executed,response-no-error,response-no-excess,response-not-empty,response-with-code
						symflower/symbolic-execution,golang,golang/plain,6,1,1,1,1,1,1
					`), data)
				},
				"evaluation.log": nil,
				"README.md": func(t *testing.T, filePath, data string) {
					validateReportLinks(t, data, []string{"symflower_symbolic-execution"})
				},
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
				"categories.svg": func(t *testing.T, filePath, data string) {
					validateSVGContent(t, data, []*metrics.AssessmentCategory{metrics.AssessmentCategoryCodeNoExcess}, 1)
				},
				"evaluation.csv": func(t *testing.T, filePath, data string) {
					assert.Equal(t, bytesutil.StringTrimIndentations(`
						model,language,repository,score,coverage-statement,files-executed,response-no-error,response-no-excess,response-not-empty,response-with-code
						symflower/symbolic-execution,golang,golang/plain,6,1,1,1,1,1,1
						symflower/symbolic-execution,java,java/plain,6,1,1,1,1,1,1
					`), data)
				},
				"evaluation.log": nil,
				"README.md": func(t *testing.T, filePath, data string) {
					validateReportLinks(t, data, []string{"symflower_symbolic-execution"})
				},
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
					"categories.svg": func(t *testing.T, filePath, data string) {
						validateSVGContent(t, data, []*metrics.AssessmentCategory{metrics.AssessmentCategoryCodeNoExcess}, 1)
					},
					"evaluation.csv": func(t *testing.T, filePath, data string) {
						assert.Equal(t, bytesutil.StringTrimIndentations(`
							model,language,repository,score,coverage-statement,files-executed,response-no-error,response-no-excess,response-not-empty,response-with-code
							symflower/symbolic-execution,golang,golang/plain,6,1,1,1,1,1,1
						`), data)
					},
					"evaluation.log": nil,
					"README.md": func(t *testing.T, filePath, data string) {
						validateReportLinks(t, data, []string{"symflower_symbolic-execution"})
					},
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
					"categories.svg": func(t *testing.T, filePath, data string) {
						validateSVGContent(t, data, []*metrics.AssessmentCategory{metrics.AssessmentCategoryCodeNoExcess}, 1)
					},
					"evaluation.csv": func(t *testing.T, filePath, data string) {
						assert.Equal(t, bytesutil.StringTrimIndentations(`
							model,language,repository,score,coverage-statement,files-executed,response-no-error,response-no-excess,response-not-empty,response-with-code
							symflower/symbolic-execution,golang,golang/plain,6,1,1,1,1,1,1
						`), data)
					},
					"evaluation.log": nil,
					"README.md": func(t *testing.T, filePath, data string) {
						validateReportLinks(t, data, []string{"symflower_symbolic-execution"})
					},
					"symflower_symbolic-execution/golang/golang/plain.log": nil,
				},
			})
		})
	})

	// This case cehcks a beautiful bug where the Markdown export crashed when the current working directory contained a README.md file. While this is not the case during the tests (as the current work directory is the directory of this file), it certainly caused problems when our binary was executed from the repository root (which of course contained a README.md). Therefore, we sadly have to modify the current work directory right within the tests of this case to reproduce the problem and fix it forever.
	validate(t, &testCase{
		Name: "Current work directory contains a README.md",

		Before: func(t *testing.T, resultPath string) {
			if err := os.Remove("README.md"); err != nil {
				require.Contains(t, err.Error(), "no such file or directory")
			}
			require.NoError(t, os.WriteFile("README.md", []byte(""), 0644))
		},
		After: func(t *testing.T, resultPath string) {
			require.NoError(t, os.Remove("README.md"))
		},

		Arguments: []string{
			"--language", "golang",
			"--model", "symflower/symbolic-execution",
		},

		ExpectedResultFiles: map[string]func(t *testing.T, filePath string, data string){
			"categories.svg": nil,
			"evaluation.csv": nil,
			"evaluation.log": nil,
			"README.md":      nil,
			"symflower_symbolic-execution/golang/golang/plain.log": nil,
		},
	})
}
