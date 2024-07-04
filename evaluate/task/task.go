package task

import (
	"fmt"
	"path/filepath"

	pkgerrors "github.com/pkg/errors"
	"github.com/symflower/eval-dev-quality/log"
	"github.com/symflower/eval-dev-quality/model"
	evaltask "github.com/symflower/eval-dev-quality/task"
)

var (
	// AllIdentifiers holds all available task identifiers.
	AllIdentifiers []evaltask.Identifier
	// LookupIdentifier holds a map of all available task identifiers.
	LookupIdentifier = map[evaltask.Identifier]bool{}
)

// registerIdentifier registers the given identifier and makes it available.
func registerIdentifier(name string) (identifier evaltask.Identifier) {
	identifier = evaltask.Identifier(name)
	AllIdentifiers = append(AllIdentifiers, identifier)

	if _, ok := LookupIdentifier[identifier]; ok {
		panic(fmt.Sprintf("task identifier already registered: %s", identifier))
	}
	LookupIdentifier[identifier] = true

	return identifier
}

var (
	// IdentifierWriteTests holds the identifier for the "write test" task.
	IdentifierWriteTests = registerIdentifier("write-tests")
	// IdentifierWriteTestsSymflowerFix holds the identifier for the "write test" task with the "symflower fix" applied.
	IdentifierWriteTestsSymflowerFix = registerIdentifier("write-tests-symflower-fix")
	// IdentifierCodeRepair holds the identifier for the "code repair" task.
	IdentifierCodeRepair = registerIdentifier("code-repair")
)

// TaskForIdentifier returns a task based on the task identifier.
func TaskForIdentifier(taskIdentifier evaltask.Identifier) (task evaltask.Task, err error) {
	switch taskIdentifier {
	case IdentifierWriteTests:
		return &TaskWriteTests{}, nil
	case IdentifierCodeRepair:
		return &TaskCodeRepair{}, nil
	default:
		return nil, pkgerrors.Wrap(evaltask.ErrTaskUnknown, string(taskIdentifier))
	}
}

// taskLogger holds common logging functionality.
type taskLogger struct {
	*log.Logger

	logClose func()
	ctx      evaltask.Context
	task     evaltask.Task
}

// newTaskLogger initializes the logging.
func newTaskLogger(ctx evaltask.Context, task evaltask.Task) (logging *taskLogger, err error) {
	logging = &taskLogger{
		ctx:  ctx,
		task: task,
	}

	logging.Logger, logging.logClose, err = log.WithFile(ctx.Logger, filepath.Join(ctx.ResultPath, string(task.Identifier()), model.CleanModelNameForFileSystem(ctx.Model.ID()), ctx.Language.ID(), ctx.Repository.Name()+".log"))
	if err != nil {
		return nil, err
	}

	logging.Logger.Printf("Evaluating model %q on task %q using language %q and repository %q", ctx.Model.ID(), task.Identifier(), ctx.Language.ID(), ctx.Repository.Name())

	return logging, nil
}

// finalizeLogging finalizes the logging.
func (t *taskLogger) finalize(problems []error) {
	t.Logger.Printf("Evaluated model %q on task %q using language %q and repository %q: encountered %d problems: %+v", t.ctx.Model.ID(), t.task.Identifier(), t.ctx.Language.ID(), t.ctx.Repository.Name(), len(problems), problems)

	t.logClose()
}
