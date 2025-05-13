package task

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	pkgerrors "github.com/pkg/errors"
	"github.com/zimmski/osutil"

	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	"github.com/symflower/eval-dev-quality/util"
)

// RepositoryConfiguration holds the configuration of a repository.
type RepositoryConfiguration struct {
	// Tasks holds the tasks supported by the repository.
	Tasks []Identifier `json:"tasks"`
	// IgnorePaths holds the relative paths that should be ignored when searching for cases.
	IgnorePaths []string `json:"ignore,omitempty"`
	// HasTestsInSource determines if the tests for this repository are located within the corresponding implementation file.
	HasTestsInSource bool `json:"tests-in-source,omitempty"`

	// Prompt holds LLM prompt-related configuration.
	Prompt RepositoryConfigurationPrompt `json:"prompt,omitempty"`

	// Validation holds quality gates for evaluation.
	Validation RepositoryConfigurationValidation `json:"validation,omitempty"`

	// MaxScores holds the maximum scores per task type, case and metric for this repository.
	MaxScores map[Identifier]map[string]map[metrics.AssessmentKey]uint64 `json:"scores,omitempty"`
}

// RepositoryConfigurationPrompt holds LLM prompt-related configuration.
type RepositoryConfigurationPrompt struct {
	// TestFramework overwrites the language-specific test framework to use.
	TestFramework string `json:"test-framework,omitempty"`
}

// RepositoryConfigurationValidation holds quality gates for evaluation.
type RepositoryConfigurationValidation struct {
	// Execution holds execution-related validation.
	Execution RepositoryConfigurationExecution `json:"execution,omitempty"`
}

// RepositoryConfigurationExecution execution-related quality gates for evaluation.
type RepositoryConfigurationExecution struct {
	// StdOutRE holds a regular expression that must be part of execution standard output.
	StdOutRE string `json:"stdout,omitempty"`
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

// Write stores the configuration on the given path.
func (rc *RepositoryConfiguration) Write(path string) (err error) {
	data, err := json.Marshal(rc)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0666)
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

	if _, err := regexp.Compile(rc.Validation.Execution.StdOutRE); err != nil {
		return pkgerrors.WithMessagef(err, "invalid regular expression %q", rc.Validation.Execution.StdOutRE)
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

// Validate validates execution outcomes against the configured quality gates.
func (e *RepositoryConfigurationExecution) Validate(stdout string) bool {
	if e.StdOutRE != "" {
		return regexp.MustCompile(e.StdOutRE).MatchString(stdout)
	}

	return true
}
