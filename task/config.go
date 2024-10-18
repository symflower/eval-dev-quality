package task

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	pkgerrors "github.com/pkg/errors"
	"github.com/symflower/eval-dev-quality/util"
	"github.com/zimmski/osutil"
)

// RepositoryConfiguration holds the configuration of a repository.
type RepositoryConfiguration struct {
	// Tasks holds the tasks supported by the repository.
	Tasks []Identifier
	// IgnorePaths holds the relative paths that should be ignored when searching for cases.
	IgnorePaths []string `json:"ignore,omitempty"`
}

// RepositoryConfigurationFileName holds the file name for a repository configuration.
const RepositoryConfigurationFileName = "repository.json"

// LoadRepositoryConfiguration loads a repository configuration from the given path.
func LoadRepositoryConfiguration(path string, defaultTasks []Identifier) (config *RepositoryConfiguration, err error) {
	if osutil.FileExists(path) != nil { // If we don't get a valid file, assume it is a repository directory and target the default configuration file name.
		path = filepath.Join(path, RepositoryConfigurationFileName)
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) && len(defaultTasks) > 0 {
		// Set default configuration.
		return &RepositoryConfiguration{
			Tasks: defaultTasks,
		}, nil
	} else if err != nil {
		return nil, pkgerrors.Wrap(err, path)
	}

	config = &RepositoryConfiguration{}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, pkgerrors.Wrap(err, path)
	} else if err := config.validate(defaultTasks); err != nil {
		return nil, err
	}

	return config, nil
}

// validate validates the configuration.
func (rc *RepositoryConfiguration) validate(validTasks []Identifier) (err error) {
	if len(rc.Tasks) == 0 {
		return pkgerrors.Errorf("empty list of tasks in configuration")
	}

	if len(validTasks) > 0 {
		lookupTask := util.Set(validTasks)
		for _, taskIdentifier := range rc.Tasks {
			if !lookupTask[taskIdentifier] {
				return pkgerrors.Errorf("task identifier %q unknown", taskIdentifier)
			}
		}
	}

	return nil
}

// IsFilePathIgnored checks if the given relative file path is to be ignored when searching for cases.
func (rc *RepositoryConfiguration) IsFilePathIgnored(filePath string) bool {
	filePath = filepath.Clean(filePath)
	for _, ignoredFilePath := range rc.IgnorePaths {
		ignoredFilePath = filepath.Clean(ignoredFilePath)
		if strings.HasPrefix(filePath, ignoredFilePath) {
			return true
		}
	}

	return false
}
