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
	"github.com/symflower/eval-dev-quality/model"
	evaltask "github.com/symflower/eval-dev-quality/task"
	"github.com/zimmski/osutil"
)

type TestCaseTask struct {
	Name string

	Model          model.Model
	Language       language.Language
	TestDataPath   string
	RepositoryPath string

	ExpectedRepositoryAssessment map[evaltask.Identifier]metrics.Assessments
	ExpectedResultFiles          map[string]func(t *testing.T, filePath string, data string)
	ExpectedProblemContains      []string
	ExpectedError                error
}

func (tc *TestCaseTask) Validate(t *testing.T, task evaltask.Task, repository evaltask.Repository, resultPath string) {
	actualRepositoryAssessment, actualProblems, actualErr := task.Run(repository)

	metricstesting.AssertTaskAssessmentsEqual(t, tc.ExpectedRepositoryAssessment, actualRepositoryAssessment)
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
	expectedResultFiles := make([]string, 0, len(tc.ExpectedResultFiles))
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
}
