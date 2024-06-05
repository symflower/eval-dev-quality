package language

import (
	pkgerrors "github.com/pkg/errors"

	"github.com/symflower/eval-dev-quality/log"
)

// Language defines a language to evaluate a repository.
type Language interface {
	// ID returns the unique ID of this language.
	ID() (id string)
	// Name is the prose name of this language.
	Name() (id string)

	// Files returns a list of relative file paths of the repository that should be evaluated.
	Files(logger *log.Logger, repositoryPath string) (filePaths []string, err error)
	// ImportPath returns the import path of the given source file.
	ImportPath(projectRootPath string, filePath string) (importPath string)
	// TestFilePath returns the file path of a test file given the corresponding file path of the test's source file.
	TestFilePath(projectRootPath string, filePath string) (testFilePath string)
	// TestFramework returns the human-readable name of the test framework that should be used.
	TestFramework() (testFramework string)

	// Execute invokes the language specific testing on the given repository.
	Execute(logger *log.Logger, repositoryPath string) (coverage uint64, problems []error, err error)
	// Mistakes builds a repository and returns the list of mistakes found.
	Mistakes(logger *log.Logger, repositoryPath string) (mistakes []string, err error)
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
