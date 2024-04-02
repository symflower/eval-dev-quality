package provider

import (
	pkgerrors "github.com/pkg/errors"
	"github.com/symflower/eval-symflower-codegen-testing/model"
)

// ProviderModelSeparator is the the separator between a provider and a model.
const ProviderModelSeparator = "/"

// Provider defines a provider to query models such as LLMs.
type Provider interface {
	// ID returns the unique ID of this provider.
	ID() (id string)
	// Models returns which models are available to be queried via this provider.
	Models() (models []model.Model)
}

// Providers holds a register of all providers.
var Providers = map[string]Provider{}

// Register adds a provider to the common provider list.
func Register(provider Provider) {
	id := provider.ID()
	if _, ok := Providers[id]; ok {
		panic(pkgerrors.WithMessage(pkgerrors.New("provider was already registered"), id))
	}

	Providers[id] = provider
}
