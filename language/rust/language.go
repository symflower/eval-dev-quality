package rust

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	pkgerrors "github.com/pkg/errors"
	"github.com/zimmski/osutil/bytesutil"

	"github.com/symflower/eval-dev-quality/language"
	"github.com/symflower/eval-dev-quality/log"
	"github.com/symflower/eval-dev-quality/tools"
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
	if strings.HasPrefix(filePath, "tests") {
		return filePath
	}

	fileNameWithoutExtension := strings.TrimSuffix(filepath.Base(filePath), l.DefaultFileExtension())
	directoryPath := "tests" + strings.TrimPrefix(filepath.Dir(filePath), "src")

	return filepath.Join(directoryPath, fileNameWithoutExtension+l.DefaultTestFileSuffix())
}

// TestFramework returns the human-readable name of the test framework that should be used.
func (l *Language) TestFramework() (testFramework string) {
	return ""
}

// DefaultFileExtension returns the default file extension.
func (l *Language) DefaultFileExtension() string {
	return ".rs"
}

// DefaultTestFileSuffix returns the default test file suffix.
func (l *Language) DefaultTestFileSuffix() string {
	return "_test.rs"
}

// testDirectiveLinePerFile computes the line of the "#[cfg(test)]" compiler directive within each file.
// Lines are counted index-1 based.
func (l *Language) testDirectiveLinePerFile(logger *log.Logger, repositoryPath string) (linePerFile map[string]int, err error) {
	files, err := l.Files(logger, repositoryPath)
	if err != nil {
		return nil, err
	}

	linePerFile = map[string]int{}
	for _, filePath := range files {
		file, err := os.Open(filepath.Join(repositoryPath, filePath))
		if err != nil {
			return nil, pkgerrors.WithStack(err)
		}
		defer func() {
			err = errors.Join(file.Close(), err)
		}()

		scanner := bufio.NewScanner(file)
		for line := 1; scanner.Scan(); line++ {
			if err := scanner.Err(); err != nil {
				return nil, pkgerrors.WithStack(err)
			}

			if strings.Contains(scanner.Text(), "#[cfg(test)]") {
				linePerFile[filePath] = line

				break
			}
		}
	}

	return linePerFile, nil
}

// ExecuteTests invokes the language specific testing on the given repository.
func (l *Language) ExecuteTests(logger *log.Logger, repositoryPath string) (testResult *language.TestResult, problems []error, err error) {
	// HACK Tests in Rust are within the implementation file, but excluding coverage more granular than file-level is currently unstable (https://github.com/rust-lang/rust/issues/84605). Therefore, we assume that test are always at the end of the file and ignore reported coverage after the "#[cfg(test)]" compiler directive.
	testStartLinePerFile, err := l.testDirectiveLinePerFile(logger, repositoryPath)
	if err != nil {
		return nil, nil, err
	}
	logger.Info("rust test locations", "locations", testStartLinePerFile)

	ctx, cancel := context.WithTimeout(context.Background(), language.DefaultExecutionTimeout)
	defer cancel()
	coverageFilePath := filepath.Join(repositoryPath, "coverage.json")
	commandOutput, err := util.CommandWithResult(ctx, logger, &util.Command{
		Command: []string{
			tools.SymflowerPath, "test",
			"--language", "rust",
			"--workspace", repositoryPath,
			"--coverage-file", coverageFilePath,
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

	coverageData, err := language.ParseCoverage(logger, coverageFilePath)
	if err != nil {
		return testResult, problems, pkgerrors.WithMessage(pkgerrors.WithStack(err), commandOutput)
	}
	for _, block := range coverageData {
		if lineStart, ok := testStartLinePerFile[block.FilePath]; ok && block.LineEnd < lineStart && block.Count > 0 {
			testResult.Coverage++
		}
	}

	return testResult, problems, nil
}

var languageRustTestSummaryRE = regexp.MustCompile(`test result: (?:ok|FAILED). (\d+) passed; (\d+) failed;`)

func parseSymflowerTestOutput(data string) (testsTotal int, testsPassed int, err error) {
	testSummaries := languageRustTestSummaryRE.FindAllStringSubmatch(data, -1)
	if len(testSummaries) == 0 {
		return 0, 0, pkgerrors.WithMessage(pkgerrors.WithStack(language.ErrCannotParseTestSummary), data)
	}

	testsFailed := 0
	for _, testSummary := range testSummaries {
		p, _ := strconv.Atoi(testSummary[1]) // The regular expression guarantees a valid number.
		testsPassed += p
		f, _ := strconv.Atoi(testSummary[2]) // The regular expression guarantees a valid number.
		testsFailed += f
	}

	return testsPassed + testsFailed, testsPassed, nil
}

type diagnosticEntry struct {
	Reason       string            `json:"reason"`
	ManifestPath string            `json:"manifest_path"`
	File         diagnosticFile    `json:"target"`
	Message      diagnosticMessage `json:"message"`
}

type diagnosticMessage struct {
	Message   string               `json:"message"`
	Locations []diagnosticLocation `json:"spans"`
}

type diagnosticLocation struct {
	LineStart int `json:"line_start"`
	LineEnd   int `json:"line_end"`
}

type diagnosticFile struct {
	Path string `json:"src_path"`
}

// Mistakes builds a Rust repository and returns the list of mistakes found.
func (l *Language) Mistakes(logger *log.Logger, repositoryPath string) (mistakes []string, err error) {
	commandOutput, err := util.CommandWithResult(context.Background(), logger, &util.Command{
		Command: []string{
			"cargo", "check",
			"--message-format", "json",
		},

		Directory: repositoryPath,
	})
	if err != nil && !strings.Contains(err.Error(), "could not compile") {
		return nil, pkgerrors.WithMessage(pkgerrors.WithStack(err), commandOutput)
	}

	lines := strings.Split(commandOutput, "\n")

	var diagnostics []diagnosticEntry
	for _, line := range lines {
		var entry diagnosticEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			logger.Warn(fmt.Sprintf("could not parse rust compile output %q", line))
		}
		diagnostics = append(diagnostics, entry)
	}

	for _, diagnostic := range diagnostics {
		if diagnostic.Reason != "compiler-message" || len(diagnostic.Message.Locations) == 0 {
			continue
		}
		diagnostic.File.Path, err = filepath.Rel(filepath.Dir(diagnostic.ManifestPath), diagnostic.File.Path)
		if err != nil {
			return nil, pkgerrors.WithStack(pkgerrors.WithMessage(err, bytesutil.FormatToGoObject(diagnostic)))
		}

		mistakes = append(mistakes, fmt.Sprintf("%s:%d: %s", diagnostic.File.Path, diagnostic.Message.Locations[0].LineStart, diagnostic.Message.Message))
	}

	return mistakes, nil
}

// SupportsFix reports if the language is supported by "symflower fix".
func (l *Language) SupportsFix() bool {
	return false
}

// SupportsTemplate reports if the language is supported by "symflower unit-test-skeleton".
func (l *Language) SupportsTemplate() bool {
	return false
}
