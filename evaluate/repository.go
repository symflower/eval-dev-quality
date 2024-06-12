package evaluate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	pkgerrors "github.com/pkg/errors"
	"github.com/zimmski/osutil"
	"github.com/zimmski/osutil/bytesutil"

	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	"github.com/symflower/eval-dev-quality/language"
	"github.com/symflower/eval-dev-quality/log"
	evalmodel "github.com/symflower/eval-dev-quality/model"
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
		if !task.LookupIdentifier[taskIdentifier] {
			return pkgerrors.Errorf("task identifier %q unknown", taskIdentifier)
		}
	}

	return nil
}

// defaultConfiguration holds the default configuration object if there exists no configuration file.
var defaultConfiguration = repositoryConfiguration{
	Tasks: task.AllIdentifiers,
}

// Repository holds data about a repository.
type Repository struct {
	repositoryConfiguration

	// Name holds the name of the repository.
	Name string
	// DataPath holds the absolute path to the repository.
	DataPath string
}

// loadConfiguration loads the configuration from the dedicated configuration file.
func (r *Repository) loadConfiguration() (err error) {
	configurationFilePath := filepath.Join(r.DataPath, "repository.json")

	data, err := os.ReadFile(configurationFilePath)
	if errors.Is(err, os.ErrNotExist) {
		r.repositoryConfiguration = defaultConfiguration

		return nil
	} else if err != nil {
		return pkgerrors.Wrap(err, configurationFilePath)
	}

	if err := json.Unmarshal(data, &r.repositoryConfiguration); err != nil {
		return pkgerrors.Wrap(err, configurationFilePath)
	}

	return r.repositoryConfiguration.validate()
}

// Evaluate evaluates a repository with the given model and language.
func (r *Repository) Evaluate(logger *log.Logger, resultPath string, model evalmodel.Model, language language.Language, taskIdentifier task.Identifier) (repositoryAssessment metrics.Assessments, problems []error, err error) {
	log, logClose, err := log.WithFile(logger, filepath.Join(resultPath, string(taskIdentifier), evalmodel.CleanModelNameForFileSystem(model.ID()), language.ID(), r.Name+".log"))
	if err != nil {
		return nil, nil, err
	}
	defer logClose()

	log.Printf("Evaluating model %q on task %q using language %q and repository %q", model.ID(), taskIdentifier, language.ID(), r.Name)
	defer func() {
		log.Printf("Evaluated model %q on task %q using language %q and repository %q: encountered %d problems: %+v", model.ID(), taskIdentifier, language.ID(), r.Name, len(problems), problems)
	}()

	filePaths, err := language.Files(log, r.DataPath)
	if err != nil {
		return nil, problems, pkgerrors.WithStack(err)
	}

	repositoryAssessment = metrics.NewAssessments()
	for _, filePath := range filePaths {
		if err := r.Reset(logger); err != nil {
			logger.Panicf("ERROR: unable to reset temporary repository path: %s", err)
		}

		ctx := task.Context{
			Language: language,

			RepositoryPath: r.DataPath,
			FilePath:       filePath,

			Logger: log,
		}
		assessments, err := model.RunTask(ctx, taskIdentifier)
		if err != nil {
			problems = append(problems, pkgerrors.WithMessage(err, filePath))

			continue
		}
		if assessments[metrics.AssessmentKeyProcessingTime] == 0 {
			return nil, nil, pkgerrors.Errorf("no model response time measurement present for %q at repository %q", model.ID(), r.Name)
		}
		repositoryAssessment.Add(assessments)
		repositoryAssessment.Award(metrics.AssessmentKeyResponseNoError)

		coverage, ps, err := language.Execute(log, r.DataPath)
		problems = append(problems, ps...)
		if err != nil {
			problems = append(problems, pkgerrors.WithMessage(err, filePath))

			continue
		}
		log.Printf("Executes tests with %d coverage objects", coverage)
		repositoryAssessment.Award(metrics.AssessmentKeyFilesExecuted)
		repositoryAssessment.AwardPoints(metrics.AssessmentKeyCoverage, coverage)
	}

	return repositoryAssessment, problems, nil
}

// Reset resets a repository back to its "initial" commit.
func (r *Repository) Reset(logger *log.Logger) (err error) {
	out, err := util.CommandWithResult(context.Background(), logger, &util.Command{
		Command: []string{
			"git",
			"clean",
			"-df",
		},

		Directory: r.DataPath,
		Env: map[string]string{ // Overwrite the global and system configs to point to the default one.
			"GIT_CONFIG_GLOBAL": filepath.Join(r.DataPath, ".git", "config"),
			"GIT_CONFIG_SYSTEM": filepath.Join(r.DataPath, ".git", "config"),
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
		Name:     repositoryPathRelative,
		DataPath: temporaryRepositoryPath,
	}
	if err := repository.loadConfiguration(); err != nil {
		return nil, cleanup, err
	}

	return repository, cleanup, nil
}
