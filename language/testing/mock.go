package languagetesting

import (
	"github.com/stretchr/testify/mock"

	"github.com/symflower/eval-dev-quality/language"
	"github.com/symflower/eval-dev-quality/log"
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

var _ language.Language = &MockLanguage{}

// ID implements language.Language.
func (m *MockLanguage) ID() (id string) {
	return m.Called().String(0)
}

// Name implements language.Language.
func (m *MockLanguage) Name() (id string) {
	return m.Called().String(0)
}

// Files implements language.Language.
func (m *MockLanguage) Files(logger *log.Logger, repositoryPath string) (filePaths []string, err error) {
	args := m.Called(logger, repositoryPath)
	return args.Get(0).([]string), args.Error(1)
}

// ImportPath returns the import path of the given source file.
func (m *MockLanguage) ImportPath(projectRootPath string, filePath string) (importPath string) {
	return m.Called(projectRootPath, filePath).String(0)
}

// TestFilePath implements language.Language.
func (m *MockLanguage) TestFilePath(projectRootPath string, filePath string) (testFilePath string) {
	return m.Called(projectRootPath, filePath).String(0)
}

// TestFramework returns the human-readable name of the test framework that should be used.
func (m *MockLanguage) TestFramework() (testFramework string) {
	return m.Called().String(0)
}

// Execute implements language.Language.
func (m *MockLanguage) Execute(logger *log.Logger, repositoryPath string) (coverage float64, err error) {
	args := m.Called(logger, repositoryPath)
	return args.Get(0).(float64), args.Error(1)
}
