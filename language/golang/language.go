package golang

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	pkgerrors "github.com/pkg/errors"
	"github.com/zimmski/osutil"

	"github.com/symflower/eval-dev-quality/language"
	"github.com/symflower/eval-dev-quality/log"
	"github.com/symflower/eval-dev-quality/tools"
	"github.com/symflower/eval-dev-quality/util"
)

// Language holds a Go language to evaluate a repository.
type Language struct{}

func init() {
	language.Register(&Language{})
}

var _ language.Language = (*Language)(nil)

// ID returns the unique ID of this language.
func (l *Language) ID() (id string) {
	return "golang"
}

// Name is the prose name of this language.
func (l *Language) Name() (id string) {
	return "Go"
}

// Files returns a list of relative file paths of the repository that should be evaluated.
func (l *Language) Files(logger *log.Logger, repositoryPath string) (filePaths []string, err error) {
	repositoryPath, err = filepath.Abs(repositoryPath)
	if err != nil {
		return nil, pkgerrors.WithStack(err)
	}

	fs, err := osutil.FilesRecursive(repositoryPath)
	if err != nil {
		return nil, pkgerrors.WithStack(err)
	}

	repositoryPath = repositoryPath + string(os.PathSeparator)
	for _, f := range fs {
		if !strings.HasSuffix(f, ".go") {
			continue
		}

		filePaths = append(filePaths, strings.TrimPrefix(f, repositoryPath))
	}

	return filePaths, nil
}

// ImportPath returns the import path of the given source file.
func (l *Language) ImportPath(projectRootPath string, filePath string) (importPath string) {
	return filepath.Join(filepath.Base(projectRootPath), filepath.Dir(filePath))
}

// TestFilePath returns the file path of a test file given the corresponding file path of the test's source file.
func (l *Language) TestFilePath(projectRootPath string, filePath string) (testFilePath string) {
	return strings.TrimSuffix(filePath, ".go") + "_test.go"
}

// TestFramework returns the human-readable name of the test framework that should be used.
func (l *Language) TestFramework() (testFramework string) {
	return ""
}

var languageGoNoTestsMatch = regexp.MustCompile(`(?m)^DONE (\d+) tests.*in (.+?)$`)
var languageGoCoverageMatch = regexp.MustCompile(`(?m)^coverage: (\d+\.?\d+)% of statements`)
var languageGoNoCoverageMatch = regexp.MustCompile(`(?m)^coverage: \[no statements\]$`)

// Execute invokes the language specific testing on the given repository.
func (l *Language) Execute(logger *log.Logger, repositoryPath string) (coverage uint64, err error) {
	coverageFilePath := filepath.Join(repositoryPath, "coverage.json")
	commandOutput, err := util.CommandWithResult(context.Background(), logger, &util.Command{
		Command: []string{
			tools.SymflowerPath, "test",
			"--language", "golang",
			"--workspace", repositoryPath,
			"--coverage-file", coverageFilePath,
		},

		Directory: repositoryPath,
	})
	if err != nil {
		return 0, pkgerrors.WithMessage(pkgerrors.WithStack(err), commandOutput)
	}

	return language.CoverageObjectCountOfFile(coverageFilePath)
}
