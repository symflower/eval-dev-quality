package task

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	pkgerrors "github.com/pkg/errors"
	"github.com/zimmski/osutil"
	"github.com/zimmski/osutil/bytesutil"

	"github.com/symflower/eval-dev-quality/language"
	"github.com/symflower/eval-dev-quality/log"
	"github.com/symflower/eval-dev-quality/task"
	"github.com/symflower/eval-dev-quality/util"
)

// Repository holds data about a repository.
type Repository struct {
	task.RepositoryConfiguration

	// name holds the name of the repository.
	name string
	// dataPath holds the absolute path to the repository.
	dataPath string
}

var _ task.Repository = (*Repository)(nil)

// loadConfiguration loads the configuration from the dedicated configuration file.
func (r *Repository) loadConfiguration() (err error) {
	configurationFilePath := filepath.Join(r.dataPath, task.RepositoryConfigurationFileName)

	configuration, err := task.LoadRepositoryConfiguration(configurationFilePath, AllIdentifiers)
	if err != nil {
		return err
	}

	r.RepositoryConfiguration = *configuration

	return nil
}

// Name holds the name of the repository.
func (r *Repository) Name() (name string) {
	return r.name
}

// DataPath holds the absolute path to the repository.
func (r *Repository) DataPath() (dataPath string) {
	return r.dataPath
}

// Validate checks it the repository is well-formed.
func (r *Repository) Validate(logger *log.Logger, language language.Language) (err error) {
	for _, taskIdentifier := range r.RepositoryConfiguration.Tasks {
		switch taskIdentifier {
		case IdentifierCodeRepair:
			return validateCodeRepairRepository(logger, r.DataPath(), language)
		case IdentifierMigrate:
			return validateMigrateRepository(logger, r.DataPath(), language)
		case IdentifierTranspile:
			return validateTranspileRepository(logger, r.DataPath(), language)
		case IdentifierWriteTests:
			return validateWriteTestsRepository(logger, r.DataPath(), language)
		}
	}

	return nil
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

	out, err = util.CommandWithResult(context.Background(), logger, &util.Command{
		Command: []string{
			"git",
			"restore",
			".",
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

// Configuration returns the configuration of a repository.
func (r *Repository) Configuration() *task.RepositoryConfiguration {
	return &r.RepositoryConfiguration
}

// TemporaryRepository creates a temporary repository and initializes a git repo in it.
func TemporaryRepository(logger *log.Logger, testDataPath string, repositoryPathRelative string) (repository *Repository, cleanup func(), err error) {
	repositoryPathAbsolute := filepath.Join(testDataPath, repositoryPathRelative)

	temporaryPath, err := os.MkdirTemp("", "eval-dev-quality")
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
	logger.Info("creating temporary repository", "repository", repositoryPathRelative, "path", temporaryRepositoryPath)

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
