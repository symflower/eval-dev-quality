package evaluate

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/zimmski/osutil"

	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	"github.com/symflower/eval-dev-quality/evaluate/report"
	"github.com/symflower/eval-dev-quality/language"
	"github.com/symflower/eval-dev-quality/language/golang"
	"github.com/symflower/eval-dev-quality/log"
	evalmodel "github.com/symflower/eval-dev-quality/model"
	"github.com/symflower/eval-dev-quality/model/llm"
	modeltesting "github.com/symflower/eval-dev-quality/model/testing"
	providertesting "github.com/symflower/eval-dev-quality/provider/testing"
)

func TestEvaluate(t *testing.T) {
	type testCase struct {
		Name string

		Before func(t *testing.T, logger *log.Logger, resultPath string)
		After  func(t *testing.T, logger *log.Logger, resultPath string)

		Context *Context

		ExpectedAssessments    report.AssessmentPerModelPerLanguagePerRepository
		ExpectedTotalScore     uint64
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

			tc.Context.Log = logger
			if tc.Context.QueryAttempts == 0 {
				tc.Context.QueryAttempts = 1
			}
			tc.Context.ResultPath = temporaryPath
			tc.Context.TestdataPath = filepath.Join("..", "testdata")
			if tc.Context.Runs == 0 {
				tc.Context.Runs = 1
			}

			if tc.Before != nil {
				tc.Before(t, logger, temporaryPath)
			}
			if tc.After != nil {
				defer tc.After(t, logger, temporaryPath)
			}

			actualAssessments, actualTotalScore := Evaluate(tc.Context)

			// Normalize assessments.
			for _, ls := range actualAssessments {
				for _, rs := range ls {
					for _, a := range rs {
						if v, ok := a[metrics.AssessmentKeyProcessingTime]; ok {
							if assert.Greater(t, v, uint64(0)) {
								delete(a, metrics.AssessmentKeyProcessingTime)
							}
						}
					}
				}
			}

			assert.Equal(t, tc.ExpectedAssessments, actualAssessments)
			assert.Equal(t, tc.ExpectedTotalScore, actualTotalScore)

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
		mockedModel := modeltesting.NewMockModelNamed(t, "empty-response-model")

		validate(t, &testCase{
			Name: "Empty model responses are errors",

			Before: func(t *testing.T, logger *log.Logger, resultPath string) {
				// Set up mocks, when test is running.
				mockedModel.On("GenerateTestsForFile", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("empty response from model"))
			},

			Context: &Context{
				Languages: []language.Language{
					&golang.Language{},
				},

				Models: []evalmodel.Model{
					mockedModel,
				},
			},

			ExpectedAssessments: map[evalmodel.Model]map[language.Language]map[string]metrics.Assessments{
				mockedModel: map[language.Language]map[string]metrics.Assessments{
					languageGolang: map[string]metrics.Assessments{},
				},
			},
			ExpectedTotalScore: 1,
			ExpectedResultFiles: map[string]func(t *testing.T, filePath string, data string){
				filepath.Join(mockedModel.ID(), "golang", "golang", "plain.log"): nil,
			},
		})
	}

	t.Run("Failying model queries", func(t *testing.T) {
		{
			languageGolang := &golang.Language{}
			mockedModelID := "testing-provider/empty-response-model"
			mockedQuery := providertesting.NewMockQuery(t)
			mockedModel := llm.NewModel(mockedQuery, mockedModelID)

			validate(t, &testCase{
				Name: "Single try fails",

				Before: func(t *testing.T, logger *log.Logger, resultPath string) {
					// Set up mocks, when test is running.
					mockedQuery.On("Query", mock.Anything, mockedModelID, mock.Anything).Return("", errors.New("empty response from model"))
				},
				After: func(t *testing.T, logger *log.Logger, resultPath string) {
					mockedQuery.AssertNumberOfCalls(t, "Query", 1)
				},

				Context: &Context{
					Languages: []language.Language{
						languageGolang,
					},

					Models: []evalmodel.Model{
						mockedModel,
					},
					QueryAttempts: 1,
				},

				ExpectedAssessments: map[evalmodel.Model]map[language.Language]map[string]metrics.Assessments{
					mockedModel: map[language.Language]map[string]metrics.Assessments{
						languageGolang: map[string]metrics.Assessments{},
					},
				},
				ExpectedTotalScore: 1,
				ExpectedResultFiles: map[string]func(t *testing.T, filePath string, data string){
					filepath.Join(evalmodel.CleanModelNameForFileSystem(mockedModelID), "golang", "golang", "plain.log"): func(t *testing.T, filePath, data string) {
						assert.Contains(t, data, "empty response from model")
					},
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
					// Set up mocks, when test is running.
					mockedQuery.On("Query", mock.Anything, mockedModelID, mock.Anything).Return("", errors.New("empty response from model")).Once()
					mockedQuery.On("Query", mock.Anything, mockedModelID, mock.Anything).Return("model-response", nil).Once().After(10 * time.Millisecond) // Simulate a model response delay because our internal safety measures trigger when a query is done in 0 milliseconds.
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
					QueryAttempts: 3,

					RepositoryPaths: []string{
						repositoryPath,
					},
				},

				ExpectedAssessments: map[evalmodel.Model]map[language.Language]map[string]metrics.Assessments{
					mockedModel: map[language.Language]map[string]metrics.Assessments{
						languageGolang: map[string]metrics.Assessments{
							repositoryPath: map[metrics.AssessmentKey]uint64{
								metrics.AssessmentKeyResponseNoError: 1,
							},
						},
					},
				},
				ExpectedTotalScore: 1,
				ExpectedResultFiles: map[string]func(t *testing.T, filePath string, data string){
					filepath.Join(evalmodel.CleanModelNameForFileSystem(mockedModelID), "golang", "golang", "plain.log"): func(t *testing.T, filePath, data string) {
						assert.Contains(t, data, "Attempt 1/3: empty response from model")
					},
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
					// Set up mocks, when test is running.
					mockedQuery.On("Query", mock.Anything, mockedModelID, mock.Anything).Return("model-response", nil).After(10 * time.Millisecond) // Simulate a model response delay because our internal safety measures trigger when a query is done in 0 milliseconds.
				},
				After: func(t *testing.T, logger *log.Logger, resultPath string) {
					mockedQuery.AssertNumberOfCalls(t, "Query", 1)
				},

				Context: &Context{
					Languages: []language.Language{
						&golang.Language{},
					},

					Models: []evalmodel.Model{
						mockedModel,
					},
					QueryAttempts: 3,

					RepositoryPaths: []string{
						repositoryPath,
					},
				},

				ExpectedAssessments: map[evalmodel.Model]map[language.Language]map[string]metrics.Assessments{
					mockedModel: map[language.Language]map[string]metrics.Assessments{
						languageGolang: map[string]metrics.Assessments{
							repositoryPath: map[metrics.AssessmentKey]uint64{
								metrics.AssessmentKeyResponseNoError: 1,
							},
						},
					},
				},
				ExpectedTotalScore: 1,
				ExpectedResultFiles: map[string]func(t *testing.T, filePath string, data string){
					filepath.Join(evalmodel.CleanModelNameForFileSystem(mockedModelID), "golang", "golang", "plain.log"): func(t *testing.T, filePath, data string) {
						assert.Contains(t, data, "DONE 0 tests, 1 error")
					},
				},
			})
		}
	})
}
