package util

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/symflower/eval-dev-quality/log"
)

func TestCommandWithResultTimeout(t *testing.T) {
	logOutput, logger := log.Buffer()
	defer func() {
		if t.Failed() {
			t.Log(logOutput.String())
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(1*time.Second))
	defer cancel()

	start := time.Now()
	_, err := CommandWithResult(ctx, logger, &Command{
		Command: []string{
			"sleep",
			"60",
		},
	})
	duration := time.Since(start)

	assert.Error(t, err)
	assert.Less(t, duration.Seconds(), 5.0)
}

func TestFilterArgs(t *testing.T) {
	type testCase struct {
		Name string

		Args    []string
		Ignored []string

		ExpectedFiltered []string
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			actualFiltered := FilterArgs(tc.Args, tc.Ignored)

			assert.Equal(t, tc.ExpectedFiltered, actualFiltered)
		})
	}

	validate(t, &testCase{
		Name: "Filter arguments",

		Args: []string{
			"--runtime",
			"abc",
			"--runs",
			"5",
		},
		Ignored: []string{
			"--runtime",
		},

		ExpectedFiltered: []string{
			"--runs",
			"5",
		},
	})

	validate(t, &testCase{
		Name: "Filter arguments with equals sign",

		Args: []string{
			"--runtime=abc",
			"--runs=5",
			"--foo",
			"bar",
		},
		Ignored: []string{
			"--runtime",
		},

		ExpectedFiltered: []string{
			"--runs",
			"5",
			"--foo",
			"bar",
		},
	})
}
