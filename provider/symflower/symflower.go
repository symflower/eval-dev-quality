package symflower

import (
	"github.com/symflower/eval-dev-quality/model"
	"github.com/symflower/eval-dev-quality/model/symflower"
	"github.com/symflower/eval-dev-quality/provider"
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
