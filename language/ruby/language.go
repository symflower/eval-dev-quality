package ruby

import (
	"path/filepath"
	"strings"

	"github.com/symflower/eval-dev-quality/language"
	"github.com/symflower/eval-dev-quality/log"
)

// Language holds a Ruby language to evaluate a repository.
type Language struct{}

var _ language.Language = (*Language)(nil)

// ID returns the unique ID of this language.
func (l *Language) ID() (id string) {
	return "ruby"
}

// Name is the prose name of this language.
func (l *Language) Name() (id string) {
	return "Ruby"
}

// Files returns a list of relative file paths of the repository that should be evaluated.
func (l *Language) Files(logger *log.Logger, repositoryPath string) (filePaths []string, err error) {
	return language.Files(logger, l, repositoryPath)
}

// ImportPath returns the import path of the given source file.
func (l *Language) ImportPath(projectRootPath string, filePath string) (importPath string) {
	return "../lib/" + strings.TrimSuffix(filepath.Base(filePath), l.DefaultFileExtension())
}

// TestFilePath returns the file path of a test file given the corresponding file path of the test's source file.
func (l *Language) TestFilePath(projectRootPath string, filePath string) (testFilePath string) {
	filePath = strings.ReplaceAll(filePath, "lib", "test")

	return strings.TrimSuffix(filePath, l.DefaultFileExtension()) + l.DefaultTestFileSuffix()
}

// TestFramework returns the human-readable name of the test framework that should be used.
func (l *Language) TestFramework() (testFramework string) {
	return "Minitest"
}

// DefaultFileExtension returns the default file extension.
func (l *Language) DefaultFileExtension() string {
	return ".rb"
}

// DefaultTestFileSuffix returns the default test file suffix.
func (l *Language) DefaultTestFileSuffix() string {
	return "_test.rb"
}

// ExecuteTests invokes the language specific testing on the given repository.
func (l *Language) ExecuteTests(logger *log.Logger, repositoryPath string) (testResult *language.TestResult, problems []error, err error) {
	logger.Panic("not implemented")

	return testResult, problems, nil
}

// Mistakes builds a Ruby repository and returns the list of mistakes found.
func (l *Language) Mistakes(logger *log.Logger, repositoryPath string) (mistakes []string, err error) {
	logger.Panic("not implemented")

	return nil, nil
}
