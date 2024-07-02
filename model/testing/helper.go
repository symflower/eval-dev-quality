package modeltesting

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	"github.com/symflower/eval-dev-quality/model"
)

// NewMockCapabilityWriteTestsNamed returns a new named mocked model.
func NewMockCapabilityWriteTestsNamed(t *testing.T, id string) *MockCapabilityWriteTests {
	m := NewMockCapabilityWriteTests(t)
	m.On("ID").Return(id).Maybe()

	return m
}

// NewMockCapabilityRepairCodeNamed returns a new named mocked model.
func NewMockCapabilityRepairCodeNamed(t *testing.T, id string) *MockCapabilityRepairCode {
	m := NewMockCapabilityRepairCode(t)
	m.On("ID").Return(id).Maybe()

	return m
}

// NewMockModelNamed returns a new named mocked model with cost information.
func NewMockModelNamedWithCosts(t *testing.T, id string, name string, cost float64) *MockModel {
	m := NewMockModel(t)
	m.On("ID").Return(id).Maybe()
	m.On("Name").Return(name).Maybe()
	m.On("Cost").Return(cost).Maybe()

	return m
}

// RegisterGenerateSuccess registers a mock call for successful generation.
func (m *MockCapabilityWriteTests) RegisterGenerateSuccess(t *testing.T, filePath string, fileContent string, assessment metrics.Assessments) *mock.Call {
	return m.On("WriteTests", mock.Anything).Return(assessment, nil).Run(func(args mock.Arguments) {
		ctx := args.Get(0).(model.Context)
		require.NoError(t, os.WriteFile(filepath.Join(ctx.RepositoryPath, filePath), []byte(fileContent), 0600))
	})
}

// RegisterGenerateError registers a mock call that errors on generation.
func (m *MockCapabilityWriteTests) RegisterGenerateError(err error) *mock.Call {
	return m.On("WriteTests", mock.Anything).Return(nil, err)
}

// RegisterGenerateSuccess registers a mock call for successful generation.
func (m *MockCapabilityRepairCode) RegisterGenerateSuccess(t *testing.T, filePath string, fileContent string, assessment metrics.Assessments) *mock.Call {
	return m.On("RepairCode", mock.Anything).Return(assessment, nil).Run(func(args mock.Arguments) {
		ctx := args.Get(0).(model.Context)
		require.NoError(t, os.WriteFile(filepath.Join(ctx.RepositoryPath, filePath), []byte(fileContent), 0600))
	})
}

// RegisterGenerateError registers a mock call that errors on generation.
func (m *MockCapabilityRepairCode) RegisterGenerateError(err error) *mock.Call {
	return m.On("RepairCode", mock.Anything).Return(nil, err)
}
