package task

import (
	"fmt"

	"github.com/symflower/eval-dev-quality/task"
)

var (
	// AllIdentifiers holds all available task identifiers.
	AllIdentifiers []task.Identifier
	// LookupIdentifier holds a map of all available task identifiers.
	LookupIdentifier = map[task.Identifier]bool{}
)

// registerIdentifier registers the given identifier and makes it available.
func registerIdentifier(name string) (identifier task.Identifier) {
	identifier = task.Identifier(name)
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
	// IdentifierCodeRepair holds the identifier for the "code repair" task.
	IdentifierCodeRepair = registerIdentifier("code-repair")
)
