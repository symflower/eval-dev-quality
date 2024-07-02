package task

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	metricstesting "github.com/symflower/eval-dev-quality/evaluate/metrics/testing"
	tasktesting "github.com/symflower/eval-dev-quality/evaluate/task/testing"
	"github.com/symflower/eval-dev-quality/language/golang"
	"github.com/symflower/eval-dev-quality/language/java"
	"github.com/symflower/eval-dev-quality/log"
	modeltesting "github.com/symflower/eval-dev-quality/model/testing"
	"github.com/symflower/eval-dev-quality/task"
	"github.com/zimmski/osutil"
	"github.com/zimmski/osutil/bytesutil"
)

func TestTaskCodeRepairRun(t *testing.T) {
	validate := func(t *testing.T, tc *tasktesting.TestCaseTask) {
		t.Run(tc.Name, func(t *testing.T) {
			resultPath := t.TempDir()

			logOutput, logger := log.Buffer()
			defer func() {
				if t.Failed() {
					t.Logf("Logging output: %s", logOutput.String())
				}
			}()
			repository, cleanup, err := TemporaryRepository(logger, tc.TestDataPath, tc.RepositoryPath)
			assert.NoError(t, err)
			defer cleanup()

			taskWriteTests := newCodeRepairTask(logger, resultPath, tc.Model, tc.Language)
			tc.Validate(t, taskWriteTests, repository, resultPath)
		})
	}

	t.Run("Go", func(t *testing.T) {
		{
			temporaryDirectoryPath := t.TempDir()

			repositoryPath := filepath.Join(temporaryDirectoryPath, "golang", "mistakes", "openingBracketMissing")
			require.NoError(t, osutil.CopyTree(filepath.Join("..", "..", "testdata", "golang", "mistakes", "openingBracketMissing"), repositoryPath))

			modelMock := modeltesting.NewMockModelNamed(t, "mocked-model")

			// Generate valid code for the task.
			sourceFileContent := bytesutil.StringTrimIndentations(`
				package openingBracketMissing

				func openingBracketMissing(x int) int {
					if x > 0 {
						return 1
					}
					if x < 0 {
						return -1
					}

					return 0
				}
			`)
			modelMock.RegisterGenerateSuccess(t, IdentifierCodeRepair, "openingBracketMissing.go", sourceFileContent, metricstesting.AssessmentsWithProcessingTime).Once()

			validate(t, &tasktesting.TestCaseTask{
				Name: "Single test case",

				Model:          modelMock,
				Language:       &golang.Language{},
				TestDataPath:   temporaryDirectoryPath,
				RepositoryPath: filepath.Join("golang", "mistakes"),

				ExpectedRepositoryAssessment: map[task.Identifier]metrics.Assessments{
					IdentifierCodeRepair: metrics.Assessments{
						metrics.AssessmentKeyCoverage:        30,
						metrics.AssessmentKeyFilesExecuted:   1,
						metrics.AssessmentKeyResponseNoError: 1,
					},
				},
				ExpectedResultFiles: map[string]func(t *testing.T, filePath string, data string){
					filepath.Join(string(IdentifierCodeRepair), "mocked-model", "golang", "golang", "mistakes.log"): func(t *testing.T, filePath, data string) {
						assert.Contains(t, data, "TestSymflowerOpeningBracketMissing/#00")
						assert.Contains(t, data, "TestSymflowerOpeningBracketMissing/#01")
						assert.Contains(t, data, "TestSymflowerOpeningBracketMissing/#02")
					},
				},
			})
		}
		{
			temporaryDirectoryPath := t.TempDir()

			repositoryPath := filepath.Join(temporaryDirectoryPath, "golang", "mistakes")
			require.NoError(t, osutil.CopyTree(filepath.Join("..", "..", "testdata", "golang", "mistakes", "openingBracketMissing"), filepath.Join(repositoryPath, "openingBracketMissing")))
			require.NoError(t, osutil.CopyTree(filepath.Join("..", "..", "testdata", "golang", "mistakes", "typeUnknown"), filepath.Join(repositoryPath, "typeUnknown")))

			modelMock := modeltesting.NewMockModelNamed(t, "mocked-model")

			// Generate valid code for the task.
			sourceFileContent := bytesutil.StringTrimIndentations(`
				package openingBracketMissing

				func openingBracketMissing(x int) int {
					if x > 0 {
						return 1
					}
					if x < 0 {
						return -1
					}

					return 0
				}
			`)
			modelMock.RegisterGenerateSuccess(t, IdentifierCodeRepair, "openingBracketMissing.go", sourceFileContent, metricstesting.AssessmentsWithProcessingTime).Once()
			sourceFileContent = bytesutil.StringTrimIndentations(`
				package typeUnknown

				func typeUnknown(x int) int {
					if x > 0 {
						return 1
					}
					if x < 0 {
						return -1
					}

					return 0
				}
			`)
			modelMock.RegisterGenerateSuccess(t, IdentifierCodeRepair, "typeUnknown.go", sourceFileContent, metricstesting.AssessmentsWithProcessingTime).Once()

			validate(t, &tasktesting.TestCaseTask{
				Name: "Multiple test cases",

				Model:          modelMock,
				Language:       &golang.Language{},
				TestDataPath:   temporaryDirectoryPath,
				RepositoryPath: filepath.Join("golang", "mistakes"),

				ExpectedRepositoryAssessment: map[task.Identifier]metrics.Assessments{
					IdentifierCodeRepair: metrics.Assessments{
						metrics.AssessmentKeyCoverage:        60,
						metrics.AssessmentKeyFilesExecuted:   2,
						metrics.AssessmentKeyResponseNoError: 2,
					},
				},
				ExpectedResultFiles: map[string]func(t *testing.T, filePath string, data string){
					filepath.Join(string(IdentifierCodeRepair), "mocked-model", "golang", "golang", "mistakes.log"): func(t *testing.T, filePath, data string) {
						assert.Contains(t, data, "TestSymflowerOpeningBracketMissing/#00")
						assert.Contains(t, data, "TestSymflowerOpeningBracketMissing/#01")
						assert.Contains(t, data, "TestSymflowerOpeningBracketMissing/#02")
						assert.Contains(t, data, "TestSymflowerTypeUnknown/#00")
						assert.Contains(t, data, "TestSymflowerTypeUnknown/#01")
						assert.Contains(t, data, "TestSymflowerTypeUnknown/#02")
					},
				},
			})
		}
	})
	t.Run("Java", func(t *testing.T) {
		{
			temporaryDirectoryPath := t.TempDir()

			repositoryPath := filepath.Join(temporaryDirectoryPath, "java", "mistakes", "openingBracketMissing")
			require.NoError(t, osutil.CopyTree(filepath.Join("..", "..", "testdata", "java", "mistakes", "openingBracketMissing"), repositoryPath))

			modelMock := modeltesting.NewMockModelNamed(t, "mocked-model")

			// Generate valid code for the task.
			sourceFileContent := bytesutil.StringTrimIndentations(`
				package com.eval;

				public class OpeningBracketMissing {
					public static int openingBracketMissing(int x) {
						if (x > 0) {
							return 1;
						}
						if (x < 0) {
							return -1;
						}

						return 0;
					}
				}
			`)
			modelMock.RegisterGenerateSuccess(t, IdentifierCodeRepair, filepath.Join("src", "main", "java", "com", "eval", "OpeningBracketMissing.java"), sourceFileContent, metricstesting.AssessmentsWithProcessingTime).Once()

			validate(t, &tasktesting.TestCaseTask{
				Name: "Single test case",

				Model:          modelMock,
				Language:       &java.Language{},
				TestDataPath:   temporaryDirectoryPath,
				RepositoryPath: filepath.Join("java", "mistakes"),

				ExpectedRepositoryAssessment: map[task.Identifier]metrics.Assessments{
					IdentifierCodeRepair: metrics.Assessments{
						metrics.AssessmentKeyCoverage:        80,
						metrics.AssessmentKeyFilesExecuted:   1,
						metrics.AssessmentKeyResponseNoError: 1,
					},
				},
				ExpectedResultFiles: map[string]func(t *testing.T, filePath string, data string){
					filepath.Join(string(IdentifierCodeRepair), "mocked-model", "java", "java", "mistakes.log"): func(t *testing.T, filePath, data string) {
						assert.Contains(t, data, "BUILD SUCCESS")
					},
				},
			})
		}
		{
			temporaryDirectoryPath := t.TempDir()

			repositoryPath := filepath.Join(temporaryDirectoryPath, "java", "mistakes")
			require.NoError(t, osutil.CopyTree(filepath.Join("..", "..", "testdata", "java", "mistakes", "openingBracketMissing"), filepath.Join(repositoryPath, "openingBracketMissing")))
			require.NoError(t, osutil.CopyTree(filepath.Join("..", "..", "testdata", "java", "mistakes", "typeUnknown"), filepath.Join(repositoryPath, "typeUnknown")))

			modelMock := modeltesting.NewMockModelNamed(t, "mocked-model")

			// Generate valid code for the task.
			sourceFileContent := bytesutil.StringTrimIndentations(`
				package com.eval;

				public class OpeningBracketMissing {
					public static int openingBracketMissing(int x) {
						if (x > 0) {
							return 1;
						}
						if (x < 0) {
							return -1;
						}

						return 0;
					}
				}
			`)
			modelMock.RegisterGenerateSuccess(t, IdentifierCodeRepair, filepath.Join("src", "main", "java", "com", "eval", "OpeningBracketMissing.java"), sourceFileContent, metricstesting.AssessmentsWithProcessingTime).Once()
			sourceFileContent = bytesutil.StringTrimIndentations(`
				package com.eval;

				public class TypeUnknown {
					public static int typeUnknown(int x) {
						if (x > 0) {
							return 1;
						}
						if (x < 0) {
							return -1;
						}

						return 0;
					}
				}
			`)
			modelMock.RegisterGenerateSuccess(t, IdentifierCodeRepair, filepath.Join("src", "main", "java", "com", "eval", "TypeUnknown.java"), sourceFileContent, metricstesting.AssessmentsWithProcessingTime).Once()

			validate(t, &tasktesting.TestCaseTask{
				Name: "Multiple test cases",

				Model:          modelMock,
				Language:       &java.Language{},
				TestDataPath:   temporaryDirectoryPath,
				RepositoryPath: filepath.Join("java", "mistakes"),

				ExpectedRepositoryAssessment: map[task.Identifier]metrics.Assessments{
					IdentifierCodeRepair: metrics.Assessments{
						metrics.AssessmentKeyCoverage:        160,
						metrics.AssessmentKeyFilesExecuted:   2,
						metrics.AssessmentKeyResponseNoError: 2,
					},
				},
				ExpectedResultFiles: map[string]func(t *testing.T, filePath string, data string){
					filepath.Join(string(IdentifierCodeRepair), "mocked-model", "java", "java", "mistakes.log"): func(t *testing.T, filePath, data string) {
						assert.Contains(t, data, "BUILD SUCCESS")
					},
				},
			})
		}
	})
}
