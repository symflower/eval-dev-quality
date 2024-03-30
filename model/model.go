package model

import (
	pkgerrors "github.com/pkg/errors"
)

// Model defines a model that can be queried for generations.
type Model interface {
	// ID returns the unique ID of this model.
	ID() (id string)

	// GenerateTestsForFile generates test files for the given implementation file in a repository.
	GenerateTestsForFile(repositoryPath string, filePath string) (err error)
}

// Models holds a register of all models.
var Models = map[string]Model{}

// Register adds a model to the common model list.
func Register(model Model) {
	id := model.ID()
	if _, ok := Models[id]; ok {
		panic(pkgerrors.WithMessage(pkgerrors.New("model was already registered"), id))
	}

	Models[id] = model
}
