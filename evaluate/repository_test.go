package evaluate

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zimmski/osutil"
	"github.com/zimmski/osutil/bytesutil"

	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	metricstesting "github.com/symflower/eval-dev-quality/evaluate/metrics/testing"
	"github.com/symflower/eval-dev-quality/language"
	"github.com/symflower/eval-dev-quality/language/golang"
	"github.com/symflower/eval-dev-quality/log"
	"github.com/symflower/eval-dev-quality/model"
	"github.com/symflower/eval-dev-quality/model/symflower"
	modeltesting "github.com/symflower/eval-dev-quality/model/testing"
	"github.com/symflower/eval-dev-quality/task"
	"github.com/symflower/eval-dev-quality/tools"
	toolstesting "github.com/symflower/eval-dev-quality/tools/testing"
	"github.com/symflower/eval-dev-quality/util"
)

func TestRepository(t *testing.T) {
	toolstesting.RequiresTool(t, tools.NewSymflower())

	type testCase struct {
		Name string

		Model          model.Model
		Language       language.Language
		TestDataPath   string
		RepositoryPath string

		ExpectedRepositoryAssessment metrics.Assessments
		ExpectedResultFiles          map[string]func(t *testing.T, filePath string, data string)
		ExpectedProblemContains      []string
		ExpectedError                error
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			temporaryPath := t.TempDir()

			_, logger := log.Buffer()
			temporaryRepository, cleanup, err := TemporaryRepository(logger, tc.TestDataPath, tc.RepositoryPath)
			assert.NoError(t, err)
			defer cleanup()

			actualRepositoryAssessment, actualProblems, actualErr := temporaryRepository.Evaluate(logger, temporaryPath, tc.Model, tc.Language, task.IdentifierWriteTests)

			metricstesting.AssertAssessmentsEqual(t, tc.ExpectedRepositoryAssessment, actualRepositoryAssessment)
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

			actualResultFiles, err := osutil.FilesRecursive(temporaryPath)
			require.NoError(t, err)
			for i, p := range actualResultFiles {
				actualResultFiles[i], err = filepath.Rel(temporaryPath, p)
				require.NoError(t, err)
			}
			sort.Strings(actualResultFiles)
			expectedResultFiles := make([]string, 0, len(tc.ExpectedResultFiles))
			for filePath, validate := range tc.ExpectedResultFiles {
				expectedResultFiles = append(expectedResultFiles, filePath)

				if validate != nil {
					data, err := os.ReadFile(filepath.Join(temporaryPath, filePath))
					if assert.NoError(t, err) {
						validate(t, filePath, string(data))
					}
				}
			}
			sort.Strings(expectedResultFiles)
			assert.Equal(t, expectedResultFiles, actualResultFiles)
		})
	}

	validate(t, &testCase{
		Name: "Plain",

		Model:          symflower.NewModel(),
		Language:       &golang.Language{},
		TestDataPath:   filepath.Join("..", "testdata"),
		RepositoryPath: filepath.Join("golang", "plain"),

		ExpectedRepositoryAssessment: metrics.Assessments{
			metrics.AssessmentKeyGenerateTestsForFileCharacterCount: 254,
			metrics.AssessmentKeyResponseCharacterCount:             254,
			metrics.AssessmentKeyCoverage:                           10,
			metrics.AssessmentKeyFilesExecuted:                      1,
			metrics.AssessmentKeyResponseNoError:                    1,
			metrics.AssessmentKeyResponseNoExcess:                   1,
			metrics.AssessmentKeyResponseWithCode:                   1,
		},
		ExpectedResultFiles: map[string]func(t *testing.T, filePath string, data string){
			filepath.Join(string(task.IdentifierWriteTests), "symflower_symbolic-execution", "golang", "golang", "plain.log"): func(t *testing.T, filePath, data string) {
				assert.Contains(t, data, "Evaluating model \"symflower/symbolic-execution\"")
				assert.Contains(t, data, "Generated 1 test")
				assert.Contains(t, data, "PASS: TestSymflowerPlain")
				assert.Contains(t, data, "Evaluated model \"symflower/symbolic-execution\"")
			},
		},
	})
	t.Run("Clear repository on each task file", func(t *testing.T) {
		temporaryDirectoryPath := t.TempDir()

		repositoryPath := filepath.Join(temporaryDirectoryPath, "golang", "plain")
		require.NoError(t, os.MkdirAll(repositoryPath, 0700))
		require.NoError(t, os.WriteFile(filepath.Join(repositoryPath, "go.mod"), []byte("module plain\n\ngo 1.21.5"), 0600))
		require.NoError(t, os.WriteFile(filepath.Join(repositoryPath, "taskA.go"), []byte("package plain\n\nfunc TaskA(){}"), 0600))
		require.NoError(t, os.WriteFile(filepath.Join(repositoryPath, "taskB.go"), []byte("package plain\n\nfunc TaskB(){}"), 0600))

		modelMock := modeltesting.NewMockModelNamed(t, "mocked-model")

		// Generate invalid code for the first task.
		modelMock.RegisterGenerateSuccess(t, "taskA_test.go", "does not compile", metricstesting.AssessmentsWithProcessingTime).Once()
		// Generate valid code for the second task.
		modelMock.RegisterGenerateSuccess(t, "taskB_test.go", "package plain\n\nimport \"testing\"\n\nfunc TestTaskB(t *testing.T){}", metricstesting.AssessmentsWithProcessingTime).Once()

		validate(t, &testCase{
			Name: "Plain",

			Model:          modelMock,
			Language:       &golang.Language{},
			TestDataPath:   temporaryDirectoryPath,
			RepositoryPath: filepath.Join("golang", "plain"),

			ExpectedRepositoryAssessment: metrics.Assessments{
				metrics.AssessmentKeyFilesExecuted:   1,
				metrics.AssessmentKeyResponseNoError: 2,
			},
			ExpectedProblemContains: []string{
				"expected 'package', found does",
			},
			ExpectedResultFiles: map[string]func(t *testing.T, filePath string, data string){
				filepath.Join(string(task.IdentifierWriteTests), "mocked-model", "golang", "golang", "plain.log"): func(t *testing.T, filePath, data string) {
					assert.Contains(t, data, "Evaluating model \"mocked-model\"")
					assert.Contains(t, data, "PASS: TestTaskB")
				},
			},
		})
	})
}

func TestTemporaryRepository(t *testing.T) {
	type testCase struct {
		Name string

		TestDataPath   string
		RepositoryPath string

		ExpectedTempPathRegex string
		ExpectedErr           error
		ValidateAfter         func(t *testing.T, logger *log.Logger, repositoryPath string)
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			if osutil.IsWindows() {
				t.Skipf("Regex tests with paths are not supported on this OS")
			}

			logBuffer, logger := log.Buffer()
			defer func() {
				if t.Failed() {
					t.Logf("Log Output:\n%s", logBuffer.String())
				}
			}()

			actualTemporaryRepository, cleanup, actualErr := TemporaryRepository(logger, tc.TestDataPath, tc.RepositoryPath)
			defer cleanup()

			assert.Regexp(t, filepath.Join(os.TempDir(), tc.ExpectedTempPathRegex), actualTemporaryRepository, actualTemporaryRepository)
			assert.Equal(t, tc.ExpectedErr, actualErr)

			if tc.ValidateAfter != nil {
				tc.ValidateAfter(t, logger, actualTemporaryRepository.DataPath)
			}
		})
	}

	validate(t, &testCase{
		Name: "Create temporary path with git repository",

		TestDataPath:   filepath.Join("..", "testdata"),
		RepositoryPath: filepath.Join("golang", "plain"),

		ExpectedTempPathRegex: `eval-dev-quality\d+\/plain`,
		ExpectedErr:           nil,
		ValidateAfter: func(t *testing.T, logger *log.Logger, repositoryPath string) {
			output, err := util.CommandWithResult(context.Background(), logger, &util.Command{
				Command: []string{
					"git",
					"log",
				},

				Directory: repositoryPath,
			})
			require.NoError(t, err)
			assert.Contains(t, output, "Author: dummy-name-temporary-repository")
		},
	})
}

func TestResetTemporaryRepository(t *testing.T) {
	type testCase struct {
		Name string

		TestDataPath   string
		RepositoryPath string

		ExpectedErr    error
		MutationBefore func(t *testing.T, path string)
		ValidateAfter  func(t *testing.T, path string)
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			_, logger := log.Buffer()
			temporaryRepository, cleanup, err := TemporaryRepository(logger, tc.TestDataPath, tc.RepositoryPath)
			assert.NoError(t, err)
			defer cleanup()

			tc.MutationBefore(t, temporaryRepository.DataPath)

			actualErr := temporaryRepository.Reset(logger)
			assert.Equal(t, tc.ExpectedErr, actualErr)

			tc.ValidateAfter(t, temporaryRepository.DataPath)
		})
	}

	validate(t, &testCase{
		Name: "Reset changes",

		TestDataPath:   filepath.Join("..", "testdata"),
		RepositoryPath: filepath.Join("golang", "plain"),

		ExpectedErr: nil,
		MutationBefore: func(t *testing.T, path string) {
			assert.NoError(t, os.WriteFile(filepath.Join(path, "foo"), []byte("foo"), 0600))
		},
		ValidateAfter: func(t *testing.T, path string) {
			assert.Error(t, osutil.FileExists(filepath.Join(path, "foo")))
		},
	})
}

func TestRepositoryLoadConfiguration(t *testing.T) {
	type testCase struct {
		Name string

		TestDataPath   string
		RepositoryPath string

		ExpectedErrorText string
		MutationBefore    func(t *testing.T, path string)
		ValidateAfter     func(t *testing.T, repository *Repository)
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			temporaryPath := t.TempDir()
			temporaryRepositoryPath := filepath.Join(temporaryPath, tc.RepositoryPath)
			require.NoError(t, osutil.CopyTree(filepath.Join(tc.TestDataPath, tc.RepositoryPath), temporaryRepositoryPath))

			if tc.MutationBefore != nil {
				tc.MutationBefore(t, temporaryRepositoryPath)
			}

			_, logger := log.Buffer()
			actualRepository, cleanup, actualErr := TemporaryRepository(logger, temporaryPath, tc.RepositoryPath)
			defer cleanup()
			if tc.ExpectedErrorText != "" {
				assert.ErrorContains(t, actualErr, tc.ExpectedErrorText)
			} else {
				assert.NoError(t, actualErr)
			}

			if tc.ValidateAfter != nil {
				tc.ValidateAfter(t, actualRepository)
			}
		})
	}

	validate(t, &testCase{
		Name: "No configuration file",

		TestDataPath:   filepath.Join("..", "testdata"),
		RepositoryPath: filepath.Join("golang", "plain"),

		ValidateAfter: func(t *testing.T, repository *Repository) {
			assert.Equal(t, task.AllIdentifiers, repository.Tasks)
		},
	})
	validate(t, &testCase{
		Name: "Specify known task",

		TestDataPath:   filepath.Join("..", "testdata"),
		RepositoryPath: filepath.Join("golang", "plain"),

		MutationBefore: func(t *testing.T, repositoryPath string) {
			configuration := bytesutil.StringTrimIndentations(`
				{
					"tasks": [
						"write-tests"
					]
				}
			`)
			assert.NoError(t, os.WriteFile(filepath.Join(repositoryPath, "repository.json"), []byte(configuration), 0600))
		},
		ValidateAfter: func(t *testing.T, repository *Repository) {
			expectedTaskIdentifiers := []task.Identifier{
				task.IdentifierWriteTests,
			}
			assert.Equal(t, expectedTaskIdentifiers, repository.Tasks)
		},
	})
	validate(t, &testCase{
		Name: "Specify unknown task",

		TestDataPath:   filepath.Join("..", "testdata"),
		RepositoryPath: filepath.Join("golang", "plain"),

		ExpectedErrorText: "task identifier \"unknown-task\" unknown",
		MutationBefore: func(t *testing.T, repositoryPath string) {
			configuration := bytesutil.StringTrimIndentations(`
				{
					"tasks": [
						"write-tests",
						"unknown-task"
					]
				}
			`)
			assert.NoError(t, os.WriteFile(filepath.Join(repositoryPath, "repository.json"), []byte(configuration), 0600))
		},
	})
}
