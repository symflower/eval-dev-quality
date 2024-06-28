package task

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	pkgerrors "github.com/pkg/errors"
	"github.com/zimmski/osutil"
	"github.com/zimmski/osutil/bytesutil"

	"github.com/symflower/eval-dev-quality/log"
	"github.com/symflower/eval-dev-quality/task"
	"github.com/symflower/eval-dev-quality/util"
)

// repositoryConfiguration holds the configuration of a repository.
type repositoryConfiguration struct {
	Tasks []task.Identifier
}

// validate validates the configuration.
func (rc *repositoryConfiguration) validate() (err error) {
	if len(rc.Tasks) == 0 {
		return pkgerrors.Errorf("empty list of tasks in configuration")
	}

	for _, taskIdentifier := range rc.Tasks {
		if !LookupIdentifier[taskIdentifier] {
			return pkgerrors.Errorf("task identifier %q unknown", taskIdentifier)
		}
	}

	return nil
}

// Repository holds data about a repository.
type Repository struct {
	repositoryConfiguration

	// name holds the name of the repository.
	name string
	// dataPath holds the absolute path to the repository.
	dataPath string
}

var _ task.Repository = (*Repository)(nil)

// loadConfiguration loads the configuration from the dedicated configuration file.
func (r *Repository) loadConfiguration() (err error) {
	configurationFilePath := filepath.Join(r.dataPath, "repository.json")

	data, err := os.ReadFile(configurationFilePath)
	if errors.Is(err, os.ErrNotExist) {
		// Set default configuration.
		r.repositoryConfiguration = repositoryConfiguration{
			Tasks: AllIdentifiers,
		}

		return nil
	} else if err != nil {
		return pkgerrors.Wrap(err, configurationFilePath)
	}

	if err := json.Unmarshal(data, &r.repositoryConfiguration); err != nil {
		return pkgerrors.Wrap(err, configurationFilePath)
	}

	return r.repositoryConfiguration.validate()
}

// Name holds the name of the repository.
func (r *Repository) Name() (name string) {
	return r.name
}

// DataPath holds the absolute path to the repository.
func (r *Repository) DataPath() (dataPath string) {
	return r.dataPath
}

// SupportedTasks returns the list of task identifiers the repository supports.
func (r *Repository) SupportedTasks() (tasks []task.Identifier) {
	return r.Tasks
}

// Reset resets a repository back to its "initial" commit.
func (r *Repository) Reset(logger *log.Logger) (err error) {
	out, err := util.CommandWithResult(context.Background(), logger, &util.Command{
		Command: []string{
			"git",
			"clean",
			"-df",
		},

		Directory: r.dataPath,
		Env: map[string]string{ // Overwrite the global and system configs to point to the default one.
			"GIT_CONFIG_GLOBAL": filepath.Join(r.dataPath, ".git", "config"),
			"GIT_CONFIG_SYSTEM": filepath.Join(r.dataPath, ".git", "config"),
		},
	})
	if err != nil {
		return pkgerrors.WithStack(pkgerrors.Wrap(err, fmt.Sprintf("%s - %s", "unable to clean git repository", out)))
	}

	return nil
}

// TemporaryRepository creates a temporary repository and initializes a git repo in it.
func TemporaryRepository(logger *log.Logger, testDataPath string, repositoryPathRelative string) (repository *Repository, cleanup func(), err error) {
	repositoryPathAbsolute := filepath.Join(testDataPath, repositoryPathRelative)

	temporaryPath, err := os.MkdirTemp("", "eval-dev-quality")
	copyFileToDir(".tool-versions", temporaryPath)
	if err != nil {
		return nil, cleanup, pkgerrors.WithStack(err)
	}

	cleanup = func() {
		if e := os.RemoveAll(temporaryPath); e != nil {
			if err != nil {
				err = errors.Join(err, pkgerrors.WithStack(e))
			} else {
				err = pkgerrors.WithStack(e)
			}
		}
	}

	temporaryRepositoryPath := filepath.Join(temporaryPath, filepath.Base(repositoryPathAbsolute))
	if err := osutil.CopyTree(repositoryPathAbsolute, temporaryRepositoryPath); err != nil {
		return nil, cleanup, pkgerrors.WithStack(err)
	}
	logger.Printf("Creating temporary repository for %q within %q", repositoryPathRelative, temporaryRepositoryPath)

	// Add a default git configuration.
	if err := os.MkdirAll(filepath.Join(temporaryRepositoryPath, ".git"), 0700); err != nil {
		return nil, cleanup, pkgerrors.WithStack(err)
	}
	if err := os.WriteFile(filepath.Join(temporaryRepositoryPath, ".git", "config"), bytesutil.TrimIndentations([]byte(`
		[user]
			name = dummy-name-temporary-repository
			email = dummy.mail@temporary.repository
			username = dummy_username_temporary_repository
		[init]
			defaultBranch = main
	`)), 0600); err != nil {
		return nil, cleanup, pkgerrors.WithStack(err)
	}
	// Overwrite the global and system configs to point to the default one.
	environment := map[string]string{
		"GIT_CONFIG_GLOBAL": filepath.Join(temporaryRepositoryPath, ".git", "config"),
		"GIT_CONFIG_SYSTEM": filepath.Join(temporaryRepositoryPath, ".git", "config"),
	}

	// Add git repository and commit changes.
	out, err := util.CommandWithResult(context.Background(), logger, &util.Command{
		Command: []string{
			"git",
			"init",
		},

		Directory: temporaryRepositoryPath,
		Env:       environment,
	})
	if err != nil {
		return nil, cleanup, pkgerrors.WithStack(pkgerrors.Wrap(err, fmt.Sprintf("%s - %s", "unable to initialize git repository", out)))
	}

	out, err = util.CommandWithResult(context.Background(), logger, &util.Command{
		Command: []string{
			"git",
			"add",
			".",
		},

		Directory: temporaryRepositoryPath,
		Env:       environment,
	})
	if err != nil {
		return nil, cleanup, pkgerrors.WithStack(pkgerrors.Wrap(err, fmt.Sprintf("%s - %s", "unable to add files", out)))
	}

	out, err = util.CommandWithResult(context.Background(), logger, &util.Command{
		Command: []string{
			"git",
			"commit",
			"-m",
			"initial",
		},

		Directory: temporaryRepositoryPath,
		Env:       environment,
	})
	if err != nil {
		return nil, cleanup, pkgerrors.WithStack(pkgerrors.Wrap(err, fmt.Sprintf("%s - %s", "unable to commit", out)))
	}

	repository = &Repository{
		name:     repositoryPathRelative,
		dataPath: temporaryRepositoryPath,
	}
	if err := repository.loadConfiguration(); err != nil {
		return nil, cleanup, err
	}

	return repository, cleanup, nil
}
func copyFileToDir(srcFile, dstDir string) error {
	// Open source file for reading
	src, err := os.Open(srcFile)
	if err != nil {
		return err
	}
	defer src.Close()

	// Ensure destination directory exists
	if _, err := os.Stat(dstDir); os.IsNotExist(err) {
		os.MkdirAll(dstDir, 0755)
	}

	// Create destination file
	dstFile := filepath.Join(dstDir, filepath.Base(srcFile))
	dst, err := os.Create(dstFile)
	if err != nil {
		return err
	}
	defer dst.Close()

	// Copy the src file to dst file
	_, err = io.Copy(dst, src)
	if err != nil {
		return err
	}

	// Commit the file contents
	// Flushes memory to disk
	err = dst.Sync()
	if err != nil {
		return err
	}

	return nil
}
