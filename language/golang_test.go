package language

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zimmski/osutil"
	"github.com/zimmski/osutil/bytesutil"
)

func TestLanguageGolangFiles(t *testing.T) {
	type testCase struct {
		Name string

		LanguageGolang *LanguageGolang

		RepositoryPath string

		ExpectedFilePaths []string
		ExpectedError     error
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			if tc.LanguageGolang == nil {
				tc.LanguageGolang = &LanguageGolang{}
			}
			actualFilePaths, actualError := tc.LanguageGolang.Files(tc.RepositoryPath)

			assert.Equal(t, tc.ExpectedFilePaths, actualFilePaths)
			assert.Equal(t, tc.ExpectedError, actualError)
		})
	}

	validate(t, &testCase{
		Name: "Plain",

		RepositoryPath: "../testdata/golang/plain/",

		ExpectedFilePaths: []string{
			"plain.go",
		},
	})
}

func TestLanguageGolangExecute(t *testing.T) {
	type testCase struct {
		Name string

		LanguageGolang *LanguageGolang

		RepositoryPath   string
		RepositoryChange func(t *testing.T, repositoryPath string)

		ExpectedCoverage  float64
		ExpectedError     error
		ExpectedErrorText string
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			temporaryPath := t.TempDir()
			repositoryPath := filepath.Join(temporaryPath, filepath.Base(tc.RepositoryPath))
			require.NoError(t, osutil.CopyTree(tc.RepositoryPath, repositoryPath))

			if tc.RepositoryChange != nil {
				tc.RepositoryChange(t, repositoryPath)
			}

			if tc.LanguageGolang == nil {
				tc.LanguageGolang = &LanguageGolang{}
			}
			actualCoverage, actualError := tc.LanguageGolang.Execute(repositoryPath)

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

		RepositoryPath: "../testdata/golang/plain/",

		ExpectedError: ErrNoTestFound,
	})

	t.Run("With test file", func(t *testing.T) {
		validate(t, &testCase{
			Name: "Valid",

			RepositoryPath: "../testdata/golang/plain/",
			RepositoryChange: func(t *testing.T, repositoryPath string) {
				require.NoError(t, os.WriteFile(filepath.Join(repositoryPath, "plain_test.go"), []byte(bytesutil.StringTrimIndentations(`
					package native

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

			RepositoryPath: "../testdata/golang/plain/",
			RepositoryChange: func(t *testing.T, repositoryPath string) {
				require.NoError(t, os.WriteFile(filepath.Join(repositoryPath, "plain_test.go"), []byte(bytesutil.StringTrimIndentations(`
					foobar
				`)), 0660))
			},

			ExpectedErrorText: "exit status 1",
		})
	})
}
