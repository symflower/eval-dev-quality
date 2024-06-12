package golang

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

		RepositoryPath: filepath.Join("..", "..", "testdata", "golang", "plain"),

		ExpectedFilePaths: []string{
			"plain.go",
		},
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
					t.Log(logOutput.String())
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

		RepositoryPath: filepath.Join("..", "..", "testdata", "golang", "plain"),

		ExpectedCoverage:  0,
		ExpectedErrorText: "exit status 1",
	})

	t.Run("With test file", func(t *testing.T) {
		validate(t, &testCase{
			Name: "Valid",

			RepositoryPath: filepath.Join("..", "..", "testdata", "golang", "plain"),
			RepositoryChange: func(t *testing.T, repositoryPath string) {
				require.NoError(t, os.WriteFile(filepath.Join(repositoryPath, "plain_test.go"), []byte(bytesutil.StringTrimIndentations(`
					package plain

					import (
						"testing"
					)

					func TestPlain(t *testing.T) {
						plain()
					}
				`)), 0660))
			},

			ExpectedCoverage: 1,
		})

		validate(t, &testCase{
			Name: "Failing tests",

			RepositoryPath: filepath.Join("..", "..", "testdata", "golang", "light"),
			RepositoryChange: func(t *testing.T, repositoryPath string) {
				require.NoError(t, os.WriteFile(filepath.Join(repositoryPath, "simpleIfElse_test.go"), []byte(bytesutil.StringTrimIndentations(`
					package light

					import (
						"testing"
					)

					func TestSimpleIfElse(t *testing.T) {
						simpleIfElse(1) // Get some coverage...
						t.Fail() // ...and then fail.
					}
				`)), 0660))
			},

			ExpectedCoverage: 1,
			ExpectedProblemTexts: []string{
				"exit status 1", // Test execution fails.
			},
		})

		validate(t, &testCase{
			Name: "Syntax error",

			RepositoryPath: filepath.Join("..", "..", "testdata", "golang", "plain"),
			RepositoryChange: func(t *testing.T, repositoryPath string) {
				require.NoError(t, os.WriteFile(filepath.Join(repositoryPath, "plain_test.go"), []byte(bytesutil.StringTrimIndentations(`
					foobar
				`)), 0660))
			},

			ExpectedErrorText: "exit status 1",
		})
	})
}

func TestMistakes(t *testing.T) {
	type testCase struct {
		Name string

		RepositoryPath string

		ExpectedMistakes []string
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			temporaryPath := t.TempDir()
			repositoryPath := filepath.Join(temporaryPath, filepath.Base(tc.RepositoryPath))
			require.NoError(t, osutil.CopyTree(tc.RepositoryPath, repositoryPath))

			buffer, logger := log.Buffer()
			defer func() {
				if t.Failed() {
					t.Log(buffer.String())
				}
			}()

			golang := &Language{}
			actualMistakes, actualErr := golang.Mistakes(logger, repositoryPath)
			require.NoError(t, actualErr)

			assert.Equal(t, tc.ExpectedMistakes, actualMistakes)
		})
	}

	validate(t, &testCase{
		Name: "Function without opening bracket",

		RepositoryPath: filepath.Join("..", "..", "testdata", "golang", "mistakes"),

		ExpectedMistakes: []string{
			"." + string(os.PathSeparator) + "functionWithoutOpeningBracket.go" + ":4:6: syntax error: non-declaration statement outside function body",
		},
	})
}

func TestExtractMistakes(t *testing.T) {
	type testCase struct {
		Name string

		RawMistakes string

		ExpectedMistakes []string
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			actualMistakes := extractMistakes(tc.RawMistakes)

			assert.Equal(t, tc.ExpectedMistakes, actualMistakes)
		})
	}

	validate(t, &testCase{
		Name: "Single mistake",

		RawMistakes: bytesutil.StringTrimIndentations(`
			foobar
			# foobar
			./foobar.go:4:2: syntax error: non-declaration statement outside function body
		`),

		ExpectedMistakes: []string{
			"./foobar.go:4:2: syntax error: non-declaration statement outside function body",
		},
	})
	validate(t, &testCase{
		Name: "Multiple mistakes",

		RawMistakes: bytesutil.StringTrimIndentations(`
			foobar
			# foobar
			./foobar.go:3:1: expected 'IDENT', found 'func'
			./foobar.go:4:2: syntax error: non-declaration statement outside function body
			./foobar.go:5:1: missing return
		`),

		ExpectedMistakes: []string{
			"./foobar.go:3:1: expected 'IDENT', found 'func'",
			"./foobar.go:4:2: syntax error: non-declaration statement outside function body",
			"./foobar.go:5:1: missing return",
		},
	})
}
