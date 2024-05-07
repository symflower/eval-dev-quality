package providertesting

import (
	"testing"

	"github.com/symflower/eval-dev-quality/model"
)

// NewMockProviderNamedWithModels returns a new mocked provider with models.
func NewMockProviderNamedWithModels(t *testing.T, id string, models []model.Model) *MockProvider {
	m := NewMockProvider(t)
	m.On("ID").Return(id).Maybe()
	m.On("Models").Return(models, nil).Maybe()

	return m
}
