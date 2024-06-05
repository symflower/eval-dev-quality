package java

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zimmski/osutil"
	"github.com/zimmski/osutil/bytesutil"

	"github.com/symflower/eval-dev-quality/log"
)

func TestLanguageFiles(t *testing.T) {
	type testCase struct {
		Name string

		Language *Language

		RepositoryPath string

		ExpectedFilePaths []string
		ExpectedError     error
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			logOutput, logger := log.Buffer()
			defer func() {
				if t.Failed() {
					t.Log(logOutput.String())
				}
			}()

			if tc.Language == nil {
				tc.Language = &Language{}
			}
			actualFilePaths, actualError := tc.Language.Files(logger, tc.RepositoryPath)

			assert.Equal(t, tc.ExpectedFilePaths, actualFilePaths)
			assert.Equal(t, tc.ExpectedError, actualError)
		})
	}

	validate(t, &testCase{
		Name: "Plain",

		RepositoryPath: filepath.Join("..", "..", "testdata", "java", "plain"),

		ExpectedFilePaths: []string{
			filepath.Join("src", "main", "java", "com", "eval", "Plain.java"),
		},
	})
}

func TestLanguageImportPath(t *testing.T) {
	type testCase struct {
		Name string

		Language *Language

		ProjectRootPath string
		FilePath        string

		ExpectedImportPath string
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			if tc.Language == nil {
				tc.Language = &Language{}
			}

			actualImportPath := tc.Language.ImportPath(tc.ProjectRootPath, tc.FilePath)

			assert.Equal(t, tc.ExpectedImportPath, actualImportPath)
		})
	}

	validate(t, &testCase{
		Name: "Source file",

		FilePath: "src/main/java/com/eval/Plain.java",

		ExpectedImportPath: "com.eval",
	})
}

func TestLanguageTestFilePath(t *testing.T) {
	type testCase struct {
		Name string

		Language *Language

		ProjectRootPath string
		FilePath        string

		ExpectedTestFilePath string
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			if tc.Language == nil {
				tc.Language = &Language{}
			}

			actualTestFilePath := tc.Language.TestFilePath(tc.ProjectRootPath, tc.FilePath)

			assert.Equal(t, tc.ExpectedTestFilePath, actualTestFilePath)
		})
	}

	validate(t, &testCase{
		Name: "Source file",

		FilePath: filepath.Join("src", "main", "java", "com", "eval", "Plain.java"),

		ExpectedTestFilePath: filepath.Join("src", "test", "java", "com", "eval", "PlainTest.java"),
	})
}

func TestLanguageExecute(t *testing.T) {
	type testCase struct {
		Name string

		Language *Language

		RepositoryPath   string
		RepositoryChange func(t *testing.T, repositoryPath string)

		ExpectedCoverage     uint64
		ExpectedProblemTexts []string
		ExpectedError        error
		ExpectedErrorText    string
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			logOutput, logger := log.Buffer()
			defer func() {
				if t.Failed() {
					t.Logf("Logging output: %s", logOutput.String())
				}
			}()

			temporaryPath := t.TempDir()
			repositoryPath := filepath.Join(temporaryPath, filepath.Base(tc.RepositoryPath))
			require.NoError(t, osutil.CopyTree(tc.RepositoryPath, repositoryPath))

			if tc.RepositoryChange != nil {
				tc.RepositoryChange(t, repositoryPath)
			}

			if tc.Language == nil {
				tc.Language = &Language{}
			}
			actualCoverage, actualProblems, actualError := tc.Language.Execute(logger, repositoryPath)

			require.Equal(t, len(tc.ExpectedProblemTexts), len(actualProblems), "the number of expected problems need to match the number of actual problems")
			for i, expectedProblemText := range tc.ExpectedProblemTexts {
				assert.ErrorContains(t, actualProblems[i], expectedProblemText)
			}

			if tc.ExpectedError != nil {
				assert.ErrorIs(t, actualError, tc.ExpectedError)
			} else if actualError != nil && tc.ExpectedErrorText != "" {
				assert.ErrorContains(t, actualError, tc.ExpectedErrorText)
			} else {
				assert.NoError(t, actualError)
				assert.Equal(t, tc.ExpectedCoverage, actualCoverage)
			}
		})
	}

	validate(t, &testCase{
		Name: "No test files",

		RepositoryPath: filepath.Join("..", "..", "testdata", "java", "plain"),

		ExpectedCoverage:  0,
		ExpectedErrorText: "exit status 1",
	})

	t.Run("With test file", func(t *testing.T) {
		validate(t, &testCase{
			Name: "Valid",

			RepositoryPath: filepath.Join("..", "..", "testdata", "java", "plain"),
			RepositoryChange: func(t *testing.T, repositoryPath string) {
				javaTestFilePath := filepath.Join(repositoryPath, "src/test/java/com/eval/PlainSymflowerTest.java")
				require.NoError(t, os.MkdirAll(filepath.Dir(javaTestFilePath), 0755))
				require.NoError(t, os.WriteFile(javaTestFilePath, []byte(bytesutil.StringTrimIndentations(`
					package com.eval;

					import org.junit.jupiter.api.*;

					public class PlainSymflowerTest {
						@Test
						public void plain1() {
							Plain.plain();
						}
					}
				`)), 0660))
			},

			ExpectedCoverage: 1,
		})

		validate(t, &testCase{
			Name: "Failing tests",

			RepositoryPath: filepath.Join("..", "..", "testdata", "java", "light"),
			RepositoryChange: func(t *testing.T, repositoryPath string) {
				javaTestFilePath := filepath.Join(repositoryPath, "src/test/java/com/eval/SimpleIfElseSymflowerTest.java")
				require.NoError(t, os.MkdirAll(filepath.Dir(javaTestFilePath), 0755))
				require.NoError(t, os.WriteFile(javaTestFilePath, []byte(bytesutil.StringTrimIndentations(`
					package com.eval;

					import org.junit.jupiter.api.*;

					public class SimpleIfElseSymflowerTest {
						@Test
						public void simpleIfElse() {
							int actual = SimpleIfElse.simpleIfElse(1); // Get some coverage...
							Assertions.assertEquals(true, false); // ... and then fail.
						}
					}
				`)), 0660))
			},

			ExpectedCoverage: 3,
		})

		validate(t, &testCase{
			Name: "Syntax error",

			RepositoryPath: filepath.Join("..", "..", "testdata", "java", "plain"),
			RepositoryChange: func(t *testing.T, repositoryPath string) {
				javaTestFilePath := filepath.Join(repositoryPath, "src/test/java/com/eval/PlainSymflowerTest.java")
				require.NoError(t, os.MkdirAll(filepath.Dir(javaTestFilePath), 0755))
				require.NoError(t, os.WriteFile(javaTestFilePath, []byte(bytesutil.StringTrimIndentations(`
					foobar
				`)), 0660))
			},

			ExpectedErrorText: "exit status 1",
		})
	})
}
