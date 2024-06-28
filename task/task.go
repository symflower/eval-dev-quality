package task

import (
	"errors"
	"log"

	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	"github.com/symflower/eval-dev-quality/language"
)

var (
	// ErrTaskUnsupported indicates that a task is unsupported.
	ErrTaskUnsupported = errors.New("task unsupported")
)

// Identifier holds the identifier of a task.
type Identifier string

// Context holds the data needed for running a task.
type Context struct {
	// Language holds the language for which the task should be evaluated.
	Language language.Language

	// RepositoryPath holds the absolute path to the repository.
	RepositoryPath string
	// FilePath holds the path the file under test relative to the repository path.
	FilePath string

	// Arguments holds extra data that can be used in a query prompt.
	Arguments any

	// Logger is used for logging during evaluation.
	Logger *log.Logger
}

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
