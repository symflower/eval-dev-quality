package symflower

import (
	"github.com/symflower/eval-codegen-testing/model"
	"github.com/symflower/eval-codegen-testing/model/symflower"
	"github.com/symflower/eval-codegen-testing/provider"
)

type symflowerProvider struct{}

func init() {
	provider.Register(&symflowerProvider{})
}

// ID returns the unique ID of this provider.
func (p *symflowerProvider) ID() (id string) {
	return "symflower"
}

// Model implements provider.Provider.
func (p *symflowerProvider) Models() (models []model.Model, err error) {
	return []model.Model{
		&symflower.ModelSymflower{},
	}, nil
}
