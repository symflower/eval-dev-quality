package cmd

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zimmski/osutil"

	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	metricstesting "github.com/symflower/eval-dev-quality/evaluate/metrics/testing"
	"github.com/symflower/eval-dev-quality/log"
	providertesting "github.com/symflower/eval-dev-quality/provider/testing"
	"github.com/symflower/eval-dev-quality/tools"
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

func atoiUint64(t *testing.T, s string) uint64 {
	value, err := strconv.ParseUint(s, 10, 64)
	assert.NoErrorf(t, err, "parsing unsigned integer from: %q", s)

	return uint64(value)
}

// extractMetricsMatch is a regular expression that maps metrics to it's subgroups.
type extractMetricsMatch *regexp.Regexp

// extractMetricsLogsMatch is a regular expression to extract metrics from log messages.
var extractMetricsLogsMatch = extractMetricsMatch(regexp.MustCompile(`score=(\d+), coverage-statement=(\d+), files-executed=(\d+), generate-tests-for-file-character-count=(\d+), processing-time=(\d+), response-character-count=(\d+), response-no-error=(\d+), response-no-excess=(\d+), response-with-code=(\d+)`))

// extractMetricsCSVMatch is a regular expression to extract metrics from CSV rows.
var extractMetricsCSVMatch = extractMetricsMatch(regexp.MustCompile(`(\d+),(\d+),(\d+),(\d+),(\d+),(\d+),(\d+),(\d+),(\d+)`))

// extractMetrics extracts multiple assessment metrics from the given string according to a given regular expression.
func extractMetrics(t *testing.T, regex extractMetricsMatch, data string) (assessments []metrics.Assessments, scores []uint64) {
	matches := (*regexp.Regexp)(regex).FindAllStringSubmatch(data, -1)

	for _, match := range matches {
		assessments = append(assessments, metrics.Assessments{
			metrics.AssessmentKeyCoverageStatement:                  atoiUint64(t, match[2]),
			metrics.AssessmentKeyFilesExecuted:                      atoiUint64(t, match[3]),
			metrics.AssessmentKeyGenerateTestsForFileCharacterCount: atoiUint64(t, match[4]),
			metrics.AssessmentKeyProcessingTime:                     atoiUint64(t, match[5]),
			metrics.AssessmentKeyResponseCharacterCount:             atoiUint64(t, match[6]),
			metrics.AssessmentKeyResponseNoError:                    atoiUint64(t, match[7]),
			metrics.AssessmentKeyResponseNoExcess:                   atoiUint64(t, match[8]),
			metrics.AssessmentKeyResponseWithCode:                   atoiUint64(t, match[9]),
		})
		scores = append(scores, atoiUint64(t, match[1]))
	}

	return assessments, scores
}

func validateMetrics(t *testing.T, regex *regexp.Regexp, data string, expectedAssessments []metrics.Assessments, expectedScores []uint64) (actual []metrics.Assessments) {
	require.Equal(t, len(expectedAssessments), len(expectedScores), "expected assessment and scores length")

	actualAssessments, actualScores := extractMetrics(t, regex, data)
	require.Equal(t, len(expectedAssessments), len(actualAssessments), "expected and actual assessment length")
	for i := range actualAssessments {
		metricstesting.AssertAssessmentsEqual(t, expectedAssessments[i], actualAssessments[i])
	}
	assert.Equal(t, expectedScores, actualScores)

	return actualAssessments
}

func TestEvaluateExecute(t *testing.T) {
	type testCase struct {
		Name string

		Before func(t *testing.T, logger *log.Logger, resultPath string)
		After  func(t *testing.T, logger *log.Logger, resultPath string)

		Arguments []string

		ExpectedOutputValidate func(t *testing.T, output string, resultPath string)
		ExpectedResultFiles    map[string]func(t *testing.T, filePath string, data string)
		ExpectedPanicContains  string
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

			if tc.Before != nil {
				tc.Before(t, logger, temporaryPath)
			}
			if tc.After != nil {
				defer tc.After(t, logger, temporaryPath)
			}

			arguments := append([]string{
				"evaluate",
				"--result-path", temporaryPath,
				"--testdata", filepath.Join("..", "..", "..", "testdata"),
			}, tc.Arguments...)

			if tc.ExpectedPanicContains == "" {
				assert.NotPanics(t, func() {
					Execute(logger, arguments)
				})
			} else {
				didPanic := true
				var recovered any
				func() {
					defer func() {
						recovered = recover()
					}()

					Execute(logger, arguments)

					didPanic = false
				}()
				assert.True(t, didPanic)
				assert.Contains(t, recovered, tc.ExpectedPanicContains)
			}

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
				actualAssessments := validateMetrics(t, extractMetricsLogsMatch, output, []metrics.Assessments{
					metrics.Assessments{
						metrics.AssessmentKeyCoverageStatement: 10,
						metrics.AssessmentKeyFilesExecuted:     1,
						metrics.AssessmentKeyResponseNoError:   1,
						metrics.AssessmentKeyResponseNoExcess:  1,
						metrics.AssessmentKeyResponseWithCode:  1,
					},
				}, []uint64{14})
				// Assert non-deterministic behavior.
				assert.Greater(t, actualAssessments[0][metrics.AssessmentKeyProcessingTime], uint64(0))
				assert.Equal(t, actualAssessments[0][metrics.AssessmentKeyGenerateTestsForFileCharacterCount], uint64(254))
				assert.Equal(t, actualAssessments[0][metrics.AssessmentKeyResponseCharacterCount], uint64(254))
				assert.Equal(t, 1, strings.Count(output, "Evaluation score for"))
			},
			ExpectedResultFiles: map[string]func(t *testing.T, filePath string, data string){
				"categories.svg": func(t *testing.T, filePath, data string) {
					validateSVGContent(t, data, []*metrics.AssessmentCategory{metrics.AssessmentCategoryCodeNoExcess}, 1)
				},
				"evaluation.csv": func(t *testing.T, filePath, data string) {
					actualAssessments := validateMetrics(t, extractMetricsCSVMatch, data, []metrics.Assessments{
						metrics.Assessments{
							metrics.AssessmentKeyCoverageStatement: 10,
							metrics.AssessmentKeyFilesExecuted:     1,
							metrics.AssessmentKeyResponseNoError:   1,
							metrics.AssessmentKeyResponseNoExcess:  1,
							metrics.AssessmentKeyResponseWithCode:  1,
						},
					}, []uint64{14})
					// Assert non-deterministic behavior.
					assert.Greater(t, actualAssessments[0][metrics.AssessmentKeyProcessingTime], uint64(0))
					assert.Equal(t, actualAssessments[0][metrics.AssessmentKeyGenerateTestsForFileCharacterCount], uint64(254))
					assert.Equal(t, actualAssessments[0][metrics.AssessmentKeyResponseCharacterCount], uint64(254))
				},
				"evaluation.log": nil,
				"golang-summed.csv": func(t *testing.T, filePath, data string) {
					actualAssessments := validateMetrics(t, extractMetricsCSVMatch, data, []metrics.Assessments{
						metrics.Assessments{
							metrics.AssessmentKeyCoverageStatement: 10,
							metrics.AssessmentKeyFilesExecuted:     1,
							metrics.AssessmentKeyResponseNoError:   1,
							metrics.AssessmentKeyResponseNoExcess:  1,
							metrics.AssessmentKeyResponseWithCode:  1,
						},
					}, []uint64{14})
					// Assert non-deterministic behavior.
					assert.Greater(t, actualAssessments[0][metrics.AssessmentKeyProcessingTime], uint64(0))
					assert.Equal(t, actualAssessments[0][metrics.AssessmentKeyGenerateTestsForFileCharacterCount], uint64(254))
					assert.Equal(t, actualAssessments[0][metrics.AssessmentKeyResponseCharacterCount], uint64(254))
				},
				"models-summed.csv": func(t *testing.T, filePath, data string) {
					actualAssessments := validateMetrics(t, extractMetricsCSVMatch, data, []metrics.Assessments{
						metrics.Assessments{
							metrics.AssessmentKeyCoverageStatement: 10,
							metrics.AssessmentKeyFilesExecuted:     1,
							metrics.AssessmentKeyResponseNoError:   1,
							metrics.AssessmentKeyResponseNoExcess:  1,
							metrics.AssessmentKeyResponseWithCode:  1,
						},
					}, []uint64{14})
					// Assert non-deterministic behavior.
					assert.Greater(t, actualAssessments[0][metrics.AssessmentKeyProcessingTime], uint64(0))
					assert.Equal(t, actualAssessments[0][metrics.AssessmentKeyGenerateTestsForFileCharacterCount], uint64(254))
					assert.Equal(t, actualAssessments[0][metrics.AssessmentKeyResponseCharacterCount], uint64(254))
				},
				"README.md": func(t *testing.T, filePath, data string) {
					validateReportLinks(t, data, []string{"symflower_symbolic-execution"})
				},
				filepath.Join("symflower_symbolic-execution", "golang", "golang", "plain.log"): nil,
			},
		})
		validate(t, &testCase{
			Name: "Multiple",

			Arguments: []string{
				"--model", "symflower/symbolic-execution",
			},

			ExpectedOutputValidate: func(t *testing.T, output string, resultPath string) {
				actualAssessments := validateMetrics(t, extractMetricsLogsMatch, output, []metrics.Assessments{
					metrics.Assessments{
						metrics.AssessmentKeyCoverageStatement: 20,
						metrics.AssessmentKeyFilesExecuted:     2,
						metrics.AssessmentKeyResponseNoError:   2,
						metrics.AssessmentKeyResponseNoExcess:  2,
						metrics.AssessmentKeyResponseWithCode:  2,
					},
				}, []uint64{28})
				// Assert non-deterministic behavior.
				assert.Greater(t, actualAssessments[0][metrics.AssessmentKeyProcessingTime], uint64(0))
				assert.Equal(t, actualAssessments[0][metrics.AssessmentKeyGenerateTestsForFileCharacterCount], uint64(393))
				assert.Equal(t, actualAssessments[0][metrics.AssessmentKeyResponseCharacterCount], uint64(393))
				assert.Equal(t, 1, strings.Count(output, "Evaluation score for"))
			},
			ExpectedResultFiles: map[string]func(t *testing.T, filePath string, data string){
				"categories.svg": func(t *testing.T, filePath, data string) {
					validateSVGContent(t, data, []*metrics.AssessmentCategory{metrics.AssessmentCategoryCodeNoExcess}, 1)
				},
				"evaluation.csv": func(t *testing.T, filePath, data string) {
					actualAssessments := validateMetrics(t, extractMetricsCSVMatch, data, []metrics.Assessments{
						metrics.Assessments{
							metrics.AssessmentKeyCoverageStatement: 10,
							metrics.AssessmentKeyFilesExecuted:     1,
							metrics.AssessmentKeyResponseNoError:   1,
							metrics.AssessmentKeyResponseNoExcess:  1,
							metrics.AssessmentKeyResponseWithCode:  1,
						},
						metrics.Assessments{
							metrics.AssessmentKeyCoverageStatement: 10,
							metrics.AssessmentKeyFilesExecuted:     1,
							metrics.AssessmentKeyResponseNoError:   1,
							metrics.AssessmentKeyResponseNoExcess:  1,
							metrics.AssessmentKeyResponseWithCode:  1,
						},
					}, []uint64{14, 14})
					// Assert non-deterministic behavior.
					assert.Greater(t, actualAssessments[0][metrics.AssessmentKeyProcessingTime], uint64(0))
					assert.Equal(t, actualAssessments[0][metrics.AssessmentKeyGenerateTestsForFileCharacterCount], uint64(254))
					assert.Equal(t, actualAssessments[0][metrics.AssessmentKeyResponseCharacterCount], uint64(254))
					assert.Greater(t, actualAssessments[1][metrics.AssessmentKeyProcessingTime], uint64(0))
					assert.Equal(t, actualAssessments[1][metrics.AssessmentKeyGenerateTestsForFileCharacterCount], uint64(139))
					assert.Equal(t, actualAssessments[1][metrics.AssessmentKeyResponseCharacterCount], uint64(139))
				},
				"golang-summed.csv": func(t *testing.T, filePath, data string) {
					actualAssessments := validateMetrics(t, extractMetricsCSVMatch, data, []metrics.Assessments{
						metrics.Assessments{
							metrics.AssessmentKeyCoverageStatement: 10,
							metrics.AssessmentKeyFilesExecuted:     1,
							metrics.AssessmentKeyResponseNoError:   1,
							metrics.AssessmentKeyResponseNoExcess:  1,
							metrics.AssessmentKeyResponseWithCode:  1,
						},
					}, []uint64{14})
					// Assert non-deterministic behavior.
					assert.Greater(t, actualAssessments[0][metrics.AssessmentKeyProcessingTime], uint64(0))
					assert.Equal(t, actualAssessments[0][metrics.AssessmentKeyGenerateTestsForFileCharacterCount], uint64(254))
					assert.Equal(t, actualAssessments[0][metrics.AssessmentKeyResponseCharacterCount], uint64(254))
				},
				"java-summed.csv": func(t *testing.T, filePath, data string) {
					actualAssessments := validateMetrics(t, extractMetricsCSVMatch, data, []metrics.Assessments{
						metrics.Assessments{
							metrics.AssessmentKeyCoverageStatement: 10,
							metrics.AssessmentKeyFilesExecuted:     1,
							metrics.AssessmentKeyResponseNoError:   1,
							metrics.AssessmentKeyResponseNoExcess:  1,
							metrics.AssessmentKeyResponseWithCode:  1,
						},
					}, []uint64{14})
					// Assert non-deterministic behavior.
					assert.Greater(t, actualAssessments[0][metrics.AssessmentKeyProcessingTime], uint64(0))
					assert.Equal(t, actualAssessments[0][metrics.AssessmentKeyGenerateTestsForFileCharacterCount], uint64(139))
					assert.Equal(t, actualAssessments[0][metrics.AssessmentKeyResponseCharacterCount], uint64(139))
				},
				"models-summed.csv": func(t *testing.T, filePath, data string) {
					actualAssessments := validateMetrics(t, extractMetricsCSVMatch, data, []metrics.Assessments{
						metrics.Assessments{
							metrics.AssessmentKeyCoverageStatement: 20,
							metrics.AssessmentKeyFilesExecuted:     2,
							metrics.AssessmentKeyResponseNoError:   2,
							metrics.AssessmentKeyResponseNoExcess:  2,
							metrics.AssessmentKeyResponseWithCode:  2,
						},
					}, []uint64{28})
					// Assert non-deterministic behavior.
					assert.Greater(t, actualAssessments[0][metrics.AssessmentKeyProcessingTime], uint64(0))
					assert.Equal(t, actualAssessments[0][metrics.AssessmentKeyGenerateTestsForFileCharacterCount], uint64(393))
					assert.Equal(t, actualAssessments[0][metrics.AssessmentKeyResponseCharacterCount], uint64(393))
				},
				"evaluation.log": nil,
				"README.md": func(t *testing.T, filePath, data string) {
					validateReportLinks(t, data, []string{"symflower_symbolic-execution"})
				},
				filepath.Join("symflower_symbolic-execution", "golang", "golang", "plain.log"): nil,
				filepath.Join("symflower_symbolic-execution", "java", "java", "plain.log"):     nil,
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
					"--repository", filepath.Join("golang", "plain"),
				},

				ExpectedOutputValidate: func(t *testing.T, output string, resultPath string) {
					actualAssessments := validateMetrics(t, extractMetricsLogsMatch, output, []metrics.Assessments{
						metrics.Assessments{
							metrics.AssessmentKeyCoverageStatement: 10,
							metrics.AssessmentKeyFilesExecuted:     1,
							metrics.AssessmentKeyResponseNoError:   1,
							metrics.AssessmentKeyResponseNoExcess:  1,
							metrics.AssessmentKeyResponseWithCode:  1,
						},
					}, []uint64{14})
					// Assert non-deterministic behavior.
					assert.Greater(t, actualAssessments[0][metrics.AssessmentKeyProcessingTime], uint64(0))
					assert.Equal(t, actualAssessments[0][metrics.AssessmentKeyGenerateTestsForFileCharacterCount], uint64(254))
					assert.Equal(t, actualAssessments[0][metrics.AssessmentKeyResponseCharacterCount], uint64(254))
					assert.Equal(t, 1, strings.Count(output, "Evaluation score for"))
				},
				ExpectedResultFiles: map[string]func(t *testing.T, filePath string, data string){
					"categories.svg": func(t *testing.T, filePath, data string) {
						validateSVGContent(t, data, []*metrics.AssessmentCategory{metrics.AssessmentCategoryCodeNoExcess}, 1)
					},
					"evaluation.csv": func(t *testing.T, filePath, data string) {
						actualAssessments := validateMetrics(t, extractMetricsCSVMatch, data, []metrics.Assessments{
							metrics.Assessments{
								metrics.AssessmentKeyCoverageStatement: 10,
								metrics.AssessmentKeyFilesExecuted:     1,
								metrics.AssessmentKeyResponseNoError:   1,
								metrics.AssessmentKeyResponseNoExcess:  1,
								metrics.AssessmentKeyResponseWithCode:  1,
							},
						}, []uint64{14})
						// Assert non-deterministic behavior.
						assert.Greater(t, actualAssessments[0][metrics.AssessmentKeyProcessingTime], uint64(0))
						assert.Equal(t, actualAssessments[0][metrics.AssessmentKeyGenerateTestsForFileCharacterCount], uint64(254))
						assert.Equal(t, actualAssessments[0][metrics.AssessmentKeyResponseCharacterCount], uint64(254))
					},
					"evaluation.log": nil,
					"golang-summed.csv": func(t *testing.T, filePath, data string) {
						actualAssessments := validateMetrics(t, extractMetricsCSVMatch, data, []metrics.Assessments{
							metrics.Assessments{
								metrics.AssessmentKeyCoverageStatement: 10,
								metrics.AssessmentKeyFilesExecuted:     1,
								metrics.AssessmentKeyResponseNoError:   1,
								metrics.AssessmentKeyResponseNoExcess:  1,
								metrics.AssessmentKeyResponseWithCode:  1,
							},
						}, []uint64{14})
						// Assert non-deterministic behavior.
						assert.Greater(t, actualAssessments[0][metrics.AssessmentKeyProcessingTime], uint64(0))
						assert.Equal(t, actualAssessments[0][metrics.AssessmentKeyGenerateTestsForFileCharacterCount], uint64(254))
						assert.Equal(t, actualAssessments[0][metrics.AssessmentKeyResponseCharacterCount], uint64(254))
					},
					"models-summed.csv": func(t *testing.T, filePath, data string) {
						actualAssessments := validateMetrics(t, extractMetricsCSVMatch, data, []metrics.Assessments{
							metrics.Assessments{
								metrics.AssessmentKeyCoverageStatement: 10,
								metrics.AssessmentKeyFilesExecuted:     1,
								metrics.AssessmentKeyResponseNoError:   1,
								metrics.AssessmentKeyResponseNoExcess:  1,
								metrics.AssessmentKeyResponseWithCode:  1,
							},
						}, []uint64{14})
						// Assert non-deterministic behavior.
						assert.Greater(t, actualAssessments[0][metrics.AssessmentKeyProcessingTime], uint64(0))
						assert.Equal(t, actualAssessments[0][metrics.AssessmentKeyGenerateTestsForFileCharacterCount], uint64(254))
						assert.Equal(t, actualAssessments[0][metrics.AssessmentKeyResponseCharacterCount], uint64(254))
					},
					"README.md": func(t *testing.T, filePath, data string) {
						validateReportLinks(t, data, []string{"symflower_symbolic-execution"})
					},
					filepath.Join("symflower_symbolic-execution", "golang", "golang", "plain.log"): nil,
				},
			})
			validate(t, &testCase{
				Name: "Multiple Languages",

				Arguments: []string{
					"--model", "symflower/symbolic-execution",
					"--repository", filepath.Join("golang", "plain"),
				},

				ExpectedOutputValidate: func(t *testing.T, output string, resultPath string) {
					assert.Regexp(t, `Evaluation score for "symflower/symbolic-execution" \("code-no-excess"\): score=14, coverage-statement=10, files-executed=1, generate-tests-for-file-character-count=254, processing-time=\d+, response-character-count=254, response-no-error=1, response-no-excess=1, response-with-code=1`, output)
					assert.Equal(t, 1, strings.Count(output, "Evaluation score for"))
				},
				ExpectedResultFiles: map[string]func(t *testing.T, filePath string, data string){
					"categories.svg": func(t *testing.T, filePath, data string) {
						validateSVGContent(t, data, []*metrics.AssessmentCategory{metrics.AssessmentCategoryCodeNoExcess}, 1)
					},
					"evaluation.csv": func(t *testing.T, filePath, data string) {
						actualAssessments := validateMetrics(t, extractMetricsCSVMatch, data, []metrics.Assessments{
							metrics.Assessments{
								metrics.AssessmentKeyCoverageStatement: 10,
								metrics.AssessmentKeyFilesExecuted:     1,
								metrics.AssessmentKeyResponseNoError:   1,
								metrics.AssessmentKeyResponseNoExcess:  1,
								metrics.AssessmentKeyResponseWithCode:  1,
							},
						}, []uint64{14})
						// Assert non-deterministic behavior.
						assert.Greater(t, actualAssessments[0][metrics.AssessmentKeyProcessingTime], uint64(0))
						assert.Equal(t, actualAssessments[0][metrics.AssessmentKeyGenerateTestsForFileCharacterCount], uint64(254))
						assert.Equal(t, actualAssessments[0][metrics.AssessmentKeyResponseCharacterCount], uint64(254))
					},
					"evaluation.log": nil,
					"golang-summed.csv": func(t *testing.T, filePath, data string) {
						actualAssessments := validateMetrics(t, extractMetricsCSVMatch, data, []metrics.Assessments{
							metrics.Assessments{
								metrics.AssessmentKeyCoverageStatement: 10,
								metrics.AssessmentKeyFilesExecuted:     1,
								metrics.AssessmentKeyResponseNoError:   1,
								metrics.AssessmentKeyResponseNoExcess:  1,
								metrics.AssessmentKeyResponseWithCode:  1,
							},
						}, []uint64{14})
						// Assert non-deterministic behavior.
						assert.Greater(t, actualAssessments[0][metrics.AssessmentKeyProcessingTime], uint64(0))
						assert.Equal(t, actualAssessments[0][metrics.AssessmentKeyGenerateTestsForFileCharacterCount], uint64(254))
						assert.Equal(t, actualAssessments[0][metrics.AssessmentKeyResponseCharacterCount], uint64(254))
					},
					"models-summed.csv": func(t *testing.T, filePath, data string) {
						actualAssessments := validateMetrics(t, extractMetricsCSVMatch, data, []metrics.Assessments{
							metrics.Assessments{
								metrics.AssessmentKeyCoverageStatement: 10,
								metrics.AssessmentKeyFilesExecuted:     1,
								metrics.AssessmentKeyResponseNoError:   1,
								metrics.AssessmentKeyResponseNoExcess:  1,
								metrics.AssessmentKeyResponseWithCode:  1,
							},
						}, []uint64{14})
						// Assert non-deterministic behavior.
						assert.Greater(t, actualAssessments[0][metrics.AssessmentKeyProcessingTime], uint64(0))
						assert.Equal(t, actualAssessments[0][metrics.AssessmentKeyGenerateTestsForFileCharacterCount], uint64(254))
						assert.Equal(t, actualAssessments[0][metrics.AssessmentKeyResponseCharacterCount], uint64(254))
					},
					"README.md": func(t *testing.T, filePath, data string) {
						validateReportLinks(t, data, []string{"symflower_symbolic-execution"})
					},
					filepath.Join("symflower_symbolic-execution", "golang", "golang", "plain.log"): nil,
				},
			})
		})
	})
	t.Run("Model filter", func(t *testing.T) {
		t.Run("openrouter.ai", func(t *testing.T) {
			validate(t, &testCase{
				Name: "Unavailable",

				Arguments: []string{
					"--model", "openrouter/auto",
					"--tokens", "openrouter:",
				},

				ExpectedOutputValidate: func(t *testing.T, output, resultPath string) {
					assert.Contains(t, output, "Skipping unavailable provider \"openrouter\"")
				},
				ExpectedResultFiles: map[string]func(t *testing.T, filePath string, data string){
					"evaluation.log": nil,
				},
				ExpectedPanicContains: "ERROR: model openrouter/auto does not exist",
			})
		})
		t.Run("Ollama", func(t *testing.T) {
			if !osutil.IsLinux() {
				t.Skipf("Installation of Ollama is not supported on this OS")
			}

			{
				var shutdown func() (err error)
				defer func() { // Defer the shutdown in case there is a panic.
					if shutdown != nil {
						require.NoError(t, shutdown())
					}
				}()
				validate(t, &testCase{
					Name: "Pulled Model",

					Before: func(t *testing.T, logger *log.Logger, resultPath string) {
						var err error
						shutdown, err = tools.OllamaStart(logger, tools.OllamaPath, tools.OllamaURL)
						require.NoError(t, err)

						require.NoError(t, tools.OllamaPull(logger, tools.OllamaPath, tools.OllamaURL, providertesting.OllamaTestModel))
					},

					Arguments: []string{
						"--language", "golang",
						"--model", "ollama/" + providertesting.OllamaTestModel,
						"--repository", filepath.Join("golang", "plain"),
					},

					ExpectedResultFiles: map[string]func(t *testing.T, filePath string, data string){
						"categories.svg": nil,
						"evaluation.csv": nil,
						"evaluation.log": func(t *testing.T, filePath, data string) {
							// Since the model is non-deterministic, we can only assert that the model did at least not error.
							assert.Contains(t, data, fmt.Sprintf(`Evaluation score for "ollama/%s"`, providertesting.OllamaTestModel))
							assert.Contains(t, data, "response-no-error=1")
							assert.Contains(t, data, "preloading model")
							assert.Contains(t, data, "unloading model")
						},
						"golang-summed.csv": nil,
						"models-summed.csv": nil,
						"README.md":         nil,
						"ollama_" + providertesting.OllamaTestModel + "/golang/golang/plain.log": nil,
					},
				})
			}
		})
		t.Run("OpenAI API", func(t *testing.T) {
			if !osutil.IsLinux() {
				t.Skipf("Installation of Ollama is not supported on this OS")
			}

			{
				var shutdown func() (err error)
				defer func() {
					if shutdown != nil {
						require.NoError(t, shutdown())
					}
				}()
				ollamaOpenAIAPIUrl, err := url.JoinPath(tools.OllamaURL, "v1")
				require.NoError(t, err)
				validate(t, &testCase{
					Name: "Ollama",

					Before: func(t *testing.T, logger *log.Logger, resultPath string) {
						var err error
						shutdown, err = tools.OllamaStart(logger, tools.OllamaPath, tools.OllamaURL)
						require.NoError(t, err)

						require.NoError(t, tools.OllamaPull(logger, tools.OllamaPath, tools.OllamaURL, providertesting.OllamaTestModel))
					},

					Arguments: []string{
						"--language", "golang",
						"--urls", "custom-ollama:" + ollamaOpenAIAPIUrl,
						"--model", "custom-ollama/" + providertesting.OllamaTestModel,
						"--repository", filepath.Join("golang", "plain"),
					},

					ExpectedResultFiles: map[string]func(t *testing.T, filePath string, data string){
						"categories.svg": nil,
						"evaluation.csv": nil,
						"evaluation.log": func(t *testing.T, filePath, data string) {
							// Since the model is non-deterministic, we can only assert that the model did at least not error.
							assert.Contains(t, data, fmt.Sprintf(`Evaluation score for "custom-ollama/%s"`, providertesting.OllamaTestModel))
							assert.Contains(t, data, "response-no-error=1")
						},
						"golang-summed.csv": nil,
						"models-summed.csv": nil,
						"README.md":         nil,
						"custom-ollama_" + providertesting.OllamaTestModel + "/golang/golang/plain.log": nil,
					},
				})
			}
		})
	})

	t.Run("Runs", func(t *testing.T) {
		validate(t, &testCase{
			Name: "Multiple",

			Arguments: []string{
				"--model", "symflower/symbolic-execution",
				"--repository", filepath.Join("golang", "plain"),
				"--runs=3",
			},

			ExpectedOutputValidate: func(t *testing.T, output string, resultPath string) {
				actualAssessments := validateMetrics(t, extractMetricsLogsMatch, output, []metrics.Assessments{
					metrics.Assessments{
						metrics.AssessmentKeyCoverageStatement: 30,
						metrics.AssessmentKeyFilesExecuted:     3,
						metrics.AssessmentKeyResponseNoError:   3,
						metrics.AssessmentKeyResponseNoExcess:  3,
						metrics.AssessmentKeyResponseWithCode:  3,
					},
				}, []uint64{42})
				// Assert non-deterministic behavior.
				assert.Greater(t, actualAssessments[0][metrics.AssessmentKeyProcessingTime], uint64(0))
				assert.Equal(t, actualAssessments[0][metrics.AssessmentKeyGenerateTestsForFileCharacterCount], uint64(762))
				assert.Equal(t, actualAssessments[0][metrics.AssessmentKeyResponseCharacterCount], uint64(762))
				assert.Equal(t, 1, strings.Count(output, "Evaluation score for"))
			},
			ExpectedResultFiles: map[string]func(t *testing.T, filePath string, data string){
				"categories.svg": nil,
				"evaluation.csv": func(t *testing.T, filePath, data string) {
					actualAssessments := validateMetrics(t, extractMetricsCSVMatch, data, []metrics.Assessments{
						metrics.Assessments{
							metrics.AssessmentKeyCoverageStatement: 30,
							metrics.AssessmentKeyFilesExecuted:     3,
							metrics.AssessmentKeyResponseNoError:   3,
							metrics.AssessmentKeyResponseNoExcess:  3,
							metrics.AssessmentKeyResponseWithCode:  3,
						},
					}, []uint64{42})
					// Assert non-deterministic behavior.
					assert.Greater(t, actualAssessments[0][metrics.AssessmentKeyProcessingTime], uint64(0))
					assert.Equal(t, actualAssessments[0][metrics.AssessmentKeyGenerateTestsForFileCharacterCount], uint64(762))
					assert.Equal(t, actualAssessments[0][metrics.AssessmentKeyResponseCharacterCount], uint64(762))
				},
				"evaluation.log": func(t *testing.T, filePath, data string) {
					assert.Contains(t, data, "Run 1/3")
					assert.Contains(t, data, "Run 2/3")
					assert.Contains(t, data, "Run 3/3")
				},
				"golang-summed.csv": nil,
				"models-summed.csv": nil,
				"README.md":         nil,
				filepath.Join("symflower_symbolic-execution", "golang", "golang", "plain.log"): func(t *testing.T, filePath, data string) {
					assert.Equal(t, 3, strings.Count(data, `Evaluating model "symflower/symbolic-execution"`))
				},
			},
		})
	})

	// This case checks a beautiful bug where the Markdown export crashed when the current working directory contained a README.md file. While this is not the case during the tests (as the current work directory is the directory of this file), it certainly caused problems when our binary was executed from the repository root (which of course contained a README.md). Therefore, we sadly have to modify the current work directory right within the tests of this case to reproduce the problem and fix it forever.
	validate(t, &testCase{
		Name: "Current work directory contains a README.md",

		Before: func(t *testing.T, logger *log.Logger, resultPath string) {
			if err := os.Remove("README.md"); err != nil {
				if osutil.IsWindows() {
					require.Contains(t, err.Error(), "The system cannot find the file specified")
				} else {
					require.Contains(t, err.Error(), "no such file or directory")
				}
			}
			require.NoError(t, os.WriteFile("README.md", []byte(""), 0644))
		},
		After: func(t *testing.T, logger *log.Logger, resultPath string) {
			require.NoError(t, os.Remove("README.md"))
		},

		Arguments: []string{
			"--language", "golang",
			"--model", "symflower/symbolic-execution",
		},

		ExpectedResultFiles: map[string]func(t *testing.T, filePath string, data string){
			"categories.svg":    nil,
			"evaluation.csv":    nil,
			"evaluation.log":    nil,
			"golang-summed.csv": nil,
			"models-summed.csv": nil,
			"README.md":         nil,
			filepath.Join("symflower_symbolic-execution", "golang", "golang", "plain.log"): nil,
		},
	})
}
