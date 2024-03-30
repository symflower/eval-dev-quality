package language

import (
	pkgerrors "github.com/pkg/errors"
)

// Language defines a language to evaluate a repository.
type Language interface {
	// ID returns the unique ID of this language.
	ID() (id string)

	// Files returns a list of relative file paths of the repository that should be evaluated.
	Files(repositoryPath string) (filePaths []string, err error)

	// Execute invokes the language specific testing on the given repository.
	Execute(repositoryPath string) (err error)
}

// Languages holds a register of all languages.
var Languages = map[string]Language{}

// Register adds a language to the common language list.
func Register(language Language) {
	id := language.ID()
	if _, ok := Languages[id]; ok {
		panic(pkgerrors.WithMessage(pkgerrors.New("language was already registered"), id))
	}

	Languages[id] = language
}
