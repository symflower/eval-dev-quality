package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCleanModelNameForFileSystem(t *testing.T) {
	type testCase struct {
		Name string

		ModelName string

		ExpectedModelNameCleaned string
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			actualModelNameCleaned := CleanModelNameForFileSystem(tc.ModelName)

			assert.Equal(t, tc.ExpectedModelNameCleaned, actualModelNameCleaned)
		})
	}

	validate(t, &testCase{
		Name: "Simple",

		ModelName: "openrouter/anthropic/claude-2.0:beta",

		ExpectedModelNameCleaned: "openrouter_anthropic_claude-2.0_beta",
	})
}
