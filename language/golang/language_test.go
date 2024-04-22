package golang

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zimmski/osutil"
	"github.com/zimmski/osutil/bytesutil"

	"github.com/symflower/eval-dev-quality/language"
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

		RepositoryPath: "../../testdata/golang/plain/",

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

		ExpectedCoverage  float64
		ExpectedError     error
		ExpectedErrorText string
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
			actualCoverage, actualError := tc.Language.Execute(logger, repositoryPath)

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

		RepositoryPath: "../../testdata/golang/plain/",

		ExpectedError: language.ErrNoTestFound,
	})

	t.Run("With test file", func(t *testing.T) {
		validate(t, &testCase{
			Name: "Valid",

			RepositoryPath: "../../testdata/golang/plain/",
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

			ExpectedCoverage: 100,
		})

		validate(t, &testCase{
			Name: "Syntax error",

			RepositoryPath: "../../testdata/golang/plain/",
			RepositoryChange: func(t *testing.T, repositoryPath string) {
				require.NoError(t, os.WriteFile(filepath.Join(repositoryPath, "plain_test.go"), []byte(bytesutil.StringTrimIndentations(`
					foobar
				`)), 0660))
			},

			ExpectedErrorText: "exit status 1",
		})
	})
}
