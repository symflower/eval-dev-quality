package modeltesting

import (
	"github.com/stretchr/testify/mock"

	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	"github.com/symflower/eval-dev-quality/language"
	"github.com/symflower/eval-dev-quality/log"
	"github.com/symflower/eval-dev-quality/model"
)

// MockModel is a mocked model.
type MockModel struct {
	mock.Mock
}

// NewMockModelNamed returns a new named mocked model.
func NewMockModelNamed(id string) *MockModel {
	m := &MockModel{}
	m.On("ID").Return(id)

	return m
}

// GenerateTestsForFile implements model.Model.
func (m *MockModel) GenerateTestsForFile(logger *log.Logger, language language.Language, repositoryPath string, filePath string) (assessments metrics.Assessments, err error) {
	args := m.Called(logger, language, repositoryPath, filePath)
	return args.Get(0).(metrics.Assessments), args.Error(1)
}

// ID implements model.Model.
func (m *MockModel) ID() (id string) {
	return m.Called().String(0)
}

var _ model.Model = &MockModel{}
