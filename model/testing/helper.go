package modeltesting

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	evaluatetask "github.com/symflower/eval-dev-quality/evaluate/task"
	task "github.com/symflower/eval-dev-quality/task"
)

// NewMockModelNamed returns a new named mocked model.
func NewMockModelNamed(t *testing.T, id string) *MockModel {
	m := NewMockModel(t)
	m.On("ID").Return(id).Maybe()

	return m
}

// RegisterGenerateSuccess registers a mock call for successful generation.
func (m *MockModel) RegisterGenerateSuccess(t *testing.T, filePath string, fileContent string, assessment metrics.Assessments) *mock.Call {
	return m.On("RunTask", mock.Anything, evaluatetask.IdentifierWriteTests).Return(assessment, nil).Run(func(args mock.Arguments) {
		ctx := args.Get(0).(task.Context)
		require.NoError(t, os.WriteFile(filepath.Join(ctx.RepositoryPath, filePath), []byte(fileContent), 0600))
	})
}

// RegisterGenerateError registers a mock call that errors on generation.
func (m *MockModel) RegisterGenerateError(err error) *mock.Call {
	return m.On("RunTask", mock.Anything, evaluatetask.IdentifierWriteTests).Return(nil, err)
}
