package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zimmski/osutil"
	"github.com/zimmski/osutil/bytesutil"
)

func TestExecute(t *testing.T) {
	type testCase struct {
		Name string

		Arguments []string

		ExpectedOutput string
		ExpectedError  error
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			actualOutput, actualError := osutil.Capture(func() {
				Execute(tc.Arguments)
			})

			assert.Equal(t, tc.ExpectedOutput, string(actualOutput))
			assert.Equal(t, tc.ExpectedError, actualError)
		})
	}

	validate(t, &testCase{
		Name: "No arguments should show help",

		ExpectedOutput: bytesutil.StringTrimIndentations(`
			Usage:
			  eval-dev-quality [OPTIONS] [evaluate | install-tools]

			Command to manage, update and actually execute the ` + "`" + `eval-dev-quality` + "`" + `
			evaluation benchmark.

			Help Options:
			  -h, --help  Show this help message

			Available commands:
			  evaluate       Run an evaluation, by default with all defined models, repositories and tasks.
			  install-tools  Checks and installs all tools required for the evaluation benchmark.
		`),
	})
}
