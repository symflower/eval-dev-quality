package rust

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	pkgerrors "github.com/pkg/errors"

	"github.com/symflower/eval-dev-quality/language"
	"github.com/symflower/eval-dev-quality/log"
	"github.com/symflower/eval-dev-quality/util"
)

// Language holds a Rust language to evaluate a repository.
type Language struct{}

func init() {
	language.Register(&Language{})
}

var _ language.Language = (*Language)(nil)

// ID returns the unique ID of this language.
func (l *Language) ID() (id string) {
	return "rust"
}

// Name is the prose name of this language.
func (l *Language) Name() (id string) {
	return "Rust"
}

// Files returns a list of relative file paths of the repository that should be evaluated.
func (l *Language) Files(logger *log.Logger, repositoryPath string) (filePaths []string, err error) {
	return language.Files(logger, l, repositoryPath)
}

// ImportPath returns the import path of the given source file.
func (l *Language) ImportPath(projectRootPath string, filePath string) (importPath string) {
	importPath = strings.ReplaceAll(filepath.Dir(filePath), string(os.PathSeparator), "::")

	return strings.TrimPrefix(strings.TrimPrefix(importPath, "src"), "::")
}

// TestFilePath returns the file path of a test file given the corresponding file path of the test's source file.
func (l *Language) TestFilePath(projectRootPath string, filePath string) (testFilePath string) {
	return filePath
}

// TestFramework returns the human-readable name of the test framework that should be used.
func (l *Language) TestFramework() (testFramework string) {
	return ""
}

// HasTestsInSource returns if the tests for this language are commonly located within the corresponding implementation file.
func (l *Language) HasTestsInSource() bool {
	return true
}

// DefaultFileExtension returns the default file extension.
func (l *Language) DefaultFileExtension() string {
	return ".rs"
}

// DefaultTestFileSuffix returns the default test file suffix.
func (l *Language) DefaultTestFileSuffix() string {
	return ".rs"
}

// ExecuteTests invokes the language specific testing on the given repository.
func (l *Language) ExecuteTests(logger *log.Logger, repositoryPath string) (testResult *language.TestResult, problems []error, err error) {
	commandOutput, err := util.CommandWithResult(context.Background(), logger, &util.Command{
		Command: []string{ // TODO Move this to `symflower test` to get coverage information.
			"cargo",
			"llvm-cov",
		},

		Directory: repositoryPath,
	})
	if err != nil {
		return nil, nil, pkgerrors.WithMessage(pkgerrors.WithStack(err), commandOutput)
	}

	testsTotal, testsPass, e := parseSymflowerTestOutput(commandOutput)
	if e != nil {
		problems = append(problems, pkgerrors.WithMessage(pkgerrors.WithStack(e), commandOutput))
	}
	// If there are test failures, then this is just a soft error since we still are able to receive coverage data.
	if err != nil {
		if testsTotal-testsPass <= 0 {
			return nil, nil, pkgerrors.WithMessage(pkgerrors.WithStack(err), commandOutput)
		}

		problems = append(problems, pkgerrors.WithMessage(pkgerrors.WithStack(err), commandOutput))
	}

	testResult = &language.TestResult{
		TestsTotal: uint(testsTotal),
		TestsPass:  uint(testsPass),

		StdOut: commandOutput,
	}

	// coverageFilePath := "" // TODO Get coverage information.
	// testResult.Coverage, err = language.CoverageObjectCountOfFile(logger, coverageFilePath)
	// if err != nil {
	// 	return testResult, problems, pkgerrors.WithMessage(pkgerrors.WithStack(err), commandOutput)
	// }

	return testResult, problems, nil
}

var languageRustTestSummaryRE = regexp.MustCompile(`test result: (?:ok|FAILED). (\d+) passed; (\d+) failed;`)

func parseSymflowerTestOutput(data string) (testsTotal int, testsPass int, err error) {
	testSummary := languageRustTestSummaryRE.FindStringSubmatch(data)
	if len(testSummary) == 0 {
		return 0, 0, pkgerrors.WithMessage(pkgerrors.WithStack(language.ErrCannotParseTestSummary), data)
	}

	testsTotal, err = strconv.Atoi(testSummary[1])
	if err != nil {
		return 0, 0, pkgerrors.WithStack(err)
	}

	testsPass, err = strconv.Atoi(testSummary[2])
	if err != nil {
		return 0, 0, pkgerrors.WithStack(err)
	}

	return testsTotal, testsPass, nil
}

// Mistakes builds a Rust repository and returns the list of mistakes found.
func (l *Language) Mistakes(logger *log.Logger, repositoryPath string) (mistakes []string, err error) {
	// TODO
	return []string{}, nil
}

// SupportsFix reports if the language is supported by "symflower fix".
func (l *Language) SupportsFix() bool {
	return false
}

// SupportsTemplate reports if the language is supported by "symflower unit-test-skeleton".
func (l *Language) SupportsTemplate() bool {
	return false
}
