package task

import (
	"errors"
	"log"

	"github.com/symflower/eval-dev-quality/evaluate/metrics"
)

var (
	// ErrTaskUnknown indicates that a task is unknown.
	ErrTaskUnknown = errors.New("task unknown")
	// ErrTaskUnsupportedByModel indicates that the model does not support the task.
	ErrTaskUnsupportedByModel = errors.New("model does not support task")
)

// Identifier holds the identifier of a task.
type Identifier string

// Task defines an evaluation task.
type Task interface {
	// Identifier returns the task identifier.
	Identifier() (identifier Identifier)

	// Run runs a task in a given repository.
	Run(repository Repository) (assessments map[Identifier]metrics.Assessments, problems []error, err error)
}

// Repository defines a repository to be evaluated.
type Repository interface {
	// Name holds the name of the repository.
	Name() (name string)
	// DataPath holds the absolute path to the repository.
	DataPath() (dataPath string)

	// SupportedTasks returns the list of task identifiers the repository supports.
	SupportedTasks() (tasks []Identifier)

	// Reset resets the repository to its initial state.
	Reset(logger *log.Logger) (err error)
}
