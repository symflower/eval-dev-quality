package language

import (
	"os"
	"path/filepath"
	"strings"

	pkgerrors "github.com/pkg/errors"
	"github.com/zimmski/osutil"

	"github.com/symflower/eval-symflower-codegen-testing/util"
)

// LanguageGolang holds a Go language to evaluate a repository.
type LanguageGolang struct{}

func init() {
	l := &LanguageGolang{}
	Languages[l.ID()] = l
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

// Execute invokes the language specific testing on the given repository.
func (language *LanguageGolang) Execute(repositoryPath string) (err error) {
	_, _, err = util.CommandWithResult(&util.Command{
		Command: []string{
			"go", "test",
			"-v",    // Output with the maximum information for easier debugging.
			"./...", // Always execute all tests of the repository in case multiple test files have been generated.
		},

		Directory: repositoryPath,
	})
	if err != nil {
		return pkgerrors.WithStack(err)
	}

	return nil
}
