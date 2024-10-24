package task

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zimmski/osutil"
	"github.com/zimmski/osutil/bytesutil"

	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	metricstesting "github.com/symflower/eval-dev-quality/evaluate/metrics/testing"
	tasktesting "github.com/symflower/eval-dev-quality/evaluate/task/testing"
	"github.com/symflower/eval-dev-quality/language"
	"github.com/symflower/eval-dev-quality/language/golang"
	"github.com/symflower/eval-dev-quality/language/java"
	"github.com/symflower/eval-dev-quality/log"
	modeltesting "github.com/symflower/eval-dev-quality/model/testing"
	evaltask "github.com/symflower/eval-dev-quality/task"
)

func TestMigrateRun(t *testing.T) {
	validate := func(t *testing.T, tc *tasktesting.TestCaseTask) {
		t.Run(tc.Name, func(t *testing.T) {
			task, err := ForIdentifier(IdentifierMigrate)
			require.NoError(t, err)
			tc.Task = task

			tc.Validate(t,
				func(logger *log.Logger, testDataPath string, repositoryPathRelative string) (repository evaltask.Repository, cleanup func(), err error) {
					return TemporaryRepository(logger, testDataPath, repositoryPathRelative)
				},
			)
		})
	}

	t.Run("Java", func(t *testing.T) {
		{
			temporaryDirectoryPath := t.TempDir()

			repositoryPath := filepath.Join(temporaryDirectoryPath, "java", "migrate-plain")
			require.NoError(t, osutil.CopyTree(filepath.Join("..", "..", "testdata", "java", "migrate-plain"), repositoryPath))

			modelMock := modeltesting.NewMockCapabilityMigrateNamed(t, "mocked-model")

			decrementTestFileContent := bytesutil.StringTrimIndentations(`
				package com.eval;

				import org.junit.jupiter.api.Test;
				import static org.junit.jupiter.api.Assertions.assertEquals;

				public class DecrementTest {
					@Test
					public void decrement() {
						int i = 1;
						int expected = 0;
						int actual = Decrement.decrement(i);

						assertEquals(expected, actual);
					}
				}
			`)
			modelMock.RegisterGenerateSuccess(t, filepath.Join("src", "test", "java", "com", "eval", "DecrementTest.java"), decrementTestFileContent, metricstesting.AssessmentsWithProcessingTime).Once()

			incrementTestFileContent := bytesutil.StringTrimIndentations(`
				package com.eval;

				import org.junit.jupiter.api.Test;
				import static org.junit.jupiter.api.Assertions.assertEquals;

				public class IncrementTest {
					@Test
					public void increment() {
						int i = 1;
						int expected = 2;
						int actual = Increment.increment(i);

						assertEquals(expected, actual);
					}
				}
			`)
			modelMock.RegisterGenerateSuccess(t, filepath.Join("src", "test", "java", "com", "eval", "IncrementTest.java"), incrementTestFileContent, metricstesting.AssessmentsWithProcessingTime).Once()

			validate(t, &tasktesting.TestCaseTask{
				Name: "Plain",

				Model:          modelMock,
				Language:       &java.Language{},
				TestDataPath:   temporaryDirectoryPath,
				RepositoryPath: filepath.Join("java", "migrate-plain"),

				ExpectedRepositoryAssessment: map[evaltask.Identifier]metrics.Assessments{
					IdentifierMigrate: metrics.Assessments{
						metrics.AssessmentKeyFilesExecutedMaximumReachable: 2,
						metrics.AssessmentKeyFilesExecuted:                 2,
						metrics.AssessmentKeyResponseNoError:               2,
						metrics.AssessmentKeyCoverage:                      4,
					},
					IdentifierMigrateSymflowerFix: metrics.Assessments{
						metrics.AssessmentKeyFilesExecutedMaximumReachable: 2,
						metrics.AssessmentKeyFilesExecuted:                 2,
						metrics.AssessmentKeyResponseNoError:               2,
						metrics.AssessmentKeyCoverage:                      4,
					},
				},
				ValidateLog: func(t *testing.T, data string) {
					assert.Contains(t, data, "BUILD SUCCESS")
				},
			})
		}
	})
}

func TestClearRepositoryForMigration(t *testing.T) {
	type testCase struct {
		Name string

		Language     language.Language
		AllFilePaths []string
		FilePath     string

		ExpectedFilePaths []string
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			repositoryPath := t.TempDir()

			for _, filePath := range tc.AllFilePaths {
				require.NoError(t, osutil.MkdirAll(filepath.Join(repositoryPath, filepath.Dir(filePath))))
				require.NoError(t, os.WriteFile(filepath.Join(repositoryPath, filePath), []byte(filePath), 0700))
			}

			require.NoError(t, clearRepositoryForMigration(tc.Language, repositoryPath, tc.AllFilePaths, tc.FilePath))

			actualFilePaths, err := osutil.FilesRecursive(repositoryPath)
			require.NoError(t, err)
			for i, filePath := range actualFilePaths {
				filePath, err := filepath.Rel(repositoryPath, filePath)
				require.NoError(t, err)
				actualFilePaths[i] = filePath
			}
			assert.Equal(t, tc.ExpectedFilePaths, actualFilePaths)
		})
	}

	t.Run("Go", func(t *testing.T) {
		validate(t, &testCase{
			Name: "Single",

			Language: &golang.Language{},
			AllFilePaths: []string{
				"file.go",
				"file_test.go",
			},
			FilePath: "file_test.go",

			ExpectedFilePaths: []string{
				"file.go",
				"file_test.go",
			},
		})
		validate(t, &testCase{
			Name: "Multiple",

			Language: &golang.Language{},
			AllFilePaths: []string{
				"fileA.go",
				"fileA_test.go",
				"fileB.go",
				"fileB_test.go",
				"fileC.go",
				"fileC_test.go",
			},
			FilePath: "fileB_test.go",

			ExpectedFilePaths: []string{
				"fileA.go",
				"fileB.go",
				"fileB_test.go",
				"fileC.go",
			},
		})
	})
	t.Run("Java", func(t *testing.T) {
		validate(t, &testCase{
			Name: "Single",

			Language: &java.Language{},
			AllFilePaths: []string{
				filepath.Join("src", "main", "java", "com", "eval", "File.java"),
				filepath.Join("src", "test", "java", "com", "eval", "FileTest.java"),
			},
			FilePath: filepath.Join("src", "test", "java", "com", "eval", "FileTest.java"),

			ExpectedFilePaths: []string{
				filepath.Join("src", "main", "java", "com", "eval", "File.java"),
				filepath.Join("src", "test", "java", "com", "eval", "FileTest.java"),
			},
		})
		validate(t, &testCase{
			Name: "Multiple",

			Language: &java.Language{},
			AllFilePaths: []string{
				filepath.Join("src", "main", "java", "com", "eval", "FileA.java"),
				filepath.Join("src", "main", "java", "com", "eval", "FileB.java"),
				filepath.Join("src", "main", "java", "com", "eval", "FileC.java"),
				filepath.Join("src", "test", "java", "com", "eval", "FileATest.java"),
				filepath.Join("src", "test", "java", "com", "eval", "FileBTest.java"),
				filepath.Join("src", "test", "java", "com", "eval", "FileCTest.java"),
			},
			FilePath: filepath.Join("src", "test", "java", "com", "eval", "FileBTest.java"),

			ExpectedFilePaths: []string{
				filepath.Join("src", "main", "java", "com", "eval", "FileA.java"),
				filepath.Join("src", "main", "java", "com", "eval", "FileB.java"),
				filepath.Join("src", "main", "java", "com", "eval", "FileC.java"),
				filepath.Join("src", "test", "java", "com", "eval", "FileBTest.java"),
			},
		})
	})
}

func TestValidateMigrateRepository(t *testing.T) {
	type testCase struct {
		Name string

		Before func(repositoryPath string)

		ExpectedError func(t *testing.T, err error)
	}

	validateJava := func(t *testing.T, tc *testCase) {
		validateRepository := &tasktesting.TestCaseValidateRepository{
			Name: tc.Name,

			Before: tc.Before,

			TestdataPath:   filepath.Join("..", "..", "testdata"),
			RepositoryPath: filepath.Join("java", "migrate-plain"),
			Language:       &java.Language{},

			ExpectedError: tc.ExpectedError,
		}

		validateRepository.Validate(t, validateMigrateRepository)
	}

	t.Run("Java", func(t *testing.T) {
		validateJava(t, &testCase{
			Name: "No test files",

			Before: func(repositoryPath string) {
				require.NoError(t, os.Remove(filepath.Join(repositoryPath, "src", "test", "java", "com", "eval", "IncrementTest.java")))
				require.NoError(t, os.Remove(filepath.Join(repositoryPath, "src", "test", "java", "com", "eval", "DecrementTest.java")))
			},

			ExpectedError: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "must contain test files but found none")
			},
		})
		validateJava(t, &testCase{
			Name: "No implementation files",

			Before: func(repositoryPath string) {
				require.NoError(t, os.Remove(filepath.Join(repositoryPath, "src", "main", "java", "com", "eval", "Increment.java")))
				require.NoError(t, os.Remove(filepath.Join(repositoryPath, "src", "main", "java", "com", "eval", "Decrement.java")))
			},

			ExpectedError: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "must contain implementation files but found none")
			},
		})
		validateJava(t, &testCase{
			Name: "Implementation files does not have a corresponding test file",

			Before: func(repositoryPath string) {
				filePath := filepath.Join(repositoryPath, "src", "main", "java", "com", "eval", "File.java")
				require.NoError(t, os.WriteFile(filePath, []byte(`content`), 0700))
			},

			ExpectedError: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "must contain a test file for each implementation file")
			},
		})
		validateJava(t, &testCase{
			Name: "Test file does not have a corresponding implementation file",

			Before: func(repositoryPath string) {
				filePath := filepath.Join(repositoryPath, "src", "test", "java", "com", "eval", "FileTest.java")
				require.NoError(t, os.WriteFile(filePath, []byte(`content`), 0700))
			},

			ExpectedError: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "must contain a test file for each implementation file")
			},
		})
		validateJava(t, &testCase{
			Name: "Valid",

			// The testdata repository is valid by default.
		})
	})
}
