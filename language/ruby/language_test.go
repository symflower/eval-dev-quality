package ruby

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLanguageTestFilePath(t *testing.T) {
	type testCase struct {
		Name string

		ProjectRootPath string
		FilePath        string

		ExpectedTestFilePath string
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			ruby := Language{}
			actualTestFilePath := ruby.TestFilePath(tc.ProjectRootPath, tc.FilePath)

			assert.Equal(t, tc.ExpectedTestFilePath, actualTestFilePath)
		})
	}

	validate(t, &testCase{
		Name: "Source file",

		FilePath: filepath.Join("testdata", "ruby", "plain", "lib", "plain.rb"),

		ExpectedTestFilePath: filepath.Join("testdata", "ruby", "plain", "test", "plain_test.rb"),
	})
}

func TestLanguageImportPath(t *testing.T) {
	type testCase struct {
		Name string

		ProjectRootPath string
		FilePath        string

		ExpectedImportPath string
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			ruby := Language{}
			actualImportPath := ruby.ImportPath(tc.ProjectRootPath, tc.FilePath)

			assert.Equal(t, tc.ExpectedImportPath, actualImportPath)
		})
	}

	validate(t, &testCase{
		Name: "Source file",

		FilePath: filepath.Join("testdata", "ruby", "plain", "lib", "plain.rb"),

		ExpectedImportPath: "../lib/plain",
	})
}
