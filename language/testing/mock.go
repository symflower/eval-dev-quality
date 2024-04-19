package languagetesting

import (
	"log"

	"github.com/stretchr/testify/mock"

	"github.com/symflower/eval-dev-quality/language"
)

// MockLanguage is a mocked language.
type MockLanguage struct {
	mock.Mock
}

// NewMockLanguageNamed returns a new named mocked language.
func NewMockLanguageNamed(id string) *MockLanguage {
	m := &MockLanguage{}
	m.On("ID").Return(id)

	return m
}

// Execute implements language.Language.
func (m *MockLanguage) Execute(log *log.Logger, repositoryPath string) (coverage float64, err error) {
	args := m.Called(log, repositoryPath)
	return args.Get(0).(float64), args.Error(1)
}

// Files implements language.Language.
func (m *MockLanguage) Files(log *log.Logger, repositoryPath string) (filePaths []string, err error) {
	args := m.Called(log, repositoryPath)
	return args.Get(0).([]string), args.Error(1)
}

// ID implements language.Language.
func (m *MockLanguage) ID() (id string) {
	return m.Called().String(0)
}

// Name implements language.Language.
func (m *MockLanguage) Name() (id string) {
	return m.Called().String(0)
}

var _ language.Language = &MockLanguage{}
