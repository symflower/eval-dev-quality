package task

import (
	"fmt"

	pkgerrors "github.com/pkg/errors"
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
