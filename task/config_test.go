package task

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigurationIsFilePathIgnored(t *testing.T) {
	type testCase struct {
		Name string

		IgnoredPaths []string
		FilePath     string

		ExpectedBool bool
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			actualBool := (&RepositoryConfiguration{
				IgnorePaths: tc.IgnoredPaths,
			}).IsFilePathIgnored(tc.FilePath)

			assert.Equal(t, tc.ExpectedBool, actualBool)
		})
	}

	validate(t, &testCase{
		Name: "Exact Match",

		IgnoredPaths: []string{
			"foo/bar.txt",
		},
		FilePath: "foo/bar.txt",

		ExpectedBool: true,
	})
	validate(t, &testCase{
		Name: "No Match",

		IgnoredPaths: []string{
			"foo/bar.txt",
		},
		FilePath: "foo/baz.txt",

		ExpectedBool: false,
	})
	validate(t, &testCase{
		Name: "Folder",

		IgnoredPaths: []string{
			"foo",
		},
		FilePath: "foo/bar.txt",

		ExpectedBool: true,
	})
}
