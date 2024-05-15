package golang

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
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
func (l *Language) Execute(logger *log.Logger, repositoryPath string) (coverage float64, err error) {
	commandOutput, err := util.CommandWithResult(context.Background(), logger, &util.Command{
		Command: []string{
			tools.SymflowerPath, "test",
			"--language", "golang",
			"--workspace", repositoryPath,
		},

		Directory: repositoryPath,
	})
	if err != nil {
		return 0.0, pkgerrors.WithStack(err)
	}

	ms := languageGoNoTestsMatch.FindStringSubmatch(commandOutput)
	if ms == nil {
		return 0.0, pkgerrors.WithStack(errors.New("could not find Go test summary"))
	}
	testCount, err := strconv.ParseUint(ms[1], 10, 64)
	if err != nil {
		return 0.0, pkgerrors.WithStack(err)
	} else if testCount == 0 {
		return 0.0, pkgerrors.WithStack(language.ErrNoTestFound)
	}

	if languageGoNoCoverageMatch.MatchString(commandOutput) {
		return 0.0, nil
	}

	mc := languageGoCoverageMatch.FindStringSubmatch(commandOutput)
	if mc == nil {
		return 0.0, pkgerrors.WithStack(pkgerrors.WithMessage(errors.New("could not find coverage report"), commandOutput))
	}
	coverage, err = strconv.ParseFloat(mc[1], 64)
	if err != nil {
		return 0.0, pkgerrors.WithStack(err)
	}

	return coverage, nil
}
