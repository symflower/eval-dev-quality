package testing

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	metricstesting "github.com/symflower/eval-dev-quality/evaluate/metrics/testing"
	"github.com/symflower/eval-dev-quality/language"
	"github.com/symflower/eval-dev-quality/log"
	"github.com/symflower/eval-dev-quality/model"
	evaltask "github.com/symflower/eval-dev-quality/task"
	"github.com/zimmski/osutil"
	"golang.org/x/exp/maps"
)

// TestCaseTask holds a test case for a task.
type TestCaseTask struct {
	Name string

	Setup func(t *testing.T)

	Task           evaltask.Task
	Model          model.Model
	Language       language.Language
	TestDataPath   string
	RepositoryPath string

	ExpectedRepositoryAssessment map[string]map[evaltask.Identifier]metrics.Assessments
	ExpectedResultFiles          map[string]func(t *testing.T, filePath string, data string)
	ExpectedProblemContains      []string
	ExpectedError                error
	ValidateLog                  func(t *testing.T, data string)
}

type createRepositoryFunction func(logger *log.Logger, testDataPath string, repositoryPathRelative string) (repository evaltask.Repository, cleanup func(), err error)

// Validate validates the object.
func (tc *TestCaseTask) Validate(t *testing.T, createRepository createRepositoryFunction) {
	resultPath := t.TempDir()

	logOutput, logger := log.Buffer()
	defer func() {
		if t.Failed() {
			t.Logf("Logging output: %s", logOutput.String())
		}
	}()
	repository, cleanup, err := createRepository(logger, tc.TestDataPath, tc.RepositoryPath)
	assert.NoError(t, err)
	defer cleanup()

	if tc.Setup != nil {
		tc.Setup(t)
	}

	taskContext := evaltask.Context{
		Language:   tc.Language,
		Repository: repository,
		Model:      tc.Model,

		ResultPath: resultPath,

		Logger: logger,
	}
	actualRepositoryAssessment, actualProblems, actualErr := tc.Task.Run(taskContext)

	for _, caseName := range maps.Keys(tc.ExpectedRepositoryAssessment) {
		tc.ExpectedRepositoryAssessment[caseName] = metricstesting.CleanMap(tc.ExpectedRepositoryAssessment[caseName])
	}
	for _, caseName := range maps.Keys(actualRepositoryAssessment) {
		actualRepositoryAssessment[caseName] = metricstesting.CleanMap(actualRepositoryAssessment[caseName])
	}

	assert.Equal(t, tc.ExpectedRepositoryAssessment, actualRepositoryAssessment)

	if assert.Equal(t, len(tc.ExpectedProblemContains), len(actualProblems), "problems count") {
		for i, expectedProblem := range tc.ExpectedProblemContains {
			actualProblem := actualProblems[i]
			assert.Containsf(t, actualProblem.Error(), expectedProblem, "Problem %d", i)
		}
	} else {
		for i, problem := range actualProblems {
			t.Logf("Actual problem %d:\n%+v", i, problem)
		}
	}
	assert.Equal(t, tc.ExpectedError, actualErr)

	actualResultFiles, err := osutil.FilesRecursive(resultPath)
	require.NoError(t, err)
	for i, p := range actualResultFiles {
		actualResultFiles[i], err = filepath.Rel(resultPath, p)
		require.NoError(t, err)
	}
	sort.Strings(actualResultFiles)
	var expectedResultFiles []string
	for filePath, validate := range tc.ExpectedResultFiles {
		expectedResultFiles = append(expectedResultFiles, filePath)

		if validate != nil {
			data, err := os.ReadFile(filepath.Join(resultPath, filePath))
			if assert.NoError(t, err) {
				validate(t, filePath, string(data))
			}
		}
	}
	sort.Strings(expectedResultFiles)
	assert.Equal(t, expectedResultFiles, actualResultFiles)

	if tc.ValidateLog != nil {
		tc.ValidateLog(t, logOutput.String())
	}
}

// TestCaseValidateRepository holds a test case for validating a repository.
type TestCaseValidateRepository struct {
	Name string

	Before func(repositoryPath string)

	TestdataPath   string
	RepositoryPath string
	Language       language.Language

	ExpectedError func(t *testing.T, err error)
}

type validateRepositoryForTask func(logger *log.Logger, repositoryPath string, language language.Language) (err error)

// Validate validates the object.
func (tc *TestCaseValidateRepository) Validate(t *testing.T, validateRepositoryForTask validateRepositoryForTask) {
	t.Run(tc.Name, func(t *testing.T) {
		logOutput, logger := log.Buffer()
		defer func() {
			if t.Failed() {
				t.Logf("Logging output: %s", logOutput.String())
			}
		}()

		temporaryDirectory := t.TempDir()
		repositoryPath := filepath.Join(temporaryDirectory, "testdata", tc.RepositoryPath)
		require.NoError(t, osutil.CopyTree(filepath.Join(tc.TestdataPath, tc.RepositoryPath), repositoryPath))

		if tc.Before != nil {
			tc.Before(repositoryPath)
		}

		actualErr := validateRepositoryForTask(logger, repositoryPath, tc.Language)
		if tc.ExpectedError != nil {
			tc.ExpectedError(t, actualErr)
		} else {
			require.NoError(t, actualErr)
		}
	})
}
