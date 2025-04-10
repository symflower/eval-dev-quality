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

// NewMockModelNamed returns a new named mocked model.
func NewMockModelNamed(t *testing.T, id string) *MockModel {
	m := NewMockModel(t)
	m.On("ID").Return(id).Maybe()

	return m
}

// RegisterGenerateSuccess registers a mock call for successful generation.
func (m *MockCapabilityWriteTests) RegisterGenerateSuccess(t *testing.T, filePath string, fileContent string, assessment metrics.Assessments) *mock.Call {
	return m.RegisterGenerateSuccessValidateContext(t, nil, filePath, fileContent, assessment)
}

// RegisterGenerateSuccessValidateContext registers a mock call for successful generation.
func (m *MockCapabilityWriteTests) RegisterGenerateSuccessValidateContext(t *testing.T, validateContext func(t *testing.T, c model.Context), filePath string, fileContent string, assessment metrics.Assessments) *mock.Call {
	return m.On("WriteTests", mock.Anything).Return(assessment, nil).Run(func(args mock.Arguments) {
		ctx, _ := args.Get(0).(model.Context)
		if validateContext != nil {
			validateContext(t, ctx)
		}
		testFilePath := filepath.Join(ctx.RepositoryPath, filePath)
		require.NoError(t, os.MkdirAll(filepath.Dir(testFilePath), 0700))
		require.NoError(t, os.WriteFile(testFilePath, []byte(fileContent), 0600))
	})
}

// RegisterGenerateError registers a mock call that errors on generation.
func (m *MockCapabilityWriteTests) RegisterGenerateError(err error) *mock.Call {
	return m.On("WriteTests", mock.Anything).Return(nil, err)
}

// RegisterGenerateSuccess registers a mock call for successful generation.
func (m *MockCapabilityRepairCode) RegisterGenerateSuccess(t *testing.T, filePath string, fileContent string, assessment metrics.Assessments) *mock.Call {
	return m.On("RepairCode", mock.Anything).Return(assessment, nil).Run(func(args mock.Arguments) {
		ctx, _ := args.Get(0).(model.Context)
		require.NoError(t, os.WriteFile(filepath.Join(ctx.RepositoryPath, filePath), []byte(fileContent), 0600))
	})
}

// RegisterGenerateSuccess registers a mock call for successful generation.
func (m *MockCapabilityMigrate) RegisterGenerateSuccess(t *testing.T, filePath string, fileContent string, assessment metrics.Assessments) *mock.Call {
	return m.On("Migrate", mock.Anything).Return(assessment, nil).Run(func(args mock.Arguments) {
		ctx, _ := args.Get(0).(model.Context)
		require.NoError(t, os.WriteFile(filepath.Join(ctx.RepositoryPath, filePath), []byte(fileContent), 0600))
	})
}

// RegisterGenerateError registers a mock call that errors on generation.
func (m *MockCapabilityRepairCode) RegisterGenerateError(err error) *mock.Call {
	return m.On("RepairCode", mock.Anything).Return(nil, err)
}

// RegisterGenerateSuccess registers a mock call for successful generation.
func (m *MockCapabilityTranspile) RegisterGenerateSuccess(t *testing.T, validateContext func(t *testing.T, c model.Context), filePath string, fileContent string, assessment metrics.Assessments) *mock.Call {
	return m.On("Transpile", mock.Anything).Return(assessment, nil).Run(func(args mock.Arguments) {
		ctx, _ := args.Get(0).(model.Context)
		if validateContext != nil {
			validateContext(t, ctx)
		}
		require.NoError(t, os.WriteFile(filepath.Join(ctx.RepositoryPath, filePath), []byte(fileContent), 0600))
	})
}

// MockModelCapabilityWriteTests holds a mock implementing the "Model" and the "CapabilityWriteTests" interface.
type MockModelCapabilityWriteTests struct {
	*MockModel
	*MockCapabilityWriteTests
}

// NewMockCapabilityWriteTestsNamed returns a new named mocked model.
func NewMockCapabilityWriteTestsNamed(t *testing.T, id string) *MockModelCapabilityWriteTests {
	return &MockModelCapabilityWriteTests{
		MockModel:                NewMockModelNamed(t, id),
		MockCapabilityWriteTests: NewMockCapabilityWriteTests(t),
	}
}

// MockModelCapabilityRepairCode holds a mock implementing the "Model" and the "CapabilityRepairCode" interface.
type MockModelCapabilityRepairCode struct {
	*MockModel
	*MockCapabilityRepairCode
}

// NewMockCapabilityRepairCodeNamed returns a new named mocked model.
func NewMockCapabilityRepairCodeNamed(t *testing.T, id string) *MockModelCapabilityRepairCode {
	return &MockModelCapabilityRepairCode{
		MockModel:                NewMockModelNamed(t, id),
		MockCapabilityRepairCode: NewMockCapabilityRepairCode(t),
	}
}

// MockModelCapabilityMigrate holds a mock implementing the "Model" and the "CapabilityMigrate" interface.
type MockModelCapabilityMigrate struct {
	*MockModel
	*MockCapabilityMigrate
}

// NewMockCapabilityMigrateNamed returns a new named mocked model.
func NewMockCapabilityMigrateNamed(t *testing.T, id string) *MockModelCapabilityMigrate {
	return &MockModelCapabilityMigrate{
		MockModel:             NewMockModelNamed(t, id),
		MockCapabilityMigrate: NewMockCapabilityMigrate(t),
	}
}

// MockModelCapabilityTranspile holds a mock implementing the "Model" and the "CapabilityTranspile" interface.
type MockModelCapabilityTranspile struct {
	*MockModel
	*MockCapabilityTranspile
}

// NewMockCapabilityTranspileNamed returns a new named mocked model.
func NewMockCapabilityTranspileNamed(t *testing.T, id string) *MockModelCapabilityTranspile {
	return &MockModelCapabilityTranspile{
		MockModel:               NewMockModelNamed(t, id),
		MockCapabilityTranspile: NewMockCapabilityTranspile(t),
	}
}
