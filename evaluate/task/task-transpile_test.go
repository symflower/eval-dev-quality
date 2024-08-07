package task

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	metricstesting "github.com/symflower/eval-dev-quality/evaluate/metrics/testing"
	tasktesting "github.com/symflower/eval-dev-quality/evaluate/task/testing"
	"github.com/symflower/eval-dev-quality/language"
	"github.com/symflower/eval-dev-quality/language/golang"
	"github.com/symflower/eval-dev-quality/language/java"
	"github.com/symflower/eval-dev-quality/language/ruby"
	"github.com/symflower/eval-dev-quality/log"
	modeltesting "github.com/symflower/eval-dev-quality/model/testing"
	evaltask "github.com/symflower/eval-dev-quality/task"
	"github.com/zimmski/osutil"
	"github.com/zimmski/osutil/bytesutil"
)

func TestTaskTranspileRun(t *testing.T) {
	validate := func(t *testing.T, tc *tasktesting.TestCaseTask) {
		t.Run(tc.Name, func(t *testing.T) {
			task, err := TaskForIdentifier(IdentifierTranspile)
			require.NoError(t, err)
			tc.Task = task

			tc.Validate(t,
				func(logger *log.Logger, testDataPath string, repositoryPathRelative string) (repository evaltask.Repository, cleanup func(), err error) {
					return TemporaryRepository(logger, testDataPath, repositoryPathRelative)
				},
			)
		})
	}

	t.Run("Transpile Java into Go", func(t *testing.T) {
		{
			temporaryDirectoryPath := t.TempDir()

			repositoryPath := filepath.Join(temporaryDirectoryPath, "golang", "transpile", "cascadingIfElse")
			require.NoError(t, osutil.CopyTree(filepath.Join("..", "..", "testdata", "golang", "transpile", "cascadingIfElse"), repositoryPath))

			modelMock := modeltesting.NewMockCapabilityTranspileNamed(t, "mocked-model")

			transpiledSourceFilePath := "cascadingIfElse.go"
			transpiledSourceFileContent := bytesutil.StringTrimIndentations(`
				package cascadingIfElse

			 	func cascadingIfElse(i int) int {
			 		if i == 1 {
			 			return 2
			 		} else if i == 3 {
			 			return 4
			 		} else {
			 			return 5
			 		}
			 	}
			`)
			modelMock.RegisterGenerateSuccess(t, transpiledSourceFilePath, transpiledSourceFileContent, metricstesting.AssessmentsWithProcessingTime).Times(2)

			validate(t, &tasktesting.TestCaseTask{
				Name: "Single test case",

				Model:          modelMock,
				Language:       &golang.Language{},
				TestDataPath:   temporaryDirectoryPath,
				RepositoryPath: filepath.Join("golang", "transpile"),

				ExpectedRepositoryAssessment: map[evaltask.Identifier]metrics.Assessments{
					IdentifierTranspile: metrics.Assessments{
						metrics.AssessmentKeyTestsPassing:                  80,
						metrics.AssessmentKeyFilesExecuted:                 2,
						metrics.AssessmentKeyFilesExecutedMaximumReachable: 2,
						metrics.AssessmentKeyResponseNoError:               2,
					},
					IdentifierTranspileSymflowerFix: metrics.Assessments{
						metrics.AssessmentKeyTestsPassing:                  80,
						metrics.AssessmentKeyFilesExecuted:                 2,
						metrics.AssessmentKeyFilesExecutedMaximumReachable: 2,
						metrics.AssessmentKeyResponseNoError:               2,
					},
				},
				ValidateLog: func(t *testing.T, data string) {
					assert.Contains(t, data, "PASS: TestSymflowerCascadingIfElse/#00")
					assert.Contains(t, data, "PASS: TestSymflowerCascadingIfElse/#01")
					assert.Contains(t, data, "PASS: TestSymflowerCascadingIfElse/#02")
				},
			})
		}
		{
			temporaryDirectoryPath := t.TempDir()

			repositoryPath := filepath.Join(temporaryDirectoryPath, "golang", "transpile")
			require.NoError(t, osutil.CopyTree(filepath.Join("..", "..", "testdata", "golang", "transpile", "cascadingIfElse"), filepath.Join(repositoryPath, "cascadingIfElse")))
			require.NoError(t, osutil.CopyTree(filepath.Join("..", "..", "testdata", "golang", "transpile", "isSorted"), filepath.Join(repositoryPath, "isSorted")))

			modelMock := modeltesting.NewMockCapabilityTranspileNamed(t, "mocked-model")

			transpiledSourceFilePath := "cascadingIfElse.go"
			transpiledSourceFileContent := bytesutil.StringTrimIndentations(`
				package cascadingIfElse

				func cascadingIfElse(i int) int {
					if i == 1 {
						return 2
					} else if i == 3 {
						return 4
					} else {
						return 5
					}
				}
			`)
			modelMock.RegisterGenerateSuccess(t, transpiledSourceFilePath, transpiledSourceFileContent, metricstesting.AssessmentsWithProcessingTime).Times(2)

			transpiledSourceFilePath = "isSorted.go"
			transpiledSourceFileContent = bytesutil.StringTrimIndentations(`
				package isSorted

				func isSorted(a []int) bool {
					i := 0
					for i < len(a)-1 && a[i] <= a[i+1] {
						i++
					}

					return i == len(a)-1
				}
			`)
			modelMock.RegisterGenerateSuccess(t, transpiledSourceFilePath, transpiledSourceFileContent, metricstesting.AssessmentsWithProcessingTime).Times(2)

			validate(t, &tasktesting.TestCaseTask{
				Name: "Multiple test cases",

				Model:          modelMock,
				Language:       &golang.Language{},
				TestDataPath:   temporaryDirectoryPath,
				RepositoryPath: filepath.Join("golang", "transpile"),

				ExpectedRepositoryAssessment: map[evaltask.Identifier]metrics.Assessments{
					IdentifierTranspile: metrics.Assessments{
						metrics.AssessmentKeyTestsPassing:                  200,
						metrics.AssessmentKeyFilesExecuted:                 4,
						metrics.AssessmentKeyFilesExecutedMaximumReachable: 4,
						metrics.AssessmentKeyResponseNoError:               4,
					},
					IdentifierTranspileSymflowerFix: metrics.Assessments{
						metrics.AssessmentKeyTestsPassing:                  200,
						metrics.AssessmentKeyFilesExecuted:                 4,
						metrics.AssessmentKeyFilesExecutedMaximumReachable: 4,
						metrics.AssessmentKeyResponseNoError:               4,
					},
				},
				ValidateLog: func(t *testing.T, data string) {
					assert.Contains(t, data, "PASS: TestSymflowerCascadingIfElse/#00")
					assert.Contains(t, data, "PASS: TestSymflowerCascadingIfElse/#01")
					assert.Contains(t, data, "PASS: TestSymflowerCascadingIfElse/#02")

					assert.Contains(t, data, "PASS: TestSymflowerIsSorted/#00")
					assert.Contains(t, data, "PASS: TestSymflowerIsSorted/#01")
					assert.Contains(t, data, "PASS: TestSymflowerIsSorted/#02")
					assert.Contains(t, data, "PASS: TestSymflowerIsSorted/#03")
					assert.Contains(t, data, "PASS: TestSymflowerIsSorted/#04")
				},
			})
		}
	})
	t.Run("Transpile Go into Java", func(t *testing.T) {
		{
			temporaryDirectoryPath := t.TempDir()

			repositoryPath := filepath.Join(temporaryDirectoryPath, "java", "transpile", "cascadingIfElse")
			require.NoError(t, osutil.CopyTree(filepath.Join("..", "..", "testdata", "java", "transpile", "cascadingIfElse"), repositoryPath))

			modelMock := modeltesting.NewMockCapabilityTranspileNamed(t, "mocked-model")

			transpiledSourceFilePath := filepath.Join("src", "main", "java", "com", "eval", "CascadingIfElse.java")
			transpiledSourceFileContent := bytesutil.StringTrimIndentations(`
				package com.eval;

				class CascadingIfElse {
					static int cascadingIfElse(int i) {
						if (i == 1) {
							return 2;
						} else if (i == 3) {
							return 4;
						} else {
							return 5;
						}
					}
				}
			`)
			modelMock.RegisterGenerateSuccess(t, transpiledSourceFilePath, transpiledSourceFileContent, metricstesting.AssessmentsWithProcessingTime).Times(2)

			validate(t, &tasktesting.TestCaseTask{
				Name: "Single test case",

				Model:          modelMock,
				Language:       &java.Language{},
				TestDataPath:   temporaryDirectoryPath,
				RepositoryPath: filepath.Join("java", "transpile"),

				ExpectedRepositoryAssessment: map[evaltask.Identifier]metrics.Assessments{
					IdentifierTranspile: metrics.Assessments{
						metrics.AssessmentKeyTestsPassing:                  60,
						metrics.AssessmentKeyFilesExecuted:                 2,
						metrics.AssessmentKeyFilesExecutedMaximumReachable: 2,
						metrics.AssessmentKeyResponseNoError:               2,
					},
					IdentifierTranspileSymflowerFix: metrics.Assessments{
						metrics.AssessmentKeyTestsPassing:                  60,
						metrics.AssessmentKeyFilesExecuted:                 2,
						metrics.AssessmentKeyFilesExecutedMaximumReachable: 2,
						metrics.AssessmentKeyResponseNoError:               2,
					},
				},
				ValidateLog: func(t *testing.T, data string) {
					assert.Contains(t, data, "BUILD SUCCESS")
				},
			})
		}
		{
			temporaryDirectoryPath := t.TempDir()

			repositoryPath := filepath.Join(temporaryDirectoryPath, "java", "transpile")
			require.NoError(t, osutil.CopyTree(filepath.Join("..", "..", "testdata", "java", "transpile", "cascadingIfElse"), filepath.Join(repositoryPath, "cascadingIfElse")))
			require.NoError(t, osutil.CopyTree(filepath.Join("..", "..", "testdata", "java", "transpile", "isSorted"), filepath.Join(repositoryPath, "isSorted")))

			modelMock := modeltesting.NewMockCapabilityTranspileNamed(t, "mocked-model")

			transpiledSourceFilePath := filepath.Join("src", "main", "java", "com", "eval", "CascadingIfElse.java")
			transpiledSourceFileContent := bytesutil.StringTrimIndentations(`
				package com.eval;

				class CascadingIfElse {
					static int cascadingIfElse(int i) {
						if (i == 1) {
							return 2;
						} else if (i == 3) {
							return 4;
						} else {
							return 5;
						}
					}
				}
			`)
			modelMock.RegisterGenerateSuccess(t, transpiledSourceFilePath, transpiledSourceFileContent, metricstesting.AssessmentsWithProcessingTime).Times(2)

			transpiledSourceFilePath = filepath.Join("src", "main", "java", "com", "eval", "IsSorted.java")
			transpiledSourceFileContent = bytesutil.StringTrimIndentations(`
				package com.eval;

				class IsSorted {
					static boolean isSorted(int[] a) {
						int i = 0;
						while (i < a.length - 1 && a[i] <= a[i + 1]) {
							i++;
						}

						return i == a.length - 1;
					}
				}
			`)
			modelMock.RegisterGenerateSuccess(t, transpiledSourceFilePath, transpiledSourceFileContent, metricstesting.AssessmentsWithProcessingTime).Times(2)

			validate(t, &tasktesting.TestCaseTask{
				Name: "Multiple test cases",

				Model:          modelMock,
				Language:       &java.Language{},
				TestDataPath:   temporaryDirectoryPath,
				RepositoryPath: filepath.Join("java", "transpile"),

				ExpectedRepositoryAssessment: map[evaltask.Identifier]metrics.Assessments{
					IdentifierTranspile: metrics.Assessments{
						metrics.AssessmentKeyTestsPassing:                  160,
						metrics.AssessmentKeyFilesExecuted:                 4,
						metrics.AssessmentKeyFilesExecutedMaximumReachable: 4,
						metrics.AssessmentKeyResponseNoError:               4,
					},
					IdentifierTranspileSymflowerFix: metrics.Assessments{
						metrics.AssessmentKeyTestsPassing:                  160,
						metrics.AssessmentKeyFilesExecuted:                 4,
						metrics.AssessmentKeyFilesExecutedMaximumReachable: 4,
						metrics.AssessmentKeyResponseNoError:               4,
					},
				},
				ValidateLog: func(t *testing.T, data string) {
					assert.Contains(t, data, "BUILD SUCCESS")
				},
			})
		}
	})
	t.Run("Symflower fix", func(t *testing.T) {
		{
			temporaryDirectoryPath := t.TempDir()

			repositoryPath := filepath.Join(temporaryDirectoryPath, "golang", "transpile", "cascadingIfElse")
			require.NoError(t, osutil.CopyTree(filepath.Join("..", "..", "testdata", "golang", "transpile", "cascadingIfElse"), repositoryPath))

			modelMock := modeltesting.NewMockCapabilityTranspileNamed(t, "mocked-model")

			transpiledSourceFilePath := "cascadingIfElse.go"
			transpiledSourceFileContent := bytesutil.StringTrimIndentations(`
				package cascadingIfElse

				import "strings"

			 	func cascadingIfElse(i int) int {
			 		if i == 1 {
			 			return 2
			 		} else if i == 3 {
			 			return 4
			 		} else {
			 			return 5
			 		}
			 	}
			`)
			modelMock.RegisterGenerateSuccess(t, transpiledSourceFilePath, transpiledSourceFileContent, metricstesting.AssessmentsWithProcessingTime).Times(2)

			validate(t, &tasktesting.TestCaseTask{
				Name: "Model generated test with unused import",

				Model:          modelMock,
				Language:       &golang.Language{},
				TestDataPath:   temporaryDirectoryPath,
				RepositoryPath: filepath.Join("golang", "transpile"),

				ExpectedRepositoryAssessment: map[evaltask.Identifier]metrics.Assessments{
					IdentifierTranspile: metrics.Assessments{
						metrics.AssessmentKeyTestsPassing:                  0,
						metrics.AssessmentKeyResponseNoError:               2,
						metrics.AssessmentKeyFilesExecutedMaximumReachable: 2,
					},
					IdentifierTranspileSymflowerFix: metrics.Assessments{
						metrics.AssessmentKeyTestsPassing:                  80,
						metrics.AssessmentKeyFilesExecuted:                 2,
						metrics.AssessmentKeyFilesExecutedMaximumReachable: 2,
						metrics.AssessmentKeyResponseNoError:               2,
					},
				},
				ExpectedProblemContains: []string{
					"imported and not used",
					"imported and not used",
				},
				ValidateLog: func(t *testing.T, data string) {
					assert.Contains(t, data, "PASS: TestSymflowerCascadingIfElse/#00")
					assert.Contains(t, data, "PASS: TestSymflowerCascadingIfElse/#01")
					assert.Contains(t, data, "PASS: TestSymflowerCascadingIfElse/#02")
				},
			})
		}
	})
}

func TestValidateTranspileRepository(t *testing.T) {
	validate := func(t *testing.T, tc *tasktesting.TestCaseValidateRepository) {
		tc.Validate(t, validateTranspileRepository)
	}

	validate(t, &tasktesting.TestCaseValidateRepository{
		Name: "Package does not contain implementation folder",

		Before: func(repositoryPath string) {
			require.NoError(t, os.MkdirAll(filepath.Join(repositoryPath, "somePackage"), 0700))
		},

		TestdataPath:   filepath.Join("..", "..", "testdata"),
		RepositoryPath: filepath.Join("golang", "transpile"),
		Language:       &golang.Language{},

		ExpectedError: func(err error) {
			var errorMessage string
			if osutil.IsWindows() {
				errorMessage = "The system cannot find the file specified"
			} else {
				errorMessage = "no such file or directory"
			}
			assert.ErrorContains(t, err, errorMessage)
		},
	})
	validate(t, &tasktesting.TestCaseValidateRepository{
		Name: "Implementation folder contains multiple files of the same language",

		Before: func(repositoryPath string) {
			require.NoError(t, os.WriteFile(filepath.Join(repositoryPath, "balancedBrackets", "implementation", "Class.java"), []byte(`content`), 0700))
		},

		TestdataPath:   filepath.Join("..", "..", "testdata"),
		RepositoryPath: filepath.Join("golang", "transpile"),
		Language:       &golang.Language{},

		ExpectedError: func(err error) {
			assert.ErrorContains(t, err, "must contain only one source file per language")
		},
	})
	validate(t, &tasktesting.TestCaseValidateRepository{
		Name: "Implementation folder does not contain all required languages",

		Before: func(repositoryPath string) {
			implementationPath := filepath.Join(repositoryPath, "somePackage", "implementation")
			require.NoError(t, os.MkdirAll(implementationPath, 0700))
			require.NoError(t, os.WriteFile(filepath.Join(implementationPath, "Class.java"), []byte(`content`), 0700))
		},

		TestdataPath:   filepath.Join("..", "..", "testdata"),
		RepositoryPath: filepath.Join("golang", "transpile"),
		Language:       &golang.Language{},

		ExpectedError: func(t *testing.T, err error) {
			assert.ErrorContains(t, err, "must contain source files for every language to prevent imbalance")
		},
	})
	validate(t, &tasktesting.TestCaseValidateRepository{
		Name: "Implementation folder must contain only files",

		Before: func(repositoryPath string) {
			require.NoError(t, os.MkdirAll(filepath.Join(repositoryPath, "somePackage", "implementation", "someFolder"), 0700))
		},

		TestdataPath:   filepath.Join("..", "..", "testdata"),
		RepositoryPath: filepath.Join("golang", "transpile"),
		Language:       &golang.Language{},

		ExpectedError: func(err error) {
			assert.ErrorContains(t, err, "must contain only source code files to transpile, but found one directory")
		},
	})
	validate(t, &tasktesting.TestCaseValidateRepository{
		Name: "Implementation folder must contain only source files",

		Before: func(repositoryPath string) {
			implementationFolderPath := filepath.Join(repositoryPath, "somePackage", "implementation")
			require.NoError(t, os.MkdirAll(implementationFolderPath, 0700))
			require.NoError(t, os.WriteFile(filepath.Join(implementationFolderPath, "ClassTest.java"), []byte(`content`), 0700))
		},

		TestdataPath:   filepath.Join("..", "..", "testdata"),
		RepositoryPath: filepath.Join("golang", "transpile"),
		Language:       &golang.Language{},

		ExpectedError: func(err error) {
			assert.ErrorContains(t, err, "must contain source files, but found a test file")
		},
	})
	validate(t, &tasktesting.TestCaseValidateRepository{
		Name: "Unsupported language",

		Before: func(repositoryPath string) {
			implementationFolderPath := filepath.Join(repositoryPath, "somePackage", "implementation")
			require.NoError(t, os.MkdirAll(implementationFolderPath, 0700))
			require.NoError(t, os.WriteFile(filepath.Join(implementationFolderPath, "file.unsupported"), []byte(`content`), 0700))
		},

		TestdataPath:   filepath.Join("..", "..", "testdata"),
		RepositoryPath: filepath.Join("golang", "transpile"),
		Language:       &golang.Language{},

		ExpectedError: func(err error) {
			assert.ErrorContains(t, err, "the language extension \".unsupported\" is not supported")
		},
	})
	t.Run("Go", func(t *testing.T) {
		validate(t, &tasktesting.TestCaseValidateRepository{
			Name: "Package without source file",

			Before: func(repositoryPath string) {
				implementationPath := filepath.Join(repositoryPath, "somePackage", "implementation")
				require.NoError(t, os.MkdirAll(implementationPath, 0700))
				require.NoError(t, os.WriteFile(filepath.Join(implementationPath, "Class.java"), []byte(`content`), 0700))
				require.NoError(t, os.WriteFile(filepath.Join(implementationPath, "file.rb"), []byte(`content`), 0700))
			},

			TestdataPath:   filepath.Join("..", "..", "testdata"),
			RepositoryPath: filepath.Join("golang", "transpile"),
			Language:       &golang.Language{},

			ExpectedError: func(err error) {
				assert.ErrorContains(t, err, "must contain exactly one Go source file")
			},
		})
		validate(t, &tasktesting.TestCaseValidateRepository{
			Name: "Package without test file",

			Before: func(repositoryPath string) {
				implementationPath := filepath.Join(repositoryPath, "somePackage", "implementation")
				require.NoError(t, os.MkdirAll(implementationPath, 0700))
				require.NoError(t, os.WriteFile(filepath.Join(implementationPath, "Class.java"), []byte(`content`), 0700))
				require.NoError(t, os.WriteFile(filepath.Join(implementationPath, "file.rb"), []byte(`content`), 0700))
				require.NoError(t, os.WriteFile(filepath.Join(repositoryPath, "somePackage", "file.go"), []byte(`content`), 0700))
			},

			TestdataPath:   filepath.Join("..", "..", "testdata"),
			RepositoryPath: filepath.Join("golang", "transpile"),
			Language:       &golang.Language{},

			ExpectedError: func(err error) {
				assert.ErrorContains(t, err, "must contain exactly one Go test file")
			},
		})
		validate(t, &tasktesting.TestCaseValidateRepository{
			Name: "Well-formed",

			Before: func(repositoryPath string) {
				require.NoError(t, osutil.MkdirAll(filepath.Join(repositoryPath, ".git")))
				require.NoError(t, os.WriteFile(filepath.Join(repositoryPath, ".git", "index"), []byte(`content`), 0700))
			},

			TestdataPath:   filepath.Join("..", "..", "testdata"),
			RepositoryPath: filepath.Join("golang", "transpile"),
			Language:       &golang.Language{},
		})
	})
	t.Run("Java", func(t *testing.T) {
		validate(t, &tasktesting.TestCaseValidateRepository{
			Name: "Package without source file",

			Before: func(repositoryPath string) {
				implementationPath := filepath.Join(repositoryPath, "somePackage", "implementation")
				require.NoError(t, os.MkdirAll(implementationPath, 0700))
				require.NoError(t, os.WriteFile(filepath.Join(implementationPath, "file.go"), []byte(`content`), 0700))
				require.NoError(t, os.WriteFile(filepath.Join(implementationPath, "file.rb"), []byte(`content`), 0700))
			},

			TestdataPath:   filepath.Join("..", "..", "testdata"),
			RepositoryPath: filepath.Join("java", "transpile"),
			Language:       &java.Language{},

			ExpectedError: func(err error) {
				assert.ErrorContains(t, err, "must contain exactly one Java source file")
			},
		})
		validate(t, &tasktesting.TestCaseValidateRepository{
			Name: "Package without test file",

			Before: func(repositoryPath string) {
				implementationPath := filepath.Join(repositoryPath, "somePackage", "implementation")
				require.NoError(t, os.MkdirAll(implementationPath, 0700))
				require.NoError(t, os.WriteFile(filepath.Join(implementationPath, "file.go"), []byte(`content`), 0700))
				require.NoError(t, os.WriteFile(filepath.Join(implementationPath, "file.rb"), []byte(`content`), 0700))
				require.NoError(t, os.WriteFile(filepath.Join(repositoryPath, "somePackage", "Class.java"), []byte(`content`), 0700))
			},

			TestdataPath:   filepath.Join("..", "..", "testdata"),
			RepositoryPath: filepath.Join("java", "transpile"),
			Language:       &java.Language{},

			ExpectedError: func(err error) {
				assert.ErrorContains(t, err, "must contain exactly one Java test file")
			},
		})
		validate(t, &tasktesting.TestCaseValidateRepository{
			Name: "Well-formed",

			Before: func(repositoryPath string) {
				require.NoError(t, osutil.MkdirAll(filepath.Join(repositoryPath, ".git")))
				require.NoError(t, os.WriteFile(filepath.Join(repositoryPath, ".git", "index"), []byte(`content`), 0700))
			},

			TestdataPath:   filepath.Join("..", "..", "testdata"),
			RepositoryPath: filepath.Join("java", "transpile"),
			Language:       &java.Language{},
		})
	})
}

func TestTaskTranspileUnpackTranspilerPackage(t *testing.T) {
	type testCase struct {
		Name string

		DestinationLanguage language.Language

		RepositoryPath string
		PackagePath    string

		ExpectedOriginFilePathsWithLanguage map[string]language.Language
		ExpectedStubFilePath                string
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			logOutput, logger := log.Buffer()
			defer func() {
				if t.Failed() {
					t.Logf("Logging output: %s", logOutput.String())
				}
			}()

			temporaryDirectory := t.TempDir()

			require.NoError(t, osutil.CopyTree(filepath.Join("..", "..", "testdata", tc.RepositoryPath), filepath.Join(temporaryDirectory, "testdata", tc.RepositoryPath)))

			repository, cleanup, err := TemporaryRepository(logger, filepath.Join(temporaryDirectory, "testdata"), tc.RepositoryPath)
			require.NoError(t, err)
			defer cleanup()

			taskTranspile := TaskTranspile{}
			ctx := evaltask.Context{
				Language:   tc.DestinationLanguage,
				Repository: repository,
			}
			actualOriginFilePathsWithLanguage, actualStubFilePath, actualErr := taskTranspile.unpackTranspilerPackage(ctx, logger, tc.PackagePath)
			require.NoError(t, actualErr)

			expectedOriginFilePathsWithLanguage := map[string]language.Language{}
			for filePath, language := range tc.ExpectedOriginFilePathsWithLanguage {
				expectedOriginFilePathsWithLanguage[filepath.Join(repository.DataPath(), tc.PackagePath, filePath)] = language
			}
			assert.Equal(t, expectedOriginFilePathsWithLanguage, actualOriginFilePathsWithLanguage)
			assert.Equal(t, tc.ExpectedStubFilePath, actualStubFilePath)
		})
	}

	t.Run("Go", func(t *testing.T) {
		validate(t, &testCase{
			Name: "Transpile",

			DestinationLanguage: &golang.Language{},

			RepositoryPath: filepath.Join("golang", "transpile"),
			PackagePath:    "binarySearch",

			ExpectedOriginFilePathsWithLanguage: map[string]language.Language{
				filepath.Join("implementation", "BinarySearch.java"): &java.Language{},
				filepath.Join("implementation", "binary_search.rb"):  &ruby.Language{},
			},
			ExpectedStubFilePath: "binarySearch.go",
		})
	})
	t.Run("Java", func(t *testing.T) {
		validate(t, &testCase{
			Name: "Transpile",

			DestinationLanguage: &java.Language{},

			RepositoryPath: filepath.Join("java", "transpile"),
			PackagePath:    "isSorted",

			ExpectedOriginFilePathsWithLanguage: map[string]language.Language{
				filepath.Join("implementation", "isSorted.go"): &golang.Language{},
				filepath.Join("implementation", "sort.rb"):     &ruby.Language{},
			},
			ExpectedStubFilePath: filepath.Join("src", "main", "java", "com", "eval", "IsSorted.java"),
		})
	})
}
