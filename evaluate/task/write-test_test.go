package task

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	metricstesting "github.com/symflower/eval-dev-quality/evaluate/metrics/testing"
	tasktesting "github.com/symflower/eval-dev-quality/evaluate/task/testing"
	"github.com/symflower/eval-dev-quality/language"
	"github.com/symflower/eval-dev-quality/language/golang"
	"github.com/symflower/eval-dev-quality/language/java"
	"github.com/symflower/eval-dev-quality/language/ruby"
	"github.com/symflower/eval-dev-quality/log"
	"github.com/symflower/eval-dev-quality/model"
	modeltesting "github.com/symflower/eval-dev-quality/model/testing"
	evaltask "github.com/symflower/eval-dev-quality/task"
	"github.com/zimmski/osutil"
	"github.com/zimmski/osutil/bytesutil"
)

func TestWriteTestsRun(t *testing.T) {
	validate := func(t *testing.T, tc *tasktesting.TestCaseTask) {
		t.Run(tc.Name, func(t *testing.T) {
			task, err := ForIdentifier(IdentifierWriteTests)
			require.NoError(t, err)
			tc.Task = task

			tc.Validate(t,
				func(logger *log.Logger, testDataPath string, repositoryPathRelative string) (repository evaltask.Repository, cleanup func(), err error) {
					return TemporaryRepository(logger, testDataPath, repositoryPathRelative)
				},
			)
		})
	}

	t.Run("Clear repository on each task file", func(t *testing.T) { // This test simulates failing tests for the first of two task cases and ensures that they don't influence test execution for the second one.
		temporaryDirectoryPath := t.TempDir()

		repositoryPath := filepath.Join(temporaryDirectoryPath, "golang", "plain")
		require.NoError(t, os.MkdirAll(repositoryPath, 0700))
		require.NoError(t, os.WriteFile(filepath.Join(repositoryPath, "go.mod"), []byte("module plain\n\ngo 1.21.5"), 0600))
		require.NoError(t, os.WriteFile(filepath.Join(repositoryPath, "caseA.go"), []byte("package plain\n\nfunc caseA(){}"), 0600))
		require.NoError(t, os.WriteFile(filepath.Join(repositoryPath, "caseB.go"), []byte("package plain\n\nfunc caseB(){}"), 0600))

		modelMock := modeltesting.NewMockCapabilityWriteTestsNamed(t, "mocked-model")

		// Generate invalid code for caseA (with and without template).
		modelMock.RegisterGenerateSuccess(t, "caseA_test.go", "does not compile", metricstesting.AssessmentsWithProcessingTime).Twice()
		// Generate valid code for caseB (with and without template).
		modelMock.RegisterGenerateSuccess(t, "caseB_test.go", "package plain\n\nimport \"testing\"\n\nfunc TestCaseB(t *testing.T){}", metricstesting.AssessmentsWithProcessingTime).Twice()

		validate(t, &tasktesting.TestCaseTask{
			Name: "Plain",

			Model:          modelMock,
			Language:       &golang.Language{},
			TestDataPath:   temporaryDirectoryPath,
			RepositoryPath: filepath.Join("golang", "plain"),

			ExpectedRepositoryAssessment: map[evaltask.Identifier]metrics.Assessments{
				IdentifierWriteTests: metrics.Assessments{
					metrics.AssessmentKeyFilesExecuted:                 1,
					metrics.AssessmentKeyFilesExecutedMaximumReachable: 2,
					metrics.AssessmentKeyResponseNoError:               2,
				},
				IdentifierWriteTestsSymflowerFix: metrics.Assessments{
					metrics.AssessmentKeyFilesExecuted:                 1,
					metrics.AssessmentKeyFilesExecutedMaximumReachable: 2,
					metrics.AssessmentKeyResponseNoError:               2,
				},
				// With the template there would be coverage but it is overwritten.
				IdentifierWriteTestsSymflowerTemplate: metrics.Assessments{
					metrics.AssessmentKeyFilesExecuted:                 1,
					metrics.AssessmentKeyFilesExecutedMaximumReachable: 2,
					metrics.AssessmentKeyResponseNoError:               2,
				},
				IdentifierWriteTestsSymflowerTemplateSymflowerFix: metrics.Assessments{
					metrics.AssessmentKeyFilesExecuted:                 1,
					metrics.AssessmentKeyFilesExecutedMaximumReachable: 2,
					metrics.AssessmentKeyResponseNoError:               2,
				},
			},
			ExpectedProblemContains: []string{
				"expected 'package', found does", // Model error.
				"exit status 1",                  // Symflower fix not applicable.
				"expected 'package', found does", // Model error (overwrote template).
				"exit status 1",                  // Symflower fix not applicable (overwrote template).
			},
			ValidateLog: func(t *testing.T, data string) {
				assert.Equal(t, 1, strings.Count(data, "Evaluating model \"mocked-model\""))
				assert.Equal(t, 4, strings.Count(data, "PASS: TestCaseB")) // Bare model result, with fix, with template, with template and fix.
			},
		})
	})

	t.Run("Symflower Fix", func(t *testing.T) {
		t.Run("Go", func(t *testing.T) {
			validateGo := func(t *testing.T, testName string, language language.Language, testFileContent string, expectedAssessments map[evaltask.Identifier]metrics.Assessments, expectedProblems []string, assertTestsPass bool) {
				temporaryDirectoryPath := t.TempDir()
				repositoryPath := filepath.Join(temporaryDirectoryPath, "golang", "plain")
				require.NoError(t, osutil.CopyTree(filepath.Join("..", "..", "testdata", "golang", "plain"), repositoryPath))

				modelMock := modeltesting.NewMockCapabilityWriteTestsNamed(t, "mocked-model")
				modelMock.RegisterGenerateSuccess(t, "plain_test.go", testFileContent, metricstesting.AssessmentsWithProcessingTime).Twice()

				validate(t, &tasktesting.TestCaseTask{
					Name: testName,

					Model:          modelMock,
					Language:       language,
					TestDataPath:   temporaryDirectoryPath,
					RepositoryPath: filepath.Join("golang", "plain"),

					ExpectedRepositoryAssessment: expectedAssessments,
					ExpectedProblemContains:      expectedProblems,
					ValidateLog: func(t *testing.T, data string) {
						assert.Contains(t, data, "Evaluating model \"mocked-model\"")
						if assertTestsPass {
							assert.Contains(t, data, "PASS: TestPlain")
						}
					},
				})
			}
			{
				expectedAssessments := map[evaltask.Identifier]metrics.Assessments{
					IdentifierWriteTests: metrics.Assessments{
						metrics.AssessmentKeyFilesExecuted:                 1,
						metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						metrics.AssessmentKeyResponseNoError:               1,
						metrics.AssessmentKeyCoverage:                      10,
					},
					IdentifierWriteTestsSymflowerFix: metrics.Assessments{
						metrics.AssessmentKeyFilesExecuted:                 1,
						metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						metrics.AssessmentKeyResponseNoError:               1,
						metrics.AssessmentKeyCoverage:                      10,
					},
					IdentifierWriteTestsSymflowerTemplate: metrics.Assessments{
						metrics.AssessmentKeyFilesExecuted:                 1,
						metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						metrics.AssessmentKeyResponseNoError:               1,
						metrics.AssessmentKeyCoverage:                      10,
					},
					IdentifierWriteTestsSymflowerTemplateSymflowerFix: metrics.Assessments{
						metrics.AssessmentKeyFilesExecuted:                 1,
						metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						metrics.AssessmentKeyResponseNoError:               1,
						metrics.AssessmentKeyCoverage:                      10,
					},
				}
				validateGo(t, "Model generated correct test", &golang.Language{}, bytesutil.StringTrimIndentations(`
					package plain

					import "testing"

					func TestPlain(t *testing.T) {
						   plain()
					}
				`), expectedAssessments, nil, true)
			}
			{
				expectedAssessments := map[evaltask.Identifier]metrics.Assessments{
					IdentifierWriteTests: metrics.Assessments{
						metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						metrics.AssessmentKeyResponseNoError:               1,
					},
					IdentifierWriteTestsSymflowerFix: metrics.Assessments{
						metrics.AssessmentKeyFilesExecuted:                 1,
						metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						metrics.AssessmentKeyResponseNoError:               1,
						metrics.AssessmentKeyCoverage:                      10,
					},
					IdentifierWriteTestsSymflowerTemplate: metrics.Assessments{
						metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						metrics.AssessmentKeyResponseNoError:               1,
					},
					IdentifierWriteTestsSymflowerTemplateSymflowerFix: metrics.Assessments{
						metrics.AssessmentKeyFilesExecuted:                 1,
						metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						metrics.AssessmentKeyResponseNoError:               1,
						metrics.AssessmentKeyCoverage:                      10,
					},
				}
				expectedProblems := []string{
					"imported and not used",
					"imported and not used",
				}
				validateGo(t, "Model generated test with unused import", &golang.Language{}, bytesutil.StringTrimIndentations(`
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
				expectedAssessments := map[evaltask.Identifier]metrics.Assessments{
					IdentifierWriteTests: metrics.Assessments{
						metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						metrics.AssessmentKeyResponseNoError:               1,
					},
					IdentifierWriteTestsSymflowerFix: metrics.Assessments{
						metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						metrics.AssessmentKeyResponseNoError:               1,
					},
					IdentifierWriteTestsSymflowerTemplate: metrics.Assessments{
						metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						metrics.AssessmentKeyResponseNoError:               1,
					},
					IdentifierWriteTestsSymflowerTemplateSymflowerFix: metrics.Assessments{
						metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						metrics.AssessmentKeyResponseNoError:               1,
					},
				}
				expectedProblems := []string{
					"expected declaration, found this",
					"unable to format source code",
					"expected declaration, found this",
					"unable to format source code",
				}
				validateGo(t, "Model generated test that is unfixable", &golang.Language{}, bytesutil.StringTrimIndentations(`
					package plain

					this is not valid go code
				`), expectedAssessments, expectedProblems, false)
			}
		})
	})

	{
		if osutil.IsWindows() {
			t.Skip("Ruby is not tested in the Windows CI")
		}

		temporaryDirectoryPath := t.TempDir()
		repositoryPath := filepath.Join(temporaryDirectoryPath, "ruby", "plain")
		require.NoError(t, osutil.CopyTree(filepath.Join("..", "..", "testdata", "ruby", "plain"), repositoryPath))

		testFileContent := bytesutil.StringTrimIndentations(`
			require_relative "../lib/plain"

			class TestPlain < Minitest::Test
				def test_plain
					plain()
				end
			end
		`)
		modelMock := modeltesting.NewMockCapabilityWriteTestsNamed(t, "mocked-model")
		modelMock.RegisterGenerateSuccess(t, filepath.Join("test", "plain_test.rb"), testFileContent, metricstesting.AssessmentsWithProcessingTime).Maybe()

		validate(t, &tasktesting.TestCaseTask{
			Name: "Ruby",

			Model:          modelMock,
			Language:       &ruby.Language{},
			TestDataPath:   temporaryDirectoryPath,
			RepositoryPath: filepath.Join("ruby", "plain"),

			ExpectedRepositoryAssessment: map[evaltask.Identifier]metrics.Assessments{
				IdentifierWriteTests: metrics.Assessments{
					metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
					metrics.AssessmentKeyFilesExecuted:                 1,
					metrics.AssessmentKeyCoverage:                      10,
					metrics.AssessmentKeyResponseNoError:               1,
				},
				IdentifierWriteTestsSymflowerFix: metrics.Assessments{
					metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
					metrics.AssessmentKeyFilesExecuted:                 1,
					metrics.AssessmentKeyCoverage:                      10,
					metrics.AssessmentKeyResponseNoError:               1,
				},
				IdentifierWriteTestsSymflowerTemplate: metrics.Assessments{
					metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
					metrics.AssessmentKeyFilesExecuted:                 1,
					metrics.AssessmentKeyCoverage:                      10,
					metrics.AssessmentKeyResponseNoError:               1,
				},
				IdentifierWriteTestsSymflowerTemplateSymflowerFix: metrics.Assessments{
					metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
					metrics.AssessmentKeyFilesExecuted:                 1,
					metrics.AssessmentKeyCoverage:                      10,
					metrics.AssessmentKeyResponseNoError:               1,
				},
			},
			ExpectedProblemContains: nil,
			ValidateLog: func(t *testing.T, data string) {
				assert.Contains(t, data, "Evaluating model \"mocked-model\"")
			},
		})
	}

	{
		temporaryDirectoryPath := t.TempDir()
		repositoryPath := filepath.Join(temporaryDirectoryPath, "golang", "plain")
		require.NoError(t, osutil.CopyTree(filepath.Join("..", "..", "testdata", "golang", "plain"), repositoryPath))
		require.NoError(t, os.WriteFile(filepath.Join(temporaryDirectoryPath, "golang", "plain", "empty.go"), []byte(bytesutil.StringTrimIndentations(`
			package plain

			// There will be no template for an empty file.
		`)), 0666))
		modelMock := modeltesting.NewMockCapabilityWriteTestsNamed(t, "mocked-model")
		modelMock.RegisterGenerateSuccess(t, "empty_test.go", "package plain\n", metricstesting.AssessmentsWithProcessingTime).Once()
		modelMock.RegisterGenerateSuccess(t, "plain_test.go", bytesutil.StringTrimIndentations(`
			package plain

			import "testing"

			func TestPlain(t *testing.T) {
					plain()
			}
		`), metricstesting.AssessmentsWithProcessingTime)
		validate(t, &tasktesting.TestCaseTask{
			Name: "Keep non-template score if template fails",

			Model:          modelMock,
			Language:       &golang.Language{},
			TestDataPath:   temporaryDirectoryPath,
			RepositoryPath: filepath.Join("golang", "plain"),

			ExpectedRepositoryAssessment: map[evaltask.Identifier]metrics.Assessments{
				IdentifierWriteTests: metrics.Assessments{
					metrics.AssessmentKeyFilesExecutedMaximumReachable: 2,
					metrics.AssessmentKeyFilesExecuted:                 2,
					metrics.AssessmentKeyCoverage:                      10,
					metrics.AssessmentKeyResponseNoError:               2,
				},
				IdentifierWriteTestsSymflowerFix: metrics.Assessments{
					metrics.AssessmentKeyFilesExecutedMaximumReachable: 2,
					metrics.AssessmentKeyFilesExecuted:                 2,
					metrics.AssessmentKeyCoverage:                      10,
					metrics.AssessmentKeyResponseNoError:               2,
				},
				IdentifierWriteTestsSymflowerTemplate: metrics.Assessments{
					metrics.AssessmentKeyFilesExecutedMaximumReachable: 2,
					metrics.AssessmentKeyFilesExecuted:                 2,
					metrics.AssessmentKeyCoverage:                      10,
					metrics.AssessmentKeyResponseNoError:               2,
				},
				IdentifierWriteTestsSymflowerTemplateSymflowerFix: metrics.Assessments{
					metrics.AssessmentKeyFilesExecutedMaximumReachable: 2,
					metrics.AssessmentKeyFilesExecuted:                 2,
					metrics.AssessmentKeyCoverage:                      10,
					metrics.AssessmentKeyResponseNoError:               2,
				},
			},
			ExpectedProblemContains: []string{
				"reading Symflower template",
			},
		})
	}

	{
		temporaryDirectoryPath := t.TempDir()
		repositoryPath := filepath.Join(temporaryDirectoryPath, "golang", "plain")
		require.NoError(t, osutil.CopyTree(filepath.Join("..", "..", "testdata", "golang", "plain"), repositoryPath))
		require.NoError(t, os.WriteFile(filepath.Join(temporaryDirectoryPath, "golang", "plain", "repository.json"), []byte(bytesutil.StringTrimIndentations(`
			{
				"tasks": [
					"write-tests"
				],
				"ignore": [
					"plain.go"
				]
			}
		`)), 0666))
		modelMock := modeltesting.NewMockCapabilityWriteTestsNamed(t, "mocked-model")
		validate(t, &tasktesting.TestCaseTask{
			Name: "Ignore Case",

			Model:          modelMock,
			Language:       &golang.Language{},
			TestDataPath:   temporaryDirectoryPath,
			RepositoryPath: filepath.Join("golang", "plain"),

			ExpectedRepositoryAssessment: map[evaltask.Identifier]metrics.Assessments{
				IdentifierWriteTests:                              metrics.Assessments{},
				IdentifierWriteTestsSymflowerFix:                  metrics.Assessments{},
				IdentifierWriteTestsSymflowerTemplate:             metrics.Assessments{},
				IdentifierWriteTestsSymflowerTemplateSymflowerFix: metrics.Assessments{},
			},
			ValidateLog: func(t *testing.T, data string) {
				assert.Contains(t, data, "Ignoring file \"plain.go\" (as configured by the repository)")
			},
		})
	}

	t.Run("Spring Boot", func(t *testing.T) {
		{
			temporaryDirectoryPath := t.TempDir()
			repositoryPath := filepath.Join(temporaryDirectoryPath, "java", "spring-plain")
			require.NoError(t, osutil.CopyTree(filepath.Join("..", "..", "testdata", "java", "spring-plain"), repositoryPath))
			modelMock := modeltesting.NewMockCapabilityWriteTestsNamed(t, "mocked-model")
			modelMock.RegisterGenerateSuccessValidateContext(t, func(t *testing.T, c model.Context) {
				args, ok := c.Arguments.(*ArgumentsWriteTest)
				require.Truef(t, ok, "unexpected type %T", c.Arguments)
				assert.Equal(t, "JUnit 5 for Spring", args.TestFramework)
			}, filepath.Join("src", "test", "java", "com", "example", "controller", "SomeControllerTest.java"), bytesutil.StringTrimIndentations(`
			package com.example.controller;

			import org.junit.jupiter.api.*;
			import org.springframework.beans.factory.annotation.Autowired;
			import org.springframework.boot.test.autoconfigure.web.servlet.WebMvcTest;
			import org.springframework.test.web.servlet.MockMvc;
			import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
			import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.content;
			import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;
			import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.view;

			@WebMvcTest(SomeController.class)
			public class SomeControllerTest {
				@Autowired
				private MockMvc mockMvc;

				@Test
				public void helloGet() throws Exception {
					this.mockMvc.perform(get("/helloGet"))
						.andExpect(status().isOk())
						.andExpect(view().name("get!"))
						.andExpect(content().string(""));
				}
			}
		`), metricstesting.AssessmentsWithProcessingTime)

			validate(t, &tasktesting.TestCaseTask{
				Name: "Spring Boot Test",

				Model:          modelMock,
				Language:       &java.Language{},
				TestDataPath:   temporaryDirectoryPath,
				RepositoryPath: filepath.Join("java", "spring-plain"),

				ExpectedRepositoryAssessment: map[evaltask.Identifier]metrics.Assessments{
					IdentifierWriteTests: metrics.Assessments{
						metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						metrics.AssessmentKeyFilesExecuted:                 1,
						metrics.AssessmentKeyCoverage:                      20,
						metrics.AssessmentKeyResponseNoError:               1,
					},
					IdentifierWriteTestsSymflowerFix: metrics.Assessments{
						metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						metrics.AssessmentKeyFilesExecuted:                 1,
						metrics.AssessmentKeyCoverage:                      20,
						metrics.AssessmentKeyResponseNoError:               1,
					},
					IdentifierWriteTestsSymflowerTemplate: metrics.Assessments{
						metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						metrics.AssessmentKeyFilesExecuted:                 1,
						metrics.AssessmentKeyCoverage:                      20,
						metrics.AssessmentKeyResponseNoError:               1,
					},
					IdentifierWriteTestsSymflowerTemplateSymflowerFix: metrics.Assessments{
						metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						metrics.AssessmentKeyFilesExecuted:                 1,
						metrics.AssessmentKeyCoverage:                      20,
						metrics.AssessmentKeyResponseNoError:               1,
					},
				},
				ValidateLog: func(t *testing.T, data string) {
					assert.Equal(t, 2, strings.Count(data, "Starting SomeControllerTest using Java"), "Expected two successful Spring startup announcements (one bare and one for template)")
				},
			})
		}
		{
			temporaryDirectoryPath := t.TempDir()
			repositoryPath := filepath.Join(temporaryDirectoryPath, "java", "spring-plain")
			require.NoError(t, osutil.CopyTree(filepath.Join("..", "..", "testdata", "java", "spring-plain"), repositoryPath))
			modelMock := modeltesting.NewMockCapabilityWriteTestsNamed(t, "mocked-model")
			modelMock.RegisterGenerateSuccessValidateContext(t, func(t *testing.T, c model.Context) {
				args, ok := c.Arguments.(*ArgumentsWriteTest)
				require.Truef(t, ok, "unexpected type %T", c.Arguments)
				assert.Equal(t, "JUnit 5 for Spring", args.TestFramework)
			}, filepath.Join("src", "test", "java", "com", "example", "controller", "SomeControllerTest.java"), bytesutil.StringTrimIndentations(`
			package com.example.controller;

			import com.example.controller.SomeController;
			import org.junit.jupiter.api.Test;

			import static org.junit.jupiter.api.Assertions.assertEquals;

			class SomeControllerTests {

				@Test // Normal JUnit tests instead of Spring Boot.
				void helloGet() {
					SomeController controller = new SomeController();
					String result = controller.helloGet();
					assertEquals("get!", result);
				}
			}
		`), metricstesting.AssessmentsWithProcessingTime)

			validate(t, &tasktesting.TestCaseTask{
				Name: "Plain JUnit Test",

				Model:          modelMock,
				Language:       &java.Language{},
				TestDataPath:   temporaryDirectoryPath,
				RepositoryPath: filepath.Join("java", "spring-plain"),

				ExpectedRepositoryAssessment: map[evaltask.Identifier]metrics.Assessments{
					IdentifierWriteTests: metrics.Assessments{
						metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						metrics.AssessmentKeyResponseNoError:               1,
					},
					IdentifierWriteTestsSymflowerFix: metrics.Assessments{
						metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						metrics.AssessmentKeyResponseNoError:               1,
					},
					IdentifierWriteTestsSymflowerTemplate: metrics.Assessments{
						metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						metrics.AssessmentKeyResponseNoError:               1,
					},
					IdentifierWriteTestsSymflowerTemplateSymflowerFix: metrics.Assessments{
						metrics.AssessmentKeyFilesExecutedMaximumReachable: 1,
						metrics.AssessmentKeyResponseNoError:               1,
					},
				},
				ValidateLog: func(t *testing.T, data string) {
					assert.Contains(t, data, "Tests run: 1") // Tests are running but they are not Spring Boot.
				},
			})
		}
	})
}

func TestValidateWriteTestsRepository(t *testing.T) {
	validate := func(t *testing.T, tc *tasktesting.TestCaseValidateRepository) {
		tc.Validate(t, validateWriteTestsRepository)
	}

	t.Run("Go", func(t *testing.T) {
		t.Run("Plain", func(t *testing.T) {
			validate(t, &tasktesting.TestCaseValidateRepository{
				Name: "Well-formed",

				TestdataPath:   filepath.Join("..", "..", "testdata"),
				RepositoryPath: filepath.Join("golang", "plain"),
				Language:       &golang.Language{},
			})
		})
		t.Run("Light", func(t *testing.T) {
			validate(t, &tasktesting.TestCaseValidateRepository{
				Name: "Repository with test files",

				Before: func(repositoryPath string) {
					fileATest, err := os.Create(filepath.Join(repositoryPath, "fileA_test.go"))
					require.NoError(t, err)
					require.NoError(t, fileATest.Close())
				},

				TestdataPath:   filepath.Join("..", "..", "testdata"),
				RepositoryPath: filepath.Join("golang", "light"),
				Language:       &golang.Language{},
				ExpectedError: func(t *testing.T, err error) {
					assert.ErrorContains(t, err, "must contain only Go source files, but found [fileA_test.go]")
				},
			})
			validate(t, &tasktesting.TestCaseValidateRepository{
				Name: "Well-formed",

				TestdataPath:   filepath.Join("..", "..", "testdata"),
				RepositoryPath: filepath.Join("golang", "light"),
				Language:       &golang.Language{},
			})
		})
	})
	t.Run("Java", func(t *testing.T) {
		t.Run("Plain", func(t *testing.T) {
			validate(t, &tasktesting.TestCaseValidateRepository{
				Name: "Well-formed",

				TestdataPath:   filepath.Join("..", "..", "testdata"),
				RepositoryPath: filepath.Join("java", "plain"),
				Language:       &java.Language{},
			})
		})
		t.Run("Light", func(t *testing.T) {
			validate(t, &tasktesting.TestCaseValidateRepository{
				Name: "Repository with test files",

				Before: func(repositoryPath string) {
					somePackage := filepath.Join(repositoryPath, "src", "test", "java", "com", "eval")
					require.NoError(t, os.MkdirAll(somePackage, 0700))

					fileATest, err := os.Create(filepath.Join(somePackage, "FileATest.java"))
					require.NoError(t, err)
					require.NoError(t, fileATest.Close())
				},

				TestdataPath:   filepath.Join("..", "..", "testdata"),
				RepositoryPath: filepath.Join("java", "light"),
				Language:       &java.Language{},

				ExpectedError: func(t *testing.T, err error) {
					assert.ErrorContains(t, err, fmt.Sprintf("must contain only Java source files, but found [%s]", filepath.Join("src", "test", "java", "com", "eval", "FileATest.java")))
				},
			})
			validate(t, &tasktesting.TestCaseValidateRepository{
				Name: "Well-formed",

				TestdataPath:   filepath.Join("..", "..", "testdata"),
				RepositoryPath: filepath.Join("java", "light"),
				Language:       &java.Language{},
			})
		})
	})
}
