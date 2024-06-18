package modeltesting

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	"github.com/symflower/eval-dev-quality/model"
	task "github.com/symflower/eval-dev-quality/task"
)

// NewMockModelNamed returns a new named mocked model.
func NewMockModelNamed(t *testing.T, id string) *MockModel {
	m := NewMockModel(t)
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
func (m *MockModel) RegisterGenerateSuccess(t *testing.T, taskIdentifier task.Identifier, filePath string, fileContent string, assessment metrics.Assessments) *mock.Call {
	return m.On("RunTask", mock.Anything, taskIdentifier).Return(assessment, nil).Run(func(args mock.Arguments) {
		ctx := args.Get(0).(model.Context)
		require.NoError(t, os.WriteFile(filepath.Join(ctx.RepositoryPath, filePath), []byte(fileContent), 0600))
	})
}

// RegisterGenerateError registers a mock call that errors on generation.
func (m *MockModel) RegisterGenerateError(taskIdentifier task.Identifier, err error) *mock.Call {
	return m.On("RunTask", mock.Anything, taskIdentifier).Return(nil, err)
}
