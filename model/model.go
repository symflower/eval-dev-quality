package model

import (
	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	"github.com/symflower/eval-dev-quality/task"
)

// Model defines a model that can be queried for generations.
type Model interface {
	// ID returns the unique ID of this model.
	ID() (id string)

	// IsTaskSupported returns whether the model supports the given task or not.
	IsTaskSupported(taskIdentifier task.Identifier) (isSupported bool)
	// RunTask runs the given task.
	RunTask(ctx task.Context, taskIdentifier task.Identifier) (assessments metrics.Assessments, err error)

	// Cost returns the cost of a model in US dollars.
	Cost() (cost float64)
}

// SetQueryAttempts defines a model that can set the number of query attempts when a model request errors in the process of solving a task.
type SetQueryAttempts interface {
	// SetQueryAttempts sets the number of query attempts to perform when a model request errors in the process of solving a task.
	SetQueryAttempts(attempts uint)
}
