package evaluate

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/zimmski/osutil"
	"github.com/zimmski/osutil/bytesutil"

	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	metricstesting "github.com/symflower/eval-dev-quality/evaluate/metrics/testing"
	reporttesting "github.com/symflower/eval-dev-quality/evaluate/report/testing"
	evaluatetask "github.com/symflower/eval-dev-quality/evaluate/task"
	"github.com/symflower/eval-dev-quality/language"
	"github.com/symflower/eval-dev-quality/language/golang"
	"github.com/symflower/eval-dev-quality/log"
	evalmodel "github.com/symflower/eval-dev-quality/model"
	"github.com/symflower/eval-dev-quality/model/llm"
	modeltesting "github.com/symflower/eval-dev-quality/model/testing"
	"github.com/symflower/eval-dev-quality/provider"
	providertesting "github.com/symflower/eval-dev-quality/provider/testing"
	"github.com/symflower/gota/dataframe"
	"github.com/symflower/gota/series"
)

var (
	// ErrEmptyResponseFromModel indicates the model returned an empty response.
	ErrEmptyResponseFromModel = errors.New("empty response from model")
)

// file represents a file with path and content.
type file struct {
	Path    string
	Content string
}

// testFiles holds common test files.
var testFiles = map[string]file{
	"plain": file{
		Path: "plain_test.go",
		Content: bytesutil.StringTrimIndentations(`
			package plain

			import "testing"

			func TestFunction(t *testing.T){}
		`),
	},
	"plain-with-assert": file{
		Path: "plain_test.go",
		Content: bytesutil.StringTrimIndentations(`
			package plain

			import (
				"testing"

				"github.com/stretchr/testify/assert"
			)

			func TestFunction(t *testing.T){
				assert.True(t, true)
			}
		`),
	},
}

func TestEvaluate(t *testing.T) {
	type testCase struct {
		Name string

		Before func(t *testing.T, logger *log.Logger, resultPath string)
		After  func(t *testing.T, logger *log.Logger, resultPath string)

		Context *Context

		ExpectedAssessments    metricstesting.AssessmentTuples
		ExpectedOutputValidate func(t *testing.T, output string, resultPath string)
		ExpectedResultFiles    map[string]func(t *testing.T, filePath string, data string)
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			temporaryPath := t.TempDir()

			logOutput, logger := log.Buffer()
			defer func() {
				log.CloseOpenLogFiles()

				if t.Failed() {
					t.Logf("Logging output: %s", logOutput.String())
				}
			}()

			resultPath := temporaryPath
			logger = logger.With(log.AttributeKeyResultPath, resultPath)

			tc.Context.Log = logger
			if tc.Context.APIReqestAttempts == 0 {
				tc.Context.APIReqestAttempts = 1
			}
			tc.Context.ResultPath = resultPath
			if tc.Context.TestdataPath == "" {
				tc.Context.TestdataPath = filepath.Join("..", "testdata")
			}
			if tc.Context.Runs == 0 {
				tc.Context.Runs = 1
			}

			if tc.Before != nil {
				tc.Before(t, logger, temporaryPath)
			}
			if tc.After != nil {
				defer tc.After(t, logger, temporaryPath)
			}

			Evaluate(tc.Context)

			csvData, err := os.ReadFile(filepath.Join(tc.Context.ResultPath, "evaluation.csv"))
			require.NoError(t, err)
			actualAssessmentTuples := reporttesting.ParseMetrics(t, string(csvData))
			for _, assessment := range actualAssessmentTuples {
				assessment.Assessment = metricstesting.Clean(assessment.Assessment)
			}

			assert.Equal(t, tc.ExpectedAssessments, actualAssessmentTuples)

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

	{
		languageGolang := &golang.Language{}
		mockedModelID := "testing-provider/empty-response-model"
		mockedQuery := providertesting.NewMockQuery(t)
		mockedModel := llm.NewModel(mockedQuery, mockedModelID)
		repositoryPath := filepath.Join("golang", "plain")

		validate(t, &testCase{
			Name: "Empty model response",

			Before: func(t *testing.T, logger *log.Logger, resultPath string) {
				queryResult1 := &provider.QueryResult{
					Message: "",
					GenerationInfo: &provider.GenerationInfo{
						TotalCost:              0.111111111,
						NativeTokensPrompt:     111,
						NativeTokensCompletion: 222,
					},
				}
				// Set up mocks, when test is running.
				mockedQuery.On("Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(queryResult1, nil).Once().After(10 * time.Millisecond) // Simulate a model response delay because our internal safety measures trigger when a query is done in 0 milliseconds.

				queryResult2 := &provider.QueryResult{
					Message: "",
					GenerationInfo: &provider.GenerationInfo{
						TotalCost:              0.222222222,
						NativeTokensPrompt:     333,
						NativeTokensCompletion: 444,
					},
				}
				// Set up mocks, when test is running.
				mockedQuery.On("Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(queryResult2, nil).Once().After(10 * time.Millisecond) // Simulate a model response delay because our internal safety measures trigger when a query is done in 0 milliseconds.
			},
			After: func(t *testing.T, logger *log.Logger, resultPath string) {
				mockedQuery.AssertNumberOfCalls(t, "Query", 2)
			},

			Context: &Context{
				Languages: []language.Language{
					&golang.Language{},
				},

				Models: []evalmodel.Model{
					mockedModel,
				},
				APIReqestAttempts: 3,

				RepositoryPaths: []string{
					repositoryPath,
				},
			},

			ExpectedAssessments: []*metricstesting.AssessmentTuple{
				&metricstesting.AssessmentTuple{
					Model:          mockedModel.ID(),
					Language:       languageGolang.ID(),
					RepositoryPath: repositoryPath,
					Case:           "plain.go",
					Task:           evaluatetask.IdentifierWriteTests,
					Assessment: map[metrics.AssessmentKey]float64{
						metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						metrics.AssessmentKeyResponseNoError:               1,
						metrics.AssessmentKeyCostsTokenActual:              0.111111111,
						metrics.AssessmentKeyNativeTokenInput:              111,
						metrics.AssessmentKeyNativeTokenOutput:             222,
					},
				},
				&metricstesting.AssessmentTuple{
					Model:          mockedModel.ID(),
					Language:       languageGolang.ID(),
					RepositoryPath: repositoryPath,
					Case:           "plain.go",
					Task:           evaluatetask.IdentifierWriteTestsSymflowerFix,
					Assessment: map[metrics.AssessmentKey]float64{
						metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						metrics.AssessmentKeyResponseNoError:               1,
						metrics.AssessmentKeyCostsTokenActual:              0.111111111,
						metrics.AssessmentKeyNativeTokenInput:              111,
						metrics.AssessmentKeyNativeTokenOutput:             222,
					},
				},
				&metricstesting.AssessmentTuple{
					Model:          mockedModel.ID(),
					Language:       languageGolang.ID(),
					RepositoryPath: repositoryPath,
					Case:           "plain.go",
					Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplate,
					Assessment: map[metrics.AssessmentKey]float64{
						metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						metrics.AssessmentKeyResponseNoError:               1,
						metrics.AssessmentKeyCostsTokenActual:              0.222222222,
						metrics.AssessmentKeyNativeTokenInput:              333,
						metrics.AssessmentKeyNativeTokenOutput:             444,
					},
				},
				&metricstesting.AssessmentTuple{
					Model:          mockedModel.ID(),
					Language:       languageGolang.ID(),
					RepositoryPath: repositoryPath,
					Case:           "plain.go",
					Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplateSymflowerFix,
					Assessment: map[metrics.AssessmentKey]float64{
						metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						metrics.AssessmentKeyResponseNoError:               1,
						metrics.AssessmentKeyCostsTokenActual:              0.222222222,
						metrics.AssessmentKeyNativeTokenInput:              333,
						metrics.AssessmentKeyNativeTokenOutput:             444,
					},
				},
			},
			ExpectedResultFiles: map[string]func(t *testing.T, filePath string, data string){
				"evaluation.log": nil,
				filepath.Join(string(evaluatetask.IdentifierWriteTests), log.CleanModelNameForFileSystem(mockedModelID), "golang", "golang", "plain", "evaluation.log"): func(t *testing.T, filePath, data string) {
					assert.Equal(t, 4, strings.Count(data, "no test files found"), "number of ocurrences of \"no test files found\" not matched")
				},
				"evaluation.csv": func(t *testing.T, filePath, data string) {
					assert.Lenf(t, strings.Split(data, "\n"), 6, "expected 6 lines: header, 4x entries and final new line:\n%s", data)
				},
			},
		})
	}

	t.Run("Failing model queries", func(t *testing.T) {
		{
			languageGolang := &golang.Language{}
			mockedModelID := "testing-provider/empty-response-model"
			mockedQuery := providertesting.NewMockQuery(t)
			mockedModel := llm.NewModel(mockedQuery, mockedModelID)
			repositoryPath := filepath.Join("golang", "plain")

			validate(t, &testCase{
				Name: "Single try fails",

				Before: func(t *testing.T, logger *log.Logger, resultPath string) {
					// Set up mocks, when test is running.
					mockedQuery.On("Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, ErrEmptyResponseFromModel)
				},
				After: func(t *testing.T, logger *log.Logger, resultPath string) {
					mockedQuery.AssertNumberOfCalls(t, "Query", 2)
				},

				Context: &Context{
					Languages: []language.Language{
						languageGolang,
					},

					Models: []evalmodel.Model{
						mockedModel,
					},
					APIReqestAttempts: 1,
				},

				ExpectedAssessments: []*metricstesting.AssessmentTuple{
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTests,
						Assessment: metrics.Assessments{
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerFix,
						Assessment: metrics.Assessments{
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplate,
						Assessment: metrics.Assessments{

							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplateSymflowerFix,
						Assessment: metrics.Assessments{
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						},
					},
				},
				ExpectedResultFiles: map[string]func(t *testing.T, filePath string, data string){
					"evaluation.log": nil,
					filepath.Join(string(evaluatetask.IdentifierWriteTests), log.CleanModelNameForFileSystem(mockedModelID), "golang", "golang", "plain", "evaluation.log"): func(t *testing.T, filePath, data string) {
						assert.Contains(t, data, ErrEmptyResponseFromModel.Error())
					},
					"evaluation.csv": nil,
				},
			})
		}
		{
			languageGolang := &golang.Language{}
			mockedModelID := "testing-provider/empty-response-model"
			mockedQuery := providertesting.NewMockQuery(t)
			mockedModel := llm.NewModel(mockedQuery, mockedModelID)
			repositoryPath := filepath.Join("golang", "plain")

			validate(t, &testCase{
				Name: "Success after retry",

				Before: func(t *testing.T, logger *log.Logger, resultPath string) {
					queryResult := &provider.QueryResult{
						Message: "model-response",
					}
					// Set up mocks, when test is running.
					mockedQuery.On("Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, ErrEmptyResponseFromModel).Once()
					mockedQuery.On("Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(queryResult, nil).Once().After(10 * time.Millisecond) // Simulate a model response delay because our internal safety measures trigger when a query is done in 0 milliseconds.
					mockedQuery.On("Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, ErrEmptyResponseFromModel).Once()
					mockedQuery.On("Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(queryResult, nil).Once().After(10 * time.Millisecond) // Simulate a model response delay because our internal safety measures trigger when a query is done in 0 milliseconds.
				},
				After: func(t *testing.T, logger *log.Logger, resultPath string) {
					mockedQuery.AssertNumberOfCalls(t, "Query", 4)
				},

				Context: &Context{
					Languages: []language.Language{
						&golang.Language{},
					},

					Models: []evalmodel.Model{
						mockedModel,
					},
					APIReqestAttempts: 3,

					RepositoryPaths: []string{
						repositoryPath,
					},
				},

				ExpectedAssessments: []*metricstesting.AssessmentTuple{
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTests,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecutedMaximumReachable:      1,
							metrics.AssessmentKeyGenerateTestsForFileCharacterCount: 14,
							metrics.AssessmentKeyResponseCharacterCount:             14,
							metrics.AssessmentKeyResponseNoError:                    1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerFix,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecutedMaximumReachable:      1,
							metrics.AssessmentKeyGenerateTestsForFileCharacterCount: 14,
							metrics.AssessmentKeyResponseCharacterCount:             14,
							metrics.AssessmentKeyResponseNoError:                    1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplate,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecutedMaximumReachable:      1,
							metrics.AssessmentKeyGenerateTestsForFileCharacterCount: 14,
							metrics.AssessmentKeyResponseCharacterCount:             14,
							metrics.AssessmentKeyResponseNoError:                    1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplateSymflowerFix,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecutedMaximumReachable:      1,
							metrics.AssessmentKeyGenerateTestsForFileCharacterCount: 14,
							metrics.AssessmentKeyResponseCharacterCount:             14,
							metrics.AssessmentKeyResponseNoError:                    1,
						},
					},
				},
				ExpectedResultFiles: map[string]func(t *testing.T, filePath string, data string){
					"evaluation.log": nil,
					filepath.Join(string(evaluatetask.IdentifierWriteTests), log.CleanModelNameForFileSystem(mockedModelID), "golang", "golang", "plain", "evaluation.log"): func(t *testing.T, filePath, data string) {
						assert.Contains(t, data, "\"msg\":\"API request attempt failed\",\"count\":1,\"total\":3,\"error\":\""+ErrEmptyResponseFromModel.Error())
					},
					"evaluation.csv": nil,
				},
			})
		}
		{
			languageGolang := &golang.Language{}
			mockedModelID := "testing-provider/empty-response-model"
			mockedQuery := providertesting.NewMockQuery(t)
			mockedModel := llm.NewModel(mockedQuery, mockedModelID)
			repositoryPath := filepath.Join("golang", "plain")

			validate(t, &testCase{
				Name: "Immediate success",

				Before: func(t *testing.T, logger *log.Logger, resultPath string) {
					queryResult := &provider.QueryResult{
						Message: "model-response",
						Usage: openai.Usage{
							PromptTokens:     123,
							CompletionTokens: 456,
						},
					}
					// Set up mocks, when test is running.
					mockedQuery.On("Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(queryResult, nil).After(10 * time.Millisecond) // Simulate a model response delay because our internal safety measures trigger when a query is done in 0 milliseconds.
				},
				After: func(t *testing.T, logger *log.Logger, resultPath string) {
					mockedQuery.AssertNumberOfCalls(t, "Query", 2)
				},

				Context: &Context{
					Languages: []language.Language{
						&golang.Language{},
					},

					Models: []evalmodel.Model{
						mockedModel,
					},
					APIReqestAttempts: 3,

					RepositoryPaths: []string{
						repositoryPath,
					},
				},

				ExpectedAssessments: []*metricstesting.AssessmentTuple{
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTests,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecutedMaximumReachable:      1,
							metrics.AssessmentKeyGenerateTestsForFileCharacterCount: 14,
							metrics.AssessmentKeyResponseCharacterCount:             14,
							metrics.AssessmentKeyResponseNoError:                    1,
							metrics.AssessmentKeyTokenInput:                         123,
							metrics.AssessmentKeyTokenOutput:                        456,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerFix,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecutedMaximumReachable:      1,
							metrics.AssessmentKeyGenerateTestsForFileCharacterCount: 14,
							metrics.AssessmentKeyResponseCharacterCount:             14,
							metrics.AssessmentKeyResponseNoError:                    1,
							metrics.AssessmentKeyTokenInput:                         123,
							metrics.AssessmentKeyTokenOutput:                        456,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplate,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecutedMaximumReachable:      1,
							metrics.AssessmentKeyGenerateTestsForFileCharacterCount: 14,
							metrics.AssessmentKeyResponseCharacterCount:             14,
							metrics.AssessmentKeyResponseNoError:                    1,
							metrics.AssessmentKeyTokenInput:                         123,
							metrics.AssessmentKeyTokenOutput:                        456,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplateSymflowerFix,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecutedMaximumReachable:      1,
							metrics.AssessmentKeyGenerateTestsForFileCharacterCount: 14,
							metrics.AssessmentKeyResponseCharacterCount:             14,
							metrics.AssessmentKeyResponseNoError:                    1,
							metrics.AssessmentKeyTokenInput:                         123,
							metrics.AssessmentKeyTokenOutput:                        456,
						},
					},
				},
				ExpectedResultFiles: map[string]func(t *testing.T, filePath string, data string){
					"evaluation.log": nil,
					filepath.Join(string(evaluatetask.IdentifierWriteTests), log.CleanModelNameForFileSystem(mockedModelID), "golang", "golang", "plain", "evaluation.log"): func(t *testing.T, filePath, data string) {
						assert.Contains(t, data, "DONE 0 tests, 1 failure, 1 error")
					},
					"evaluation.csv": nil,
				},
			})
		}
	})

	t.Run("Failing basic language checks should exclude model", func(t *testing.T) {
		repositoryPlainPath := filepath.Join("golang", "plain")
		repositoryNextPath := filepath.Join("golang", "next")

		temporaryTestdataPath := t.TempDir()
		assert.NoError(t, osutil.CopyTree(filepath.Join("..", "testdata", repositoryPlainPath), filepath.Join(temporaryTestdataPath, repositoryPlainPath)))
		assert.NoError(t, osutil.CopyTree(filepath.Join("..", "testdata", repositoryPlainPath), filepath.Join(temporaryTestdataPath, repositoryNextPath)))
		repositoryNextConfigPath := filepath.Join(temporaryTestdataPath, repositoryNextPath, "go.mod")
		d, err := os.ReadFile(repositoryNextConfigPath)
		require.NoError(t, err)
		d = bytes.ReplaceAll(d, []byte("plain"), []byte("next"))
		require.NoError(t, os.WriteFile(repositoryNextConfigPath, d, 0))

		generateTestsForFilePlainError := errors.New("generateTestsForFile error")

		generateSuccess := func(mockedModel *modeltesting.MockModelCapabilityWriteTests) *mock.Call {
			return mockedModel.RegisterGenerateSuccess(t, testFiles["plain"].Path, testFiles["plain"].Content, metricstesting.AssessmentsWithProcessingTime)
		}
		generateError := func(mockedModel *modeltesting.MockModelCapabilityWriteTests) *mock.Call {
			return mockedModel.RegisterGenerateError(generateTestsForFilePlainError)
		}

		{
			languageGolang := &golang.Language{}
			mockedModelID := "mocked-generation-model"
			mockedModel := modeltesting.NewMockCapabilityWriteTestsNamed(t, mockedModelID)

			validate(t, &testCase{
				Name: "Problems of previous runs shouldn't cancel successive runs",

				Before: func(t *testing.T, logger *log.Logger, resultPath string) {
					// Set up mocks, when test is running.
					{
						// Succeed on both "plain" runs.
						generateSuccess(mockedModel).Once() // Without template.
						generateSuccess(mockedModel).Once() // With template.
						generateSuccess(mockedModel).Once() // Without template.
						generateSuccess(mockedModel).Once() // With template.

						// Error on the first run for the "next" repository.
						generateError(mockedModel).Once() // Without template.
						generateError(mockedModel).Once() // With template.
						// Succeed on the second run for the "next" repository.
						generateSuccess(mockedModel).Once() // Without template.
						generateSuccess(mockedModel).Once() // With template.
					}
				},
				After: func(t *testing.T, logger *log.Logger, resultPath string) {
					mockedModel.MockCapabilityWriteTests.AssertNumberOfCalls(t, "WriteTests", 8)
				},

				Context: &Context{
					Languages: []language.Language{
						&golang.Language{},
					},

					Models: []evalmodel.Model{
						mockedModel,
					},

					RepositoryPaths: []string{
						repositoryPlainPath,
						repositoryNextPath,
					},
					TestdataPath: temporaryTestdataPath,

					Runs: 2,
				},

				ExpectedAssessments: []*metricstesting.AssessmentTuple{
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPlainPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTests,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPlainPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerFix,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPlainPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplate,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPlainPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplateSymflowerFix,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPlainPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTests,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPlainPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerFix,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPlainPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplate,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPlainPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplateSymflowerFix,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryNextPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTests,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryNextPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerFix,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryNextPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplate,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						},
					}, &metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryNextPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplateSymflowerFix,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryNextPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTests,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryNextPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerFix,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryNextPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplate,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					}, &metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryNextPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplateSymflowerFix,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
				},
				ExpectedResultFiles: map[string]func(t *testing.T, filePath string, data string){
					"evaluation.log": nil,
					filepath.Join(string(evaluatetask.IdentifierWriteTests), log.CleanModelNameForFileSystem(mockedModelID), "golang", "golang", "plain", "evaluation.log"): nil,
					filepath.Join(string(evaluatetask.IdentifierWriteTests), log.CleanModelNameForFileSystem(mockedModelID), "golang", "golang", "next", "evaluation.log"):  nil,
					"evaluation.csv": nil,
				},
			})
		}
		{
			languageGolang := &golang.Language{}
			mockedModelID := "mocked-generation-model"
			mockedModel := modeltesting.NewMockCapabilityWriteTestsNamed(t, mockedModelID)

			validate(t, &testCase{
				Name: "Solving basic checks once is enough",

				Before: func(t *testing.T, logger *log.Logger, resultPath string) {
					// Set up mocks, when test is running.
					{
						// Succeed on only one "plain" run.
						generateError(mockedModel).Once()   // Without template.
						generateSuccess(mockedModel).Once() // With template.
						generateError(mockedModel).Once()   // Without template.
						generateError(mockedModel).Once()   // With template.

						// Succeed on both "next" runs.
						generateSuccess(mockedModel).Once() // Without template.
						generateSuccess(mockedModel).Once() // With template.
						generateSuccess(mockedModel).Once() // Without template.
						generateSuccess(mockedModel).Once() // With template
					}
				},
				After: func(t *testing.T, logger *log.Logger, resultPath string) {
					mockedModel.MockCapabilityWriteTests.AssertNumberOfCalls(t, "WriteTests", 8)
				},

				Context: &Context{
					Languages: []language.Language{
						&golang.Language{},
					},

					Models: []evalmodel.Model{
						mockedModel,
					},

					RepositoryPaths: []string{
						repositoryPlainPath,
						repositoryNextPath,
					},
					TestdataPath: temporaryTestdataPath,

					Runs: 2,
				},

				ExpectedAssessments: []*metricstesting.AssessmentTuple{
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPlainPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTests,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPlainPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerFix,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPlainPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplate,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPlainPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplateSymflowerFix,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPlainPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTests,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPlainPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerFix,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPlainPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplate,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPlainPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplateSymflowerFix,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryNextPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTests,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryNextPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerFix,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryNextPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplate,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryNextPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplateSymflowerFix,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryNextPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTests,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryNextPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerFix,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryNextPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplate,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryNextPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplateSymflowerFix,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
				},
				ExpectedResultFiles: map[string]func(t *testing.T, filePath string, data string){
					"evaluation.log": nil,
					filepath.Join(string(evaluatetask.IdentifierWriteTests), log.CleanModelNameForFileSystem(mockedModelID), "golang", "golang", "plain", "evaluation.log"): nil,
					filepath.Join(string(evaluatetask.IdentifierWriteTests), log.CleanModelNameForFileSystem(mockedModelID), "golang", "golang", "next", "evaluation.log"):  nil,
					"evaluation.csv": nil,
				},
			})
		}
		{
			languageGolang := &golang.Language{}
			mockedModelID := "mocked-generation-model"
			mockedModel := modeltesting.NewMockCapabilityWriteTestsNamed(t, mockedModelID)

			validate(t, &testCase{
				Name: "Never solving basic checks leads to exclusion",

				Before: func(t *testing.T, logger *log.Logger, resultPath string) {
					// Set up mocks, when test is running.
					{
						// Error on every "plain" run.
						generateError(mockedModel).Once() // Without template.
						generateError(mockedModel).Once() // With template.
						generateError(mockedModel).Once() // Without template.
						generateError(mockedModel).Once() // With template.
					}
				},
				After: func(t *testing.T, logger *log.Logger, resultPath string) {
					mockedModel.MockCapabilityWriteTests.AssertNumberOfCalls(t, "WriteTests", 4)
				},

				Context: &Context{
					Languages: []language.Language{
						&golang.Language{},
					},

					Models: []evalmodel.Model{
						mockedModel,
					},

					RepositoryPaths: []string{
						repositoryPlainPath,
						repositoryNextPath,
					},
					TestdataPath: temporaryTestdataPath,

					Runs: 2,
				},

				ExpectedAssessments: []*metricstesting.AssessmentTuple{
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPlainPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTests,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPlainPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerFix,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPlainPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplate,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPlainPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplateSymflowerFix,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPlainPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTests,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPlainPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerFix,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPlainPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplate,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPlainPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplateSymflowerFix,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						},
					},
				},
				ExpectedResultFiles: map[string]func(t *testing.T, filePath string, data string){
					"evaluation.log": nil,
					filepath.Join(string(evaluatetask.IdentifierWriteTests), log.CleanModelNameForFileSystem(mockedModelID), "golang", "golang", "plain", "evaluation.log"): nil,
					"evaluation.csv": nil,
				},
			})
		}
	})
	t.Run("Runs", func(t *testing.T) {
		generateSuccess := func(mockedModel *modeltesting.MockModelCapabilityWriteTests) {
			mockedModel.RegisterGenerateSuccess(t, testFiles["plain"].Path, testFiles["plain"].Content, metricstesting.AssessmentsWithProcessingTime)
		}
		{
			languageGolang := &golang.Language{}
			mockedModelID := "mocked-generation-model"
			mockedModel := modeltesting.NewMockCapabilityWriteTestsNamed(t, mockedModelID)

			repositoryPath := filepath.Join("golang", "plain")
			validate(t, &testCase{
				Name: "Interleaved",

				Before: func(t *testing.T, logger *log.Logger, resultPath string) {
					generateSuccess(mockedModel)
				},

				Context: &Context{
					Languages: []language.Language{
						&golang.Language{},
					},

					Models: []evalmodel.Model{
						mockedModel,
					},

					RepositoryPaths: []string{
						repositoryPath,
					},

					RunIDStartsAt:  11,
					Runs:           3,
					RunsSequential: false,
				},

				ExpectedAssessments: []*metricstesting.AssessmentTuple{
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTests,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerFix,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplate,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplateSymflowerFix,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTests,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerFix,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplate,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplateSymflowerFix,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTests,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerFix,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplate,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplateSymflowerFix,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
				},
				ExpectedResultFiles: map[string]func(t *testing.T, filePath string, data string){
					"evaluation.log": func(t *testing.T, filePath string, data string) {
						assert.Contains(t, data, "\"msg\":\"starting run\",\"count\":1,\"total\":3}")
						assert.Contains(t, data, "\"msg\":\"starting run\",\"count\":2,\"total\":3}")
						assert.Contains(t, data, "\"msg\":\"starting run\",\"count\":3,\"total\":3}")
						assert.NotRegexp(t, `\\\"msg\\\":\\\"starting run\\\",\\\"count\\\":\d+,\\\"total\\\":\d+,`, data)

						assert.Equal(t, 1, strings.Count(data, "creating temporary repository"), "create only one temporary repository")
					},
					filepath.Join(string(evaluatetask.IdentifierWriteTests), log.CleanModelNameForFileSystem(mockedModelID), "golang", "golang", "plain", "evaluation.log"): nil,
					"evaluation.csv": func(t *testing.T, filePath string, data string) {
						dataFrame := dataframe.ReadCSV(strings.NewReader(data))
						assert.NoError(t, dataFrame.Err)

						expectedColumnRun := series.New([]int{11, 11, 11, 11, 12, 12, 12, 12, 13, 13, 13, 13}, series.Int, "run")
						actualColumnRun := dataFrame.Col("run")
						assert.Equal(t, expectedColumnRun, actualColumnRun)
					},
				},
			})
		}
		{
			languageGolang := &golang.Language{}
			mockedModelID := "mocked-generation-model"
			mockedModel := modeltesting.NewMockCapabilityWriteTestsNamed(t, mockedModelID)

			repositoryPath := filepath.Join("golang", "plain")
			validate(t, &testCase{
				Name: "Sequential",

				Before: func(t *testing.T, logger *log.Logger, resultPath string) {
					generateSuccess(mockedModel)
				},

				Context: &Context{
					Languages: []language.Language{
						&golang.Language{},
					},

					Models: []evalmodel.Model{
						mockedModel,
					},

					RepositoryPaths: []string{
						repositoryPath,
					},

					RunIDStartsAt:  21,
					Runs:           3,
					RunsSequential: true,
				},

				ExpectedAssessments: []*metricstesting.AssessmentTuple{
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTests,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerFix,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplate,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplateSymflowerFix,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTests,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerFix,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplate,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplateSymflowerFix,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTests,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerFix,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplate,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplateSymflowerFix,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
				},
				ExpectedResultFiles: map[string]func(t *testing.T, filePath string, data string){
					"evaluation.log": func(t *testing.T, filePath string, data string) {
						assert.Equal(t, 1, strings.Count(data, "creating temporary repository"), "create only one temporary repository")
					},
					filepath.Join(string(evaluatetask.IdentifierWriteTests), log.CleanModelNameForFileSystem(mockedModelID), "golang", "golang", "plain", "evaluation.log"): func(t *testing.T, filePath string, data string) {
						assert.Contains(t, data, "\"msg\":\"starting run\",\"count\":1,\"total\":3,")
						assert.Contains(t, data, "\"msg\":\"starting run\",\"count\":2,\"total\":3,")
						assert.Contains(t, data, "\"msg\":\"starting run\",\"count\":3,\"total\":3,")
						assert.NotRegexp(t, `\\\"msg\\\":\\\"starting run\\\",\\\"count\\\":\d+,\\\"total\\\":\d+\}`, data)
					},
					"evaluation.csv": func(t *testing.T, filePath string, data string) {
						dataFrame := dataframe.ReadCSV(strings.NewReader(data))
						assert.NoError(t, dataFrame.Err)

						expectedColumnRun := series.New([]int{21, 21, 21, 21, 22, 22, 22, 22, 23, 23, 23, 23}, series.Int, "run")
						actualColumnRun := dataFrame.Col("run")
						assert.Equal(t, expectedColumnRun, actualColumnRun)
					},
				},
			})
		}
	})

	t.Run("Preloading", func(t *testing.T) {
		generateSuccess := func(mockedModel *modeltesting.MockModelCapabilityWriteTests) {
			mockedModel.RegisterGenerateSuccess(t, testFiles["plain"].Path, testFiles["plain"].Content, metricstesting.AssessmentsWithProcessingTime)
		}

		{
			// Setup provider and model mocking.
			languageGolang := &golang.Language{}
			mockedModelID := "testing-provider/testing-model"
			mockedModel := modeltesting.NewMockCapabilityWriteTestsNamed(t, mockedModelID)
			mockedProviderID := "testing-provider"
			mockedProvider := providertesting.NewMockProviderNamedWithModels(t, mockedProviderID, []evalmodel.Model{mockedModel})
			mockedLoader := providertesting.NewMockLoader(t)
			embeddedProvider := &struct {
				provider.Provider
				provider.Loader
			}{
				Provider: mockedProvider,
				Loader:   mockedLoader,
			}
			repositoryPath := filepath.Join("golang", "plain")

			validate(t, &testCase{
				Name: "Once for combined runs",

				Before: func(t *testing.T, logger *log.Logger, resultPath string) {
					generateSuccess(mockedModel)
					mockedLoader.On("Load", mockedModelID).Return(nil)
					mockedLoader.On("Unload", mockedModelID).Return(nil)
				},
				After: func(t *testing.T, logger *log.Logger, resultPath string) {
					delete(provider.Providers, mockedProviderID)

					mockedLoader.AssertNumberOfCalls(t, "Load", 1)
					mockedLoader.AssertNumberOfCalls(t, "Unload", 1)
				},

				Context: &Context{
					Languages: []language.Language{
						languageGolang,
					},

					Models: []evalmodel.Model{
						mockedModel,
					},
					ProviderForModel: map[evalmodel.Model]provider.Provider{
						mockedModel: embeddedProvider,
					},

					RepositoryPaths: []string{
						repositoryPath,
					},

					Runs:           3,
					RunsSequential: true,
				},

				ExpectedAssessments: []*metricstesting.AssessmentTuple{
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTests,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerFix,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplate,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplateSymflowerFix,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTests,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerFix,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplate,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplateSymflowerFix,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTests,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerFix,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplate,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplateSymflowerFix,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
				},
				ExpectedResultFiles: map[string]func(t *testing.T, filePath string, data string){
					"evaluation.log": nil,
					filepath.Join(string(evaluatetask.IdentifierWriteTests), log.CleanModelNameForFileSystem(mockedModelID), "golang", "golang", "plain", "evaluation.log"): nil,
					"evaluation.csv": nil,
				},
			})
		}
		{
			// Setup provider and model mocking.
			languageGolang := &golang.Language{}
			mockedModelID := "testing-provider/testing-model"
			mockedModel := modeltesting.NewMockCapabilityWriteTestsNamed(t, mockedModelID)
			mockedProviderID := "testing-provider"
			mockedProvider := providertesting.NewMockProviderNamedWithModels(t, mockedProviderID, []evalmodel.Model{mockedModel})
			mockedLoader := providertesting.NewMockLoader(t)
			embeddedProvider := &struct {
				provider.Provider
				provider.Loader
			}{
				Provider: mockedProvider,
				Loader:   mockedLoader,
			}
			repositoryPath := filepath.Join("golang", "plain")
			validate(t, &testCase{
				Name: "Multiple times for interleaved runs",

				Before: func(t *testing.T, logger *log.Logger, resultPath string) {
					generateSuccess(mockedModel)
					mockedLoader.On("Load", mockedModelID).Return(nil)
					mockedLoader.On("Unload", mockedModelID).Return(nil)
				},
				After: func(t *testing.T, logger *log.Logger, resultPath string) {
					delete(provider.Providers, "testing-provider")

					mockedLoader.AssertNumberOfCalls(t, "Load", 3)
					mockedLoader.AssertNumberOfCalls(t, "Unload", 3)
				},

				Context: &Context{
					Languages: []language.Language{
						languageGolang,
					},

					Models: []evalmodel.Model{
						mockedModel,
					},
					ProviderForModel: map[evalmodel.Model]provider.Provider{
						mockedModel: embeddedProvider,
					},

					RepositoryPaths: []string{
						repositoryPath,
					},

					Runs: 3,
				},

				ExpectedAssessments: []*metricstesting.AssessmentTuple{
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTests,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerFix,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplate,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplateSymflowerFix,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTests,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerFix,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplate,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplateSymflowerFix,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTests,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerFix,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplate,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
					&metricstesting.AssessmentTuple{
						Model:          mockedModel.ID(),
						Language:       languageGolang.ID(),
						RepositoryPath: repositoryPath,
						Case:           "plain.go",
						Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplateSymflowerFix,
						Assessment: map[metrics.AssessmentKey]float64{
							metrics.AssessmentKeyFilesExecuted:                 1,
							metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
							metrics.AssessmentKeyResponseNoError:               1,
						},
					},
				},
				ExpectedResultFiles: map[string]func(t *testing.T, filePath string, data string){
					"evaluation.log": nil,
					filepath.Join(string(evaluatetask.IdentifierWriteTests), log.CleanModelNameForFileSystem(mockedModelID), "golang", "golang", "plain", "evaluation.log"): nil,
					"evaluation.csv": nil,
				},
			})
		}
	})
	{
		// Setup provider and model mocking.
		languageGolang := &golang.Language{}
		mockedModelID := "testing-provider/testing-model"
		mockedModel := modeltesting.NewMockCapabilityWriteTestsNamed(t, mockedModelID)

		repositoryPath := filepath.Join("golang", "plain")

		validate(t, &testCase{
			Name: "Download Go dependencies",

			Before: func(t *testing.T, logger *log.Logger, resultPath string) {
				mockedModel.RegisterGenerateSuccess(t, testFiles["plain-with-assert"].Path, testFiles["plain-with-assert"].Content, metricstesting.AssessmentsWithProcessingTime)
			},

			Context: &Context{
				Languages: []language.Language{
					languageGolang,
				},

				Models: []evalmodel.Model{
					mockedModel,
				},

				RepositoryPaths: []string{
					repositoryPath,
				},

				Runs: 1,
			},

			ExpectedAssessments: []*metricstesting.AssessmentTuple{
				&metricstesting.AssessmentTuple{
					Model:          mockedModel.ID(),
					Language:       languageGolang.ID(),
					RepositoryPath: repositoryPath,
					Case:           "plain.go",
					Task:           evaluatetask.IdentifierWriteTests,
					Assessment: map[metrics.AssessmentKey]float64{
						metrics.AssessmentKeyFilesExecuted:                 1,
						metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						metrics.AssessmentKeyResponseNoError:               1,
					},
				},
				&metricstesting.AssessmentTuple{
					Model:          mockedModel.ID(),
					Language:       languageGolang.ID(),
					RepositoryPath: repositoryPath,
					Case:           "plain.go",
					Task:           evaluatetask.IdentifierWriteTestsSymflowerFix,
					Assessment: map[metrics.AssessmentKey]float64{
						metrics.AssessmentKeyFilesExecuted:                 1,
						metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						metrics.AssessmentKeyResponseNoError:               1,
					},
				},
				&metricstesting.AssessmentTuple{
					Model:          mockedModel.ID(),
					Language:       languageGolang.ID(),
					RepositoryPath: repositoryPath,
					Case:           "plain.go",
					Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplate,
					Assessment: map[metrics.AssessmentKey]float64{
						metrics.AssessmentKeyFilesExecuted:                 1,
						metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						metrics.AssessmentKeyResponseNoError:               1,
					},
				},
				&metricstesting.AssessmentTuple{
					Model:          mockedModel.ID(),
					Language:       languageGolang.ID(),
					RepositoryPath: repositoryPath,
					Case:           "plain.go",
					Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplateSymflowerFix,
					Assessment: map[metrics.AssessmentKey]float64{
						metrics.AssessmentKeyFilesExecuted:                 1,
						metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						metrics.AssessmentKeyResponseNoError:               1,
					},
				},
			},
			ExpectedResultFiles: map[string]func(t *testing.T, filePath string, data string){
				"evaluation.log": nil,
				filepath.Join(string(evaluatetask.IdentifierWriteTests), log.CleanModelNameForFileSystem(mockedModelID), "golang", "golang", "plain", "evaluation.log"): nil,
				"evaluation.csv": nil,
			},
		})
	}
	{
		// Setup provider and model mocking.
		languageGolang := &golang.Language{}
		mockedModelID := "testing-provider/testing-model"
		mockedModel := modeltesting.NewMockCapabilityWriteTestsNamed(t, mockedModelID)

		repositoryPathPlain := filepath.Join("golang", "plain")
		repositoryPathSomePlain := filepath.Join("golang", "some-plain")
		temporaryTestdataPath := t.TempDir()
		require.NoError(t, osutil.CopyTree(filepath.Join("..", "testdata", repositoryPathPlain), filepath.Join(temporaryTestdataPath, repositoryPathSomePlain)))
		require.NoError(t, osutil.CopyTree(filepath.Join("..", "testdata", repositoryPathPlain), filepath.Join(temporaryTestdataPath, repositoryPathPlain)))

		validate(t, &testCase{
			Name: "Repository with -plain suffix",

			Before: func(t *testing.T, logger *log.Logger, resultPath string) {
				mockedModel.RegisterGenerateSuccess(t, testFiles["plain"].Path, testFiles["plain"].Content, metricstesting.AssessmentsWithProcessingTime)
			},

			Context: &Context{
				Languages: []language.Language{
					languageGolang,
				},

				Models: []evalmodel.Model{
					mockedModel,
				},

				RepositoryPaths: []string{
					repositoryPathSomePlain,
				},

				TestdataPath: temporaryTestdataPath,

				Runs: 1,
			},

			ExpectedAssessments: []*metricstesting.AssessmentTuple{
				&metricstesting.AssessmentTuple{
					Model:          mockedModel.ID(),
					Language:       languageGolang.ID(),
					RepositoryPath: repositoryPathPlain,
					Case:           "plain.go",
					Task:           evaluatetask.IdentifierWriteTests,
					Assessment: map[metrics.AssessmentKey]float64{
						metrics.AssessmentKeyFilesExecuted:                 1,
						metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						metrics.AssessmentKeyResponseNoError:               1,
					},
				},
				&metricstesting.AssessmentTuple{
					Model:          mockedModel.ID(),
					Language:       languageGolang.ID(),
					RepositoryPath: repositoryPathPlain,
					Case:           "plain.go",
					Task:           evaluatetask.IdentifierWriteTestsSymflowerFix,
					Assessment: map[metrics.AssessmentKey]float64{
						metrics.AssessmentKeyFilesExecuted:                 1,
						metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						metrics.AssessmentKeyResponseNoError:               1,
					},
				},
				&metricstesting.AssessmentTuple{
					Model:          mockedModel.ID(),
					Language:       languageGolang.ID(),
					RepositoryPath: repositoryPathPlain,
					Case:           "plain.go",
					Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplate,
					Assessment: map[metrics.AssessmentKey]float64{
						metrics.AssessmentKeyFilesExecuted:                 1,
						metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						metrics.AssessmentKeyResponseNoError:               1,
					},
				},
				&metricstesting.AssessmentTuple{
					Model:          mockedModel.ID(),
					Language:       languageGolang.ID(),
					RepositoryPath: repositoryPathPlain,
					Case:           "plain.go",
					Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplateSymflowerFix,
					Assessment: map[metrics.AssessmentKey]float64{
						metrics.AssessmentKeyFilesExecuted:                 1,
						metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						metrics.AssessmentKeyResponseNoError:               1,
					},
				},
				&metricstesting.AssessmentTuple{
					Model:          mockedModel.ID(),
					Language:       languageGolang.ID(),
					RepositoryPath: repositoryPathSomePlain,
					Case:           "plain.go",
					Task:           evaluatetask.IdentifierWriteTests,
					Assessment: map[metrics.AssessmentKey]float64{
						metrics.AssessmentKeyFilesExecuted:                 1,
						metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						metrics.AssessmentKeyResponseNoError:               1,
					},
				},
				&metricstesting.AssessmentTuple{
					Model:          mockedModel.ID(),
					Language:       languageGolang.ID(),
					RepositoryPath: repositoryPathSomePlain,
					Case:           "plain.go",
					Task:           evaluatetask.IdentifierWriteTestsSymflowerFix,
					Assessment: map[metrics.AssessmentKey]float64{
						metrics.AssessmentKeyFilesExecuted:                 1,
						metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						metrics.AssessmentKeyResponseNoError:               1,
					},
				},
				&metricstesting.AssessmentTuple{
					Model:          mockedModel.ID(),
					Language:       languageGolang.ID(),
					RepositoryPath: repositoryPathSomePlain,
					Case:           "plain.go",
					Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplate,
					Assessment: map[metrics.AssessmentKey]float64{
						metrics.AssessmentKeyFilesExecuted:                 1,
						metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						metrics.AssessmentKeyResponseNoError:               1,
					},
				},
				&metricstesting.AssessmentTuple{
					Model:          mockedModel.ID(),
					Language:       languageGolang.ID(),
					RepositoryPath: repositoryPathSomePlain,
					Case:           "plain.go",
					Task:           evaluatetask.IdentifierWriteTestsSymflowerTemplateSymflowerFix,
					Assessment: map[metrics.AssessmentKey]float64{
						metrics.AssessmentKeyFilesExecuted:                 1,
						metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						metrics.AssessmentKeyResponseNoError:               1,
					},
				},
			},
			ExpectedResultFiles: map[string]func(t *testing.T, filePath string, data string){
				"evaluation.log": nil,
				filepath.Join(string(evaluatetask.IdentifierWriteTests), log.CleanModelNameForFileSystem(mockedModelID), "golang", "golang", "plain", "evaluation.log"):      nil,
				filepath.Join(string(evaluatetask.IdentifierWriteTests), log.CleanModelNameForFileSystem(mockedModelID), "golang", "golang", "some-plain", "evaluation.log"): nil,
				"evaluation.csv": nil,
			},
		})
	}
}
