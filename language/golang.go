package language

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	pkgerrors "github.com/pkg/errors"
	"github.com/zimmski/osutil"

	"github.com/symflower/eval-symflower-codegen-testing/util"
)

// LanguageGolang holds a Go language to evaluate a repository.
type LanguageGolang struct{}

func init() {
	Register(&LanguageGolang{})
}

var _ Language = (*LanguageGolang)(nil)

// ID returns the unique ID of this language.
func (language *LanguageGolang) ID() (id string) {
	return "golang"
}

// Files returns a list of relative file paths of the repository that should be evaluated.
func (language *LanguageGolang) Files(repositoryPath string) (filePaths []string, err error) {
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

var languageGoNoTestsMatch = regexp.MustCompile(`(?m)^DONE (\d+) tests.*in (.+?)$`)
var languageGoCoverageMatch = regexp.MustCompile(`(?m)^coverage: (\d+\.?\d+)% of statements`)
var languageGoNoCoverageMatch = regexp.MustCompile(`(?m)^coverage: \[no statements\]$`)

// Execute invokes the language specific testing on the given repository.
func (language *LanguageGolang) Execute(repositoryPath string) (coverage float64, err error) {
	stdout, _, err := util.CommandWithResult(&util.Command{
		Command: []string{
			"gotestsum",
			"--format", "standard-verbose", // Keep formatting consistent.
			"--hide-summary", "skipped", // We are not interested in skipped tests, because they are the same as no tests at all.
			"--",       // Let the real Go "test" tool options begin.
			"-cover",   // Enable statement coverage.
			"-v",       // Output with the maximum information for easier debugging.
			"-vet=off", // Disable all linter checks, because those should be part of a different task.
			"./...",    // Always execute all tests of the repository in case multiple test files have been generated.
		},

		Directory: repositoryPath,
	})
	if err != nil {
		return 0.0, pkgerrors.WithStack(err)
	}

	ms := languageGoNoTestsMatch.FindStringSubmatch(stdout)
	if ms == nil {
		return 0.0, pkgerrors.WithStack(errors.New("could not find Go test summary"))
	}
	testCount, err := strconv.ParseUint(ms[1], 10, 64)
	if err != nil {
		return 0.0, pkgerrors.WithStack(err)
	} else if testCount == 0 {
		return 0.0, pkgerrors.WithStack(ErrNoTestFound)
	}

	if languageGoNoCoverageMatch.MatchString(stdout) {
		return 0.0, nil
	}

	mc := languageGoCoverageMatch.FindStringSubmatch(stdout)
	if mc == nil {
		fmt.Println(mc, stdout)
		return 0.0, pkgerrors.WithStack(errors.New("could not find Go coverage report"))
	}
	coverage, err = strconv.ParseFloat(mc[1], 64)
	if err != nil {
		return 0.0, pkgerrors.WithStack(err)
	}

	return coverage, nil
}
