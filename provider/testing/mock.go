package testing

import (
	"context"

	"github.com/stretchr/testify/mock"

	provider "github.com/symflower/eval-symflower-codegen-testing/provider"
)

// MockQueryProvider is a mocked QueryProvider.
type MockQueryProvider struct {
	mock.Mock
}

var _ provider.QueryProvider = &MockQueryProvider{}

// Query queries the LLM with the given model name.
func (m *MockQueryProvider) Query(ctx context.Context, modelIdentifier string, promptText string) (response string, err error) {
	args := m.Called(ctx, modelIdentifier, promptText)
	return args.String(0), args.Error(1)
}
