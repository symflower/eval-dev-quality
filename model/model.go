package model

// Model defines a model that can be queried for generations.
type Model interface {
	// ID returns the unique ID of this model.
	ID() (id string)

	// GenerateTestsForFile generates test files for the given implementation file in a repository.
	GenerateTestsForFile(repositoryPath string, filePath string) (err error)
}
