package testintegration

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	evaluatetask "github.com/symflower/eval-dev-quality/evaluate/task"
	tasktesting "github.com/symflower/eval-dev-quality/evaluate/task/testing"
	"github.com/symflower/eval-dev-quality/language/golang"
	"github.com/symflower/eval-dev-quality/language/rust"
	"github.com/symflower/eval-dev-quality/log"
	"github.com/symflower/eval-dev-quality/model/llm"
	"github.com/symflower/eval-dev-quality/model/symflower"
	"github.com/symflower/eval-dev-quality/provider"
	providertesting "github.com/symflower/eval-dev-quality/provider/testing"
	evaltask "github.com/symflower/eval-dev-quality/task"
	"github.com/symflower/eval-dev-quality/tools"
	toolstesting "github.com/symflower/eval-dev-quality/tools/testing"
	"github.com/zimmski/osutil"
	"github.com/zimmski/osutil/bytesutil"
)

func TestWriteTestsRun(t *testing.T) {
	toolstesting.RequiresTool(t, tools.NewSymflower())

	validate := func(t *testing.T, tc *tasktesting.TestCaseTask) {
		t.Run(tc.Name, func(t *testing.T) {
			task, err := evaluatetask.ForIdentifier(evaluatetask.IdentifierWriteTests)
			require.NoError(t, err)
			tc.Task = task

			tc.Validate(t,
				func(logger *log.Logger, testDataPath string, repositoryPathRelative string) (repository evaltask.Repository, cleanup func(), err error) {
					return evaluatetask.TemporaryRepository(logger, testDataPath, repositoryPathRelative)
				},
			)
		})
	}

	validate(t, &tasktesting.TestCaseTask{
		Name: "Plain",

		Model:          symflower.NewModelSymbolicExecution(),
		Language:       &golang.Language{},
		TestDataPath:   filepath.Join("..", "..", "..", "testdata"),
		RepositoryPath: filepath.Join("golang", "plain"),

		ExpectedRepositoryAssessment: map[string]map[evaltask.Identifier]metrics.Assessments{
			"plain.go": {
				evaluatetask.IdentifierWriteTests: metrics.Assessments{
					metrics.AssessmentKeyGenerateTestsForFileCharacterCount: 254,
					metrics.AssessmentKeyResponseCharacterCount:             254,
					metrics.AssessmentKeyCoverage:                           1,
					metrics.AssessmentKeyFilesExecuted:                      1,
					metrics.AssessmentKeyFilesExecutedMaximumReachable:      1,
					metrics.AssessmentKeyResponseNoError:                    1,
					metrics.AssessmentKeyResponseNoExcess:                   1,
					metrics.AssessmentKeyResponseWithCode:                   1,
				},
				evaluatetask.IdentifierWriteTestsSymflowerFix: metrics.Assessments{
					metrics.AssessmentKeyGenerateTestsForFileCharacterCount: 254,
					metrics.AssessmentKeyResponseCharacterCount:             254,
					metrics.AssessmentKeyCoverage:                           1,
					metrics.AssessmentKeyFilesExecuted:                      1,
					metrics.AssessmentKeyFilesExecutedMaximumReachable:      1,
					metrics.AssessmentKeyResponseNoError:                    1,
					metrics.AssessmentKeyResponseNoExcess:                   1,
					metrics.AssessmentKeyResponseWithCode:                   1,
				},
				evaluatetask.IdentifierWriteTestsSymflowerTemplate: metrics.Assessments{
					metrics.AssessmentKeyGenerateTestsForFileCharacterCount: 254,
					metrics.AssessmentKeyResponseCharacterCount:             254,
					metrics.AssessmentKeyCoverage:                           1,
					metrics.AssessmentKeyFilesExecuted:                      1,
					metrics.AssessmentKeyFilesExecutedMaximumReachable:      1,
					metrics.AssessmentKeyResponseNoError:                    1,
					metrics.AssessmentKeyResponseNoExcess:                   1,
					metrics.AssessmentKeyResponseWithCode:                   1,
				},
				evaluatetask.IdentifierWriteTestsSymflowerTemplateSymflowerFix: metrics.Assessments{
					metrics.AssessmentKeyGenerateTestsForFileCharacterCount: 254,
					metrics.AssessmentKeyResponseCharacterCount:             254,
					metrics.AssessmentKeyCoverage:                           1,
					metrics.AssessmentKeyFilesExecuted:                      1,
					metrics.AssessmentKeyFilesExecutedMaximumReachable:      1,
					metrics.AssessmentKeyResponseNoError:                    1,
					metrics.AssessmentKeyResponseNoExcess:                   1,
					metrics.AssessmentKeyResponseWithCode:                   1,
				},
			},
		},
		ValidateLog: func(t *testing.T, data string) {
			assert.Contains(t, data, "msg=\"evaluating model\" model=symflower/symbolic-execution")
			assert.Contains(t, data, "Generated 1 test")
			assert.Contains(t, data, "PASS: TestSymflowerPlain")
			assert.Contains(t, data, "msg=\"evaluated model\" model=symflower/symbolic-execution")
		},
	})
	{
		mockProvider := providertesting.NewMockQuery(t)
		model := llm.NewModel(mockProvider, "model")
		validate(t, &tasktesting.TestCaseTask{
			Name: "Rust",

			Setup: func(t *testing.T) {
				var query any = bytesutil.StringTrimIndentations(`
					Given the following Rust code file "src/plain.rs", provide tests for this code that can be appended to the source file.
					The tests should produce 100 percent code coverage and must compile.
					The response must contain only the test code in a fenced code block and nothing else.

					` + "```" + `rust
					pub fn plain() {
					    // This does not do anything but it gives us a line to cover.
					}
					` + "```" + `
				`)
				if osutil.IsWindows() {
					// There are "carriage returns" in the file on windows, so let's spare ourselves the nightmare of matching against those separately.
					query = mock.Anything
				}
				mockProvider.On("Query", mock.Anything, mock.Anything, mock.Anything, query).Return(
					&provider.QueryResult{
						Message: bytesutil.StringTrimIndentations(`
							` + "```rust`" + `
							#[cfg(test)]
							mod tests {
								use super::*;

								#[test]
								fn test_plain() {
									plain();
								}
							}
							` + "```" + `
						`),
					},
					nil,
				).After(100 * time.Millisecond)
			},

			Model:          model,
			Language:       &rust.Language{},
			TestDataPath:   filepath.Join("..", "..", "..", "testdata"),
			RepositoryPath: filepath.Join("rust", "plain"),

			ExpectedRepositoryAssessment: map[string]map[evaltask.Identifier]metrics.Assessments{
				filepath.Join("src", "plain.rs"): {
					evaluatetask.IdentifierWriteTests: metrics.Assessments{
						metrics.AssessmentKeyGenerateTestsForFileCharacterCount: 84,
						metrics.AssessmentKeyResponseCharacterCount:             98,
						metrics.AssessmentKeyCoverage:                           3,
						metrics.AssessmentKeyFilesExecuted:                      1,
						metrics.AssessmentKeyFilesExecutedMaximumReachable:      1,
						metrics.AssessmentKeyResponseNoError:                    1,
						metrics.AssessmentKeyResponseNoExcess:                   1,
						metrics.AssessmentKeyResponseWithCode:                   1,
					},
					evaluatetask.IdentifierWriteTestsSymflowerFix: metrics.Assessments{
						metrics.AssessmentKeyGenerateTestsForFileCharacterCount: 84,
						metrics.AssessmentKeyResponseCharacterCount:             98,
						metrics.AssessmentKeyCoverage:                           3,
						metrics.AssessmentKeyFilesExecuted:                      1,
						metrics.AssessmentKeyFilesExecutedMaximumReachable:      1,
						metrics.AssessmentKeyResponseNoError:                    1,
						metrics.AssessmentKeyResponseNoExcess:                   1,
						metrics.AssessmentKeyResponseWithCode:                   1,
					},
					evaluatetask.IdentifierWriteTestsSymflowerTemplate: metrics.Assessments{
						metrics.AssessmentKeyGenerateTestsForFileCharacterCount: 84,
						metrics.AssessmentKeyResponseCharacterCount:             98,
						metrics.AssessmentKeyCoverage:                           3,
						metrics.AssessmentKeyFilesExecuted:                      1,
						metrics.AssessmentKeyFilesExecutedMaximumReachable:      1,
						metrics.AssessmentKeyResponseNoError:                    1,
						metrics.AssessmentKeyResponseNoExcess:                   1,
						metrics.AssessmentKeyResponseWithCode:                   1,
					},
					evaluatetask.IdentifierWriteTestsSymflowerTemplateSymflowerFix: metrics.Assessments{
						metrics.AssessmentKeyGenerateTestsForFileCharacterCount: 84,
						metrics.AssessmentKeyResponseCharacterCount:             98,
						metrics.AssessmentKeyCoverage:                           3,
						metrics.AssessmentKeyFilesExecuted:                      1,
						metrics.AssessmentKeyFilesExecutedMaximumReachable:      1,
						metrics.AssessmentKeyResponseNoError:                    1,
						metrics.AssessmentKeyResponseNoExcess:                   1,
						metrics.AssessmentKeyResponseWithCode:                   1,
					},
				},
			},
			ValidateLog: func(t *testing.T, data string) {
				assert.Contains(t, data, "test result: ok. 1 passed; 0 failed; 0 ignored; 0 measured; 0 filtered out;")
			},
		})
	}
}
