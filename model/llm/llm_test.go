package llm

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/zimmski/osutil"
	"github.com/zimmski/osutil/bytesutil"

	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	metricstesting "github.com/symflower/eval-dev-quality/evaluate/metrics/testing"
	evaluatetask "github.com/symflower/eval-dev-quality/evaluate/task"
	"github.com/symflower/eval-dev-quality/language"
	"github.com/symflower/eval-dev-quality/language/golang"
	"github.com/symflower/eval-dev-quality/language/java"
	"github.com/symflower/eval-dev-quality/language/rust"
	"github.com/symflower/eval-dev-quality/log"
	"github.com/symflower/eval-dev-quality/model"
	"github.com/symflower/eval-dev-quality/provider"
	providertesting "github.com/symflower/eval-dev-quality/provider/testing"
)

func TestModelGenerateTestsForFile(t *testing.T) {
	type testCase struct {
		Name string

		SetupMock func(mockedProvider *providertesting.MockQuery)

		Language          language.Language
		ModelID           string
		SourceFileContent string
		SourceFilePath    string

		ExpectedAssessment            metrics.Assessments
		ExpectedTestFileContent       string
		ExpectedTestFilePath          string
		ExpectedTestFilePathNotExists string
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			logOutput, logger := log.Buffer()
			defer func() {
				if t.Failed() {
					t.Log(logOutput.String())
				}
			}()

			temporaryPath := t.TempDir()
			temporaryPath = filepath.Join(temporaryPath, "native")
			require.NoError(t, os.Mkdir(temporaryPath, 0755))

			require.NoError(t, os.WriteFile(filepath.Join(temporaryPath, tc.SourceFilePath), []byte(bytesutil.StringTrimIndentations(tc.SourceFileContent)), 0644))

			mock := providertesting.NewMockQuery(t)
			tc.SetupMock(mock)
			llm := NewModel(mock, tc.ModelID)

			ctx := model.Context{
				Language: tc.Language,

				RepositoryPath: temporaryPath,
				FilePath:       tc.SourceFilePath,

				Logger: logger,

				Arguments: &evaluatetask.ArgumentsWriteTest{},
			}
			actualAssessment, actualError := llm.WriteTests(ctx)
			assert.NoError(t, actualError)

			assert.Equal(t, metricstesting.Clean(tc.ExpectedAssessment), metricstesting.Clean(actualAssessment))

			if tc.ExpectedTestFilePath != "" {
				actualTestFileContent, err := os.ReadFile(filepath.Join(temporaryPath, tc.ExpectedTestFilePath))
				assert.NoError(t, err)

				assert.Equal(t, strings.TrimSpace(bytesutil.StringTrimIndentations(tc.ExpectedTestFileContent)), string(actualTestFileContent))
			}

			if tc.ExpectedTestFilePathNotExists != "" {
				assert.NoFileExists(t, filepath.Join(temporaryPath, tc.ExpectedTestFilePathNotExists))
			}
		})
	}

	sourceFileContent := `
		package native

		func main() {}
	`
	sourceFilePath := "simple.go"
	promptMessage, err := (&llmWriteTestSourceFilePromptContext{
		llmSourceFilePromptContext: llmSourceFilePromptContext{
			Language: &golang.Language{},

			Code:       bytesutil.StringTrimIndentations(sourceFileContent),
			FilePath:   sourceFilePath,
			ImportPath: "native",
		},
	}).Format()
	require.NoError(t, err)
	validate(t, &testCase{
		Name: "Simple",

		SetupMock: func(mockedProvider *providertesting.MockQuery) {
			queryResult := &provider.QueryResult{
				Message: bytesutil.StringTrimIndentations(`
				` + "```" + `
				package native

				func TestMain() {}
				` + "```" + `
			`),
			}
			mockedProvider.On("Query", mock.Anything, mock.Anything, mock.Anything, promptMessage).Return(queryResult, nil)
		},

		Language:          &golang.Language{},
		ModelID:           "model-id",
		SourceFileContent: sourceFileContent,
		SourceFilePath:    sourceFilePath,

		ExpectedAssessment: metrics.Assessments{
			metrics.AssessmentKeyResponseNoExcess:                   1,
			metrics.AssessmentKeyResponseWithCode:                   1,
			metrics.AssessmentKeyGenerateTestsForFileCharacterCount: 34,
			metrics.AssessmentKeyResponseCharacterCount:             43,
		},
		ExpectedTestFileContent: `
			package native

			func TestMain() {}
		`,
		ExpectedTestFilePath: "simple_test.go",
	})
	validate(t, &testCase{
		Name: "Empty response",

		SetupMock: func(mockedProvider *providertesting.MockQuery) {
			queryResult := &provider.QueryResult{
				Duration: time.Millisecond * 123,
				GenerationInfo: &provider.GenerationInfo{
					TotalCost: 0.123456789,
				},
			}
			mockedProvider.On("Query", mock.Anything, mock.Anything, mock.Anything, promptMessage).Return(queryResult, nil)
		},

		Language:          &golang.Language{},
		ModelID:           "model-id",
		SourceFileContent: sourceFileContent,
		SourceFilePath:    sourceFilePath,

		ExpectedAssessment: metrics.Assessments{
			metrics.AssessmentKeyCostsTokenActual: 0.123456789,
		},
		ExpectedTestFilePathNotExists: "simple_test.go",
	})
}

func TestModelQuery(t *testing.T) {
	type testCase struct {
		Name string

		SetupMock func(mockedProvider *providertesting.MockQuery)

		QueryAttempts uint
		QueryTimeout  uint
		Request       string

		ExpectedResponse *provider.QueryResult
		ExpectedError    string

		ValidateLogs func(t *testing.T, logs string)
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			logOutput, logger := log.Buffer()
			defer func() {
				if t.Failed() {
					t.Log(logOutput.String())
				}
			}()

			mock := providertesting.NewMockQuery(t)
			if tc.SetupMock != nil {
				tc.SetupMock(mock)
			}
			llm := NewModel(mock, "some-model")
			llm.SetAPIRequestAttempts(tc.QueryAttempts)
			llm.SetAPIRequestTimeout(tc.QueryTimeout)

			queryResult, actualError := llm.query(logger, tc.Request)

			if tc.ExpectedError != "" {
				assert.ErrorContains(t, actualError, tc.ExpectedError)
				assert.Nil(t, queryResult)
			} else {
				assert.NoError(t, actualError)

				queryResult.Duration = 0

				assert.Equal(t, tc.ExpectedResponse, queryResult)
			}

			if tc.ValidateLogs != nil {
				tc.ValidateLogs(t, logOutput.String())
			}
		})
	}

	reLogID := regexp.MustCompile(`query-id=([a-z0-9-]*)`)
	parseLogIDs := func(logs string) (ids []string) {
		for _, match := range reLogID.FindAllStringSubmatch(logs, -1) {
			ids = append(ids, match[1])
		}

		return ids
	}
	assertAllIDsMatch := func(t *testing.T, logs string) {
		ids := parseLogIDs(logs)
		assert.Len(t,
			slices.CompactFunc(ids, func(e1 string, e2 string) bool {
				return e1 == e2
			}),
			1,
		)
	}

	validate(t, &testCase{
		Name: "Successful",
		SetupMock: func(mockedProvider *providertesting.MockQuery) {
			queryResult := &provider.QueryResult{
				Message: "test response",
			}
			mockedProvider.On("Query", mock.Anything, mock.Anything, mock.Anything, "test request").Return(queryResult, nil)
		},
		QueryAttempts: 1,
		Request:       "test request",
		ExpectedResponse: &provider.QueryResult{
			Message: "test response",
		},

		ValidateLogs: assertAllIDsMatch,
	})

	validate(t, &testCase{
		Name: "Failed query no retry",
		SetupMock: func(mockedProvider *providertesting.MockQuery) {
			mockedProvider.On("Query", mock.Anything, mock.Anything, mock.Anything, "test request").Return(nil, assert.AnError)
		},
		QueryAttempts: 1,
		Request:       "test request",
		ExpectedError: assert.AnError.Error(),
	})

	validate(t, &testCase{
		Name: "Failed query with retry",
		SetupMock: func(mockedProvider *providertesting.MockQuery) {
			mockedProvider.On("Query", mock.Anything, mock.Anything, mock.Anything, "test request").Return(nil, assert.AnError).Once()
			mockedProvider.On("Query", mock.Anything, mock.Anything, mock.Anything, "test request").Return(&provider.QueryResult{
				Message: "test response",
			}, nil).Once()
		},
		QueryAttempts: 1 + 1,
		Request:       "test request",
		ExpectedResponse: &provider.QueryResult{
			Message: "test response",
		},

		ValidateLogs: assertAllIDsMatch,
	})

	validate(t, &testCase{
		Name: "Timeout",
		SetupMock: func(mockedProvider *providertesting.MockQuery) {
			queryResult := &provider.QueryResult{
				Message: "test response",
			}
			mockedProvider.On("Query", mock.Anything, mock.Anything, mock.Anything, "test request").Return(queryResult, nil).After(time.Second * 2)
		},
		QueryAttempts: 1,
		QueryTimeout:  1,
		Request:       "test request",
		ExpectedError: "API request timed out",
	})

	validate(t, &testCase{
		Name: "Multiple Timeouts",
		SetupMock: func(mockedProvider *providertesting.MockQuery) {
			queryResult := &provider.QueryResult{
				Message: "test response",
			}
			mockedProvider.On("Query", mock.Anything, mock.Anything, mock.Anything, "test request").Return(queryResult, nil).After(time.Second * 2)
			mockedProvider.On("Query", mock.Anything, mock.Anything, mock.Anything, "test request").Return(queryResult, nil).After(time.Second * 2)
		},
		QueryAttempts: 2,
		QueryTimeout:  1,
		Request:       "test request",
		ExpectedError: "API request timed out",

		ValidateLogs: func(t *testing.T, logs string) {
			assert.Equal(t, 2, strings.Count(logs, "querying model"))
		},
	})
}

func TestModelRepairSourceCodeFile(t *testing.T) {
	type testCase struct {
		Name string

		SetupMock func(t *testing.T, mockedProvider *providertesting.MockQuery)

		Language       language.Language
		RepositoryPath string
		SourceFilePath string

		Mistakes []string

		ExpectedAssessment        metrics.Assessments
		ExpectedSourceFileContent string
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			logOutput, logger := log.Buffer()
			defer func() {
				if t.Failed() {
					t.Log(logOutput.String())
				}
			}()

			temporaryPath := t.TempDir()
			repositoryPath := filepath.Join(temporaryPath, filepath.Base(tc.RepositoryPath))
			require.NoError(t, osutil.CopyTree(tc.RepositoryPath, repositoryPath))

			modelID := "some-model"
			mock := providertesting.NewMockQuery(t)
			tc.SetupMock(t, mock)
			llm := NewModel(mock, modelID)

			ctx := model.Context{
				Language: tc.Language,

				RepositoryPath: repositoryPath,
				FilePath:       tc.SourceFilePath,

				Arguments: &evaluatetask.ArgumentsCodeRepair{
					Mistakes: tc.Mistakes,
				},

				Logger: logger,
			}
			actualAssessment, actualError := llm.RepairCode(ctx)
			assert.NoError(t, actualError)

			assert.Equal(t, metricstesting.Clean(tc.ExpectedAssessment), metricstesting.Clean(actualAssessment))

			actualSourceFileContent, err := os.ReadFile(filepath.Join(repositoryPath, tc.SourceFilePath))
			assert.NoError(t, err)

			assert.Equal(t, strings.TrimSpace(bytesutil.StringTrimIndentations(tc.ExpectedSourceFileContent)), string(actualSourceFileContent))
		})
	}
	t.Run("Go", func(t *testing.T) {
		validate(t, &testCase{
			Name: "Opening bracket is missing",

			SetupMock: func(t *testing.T, mockedProvider *providertesting.MockQuery) {
				queryResult := &provider.QueryResult{
					Message: bytesutil.StringTrimIndentations(`
						` + "```" + `
						package openingBracketMissing
						func openingBracketMissing(x int) int {
							if x > 0 {
								return 1
							}
							if x < 0 {
								return -1
							}
							return 0
						}
						` + "```" + `
					`),
				}
				mockedProvider.On("Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(queryResult, nil)
			},

			Language:       &golang.Language{},
			RepositoryPath: filepath.Join("..", "..", "testdata", "golang", "mistakes", "openingBracketMissing"),
			SourceFilePath: "openingBracketMissing.go",

			Mistakes: []string{
				"./openingBracketMissing.go:4:2: syntax error: non-declaration statement outside function body",
			},

			ExpectedAssessment: metrics.Assessments{
				metrics.AssessmentKeyResponseNoExcess:                   1,
				metrics.AssessmentKeyResponseWithCode:                   1,
				metrics.AssessmentKeyGenerateTestsForFileCharacterCount: 134,
				metrics.AssessmentKeyResponseCharacterCount:             143,
			},
			ExpectedSourceFileContent: `
				package openingBracketMissing
				func openingBracketMissing(x int) int {
					if x > 0 {
						return 1
					}
					if x < 0 {
						return -1
					}
					return 0
				}
			`,
		})
	})
	t.Run("Java", func(t *testing.T) {
		validate(t, &testCase{
			Name: "Opening bracket is missing",

			SetupMock: func(t *testing.T, mockedProvider *providertesting.MockQuery) {
				queryResult := &provider.QueryResult{
					Message: bytesutil.StringTrimIndentations(`
						` + "```" + `
						package com.eval;
						public class OpeningBracketMissing {
							public static int openingBracketMissing(int x) {
								if (x > 0) {
									return 1;
								}
								if (x < 0) {
									return -1;
								}
								return 0;
							}
						}
						` + "```" + `
					`),
				}
				mockedProvider.On("Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(queryResult, nil)
			},

			Language:       &java.Language{},
			RepositoryPath: filepath.Join("..", "..", "testdata", "java", "mistakes", "openingBracketMissing"),
			SourceFilePath: filepath.Join("src", "main", "java", "com", "eval", "OpeningBracketMissing.java"),

			Mistakes: []string{
				"/src/main/java/com/eval/OpeningBracketMissing.java:[12,17] illegal start of type",
				"/src/main/java/com/eval/OpeningBracketMissing.java:[14,1] class, interface, or enum expected",
			},

			ExpectedAssessment: metrics.Assessments{
				metrics.AssessmentKeyResponseNoExcess:                   1,
				metrics.AssessmentKeyResponseWithCode:                   1,
				metrics.AssessmentKeyGenerateTestsForFileCharacterCount: 186,
				metrics.AssessmentKeyResponseCharacterCount:             195,
			},
			ExpectedSourceFileContent: `
				package com.eval;
				public class OpeningBracketMissing {
					public static int openingBracketMissing(int x) {
						if (x > 0) {
							return 1;
						}
						if (x < 0) {
							return -1;
						}
						return 0;
					}
				}
			`,
		})
	})
}

type promptContext interface {
	Format() (string, error)
}

func TestFormatPromptContext(t *testing.T) {
	type testCase struct {
		Name string

		Context promptContext

		ExpectedMessage string
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			actualMessage, actualErr := tc.Context.Format()
			require.NoError(t, actualErr)

			assert.Equal(t, tc.ExpectedMessage, actualMessage)
		})
	}

	t.Run("Write Test", func(t *testing.T) {
		validate(t, &testCase{
			Name: "No Template",

			Context: &llmWriteTestSourceFilePromptContext{
				llmSourceFilePromptContext: llmSourceFilePromptContext{
					Language: &golang.Language{},

					Code: bytesutil.StringTrimIndentations(`
						package increment

						func increment(i int) int
							return i + 1
						}
					`),
					FilePath:   filepath.Join("path", "to", "increment.go"),
					ImportPath: "increment",
				},
			},

			ExpectedMessage: bytesutil.StringTrimIndentations(`
				Given the following Go code file "path/to/increment.go" with package "increment", provide a test file for this code.
				The tests should produce 100 percent code coverage and must compile.
				The response must contain only the test code in a fenced code block and nothing else.

				` + "```" + `golang
				package increment

				func increment(i int) int
					return i + 1
				}
				` + "```" + `
			`),
		})

		validate(t, &testCase{
			Name: "With Template",

			Context: &llmWriteTestSourceFilePromptContext{
				llmSourceFilePromptContext: llmSourceFilePromptContext{
					Language: &golang.Language{},

					Code: bytesutil.StringTrimIndentations(`
						package increment

						func increment(i int) int
							return i + 1
						}
					`),
					FilePath:   filepath.Join("path", "to", "increment.go"),
					ImportPath: "increment",
				},

				Template: bytesutil.StringTrimIndentations(`
					package increment

					import (
						"testing"

						"github.com/stretchr/testify/assert"
					)

					func TestIncrement(t *testing.T) {
						type testCase struct {
							Name string

							I int

							Expected int
						}

						validate := func(t *testing.T, tc *testCase) {
							t.Run(tc.Name, func(t *testing.T){
								assert.Equal(t, tc.Expected, increment(tc.I))
							})
						}
					}
				`),
			},

			ExpectedMessage: bytesutil.StringTrimIndentations(`
				Given the following Go code file "path/to/increment.go" with package "increment", provide a test file for this code.
				The tests should produce 100 percent code coverage and must compile.
				The response must contain only the test code in a fenced code block and nothing else.

				` + "```" + `golang
				package increment

				func increment(i int) int
					return i + 1
				}
				` + "```" + `

				The tests should be based on this template:

				` + "```" + `golang
				package increment

				import (
					"testing"

					"github.com/stretchr/testify/assert"
				)

				func TestIncrement(t *testing.T) {
					type testCase struct {
						Name string

						I int

						Expected int
					}

					validate := func(t *testing.T, tc *testCase) {
						t.Run(tc.Name, func(t *testing.T){
							assert.Equal(t, tc.Expected, increment(tc.I))
						})
					}
				}
				` + "```" + `
			`),
		})

		validate(t, &testCase{
			Name: "Custom test framework",

			Context: &llmWriteTestSourceFilePromptContext{
				llmSourceFilePromptContext: llmSourceFilePromptContext{
					Language: &java.Language{},

					Code: bytesutil.StringTrimIndentations(`
						${code}
					`),
					FilePath:   "${path}",
					ImportPath: "${pkg}",
				},

				TestFramework: "JUnit 5 for Spring",
			},

			ExpectedMessage: bytesutil.StringTrimIndentations(`
				Given the following Java code file "${path}" with package "${pkg}", provide a test file for this code with JUnit 5 for Spring as a test framework.
				The tests should produce 100 percent code coverage and must compile.
				The response must contain only the test code in a fenced code block and nothing else.

				` + "```" + `java
				${code}
				` + "```" + `
			`),
		})

		validate(t, &testCase{
			Name: "No Import path",

			Context: &llmWriteTestSourceFilePromptContext{
				llmSourceFilePromptContext: llmSourceFilePromptContext{
					Language: &golang.Language{},

					Code: bytesutil.StringTrimIndentations(`
						package increment

						func increment(i int) int
							return i + 1
						}
					`),
					FilePath:   filepath.Join("path", "to", "increment.go"),
					ImportPath: "",
				},
			},

			ExpectedMessage: bytesutil.StringTrimIndentations(`
				Given the following Go code file "path/to/increment.go", provide a test file for this code.
				The tests should produce 100 percent code coverage and must compile.
				The response must contain only the test code in a fenced code block and nothing else.

				` + "```" + `golang
				package increment

				func increment(i int) int
					return i + 1
				}
				` + "```" + `
			`),
		})

		validate(t, &testCase{
			Name: "Tests in source file",

			Context: &llmWriteTestSourceFilePromptContext{
				llmSourceFilePromptContext: llmSourceFilePromptContext{
					Language: &rust.Language{},

					Code: bytesutil.StringTrimIndentations(`
						fn main() {
						}
					`),
					FilePath:         filepath.Join("path", "to", "main.rs"),
					ImportPath:       "",
					HasTestsInSource: true,
				},
			},

			ExpectedMessage: bytesutil.StringTrimIndentations(`
				Given the following Rust code file "path/to/main.rs", provide tests for this code.
				The tests should produce 100 percent code coverage and must compile.
				The response must contain only the test code in a fenced code block and nothing else.

				` + "```" + `rust
				fn main() {
				}
				` + "```" + `
			`),
		})
	})

	t.Run("Code Repair", func(t *testing.T) {
		validate(t, &testCase{
			Name: "Default",

			Context: &llmCodeRepairSourceFilePromptContext{
				llmSourceFilePromptContext: llmSourceFilePromptContext{
					Language: &golang.Language{},

					Code: bytesutil.StringTrimIndentations(`
					package increment

					func increment(i int) int
						return i + 1
					}
				`),
					FilePath:   filepath.Join("path", "to", "increment.go"),
					ImportPath: "increment",
				},
				Mistakes: []string{
					"path/to/increment.go:3:1: expected 'IDENT', found 'func'",
					"path/to/increment.go: syntax error: non-declaration statement outside function body",
					"path/to/increment.go: missing return",
				},
			},

			ExpectedMessage: bytesutil.StringTrimIndentations(`
				Given the following Go code file "path/to/increment.go" with package "increment" and a list of compilation errors, modify the code such that the errors are resolved.
				The response must contain only the source code in a fenced code block and nothing else.

				` + "```" + `golang
				package increment

				func increment(i int) int
					return i + 1
				}
				` + "```" + `

				The list of compilation errors is the following:
				- path/to/increment.go:3:1: expected 'IDENT', found 'func'
				- path/to/increment.go: syntax error: non-declaration statement outside function body
				- path/to/increment.go: missing return
			`),
		})

		validate(t, &testCase{
			Name: "No package",

			Context: &llmCodeRepairSourceFilePromptContext{
				llmSourceFilePromptContext: llmSourceFilePromptContext{
					Language: &golang.Language{},

					Code: bytesutil.StringTrimIndentations(`
					package increment

					func increment(i int) int
						return i + 1
					}
				`),
					FilePath: filepath.Join("path", "to", "increment.go"),
				},
				Mistakes: []string{
					"path/to/increment.go:3:1: expected 'IDENT', found 'func'",
					"path/to/increment.go: syntax error: non-declaration statement outside function body",
					"path/to/increment.go: missing return",
				},
			},

			ExpectedMessage: bytesutil.StringTrimIndentations(`
				Given the following Go code file "path/to/increment.go" and a list of compilation errors, modify the code such that the errors are resolved.
				The response must contain only the source code in a fenced code block and nothing else.

				` + "```" + `golang
				package increment

				func increment(i int) int
					return i + 1
				}
				` + "```" + `

				The list of compilation errors is the following:
				- path/to/increment.go:3:1: expected 'IDENT', found 'func'
				- path/to/increment.go: syntax error: non-declaration statement outside function body
				- path/to/increment.go: missing return
			`),
		})
	})

	validate(t, &testCase{
		Name: "Transpile",

		Context: &llmTranspileSourceFilePromptContext{
			llmSourceFilePromptContext: llmSourceFilePromptContext{
				Language: &java.Language{},

				Code: bytesutil.StringTrimIndentations(`
					package com.eval;

					class Foobar {
						static int foobar(int i) {}
					}
				`),
				FilePath:   "Foobar.java",
				ImportPath: "com.eval",
			},
			OriginLanguage: &golang.Language{},
			OriginFileContent: bytesutil.StringTrimIndentations(`
				package foobar

				func foobar(i int) int {
					return i + 1
				}
			`),
		},

		ExpectedMessage: bytesutil.StringTrimIndentations(`
			Given the following Go code file, transpile it into a Java code file.
			The response must contain only the transpiled Java source code in a fenced code block and nothing else.

			` + "```" + `golang
			package foobar

			func foobar(i int) int {
				return i + 1
			}
			` + "```" + `

			The transpiled Java code file must have the package "com.eval" and the following signature:

			` + "```" + `java
			package com.eval;

			class Foobar {
				static int foobar(int i) {}
			}
			` + "```" + `
		`),
	})
	validate(t, &testCase{
		Name: "Migrate",

		Context: &llmMigrateSourceFilePromptContext{
			llmSourceFilePromptContext: llmSourceFilePromptContext{
				Language: &java.Language{},

				Code: bytesutil.StringTrimIndentations(`
					package com.eval;

					class Foobar {
						static int foobar(int i) {}
					}
				`),
				FilePath:   "Foobar.java",
				ImportPath: "com.eval",
			},
			TestFramework: "JUnit 5",
		},

		ExpectedMessage: bytesutil.StringTrimIndentations(`
			Given the following Java test file "Foobar.java" with package "com.eval", migrate the test file to JUnit 5 as the test framework.
			The tests should produce 100 percent code coverage and must compile.
			The response must contain only the test code in a fenced code block and nothing else.

			` + "```" + `java
			package com.eval;

			class Foobar {
				static int foobar(int i) {}
			}
			` + "```" + `
		`),
	})
}

func TestModelTranspile(t *testing.T) {
	type testCase struct {
		Name string

		SetupMock func(t *testing.T, mockedProvider *providertesting.MockQuery)

		Language       language.Language
		OriginLanguage language.Language

		RepositoryPath string
		OriginFilePath string
		StubFilePath   string

		ExpectedAssessment      metrics.Assessments
		ExpectedStubFileContent string
	}

	validate := func(t *testing.T, tc *testCase) {
		logOutput, logger := log.Buffer()
		defer func() {
			if t.Failed() {
				t.Log(logOutput.String())
			}
		}()

		temporaryPath := t.TempDir()
		repositoryPath := filepath.Join(temporaryPath, filepath.Base(tc.RepositoryPath))
		require.NoError(t, osutil.CopyTree(tc.RepositoryPath, repositoryPath))

		modelID := "some-model"
		mock := providertesting.NewMockQuery(t)
		tc.SetupMock(t, mock)
		llm := NewModel(mock, modelID)

		ctx := model.Context{
			Language: tc.Language,

			RepositoryPath: repositoryPath,
			FilePath:       tc.StubFilePath,

			Arguments: &evaluatetask.ArgumentsTranspile{
				OriginLanguage: tc.OriginLanguage,
				OriginFilePath: tc.OriginFilePath,
			},

			Logger: logger,
		}

		actualAssessment, actualError := llm.Transpile(ctx)
		assert.NoError(t, actualError)

		assert.Equal(t, metricstesting.Clean(tc.ExpectedAssessment), metricstesting.Clean(actualAssessment))

		actualStubFileContent, err := os.ReadFile(filepath.Join(repositoryPath, tc.StubFilePath))
		assert.NoError(t, err)

		assert.Equal(t, strings.TrimSpace(tc.ExpectedStubFileContent), string(actualStubFileContent))
	}

	t.Run("Transpile Java into Go", func(t *testing.T) {
		transpiledFileContent := bytesutil.StringTrimIndentations(`
			package binarySearch

			func binarySearch(a []int, x int) int {
				index := -1

				min := 0
				max := len(a) - 1

				for index == -1 && min <= max {
					m := (min + max) / 2

					if x == a[m] {
						index = m
					} else if x < a[m] {
						max = m - 1
					} else {
						min = m + 1
					}
				}

				return index
			}
		`)
		validate(t, &testCase{
			Name: "Binary search",

			SetupMock: func(t *testing.T, mockedProvider *providertesting.MockQuery) {
				queryResult := &provider.QueryResult{
					Message: "```\n" + transpiledFileContent + "```\n",
				}
				mockedProvider.On("Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(queryResult, nil)
			},

			Language:       &golang.Language{},
			OriginLanguage: &java.Language{},

			RepositoryPath: filepath.Join("..", "..", "testdata", "golang", "transpile", "binarySearch"),
			OriginFilePath: filepath.Join("implementation", "BinarySearch.java"),
			StubFilePath:   filepath.Join("binarySearch.go"),

			ExpectedAssessment: metrics.Assessments{
				metrics.AssessmentKeyResponseNoExcess:                   1,
				metrics.AssessmentKeyResponseWithCode:                   1,
				metrics.AssessmentKeyGenerateTestsForFileCharacterCount: 280,
				metrics.AssessmentKeyResponseCharacterCount:             289,
			},
			ExpectedStubFileContent: transpiledFileContent,
		})
	})
	t.Run("Transpile Go into Java", func(t *testing.T) {
		transpiledFileContent := bytesutil.StringTrimIndentations(`
			package com.eval;

			class BinarySearch {
				static int binarySearch(int[] a, int x) {
					int index = -1;

					int min = 0;
					int max = a.length - 1;

					while (index == -1 && min <= max) {
						int m = (min + max) / 2;

						if (x == a[m]) {
							index = m;
						} else if (x < a[m]) {
							max = m - 1;
						} else {
							min = m + 1;
						}
					}

					return index;
				}
			}
		`)
		validate(t, &testCase{
			Name: "Binary search",

			SetupMock: func(t *testing.T, mockedProvider *providertesting.MockQuery) {
				queryResult := &provider.QueryResult{
					Message: "```\n" + transpiledFileContent + "```\n",
				}
				mockedProvider.On("Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(queryResult, nil)
			},

			Language:       &java.Language{},
			OriginLanguage: &golang.Language{},

			RepositoryPath: filepath.Join("..", "..", "testdata", "java", "transpile", "binarySearch"),
			OriginFilePath: filepath.Join("implementation", "binarySearch.go"),
			StubFilePath:   filepath.Join("src", "main", "java", "com", "eval", "BinarySearch.java"),

			ExpectedAssessment: metrics.Assessments{
				metrics.AssessmentKeyResponseNoExcess:                   1,
				metrics.AssessmentKeyResponseWithCode:                   1,
				metrics.AssessmentKeyGenerateTestsForFileCharacterCount: 348,
				metrics.AssessmentKeyResponseCharacterCount:             357,
			},
			ExpectedStubFileContent: transpiledFileContent,
		})
	})
}

func TestModelMigrate(t *testing.T) {
	type testCase struct {
		Name string

		SetupMock func(t *testing.T, mockedProvider *providertesting.MockQuery)

		Language language.Language

		RepositoryPath string
		TestFilePath   string
		TestFramework  string

		ExpectedAssessment          metrics.Assessments
		ExpectedMigratedFileContent string
	}

	validate := func(t *testing.T, tc *testCase) {
		logOutput, logger := log.Buffer()
		defer func() {
			if t.Failed() {
				t.Log(logOutput.String())
			}
		}()

		temporaryPath := t.TempDir()
		repositoryPath := filepath.Join(temporaryPath, filepath.Base(tc.RepositoryPath))
		require.NoError(t, osutil.CopyTree(tc.RepositoryPath, repositoryPath))

		modelID := "some-model"
		mock := providertesting.NewMockQuery(t)
		tc.SetupMock(t, mock)
		llm := NewModel(mock, modelID)

		ctx := model.Context{
			Language: tc.Language,

			RepositoryPath: repositoryPath,
			FilePath:       tc.TestFilePath,

			Arguments: &evaluatetask.ArgumentsMigrate{
				TestFramework: tc.TestFramework,
			},

			Logger: logger,
		}

		actualAssessment, actualError := llm.Migrate(ctx)
		assert.NoError(t, actualError)

		assert.Equal(t, metricstesting.Clean(tc.ExpectedAssessment), metricstesting.Clean(actualAssessment))

		actualMigratedFileContent, err := os.ReadFile(filepath.Join(repositoryPath, tc.TestFilePath))
		assert.NoError(t, err)

		assert.Equal(t, strings.TrimSpace(tc.ExpectedMigratedFileContent), string(actualMigratedFileContent))
	}

	migratedTestFile := bytesutil.StringTrimIndentations(`
		package com.eval;

		import org.junit.jupiter.api.Test;
		import static org.junit.jupiter.api.Assertions.assertEquals;

		public class IncrementTest {
			@Test
			public void increment() {
				int i = 1;
				int expected = 2;
				int actual = Increment.increment(i);

				assertEquals(expected, actual);
			}
		}
	`)

	validate(t, &testCase{
		Name: "Increment",

		SetupMock: func(t *testing.T, mockedProvider *providertesting.MockQuery) {
			queryResult := &provider.QueryResult{
				Message: "```\n" + migratedTestFile + "```\n",
			}
			mockedProvider.On("Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(queryResult, nil)
		},

		Language: &java.Language{},

		RepositoryPath: filepath.Join("..", "..", "testdata", "java", "migrate-plain"),
		TestFilePath:   filepath.Join("src", "test", "java", "com", "eval", "IncrementTest.java"),

		ExpectedAssessment: metrics.Assessments{
			metrics.AssessmentKeyResponseNoExcess:                   1,
			metrics.AssessmentKeyResponseWithCode:                   1,
			metrics.AssessmentKeyGenerateTestsForFileCharacterCount: 290,
			metrics.AssessmentKeyResponseCharacterCount:             299,
		},
		ExpectedMigratedFileContent: migratedTestFile,
	})
}
