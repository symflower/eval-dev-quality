package symflower

import (
	"github.com/symflower/eval-dev-quality/model"
	"github.com/symflower/eval-dev-quality/model/symflower"
	"github.com/symflower/eval-dev-quality/provider"
)

// Provider holds a Symflower provider.
type Provider struct{}

func init() {
	provider.Register(&Provider{})
}

// NewProvider returns a Symflower provider.
func NewProvider() (provider provider.Provider) {
	return &Provider{}
}

var _ provider.Provider = (*Provider)(nil)

// ID returns the unique ID of this provider.
func (p *Provider) ID() (id string) {
	return "symflower"
}

// Models returns which models are available to be queried via this provider.
func (p *Provider) Models() (models []model.Model, err error) {
	return []model.Model{
		symflower.NewModel(),
	}, nil
}
