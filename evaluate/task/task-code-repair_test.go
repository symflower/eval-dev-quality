package task

import (
	"fmt"
	"os"
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
	evaltask "github.com/symflower/eval-dev-quality/task"
	"github.com/zimmski/osutil"
	"github.com/zimmski/osutil/bytesutil"
)

func TestTaskCodeRepairRun(t *testing.T) {
	validate := func(t *testing.T, tc *tasktesting.TestCaseTask) {
		t.Run(tc.Name, func(t *testing.T) {
			task, err := TaskForIdentifier(IdentifierCodeRepair)
			require.NoError(t, err)
			tc.Task = task

			tc.Validate(t,
				func(logger *log.Logger, testDataPath string, repositoryPathRelative string) (repository evaltask.Repository, cleanup func(), err error) {
					return TemporaryRepository(logger, testDataPath, repositoryPathRelative)
				},
			)
		})
	}

	t.Run("Go", func(t *testing.T) {
		{
			temporaryDirectoryPath := t.TempDir()

			repositoryPath := filepath.Join(temporaryDirectoryPath, "golang", "mistakes", "openingBracketMissing")
			require.NoError(t, osutil.CopyTree(filepath.Join("..", "..", "testdata", "golang", "mistakes", "openingBracketMissing"), repositoryPath))

			modelMock := modeltesting.NewMockCapabilityRepairCodeNamed(t, "mocked-model")

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
			modelMock.RegisterGenerateSuccess(t, "openingBracketMissing.go", sourceFileContent, metricstesting.AssessmentsWithProcessingTime).Once()

			validate(t, &tasktesting.TestCaseTask{
				Name: "Single test case",

				Model:          modelMock,
				Language:       &golang.Language{},
				TestDataPath:   temporaryDirectoryPath,
				RepositoryPath: filepath.Join("golang", "mistakes"),

				ExpectedRepositoryAssessment: map[evaltask.Identifier]metrics.Assessments{
					IdentifierCodeRepair: metrics.Assessments{
						metrics.AssessmentKeyCoverage:                      30,
						metrics.AssessmentKeyFilesExecuted:                 1,
						metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						metrics.AssessmentKeyResponseNoError:               1,
					},
				},
				ValidateLog: func(t *testing.T, data string) {
					assert.Contains(t, data, "TestSymflowerOpeningBracketMissing/#00")
					assert.Contains(t, data, "TestSymflowerOpeningBracketMissing/#01")
					assert.Contains(t, data, "TestSymflowerOpeningBracketMissing/#02")
				},
			})
		}
		{
			temporaryDirectoryPath := t.TempDir()

			repositoryPath := filepath.Join(temporaryDirectoryPath, "golang", "mistakes")
			require.NoError(t, osutil.CopyTree(filepath.Join("..", "..", "testdata", "golang", "mistakes", "openingBracketMissing"), filepath.Join(repositoryPath, "openingBracketMissing")))
			require.NoError(t, osutil.CopyTree(filepath.Join("..", "..", "testdata", "golang", "mistakes", "typeUnknown"), filepath.Join(repositoryPath, "typeUnknown")))

			modelMock := modeltesting.NewMockCapabilityRepairCodeNamed(t, "mocked-model")

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
			modelMock.RegisterGenerateSuccess(t, "openingBracketMissing.go", sourceFileContent, metricstesting.AssessmentsWithProcessingTime).Once()
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
			modelMock.RegisterGenerateSuccess(t, "typeUnknown.go", sourceFileContent, metricstesting.AssessmentsWithProcessingTime).Once()

			validate(t, &tasktesting.TestCaseTask{
				Name: "Multiple test cases",

				Model:          modelMock,
				Language:       &golang.Language{},
				TestDataPath:   temporaryDirectoryPath,
				RepositoryPath: filepath.Join("golang", "mistakes"),

				ExpectedRepositoryAssessment: map[evaltask.Identifier]metrics.Assessments{
					IdentifierCodeRepair: metrics.Assessments{
						metrics.AssessmentKeyCoverage:                      60,
						metrics.AssessmentKeyFilesExecuted:                 2,
						metrics.AssessmentKeyFilesExecutedMaximumReachable: 2,
						metrics.AssessmentKeyResponseNoError:               2,
					},
				},
				ValidateLog: func(t *testing.T, data string) {
					assert.Contains(t, data, "TestSymflowerOpeningBracketMissing/#00")
					assert.Contains(t, data, "TestSymflowerOpeningBracketMissing/#01")
					assert.Contains(t, data, "TestSymflowerOpeningBracketMissing/#02")
					assert.Contains(t, data, "TestSymflowerTypeUnknown/#00")
					assert.Contains(t, data, "TestSymflowerTypeUnknown/#01")
					assert.Contains(t, data, "TestSymflowerTypeUnknown/#02")
				},
			})
		}
	})
	t.Run("Java", func(t *testing.T) {
		{
			temporaryDirectoryPath := t.TempDir()

			repositoryPath := filepath.Join(temporaryDirectoryPath, "java", "mistakes", "openingBracketMissing")
			require.NoError(t, osutil.CopyTree(filepath.Join("..", "..", "testdata", "java", "mistakes", "openingBracketMissing"), repositoryPath))

			modelMock := modeltesting.NewMockCapabilityRepairCodeNamed(t, "mocked-model")

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
			modelMock.RegisterGenerateSuccess(t, filepath.Join("src", "main", "java", "com", "eval", "OpeningBracketMissing.java"), sourceFileContent, metricstesting.AssessmentsWithProcessingTime).Once()

			validate(t, &tasktesting.TestCaseTask{
				Name: "Single test case",

				Model:          modelMock,
				Language:       &java.Language{},
				TestDataPath:   temporaryDirectoryPath,
				RepositoryPath: filepath.Join("java", "mistakes"),

				ExpectedRepositoryAssessment: map[evaltask.Identifier]metrics.Assessments{
					IdentifierCodeRepair: metrics.Assessments{
						metrics.AssessmentKeyCoverage:                      80,
						metrics.AssessmentKeyFilesExecuted:                 1,
						metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						metrics.AssessmentKeyResponseNoError:               1,
					},
				},
				ValidateLog: func(t *testing.T, data string) {
					assert.Contains(t, data, "BUILD SUCCESS")
				},
			})
		}
		{
			temporaryDirectoryPath := t.TempDir()

			repositoryPath := filepath.Join(temporaryDirectoryPath, "java", "mistakes")
			require.NoError(t, osutil.CopyTree(filepath.Join("..", "..", "testdata", "java", "mistakes", "openingBracketMissing"), filepath.Join(repositoryPath, "openingBracketMissing")))
			require.NoError(t, osutil.CopyTree(filepath.Join("..", "..", "testdata", "java", "mistakes", "typeUnknown"), filepath.Join(repositoryPath, "typeUnknown")))

			modelMock := modeltesting.NewMockCapabilityRepairCodeNamed(t, "mocked-model")

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
			modelMock.RegisterGenerateSuccess(t, filepath.Join("src", "main", "java", "com", "eval", "OpeningBracketMissing.java"), sourceFileContent, metricstesting.AssessmentsWithProcessingTime).Once()
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
			modelMock.RegisterGenerateSuccess(t, filepath.Join("src", "main", "java", "com", "eval", "TypeUnknown.java"), sourceFileContent, metricstesting.AssessmentsWithProcessingTime).Once()

			validate(t, &tasktesting.TestCaseTask{
				Name: "Multiple test cases",

				Model:          modelMock,
				Language:       &java.Language{},
				TestDataPath:   temporaryDirectoryPath,
				RepositoryPath: filepath.Join("java", "mistakes"),

				ExpectedRepositoryAssessment: map[evaltask.Identifier]metrics.Assessments{
					IdentifierCodeRepair: metrics.Assessments{
						metrics.AssessmentKeyCoverage:                      160,
						metrics.AssessmentKeyFilesExecutedMaximumReachable: 2,
						metrics.AssessmentKeyFilesExecuted:                 2,
						metrics.AssessmentKeyResponseNoError:               2,
					},
				},
				ValidateLog: func(t *testing.T, data string) {
					assert.Contains(t, data, "BUILD SUCCESS")
				},
			})
		}
	})
}

func TestValidateCodeRepairRepository(t *testing.T) {
	validate := func(t *testing.T, tc *tasktesting.TestCaseValidateRepository) {
		tc.Validate(t, validateCodeRepairRepository)
	}

	validate(t, &tasktesting.TestCaseValidateRepository{
		Name: "Repository root path contains source files",

		Before: func(repositoryPath string) {
			someFile, err := os.Create(filepath.Join(repositoryPath, "someFile.go"))
			require.NoError(t, err)
			someFile.Close()
		},

		TestdataPath:   filepath.Join("..", "..", "testdata"),
		RepositoryPath: filepath.Join("golang", "mistakes"),
		Language:       &golang.Language{},

		ExpectedErrorContains: "must contain only packages, but found [someFile.go]",
	})
	t.Run("Go", func(t *testing.T) {
		validate(t, &tasktesting.TestCaseValidateRepository{
			Name: "Package does not contain source file",

			Before: func(repositoryPath string) {
				require.NoError(t, os.MkdirAll(filepath.Join(repositoryPath, "somePackage"), 0700))
			},

			TestdataPath:   filepath.Join("..", "..", "testdata"),
			RepositoryPath: filepath.Join("golang", "mistakes"),
			Language:       &golang.Language{},

			ExpectedErrorContains: "must contain exactly one Go source file, but found []",
		})
		validate(t, &tasktesting.TestCaseValidateRepository{
			Name: "Package contains multiple source files",

			Before: func(repositoryPath string) {
				somePackage := filepath.Join(repositoryPath, "somePackage")
				require.NoError(t, os.MkdirAll(somePackage, 0700))

				fileA, err := os.Create(filepath.Join(somePackage, "fileA.go"))
				require.NoError(t, err)
				fileA.Close()

				fileB, err := os.Create(filepath.Join(somePackage, "fileB.go"))
				require.NoError(t, err)
				fileB.Close()
			},

			TestdataPath:   filepath.Join("..", "..", "testdata"),
			RepositoryPath: filepath.Join("golang", "mistakes"),
			Language:       &golang.Language{},

			ExpectedErrorContains: "must contain exactly one Go source file, but found [fileA.go fileB.go]",
		})
		validate(t, &tasktesting.TestCaseValidateRepository{
			Name: "Package does not contain test file",

			Before: func(repositoryPath string) {
				somePackage := filepath.Join(repositoryPath, "somePackage")
				require.NoError(t, os.MkdirAll(somePackage, 0700))

				file, err := os.Create(filepath.Join(somePackage, "someFile.go"))
				require.NoError(t, err)
				defer file.Close()
			},

			TestdataPath:   filepath.Join("..", "..", "testdata"),
			RepositoryPath: filepath.Join("golang", "mistakes"),
			Language:       &golang.Language{},

			ExpectedErrorContains: "must contain exactly one Go test file, but found []",
		})
		validate(t, &tasktesting.TestCaseValidateRepository{
			Name: "Package contains multiple test files",

			Before: func(repositoryPath string) {
				somePackage := filepath.Join(repositoryPath, "somePackage")
				require.NoError(t, os.MkdirAll(somePackage, 0700))

				fileA, err := os.Create(filepath.Join(somePackage, "fileA.go"))
				require.NoError(t, err)
				fileA.Close()

				fileATest, err := os.Create(filepath.Join(somePackage, "fileA_test.go"))
				require.NoError(t, err)
				fileATest.Close()

				fileBTest, err := os.Create(filepath.Join(somePackage, "fileB_test.go"))
				require.NoError(t, err)
				fileBTest.Close()
			},

			TestdataPath:   filepath.Join("..", "..", "testdata"),
			RepositoryPath: filepath.Join("golang", "mistakes"),
			Language:       &golang.Language{},

			ExpectedErrorContains: "must contain exactly one Go test file, but found [fileA_test.go fileB_test.go]",
		})
		validate(t, &tasktesting.TestCaseValidateRepository{
			Name: "Well-formed",

			TestdataPath:   filepath.Join("..", "..", "testdata"),
			RepositoryPath: filepath.Join("golang", "mistakes"),
			Language:       &golang.Language{},
		})
	})
	t.Run("Java", func(t *testing.T) {
		validate(t, &tasktesting.TestCaseValidateRepository{
			Name: "Package does not contain source file",

			Before: func(repositoryPath string) {
				require.NoError(t, os.MkdirAll(filepath.Join(repositoryPath, "somePackage"), 0700))
			},

			TestdataPath:   filepath.Join("..", "..", "testdata"),
			RepositoryPath: filepath.Join("java", "mistakes"),
			Language:       &java.Language{},

			ExpectedErrorContains: "must contain exactly one Java source file, but found []",
		})
		validate(t, &tasktesting.TestCaseValidateRepository{
			Name: "Package contains multiple source files",

			Before: func(repositoryPath string) {
				somePackage := filepath.Join(repositoryPath, "somePackage", "src", "main", "java", "com", "eval")
				require.NoError(t, os.MkdirAll(somePackage, 0700))

				fileA, err := os.Create(filepath.Join(somePackage, "FileA.java"))
				require.NoError(t, err)
				fileA.Close()

				fileB, err := os.Create(filepath.Join(somePackage, "FileB.java"))
				require.NoError(t, err)
				fileB.Close()
			},

			TestdataPath:   filepath.Join("..", "..", "testdata"),
			RepositoryPath: filepath.Join("java", "mistakes"),
			Language:       &java.Language{},

			ExpectedErrorContains: fmt.Sprintf("must contain exactly one Java source file, but found [%s %s]", filepath.Join("src", "main", "java", "com", "eval", "FileA.java"), filepath.Join("src", "main", "java", "com", "eval", "FileB.java")),
		})

		validate(t, &tasktesting.TestCaseValidateRepository{
			Name: "Package does not contain test file",

			Before: func(repositoryPath string) {
				somePackage := filepath.Join(repositoryPath, "somePackage", "src", "main", "java", "com", "eval")
				require.NoError(t, os.MkdirAll(somePackage, 0700))

				fileA, err := os.Create(filepath.Join(somePackage, "FileA.java"))
				require.NoError(t, err)
				fileA.Close()
			},

			TestdataPath:   filepath.Join("..", "..", "testdata"),
			RepositoryPath: filepath.Join("java", "mistakes"),
			Language:       &java.Language{},

			ExpectedErrorContains: "must contain exactly one Java test file, but found []",
		})
		validate(t, &tasktesting.TestCaseValidateRepository{
			Name: "Package contains multiple test files",

			Before: func(repositoryPath string) {
				sourcePackage := filepath.Join(repositoryPath, "somePackage", "src", "main", "java", "com", "eval")
				require.NoError(t, os.MkdirAll(sourcePackage, 0700))
				testPackage := filepath.Join(repositoryPath, "somePackage", "src", "test", "java", "com", "eval")
				require.NoError(t, os.MkdirAll(testPackage, 0700))

				fileA, err := os.Create(filepath.Join(sourcePackage, "FileA.java"))
				require.NoError(t, err)
				fileA.Close()

				fileATest, err := os.Create(filepath.Join(testPackage, "FileATest.java"))
				require.NoError(t, err)
				fileATest.Close()

				fileBTest, err := os.Create(filepath.Join(testPackage, "FileBTest.java"))
				require.NoError(t, err)
				fileBTest.Close()
			},

			TestdataPath:   filepath.Join("..", "..", "testdata"),
			RepositoryPath: filepath.Join("java", "mistakes"),
			Language:       &java.Language{},

			ExpectedErrorContains: fmt.Sprintf("must contain exactly one Java test file, but found [%s %s]", filepath.Join("src", "test", "java", "com", "eval", "FileATest.java"), filepath.Join("src", "test", "java", "com", "eval", "FileBTest.java")),
		})
		validate(t, &tasktesting.TestCaseValidateRepository{
			Name: "Well-formed",

			TestdataPath:   filepath.Join("..", "..", "testdata"),
			RepositoryPath: filepath.Join("java", "mistakes"),
			Language:       &java.Language{},
		})
	})
}
