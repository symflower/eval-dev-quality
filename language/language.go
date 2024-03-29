package language

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
