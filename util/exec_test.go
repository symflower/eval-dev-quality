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
