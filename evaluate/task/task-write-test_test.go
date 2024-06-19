package task

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	metricstesting "github.com/symflower/eval-dev-quality/evaluate/metrics/testing"
	tasktesting "github.com/symflower/eval-dev-quality/evaluate/task/testing"
	"github.com/symflower/eval-dev-quality/language/golang"
	"github.com/symflower/eval-dev-quality/log"
	modeltesting "github.com/symflower/eval-dev-quality/model/testing"
	"github.com/symflower/eval-dev-quality/task"
	"github.com/zimmski/osutil"
	"github.com/zimmski/osutil/bytesutil"
)

func TestTaskWriteTestsRun(t *testing.T) {
	validate := func(t *testing.T, tc *tasktesting.TestCaseTask) {
		t.Run(tc.Name, func(t *testing.T) {
			resultPath := t.TempDir()

			logOutput, logger := log.Buffer()
			defer func() {
				if t.Failed() {
					t.Logf("Logging output: %s", logOutput.String())
				}
			}()
			repository, cleanup, err := TemporaryRepository(logger, tc.TestDataPath, tc.RepositoryPath)
			assert.NoError(t, err)
			defer cleanup()

			tc.Validate(t, &TaskWriteTests{}, repository, resultPath, logger)
		})
	}

	t.Run("Clear repository on each task file", func(t *testing.T) {
		temporaryDirectoryPath := t.TempDir()

		repositoryPath := filepath.Join(temporaryDirectoryPath, "golang", "plain")
		require.NoError(t, os.MkdirAll(repositoryPath, 0700))
		require.NoError(t, os.WriteFile(filepath.Join(repositoryPath, "go.mod"), []byte("module plain\n\ngo 1.21.5"), 0600))
		require.NoError(t, os.WriteFile(filepath.Join(repositoryPath, "taskA.go"), []byte("package plain\n\nfunc TaskA(){}"), 0600))
		require.NoError(t, os.WriteFile(filepath.Join(repositoryPath, "taskB.go"), []byte("package plain\n\nfunc TaskB(){}"), 0600))

		modelMock := modeltesting.NewMockCapabilityWriteTestsNamed(t, "mocked-model")

		// Generate invalid code for the first taskcontext.
		modelMock.RegisterGenerateSuccess(t, "taskA_test.go", "does not compile", metricstesting.AssessmentsWithProcessingTime).Once()
		// Generate valid code for the second taskcontext.
		modelMock.RegisterGenerateSuccess(t, "taskB_test.go", "package plain\n\nimport \"testing\"\n\nfunc TestTaskB(t *testing.T){}", metricstesting.AssessmentsWithProcessingTime).Once()

		validate(t, &tasktesting.TestCaseTask{
			Name: "Plain",

			Model:          modelMock,
			Language:       &golang.Language{},
			TestDataPath:   temporaryDirectoryPath,
			RepositoryPath: filepath.Join("golang", "plain"),

			ExpectedRepositoryAssessment: map[task.Identifier]metrics.Assessments{
				IdentifierWriteTests: metrics.Assessments{
					metrics.AssessmentKeyFilesExecuted:   1,
					metrics.AssessmentKeyResponseNoError: 2,
				},
				IdentifierWriteTestsSymflowerFix: metrics.Assessments{
					metrics.AssessmentKeyFilesExecuted:   1,
					metrics.AssessmentKeyResponseNoError: 2,
				},
			},
			ExpectedProblemContains: []string{
				"expected 'package', found does",
				"exit status 1",
			},
			ExpectedResultFiles: map[string]func(t *testing.T, filePath string, data string){
				filepath.Join(string(IdentifierWriteTests), "mocked-model", "golang", "golang", "plain.log"): func(t *testing.T, filePath, data string) {
					assert.Contains(t, data, "Evaluating model \"mocked-model\"")
					assert.Contains(t, data, "PASS: TestTaskB")
				},
			},
		})
	})

	t.Run("Symflower Fix", func(t *testing.T) {
		t.Run("Go", func(t *testing.T) {
			validateGo := func(t *testing.T, testName string, testFileContent string, expectedAssessments map[task.Identifier]metrics.Assessments, expectedProblems []string, assertTestsPass bool) {
				temporaryDirectoryPath := t.TempDir()
				repositoryPath := filepath.Join(temporaryDirectoryPath, "golang", "plain")
				require.NoError(t, osutil.CopyTree(filepath.Join("..", "..", "testdata", "golang", "plain"), repositoryPath))

				modelMock := modeltesting.NewMockCapabilityWriteTestsNamed(t, "mocked-model")
				modelMock.RegisterGenerateSuccess(t, "plain_test.go", testFileContent, metricstesting.AssessmentsWithProcessingTime).Once()

				validate(t, &tasktesting.TestCaseTask{
					Name: testName,

					Model:          modelMock,
					Language:       &golang.Language{},
					TestDataPath:   temporaryDirectoryPath,
					RepositoryPath: filepath.Join("golang", "plain"),

					ExpectedRepositoryAssessment: expectedAssessments,
					ExpectedProblemContains:      expectedProblems,
					ExpectedResultFiles: map[string]func(t *testing.T, filePath string, data string){
						filepath.Join(string(IdentifierWriteTests), "mocked-model", "golang", "golang", "plain.log"): func(t *testing.T, filePath, data string) {
							assert.Contains(t, data, "Evaluating model \"mocked-model\"")
							if assertTestsPass {
								assert.Contains(t, data, "PASS: TestPlain")
							}
						},
					},
				})
			}
			{
				expectedAssessments := map[task.Identifier]metrics.Assessments{
					IdentifierWriteTests: metrics.Assessments{
						metrics.AssessmentKeyFilesExecuted:   1,
						metrics.AssessmentKeyResponseNoError: 1,
						metrics.AssessmentKeyCoverage:        10,
					},
					IdentifierWriteTestsSymflowerFix: metrics.Assessments{
						metrics.AssessmentKeyFilesExecuted:   1,
						metrics.AssessmentKeyResponseNoError: 1,
						metrics.AssessmentKeyCoverage:        10,
					},
				}
				validateGo(t, "Model generated correct test", bytesutil.StringTrimIndentations(`
					package plain

					import "testing"

					func TestPlain(t *testing.T) {
						   plain()
					}
				`), expectedAssessments, nil, true)
			}
			{
				expectedAssessments := map[task.Identifier]metrics.Assessments{
					IdentifierWriteTests: metrics.Assessments{
						metrics.AssessmentKeyResponseNoError: 1,
					},
					IdentifierWriteTestsSymflowerFix: metrics.Assessments{
						metrics.AssessmentKeyFilesExecuted:   1,
						metrics.AssessmentKeyResponseNoError: 1,
						metrics.AssessmentKeyCoverage:        10,
					},
				}
				expectedProblems := []string{
					"imported and not used",
				}
				validateGo(t, "Model generated test with unused import", bytesutil.StringTrimIndentations(`
					package plain

					import (
						"testing"
						"strings"
					)

					func TestPlain(t *testing.T) {
					   	plain()
					}
				`), expectedAssessments, expectedProblems, true)
			}
			{
				expectedAssessments := map[task.Identifier]metrics.Assessments{
					IdentifierWriteTests: metrics.Assessments{
						metrics.AssessmentKeyResponseNoError: 1,
					},
					IdentifierWriteTestsSymflowerFix: metrics.Assessments{
						metrics.AssessmentKeyResponseNoError: 1,
					},
				}
				expectedProblems := []string{
					"expected declaration, found this",
					"unable to format source code",
				}
				validateGo(t, "Model generated test that is unfixable", bytesutil.StringTrimIndentations(`
					package plain

					this is not valid go code
				`), expectedAssessments, expectedProblems, false)
			}
		})
	})
}
