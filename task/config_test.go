package task

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zimmski/osutil/bytesutil"

	"github.com/symflower/eval-dev-quality/evaluate/metrics"
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

var configJSON = bytesutil.TrimIndentations([]byte(`
	{
		"tasks": ["task-identifier"],
		"ignore": ["some-path"],
		"prompt" : {
			"test-framework": "some-framework"
		},
		"validation": {
			"execution": {
				"stdout": ".*"
			}
		},
		"scores": {
			"task-identifier": {
				"case-identifier": {
					"metric-identifier": 123
				}
			}
		}
	}
`))

var configParsed = &RepositoryConfiguration{
	Tasks: []Identifier{
		Identifier("task-identifier"),
	},

	IgnorePaths: []string{
		"some-path",
	},

	Prompt: RepositoryConfigurationPrompt{
		TestFramework: "some-framework",
	},

	Validation: RepositoryConfigurationValidation{
		Execution: RepositoryConfigurationExecution{
			StdOutRE: ".*",
		},
	},

	MaxScores: map[Identifier]map[string]map[metrics.AssessmentKey]uint64{
		Identifier("task-identifier"): {
			"case-identifier": {
				metrics.AssessmentKey("metric-identifier"): 123,
			},
		},
	},
}

func TestLoadRepositoryConfiguration(t *testing.T) {
	configurationPath := filepath.Join(t.TempDir(), "repository.json")
	require.NoError(t, os.WriteFile(configurationPath, configJSON, 0666))

	configuration, err := LoadRepositoryConfiguration(configurationPath, nil)
	assert.NoError(t, err)
	assert.Equal(t, configParsed, configuration)
}

func TestWriteRepositoryConfiguration(t *testing.T) {
	configurationPath := filepath.Join(t.TempDir(), "repository.json")
	assert.NoError(t, configParsed.Write(configurationPath))

	data, err := os.ReadFile(configurationPath)
	require.NoError(t, err)
	assert.Equal(t, strings.NewReplacer("\n", "", "\t", "", " ", "").Replace(string(configJSON)), string(data))
}
