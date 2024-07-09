package report

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zimmski/osutil"
	"github.com/zimmski/osutil/bytesutil"
	"golang.org/x/exp/maps"

	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	evaluatetask "github.com/symflower/eval-dev-quality/evaluate/task"
	languagetesting "github.com/symflower/eval-dev-quality/language/testing"
	modeltesting "github.com/symflower/eval-dev-quality/model/testing"
	"github.com/symflower/eval-dev-quality/task"
)

func TestGenerateCSVForAssessmentPerModel(t *testing.T) {
	type testCase struct {
		Name string

		CSVFormatter CSVFormatter

		ExpectedString string
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			actualString, err := generateCSV(tc.CSVFormatter)
			assert.NoError(t, err)

			assert.Equal(t, bytesutil.StringTrimIndentations(tc.ExpectedString), actualString)
		})
	}

	validate(t, &testCase{
		Name: "Single empty model",

		CSVFormatter: EvaluationRecordsPerModel{
			"some-model-a": &EvaluationRecordSummary{
				ModelID:     "some-model-a",
				ModelName:   "Some Model A",
				ModelCost:   0.0001,
				LanguageID:  "golang",
				Assessments: metrics.NewAssessments(),
			},
		},

		ExpectedString: `
			model-id,model-name,cost,score,coverage,files-executed,generate-tests-for-file-character-count,processing-time,response-character-count,response-no-error,response-no-excess,response-with-code
			some-model-a,Some Model A,0.0001,0,0,0,0,0,0,0,0,0
		`,
	})
	validate(t, &testCase{
		Name: "Multiple models with assessments",

		CSVFormatter: EvaluationRecordsPerModel{
			"some-model-a": &EvaluationRecordSummary{
				ModelID:    "some-model-a",
				ModelName:  "Some Model A",
				ModelCost:  0.0001,
				LanguageID: "golang",
				Assessments: metrics.Assessments{
					metrics.AssessmentKeyGenerateTestsForFileCharacterCount: 50,
					metrics.AssessmentKeyResponseCharacterCount:             100,
					metrics.AssessmentKeyCoverage:                           1,
					metrics.AssessmentKeyFilesExecuted:                      2,
					metrics.AssessmentKeyResponseNoError:                    3,
					metrics.AssessmentKeyResponseNoExcess:                   4,
					metrics.AssessmentKeyResponseWithCode:                   5,
					metrics.AssessmentKeyProcessingTime:                     200,
				},
			},
			"some-model-b": &EvaluationRecordSummary{
				ModelID:    "some-model-b",
				ModelName:  "Some Model B",
				ModelCost:  0.0003,
				LanguageID: "java",
				Assessments: metrics.Assessments{
					metrics.AssessmentKeyGenerateTestsForFileCharacterCount: 100,
					metrics.AssessmentKeyResponseCharacterCount:             200,
					metrics.AssessmentKeyCoverage:                           6,
					metrics.AssessmentKeyFilesExecuted:                      7,
					metrics.AssessmentKeyResponseNoError:                    8,
					metrics.AssessmentKeyResponseNoExcess:                   9,
					metrics.AssessmentKeyResponseWithCode:                   10,
					metrics.AssessmentKeyProcessingTime:                     400,
				},
			},
		},

		ExpectedString: `
			model-id,model-name,cost,score,coverage,files-executed,generate-tests-for-file-character-count,processing-time,response-character-count,response-no-error,response-no-excess,response-with-code
			some-model-a,Some Model A,0.0001,15,1,2,50,200,100,3,4,5
			some-model-b,Some Model B,0.0003,40,6,7,100,400,200,8,9,10
		`,
	})
}

func TestNewEvaluationFile(t *testing.T) {
	var file strings.Builder
	_, err := NewEvaluationFile(&file)
	require.NoError(t, err)

	actualEvaluationFileContent := file.String()
	require.NoError(t, err)

	expectedEvaluationFileContent := bytesutil.StringTrimIndentations(`
		model-id,model-name,cost,language,repository,task,score,coverage,files-executed,generate-tests-for-file-character-count,processing-time,response-character-count,response-no-error,response-no-excess,response-with-code
	`)

	assert.Equal(t, expectedEvaluationFileContent, string(actualEvaluationFileContent))
}

func TestWriteEvaluationRecord(t *testing.T) {
	type testCase struct {
		Name string

		Assessments map[task.Identifier]metrics.Assessments

		ExpectedCSV string
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			var file strings.Builder
			evaluationFile, err := NewEvaluationFile(&file)
			require.NoError(t, err)

			modelMock := modeltesting.NewMockModelNamedWithCosts(t, "mocked-model", "Mocked Model", 0.0001)
			languageMock := languagetesting.NewMockLanguageNamed(t, "golang")

			err = evaluationFile.WriteEvaluationRecord(modelMock, languageMock, "golang/plain", tc.Assessments)
			require.NoError(t, err)

			assert.Equal(t, bytesutil.StringTrimIndentations(tc.ExpectedCSV), file.String())
		})
	}

	validate(t, &testCase{
		Name: "Single task with empty assessments",

		Assessments: map[task.Identifier]metrics.Assessments{
			evaluatetask.IdentifierWriteTests: metrics.NewAssessments(),
		},

		ExpectedCSV: `
			model-id,model-name,cost,language,repository,task,score,coverage,files-executed,generate-tests-for-file-character-count,processing-time,response-character-count,response-no-error,response-no-excess,response-with-code
			mocked-model,Mocked Model,0.0001,golang,golang/plain,write-tests,0,0,0,0,0,0,0,0,0
		`,
	})
	validate(t, &testCase{
		Name: "Multiple tasks with assessments",

		Assessments: map[task.Identifier]metrics.Assessments{
			evaluatetask.IdentifierWriteTests: metrics.Assessments{
				metrics.AssessmentKeyFilesExecuted:   1,
				metrics.AssessmentKeyResponseNoError: 1,
				metrics.AssessmentKeyCoverage:        0,
			},
			evaluatetask.IdentifierWriteTestsSymflowerFix: metrics.Assessments{
				metrics.AssessmentKeyFilesExecuted:   1,
				metrics.AssessmentKeyResponseNoError: 1,
				metrics.AssessmentKeyCoverage:        10,
			},
		},

		ExpectedCSV: `
			model-id,model-name,cost,language,repository,task,score,coverage,files-executed,generate-tests-for-file-character-count,processing-time,response-character-count,response-no-error,response-no-excess,response-with-code
			mocked-model,Mocked Model,0.0001,golang,golang/plain,write-tests,2,0,1,0,0,0,1,0,0
			mocked-model,Mocked Model,0.0001,golang,golang/plain,write-tests-symflower-fix,12,10,1,0,0,0,1,0,0
		`,
	})
}

func TestLoadEvaluationRecords(t *testing.T) {
	type testCase struct {
		Name string

		Before func(resultPath string)

		ExpectedEvaluationRecords EvaluationRecords
		ExpectedErr               func(err error)
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			resultPath := t.TempDir()

			if tc.Before != nil {
				tc.Before(resultPath)
			}

			actualAssessments, actualErr := loadEvaluationRecords(filepath.Join(resultPath, "evaluation.csv"))

			if tc.ExpectedErr != nil {
				tc.ExpectedErr(actualErr)
			} else {
				assert.NoError(t, actualErr)
				assert.Equal(t, tc.ExpectedEvaluationRecords, actualAssessments)
			}
		})
	}

	validate(t, &testCase{
		Name: "Evaluation file does not exist",

		ExpectedErr: func(err error) {
			if osutil.IsWindows() {
				assert.ErrorContains(t, err, "The system cannot find the file specified")
			} else {
				assert.ErrorContains(t, err, "no such file or directory")
			}
		},
	})
	validate(t, &testCase{
		Name: "Evaluation file exists but it is empty",

		Before: func(resultPath string) {
			file, err := os.Create(filepath.Join(resultPath, "evaluation.csv"))
			require.NoError(t, err)
			defer file.Close()
		},

		ExpectedErr: func(err error) {
			assert.ErrorContains(t, err, "found error while reading evaluation file")
		},
	})
	validate(t, &testCase{
		Name: "Evaluation file exists but with the wrong header",

		Before: func(resultPath string) {
			header := bytesutil.StringTrimIndentations(`
				model-id,model-name,cost
			`)
			require.NoError(t, os.WriteFile(filepath.Join(resultPath, "evaluation.csv"), []byte(header), 0644))
		},

		ExpectedErr: func(err error) {
			assert.ErrorContains(t, err, "found header [model-id model-name cost]")
		},
	})
	validate(t, &testCase{
		Name: "Single assessment",

		Before: func(resultPath string) {
			fileContent := bytesutil.StringTrimIndentations(`
				model-id,model-name,cost,language,repository,task,score,coverage,files-executed,generate-tests-for-file-character-count,processing-time,response-character-count,response-no-error,response-no-excess,response-with-code
				openrouter/anthropic/claude-1.2,Claude 1.2,0.0001,golang,golang/light,write-tests,982,750,18,70179,720571,71195,115,49,50
			`)
			require.NoError(t, os.WriteFile(filepath.Join(resultPath, "evaluation.csv"), []byte(fileContent), 0644))
		},

		ExpectedEvaluationRecords: EvaluationRecords{
			&EvaluationRecord{
				ModelID:        "openrouter/anthropic/claude-1.2",
				ModelName:      "Claude 1.2",
				ModelCost:      0.0001,
				LanguageID:     "golang",
				RepositoryName: "golang/light",
				Task:           "write-tests",
				Assessments: metrics.Assessments{
					metrics.AssessmentKeyCoverage:                           750,
					metrics.AssessmentKeyFilesExecuted:                      18,
					metrics.AssessmentKeyGenerateTestsForFileCharacterCount: 70179,
					metrics.AssessmentKeyProcessingTime:                     720571,
					metrics.AssessmentKeyResponseCharacterCount:             71195,
					metrics.AssessmentKeyResponseNoError:                    115,
					metrics.AssessmentKeyResponseNoExcess:                   49,
					metrics.AssessmentKeyResponseWithCode:                   50,
				},
			},
		},
	})
	validate(t, &testCase{
		Name: "Multiple assessments",

		Before: func(resultPath string) {
			fileContent := bytesutil.StringTrimIndentations(`
					model-id,model-name,cost,language,repository,task,score,coverage,files-executed,generate-tests-for-file-character-count,processing-time,response-character-count,response-no-error,response-no-excess,response-with-code
					openrouter/anthropic/claude-1.2,Claude 1.2,0.0001,golang,golang/light,write-tests,982,750,18,70179,720571,71195,115,49,50
					openrouter/anthropic/claude-1.2,Claude 1.2,0.0002,golang,golang/plain,transpile,37,20,2,441,11042,523,5,5,5
				`)
			require.NoError(t, os.WriteFile(filepath.Join(resultPath, "evaluation.csv"), []byte(fileContent), 0644))
		},

		ExpectedEvaluationRecords: EvaluationRecords{
			&EvaluationRecord{
				ModelID:        "openrouter/anthropic/claude-1.2",
				ModelName:      "Claude 1.2",
				ModelCost:      0.0001,
				LanguageID:     "golang",
				RepositoryName: "golang/light",
				Task:           "write-tests",
				Assessments: metrics.Assessments{
					metrics.AssessmentKeyCoverage:                           750,
					metrics.AssessmentKeyFilesExecuted:                      18,
					metrics.AssessmentKeyGenerateTestsForFileCharacterCount: 70179,
					metrics.AssessmentKeyProcessingTime:                     720571,
					metrics.AssessmentKeyResponseCharacterCount:             71195,
					metrics.AssessmentKeyResponseNoError:                    115,
					metrics.AssessmentKeyResponseNoExcess:                   49,
					metrics.AssessmentKeyResponseWithCode:                   50,
				},
			},
			&EvaluationRecord{
				ModelID:        "openrouter/anthropic/claude-1.2",
				ModelName:      "Claude 1.2",
				ModelCost:      0.0002,
				LanguageID:     "golang",
				RepositoryName: "golang/plain",
				Task:           "transpile",
				Assessments: metrics.Assessments{
					metrics.AssessmentKeyCoverage:                           20,
					metrics.AssessmentKeyFilesExecuted:                      2,
					metrics.AssessmentKeyGenerateTestsForFileCharacterCount: 441,
					metrics.AssessmentKeyProcessingTime:                     11042,
					metrics.AssessmentKeyResponseCharacterCount:             523,
					metrics.AssessmentKeyResponseNoError:                    5,
					metrics.AssessmentKeyResponseNoExcess:                   5,
					metrics.AssessmentKeyResponseWithCode:                   5,
				},
			},
		},
	})
}

func TestEvaluationRecordsGroupByModel(t *testing.T) {
	type testCase struct {
		Name string

		EvaluationRecords EvaluationRecords

		ExpectedEvaluationRecords map[string]*EvaluationRecordSummary
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			actualEvaluationRecords := tc.EvaluationRecords.GroupByModel()

			assert.ElementsMatch(t, maps.Keys(tc.ExpectedEvaluationRecords), maps.Keys(actualEvaluationRecords))

			for modelID, expectedRecord := range tc.ExpectedEvaluationRecords {
				actualRecord := actualEvaluationRecords[modelID]
				assert.Equal(t, expectedRecord, actualRecord)
				assert.Truef(t, expectedRecord.Assessments.Equal(actualRecord.Assessments), "model:%s\nexpected:%s\nactual:%s", modelID, tc.ExpectedEvaluationRecords, actualEvaluationRecords)
			}
		})
	}

	validate(t, &testCase{
		Name: "Single record",

		EvaluationRecords: EvaluationRecords{
			&EvaluationRecord{
				ModelID:    "openrouter/anthropic/claude-1.2",
				ModelName:  "Claude 1.2",
				ModelCost:  0.0001,
				LanguageID: "golang",
				Assessments: metrics.Assessments{
					metrics.AssessmentKeyCoverage:                           1,
					metrics.AssessmentKeyFilesExecuted:                      2,
					metrics.AssessmentKeyGenerateTestsForFileCharacterCount: 3,
					metrics.AssessmentKeyProcessingTime:                     4,
					metrics.AssessmentKeyResponseCharacterCount:             5,
					metrics.AssessmentKeyResponseNoError:                    6,
					metrics.AssessmentKeyResponseNoExcess:                   7,
					metrics.AssessmentKeyResponseWithCode:                   8,
				},
			},
		},
		ExpectedEvaluationRecords: map[string]*EvaluationRecordSummary{
			"openrouter/anthropic/claude-1.2": &EvaluationRecordSummary{
				ModelID:    "openrouter/anthropic/claude-1.2",
				ModelName:  "Claude 1.2",
				ModelCost:  0.0001,
				LanguageID: "golang",
				Assessments: metrics.Assessments{
					metrics.AssessmentKeyCoverage:                           1,
					metrics.AssessmentKeyFilesExecuted:                      2,
					metrics.AssessmentKeyGenerateTestsForFileCharacterCount: 3,
					metrics.AssessmentKeyProcessingTime:                     4,
					metrics.AssessmentKeyResponseCharacterCount:             5,
					metrics.AssessmentKeyResponseNoError:                    6,
					metrics.AssessmentKeyResponseNoExcess:                   7,
					metrics.AssessmentKeyResponseWithCode:                   8,
				},
			},
		},
	})
	validate(t, &testCase{
		Name: "Multiple records",

		EvaluationRecords: EvaluationRecords{
			&EvaluationRecord{
				ModelID:    "openrouter/anthropic/claude-1.2",
				ModelName:  "Claude 1.2",
				ModelCost:  0.0001,
				LanguageID: "golang",
				Assessments: metrics.Assessments{
					metrics.AssessmentKeyCoverage:                           1,
					metrics.AssessmentKeyFilesExecuted:                      2,
					metrics.AssessmentKeyGenerateTestsForFileCharacterCount: 3,
					metrics.AssessmentKeyProcessingTime:                     4,
					metrics.AssessmentKeyResponseCharacterCount:             5,
					metrics.AssessmentKeyResponseNoError:                    6,
					metrics.AssessmentKeyResponseNoExcess:                   7,
					metrics.AssessmentKeyResponseWithCode:                   8,
				},
			},
			&EvaluationRecord{
				ModelID:    "openrouter/anthropic/claude-1.2",
				ModelName:  "Claude 1.2",
				ModelCost:  0.0002,
				LanguageID: "golang",
				Assessments: metrics.Assessments{
					metrics.AssessmentKeyCoverage:                           1,
					metrics.AssessmentKeyFilesExecuted:                      2,
					metrics.AssessmentKeyGenerateTestsForFileCharacterCount: 3,
					metrics.AssessmentKeyProcessingTime:                     4,
					metrics.AssessmentKeyResponseCharacterCount:             5,
					metrics.AssessmentKeyResponseNoError:                    6,
					metrics.AssessmentKeyResponseNoExcess:                   7,
					metrics.AssessmentKeyResponseWithCode:                   8,
				},
			},
			&EvaluationRecord{
				ModelID:    "ollama/codeqwen:latest",
				ModelName:  "Code Qwen",
				ModelCost:  0.0003,
				LanguageID: "java",
				Assessments: metrics.Assessments{
					metrics.AssessmentKeyCoverage:                           1,
					metrics.AssessmentKeyFilesExecuted:                      2,
					metrics.AssessmentKeyGenerateTestsForFileCharacterCount: 3,
					metrics.AssessmentKeyProcessingTime:                     4,
					metrics.AssessmentKeyResponseCharacterCount:             5,
					metrics.AssessmentKeyResponseNoError:                    6,
					metrics.AssessmentKeyResponseNoExcess:                   7,
					metrics.AssessmentKeyResponseWithCode:                   8,
				},
			},
		},
		ExpectedEvaluationRecords: map[string]*EvaluationRecordSummary{
			"openrouter/anthropic/claude-1.2": &EvaluationRecordSummary{
				ModelID:    "openrouter/anthropic/claude-1.2",
				ModelName:  "Claude 1.2",
				ModelCost:  0.0001,
				LanguageID: "golang",
				Assessments: metrics.Assessments{
					metrics.AssessmentKeyCoverage:                           2,
					metrics.AssessmentKeyFilesExecuted:                      4,
					metrics.AssessmentKeyGenerateTestsForFileCharacterCount: 6,
					metrics.AssessmentKeyProcessingTime:                     8,
					metrics.AssessmentKeyResponseCharacterCount:             10,
					metrics.AssessmentKeyResponseNoError:                    12,
					metrics.AssessmentKeyResponseNoExcess:                   14,
					metrics.AssessmentKeyResponseWithCode:                   16,
				},
			},
			"ollama/codeqwen:latest": &EvaluationRecordSummary{
				ModelID:    "ollama/codeqwen:latest",
				ModelName:  "Code Qwen",
				ModelCost:  0.0003,
				LanguageID: "java",
				Assessments: metrics.Assessments{
					metrics.AssessmentKeyCoverage:                           1,
					metrics.AssessmentKeyFilesExecuted:                      2,
					metrics.AssessmentKeyGenerateTestsForFileCharacterCount: 3,
					metrics.AssessmentKeyProcessingTime:                     4,
					metrics.AssessmentKeyResponseCharacterCount:             5,
					metrics.AssessmentKeyResponseNoError:                    6,
					metrics.AssessmentKeyResponseNoExcess:                   7,
					metrics.AssessmentKeyResponseWithCode:                   8,
				},
			},
		},
	})
}

func TestEvaluationRecordsGroupByLanguageAndModel(t *testing.T) {
	type testCase struct {
		Name string

		EvaluationRecords EvaluationRecords

		ExpectedEvaluationRecordsPerLanguagePerModel EvaluationRecordsPerLanguagePerModel
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			actualEvaluationRecordsPerLanguagePerModel := tc.EvaluationRecords.GroupByLanguageAndModel()

			assert.Equal(t, tc.ExpectedEvaluationRecordsPerLanguagePerModel, actualEvaluationRecordsPerLanguagePerModel)
		})
	}

	validate(t, &testCase{
		Name: "Single record without assessments",

		EvaluationRecords: EvaluationRecords{
			&EvaluationRecord{
				ModelID:     "openrouter/anthropic/claude-1.2",
				ModelName:   "Claude 1.2",
				ModelCost:   0.0001,
				LanguageID:  "golang",
				Assessments: metrics.NewAssessments(),
			},
		},

		ExpectedEvaluationRecordsPerLanguagePerModel: EvaluationRecordsPerLanguagePerModel{
			"golang": EvaluationRecordsPerModel{
				"openrouter/anthropic/claude-1.2": &EvaluationRecordSummary{
					ModelID:     "openrouter/anthropic/claude-1.2",
					ModelName:   "Claude 1.2",
					ModelCost:   0.0001,
					LanguageID:  "golang",
					Assessments: metrics.NewAssessments(),
				},
			},
		},
	})
	validate(t, &testCase{
		Name: "Multiple records",

		EvaluationRecords: EvaluationRecords{
			&EvaluationRecord{
				ModelID:    "openrouter/anthropic/claude-1.2",
				ModelName:  "Claude 1.2",
				ModelCost:  0.0001,
				LanguageID: "golang",
				Assessments: metrics.Assessments{
					metrics.AssessmentKeyCoverage:                           1,
					metrics.AssessmentKeyFilesExecuted:                      2,
					metrics.AssessmentKeyGenerateTestsForFileCharacterCount: 3,
					metrics.AssessmentKeyProcessingTime:                     4,
					metrics.AssessmentKeyResponseCharacterCount:             5,
					metrics.AssessmentKeyResponseNoError:                    6,
					metrics.AssessmentKeyResponseNoExcess:                   7,
					metrics.AssessmentKeyResponseWithCode:                   8,
				},
			},
			&EvaluationRecord{
				ModelID:    "openrouter/anthropic/claude-1.2",
				ModelName:  "Claude 1.2",
				ModelCost:  0.0001,
				LanguageID: "golang",
				Assessments: metrics.Assessments{
					metrics.AssessmentKeyCoverage:                           1,
					metrics.AssessmentKeyFilesExecuted:                      2,
					metrics.AssessmentKeyGenerateTestsForFileCharacterCount: 3,
					metrics.AssessmentKeyProcessingTime:                     4,
					metrics.AssessmentKeyResponseCharacterCount:             5,
					metrics.AssessmentKeyResponseNoError:                    6,
					metrics.AssessmentKeyResponseNoExcess:                   7,
					metrics.AssessmentKeyResponseWithCode:                   8,
				},
			},
			&EvaluationRecord{
				ModelID:    "openrouter/anthropic/claude-1.2",
				ModelName:  "Claude 1.2",
				ModelCost:  0.0001,
				LanguageID: "java",
				Assessments: metrics.Assessments{
					metrics.AssessmentKeyCoverage:                           1,
					metrics.AssessmentKeyFilesExecuted:                      2,
					metrics.AssessmentKeyGenerateTestsForFileCharacterCount: 3,
					metrics.AssessmentKeyProcessingTime:                     4,
					metrics.AssessmentKeyResponseCharacterCount:             5,
					metrics.AssessmentKeyResponseNoError:                    6,
					metrics.AssessmentKeyResponseNoExcess:                   7,
					metrics.AssessmentKeyResponseWithCode:                   8,
				},
			},
			&EvaluationRecord{
				ModelID:    "ollama/codeqwen:latest",
				ModelName:  "Code Qwen",
				ModelCost:  0.0003,
				LanguageID: "golang",
				Assessments: metrics.Assessments{
					metrics.AssessmentKeyCoverage:                           1,
					metrics.AssessmentKeyFilesExecuted:                      2,
					metrics.AssessmentKeyGenerateTestsForFileCharacterCount: 3,
					metrics.AssessmentKeyProcessingTime:                     4,
					metrics.AssessmentKeyResponseCharacterCount:             5,
					metrics.AssessmentKeyResponseNoError:                    6,
					metrics.AssessmentKeyResponseNoExcess:                   7,
					metrics.AssessmentKeyResponseWithCode:                   8,
				},
			},
			&EvaluationRecord{
				ModelID:    "ollama/codeqwen:latest",
				ModelName:  "Code Qwen",
				ModelCost:  0.0003,
				LanguageID: "java",
				Assessments: metrics.Assessments{
					metrics.AssessmentKeyCoverage:                           1,
					metrics.AssessmentKeyFilesExecuted:                      2,
					metrics.AssessmentKeyGenerateTestsForFileCharacterCount: 3,
					metrics.AssessmentKeyProcessingTime:                     4,
					metrics.AssessmentKeyResponseCharacterCount:             5,
					metrics.AssessmentKeyResponseNoError:                    6,
					metrics.AssessmentKeyResponseNoExcess:                   7,
					metrics.AssessmentKeyResponseWithCode:                   8,
				},
			},
		},

		ExpectedEvaluationRecordsPerLanguagePerModel: EvaluationRecordsPerLanguagePerModel{
			"golang": EvaluationRecordsPerModel{
				"openrouter/anthropic/claude-1.2": &EvaluationRecordSummary{
					ModelID:    "openrouter/anthropic/claude-1.2",
					ModelName:  "Claude 1.2",
					ModelCost:  0.0001,
					LanguageID: "golang",
					Assessments: metrics.Assessments{
						metrics.AssessmentKeyCoverage:                           2,
						metrics.AssessmentKeyFilesExecuted:                      4,
						metrics.AssessmentKeyGenerateTestsForFileCharacterCount: 6,
						metrics.AssessmentKeyProcessingTime:                     8,
						metrics.AssessmentKeyResponseCharacterCount:             10,
						metrics.AssessmentKeyResponseNoError:                    12,
						metrics.AssessmentKeyResponseNoExcess:                   14,
						metrics.AssessmentKeyResponseWithCode:                   16,
					},
				},
				"ollama/codeqwen:latest": &EvaluationRecordSummary{
					ModelID:    "ollama/codeqwen:latest",
					ModelName:  "Code Qwen",
					ModelCost:  0.0003,
					LanguageID: "golang",
					Assessments: metrics.Assessments{
						metrics.AssessmentKeyCoverage:                           1,
						metrics.AssessmentKeyFilesExecuted:                      2,
						metrics.AssessmentKeyGenerateTestsForFileCharacterCount: 3,
						metrics.AssessmentKeyProcessingTime:                     4,
						metrics.AssessmentKeyResponseCharacterCount:             5,
						metrics.AssessmentKeyResponseNoError:                    6,
						metrics.AssessmentKeyResponseNoExcess:                   7,
						metrics.AssessmentKeyResponseWithCode:                   8,
					},
				},
			},
			"java": EvaluationRecordsPerModel{
				"openrouter/anthropic/claude-1.2": &EvaluationRecordSummary{
					ModelID:    "openrouter/anthropic/claude-1.2",
					ModelName:  "Claude 1.2",
					ModelCost:  0.0001,
					LanguageID: "java",
					Assessments: metrics.Assessments{
						metrics.AssessmentKeyCoverage:                           1,
						metrics.AssessmentKeyFilesExecuted:                      2,
						metrics.AssessmentKeyGenerateTestsForFileCharacterCount: 3,
						metrics.AssessmentKeyProcessingTime:                     4,
						metrics.AssessmentKeyResponseCharacterCount:             5,
						metrics.AssessmentKeyResponseNoError:                    6,
						metrics.AssessmentKeyResponseNoExcess:                   7,
						metrics.AssessmentKeyResponseWithCode:                   8,
					},
				},
				"ollama/codeqwen:latest": &EvaluationRecordSummary{
					ModelID:    "ollama/codeqwen:latest",
					ModelName:  "Code Qwen",
					ModelCost:  0.0003,
					LanguageID: "java",
					Assessments: metrics.Assessments{
						metrics.AssessmentKeyCoverage:                           1,
						metrics.AssessmentKeyFilesExecuted:                      2,
						metrics.AssessmentKeyGenerateTestsForFileCharacterCount: 3,
						metrics.AssessmentKeyProcessingTime:                     4,
						metrics.AssessmentKeyResponseCharacterCount:             5,
						metrics.AssessmentKeyResponseNoError:                    6,
						metrics.AssessmentKeyResponseNoExcess:                   7,
						metrics.AssessmentKeyResponseWithCode:                   8,
					},
				},
			},
		},
	})

}

func TestWriteCSVs(t *testing.T) {
	type testCase struct {
		Name string

		FileName string

		ExpectedFileContent string
	}

	resultPath := t.TempDir()

	evaluationFilePath := filepath.Join(resultPath, "evaluation.csv")
	evaluationFileContent := bytesutil.StringTrimIndentations(`
		model-id,model-name,cost,language,repository,task,score,coverage,files-executed,generate-tests-for-file-character-count,processing-time,response-character-count,response-no-error,response-no-excess,response-with-code
		openrouter/anthropic/claude-2.0,Claude 2.0,0.001,golang,golang/light,write-tests,24,1,2,3,4,5,6,7,8
		openrouter/anthropic/claude-2.0,Claude 2.0,0.001,golang,golang/plain,write-tests,24,1,2,3,4,5,6,7,8
		openrouter/anthropic/claude-2.0,Claude 2.0,0.001,java,java/light,write-tests,69,10,11,12,13,14,15,16,17
		openrouter/anthropic/claude-2.0,Claude 2.0,0.001,java,java/plain,write-tests,69,10,11,12,13,14,15,16,17
		openrouter/anthropic/claude-3-sonnet,Claude 3 Sonnet,0.003,golang,golang/light,write-tests,21,8,7,6,5,4,3,2,1
		openrouter/anthropic/claude-3-sonnet,Claude 3 Sonnet,0.003,golang,golang/plain,write-tests,21,8,7,6,5,4,3,2,1
		openrouter/anthropic/claude-3-sonnet,Claude 3 Sonnet,0.003,java,java/light,write-tests,69,10,11,12,13,14,15,16,17
		openrouter/anthropic/claude-3-sonnet,Claude 3 Sonnet,0.003,java,java/plain,write-tests,69,10,11,12,13,14,15,16,17
		openrouter/openai/gpt-4,GPT 4,0.005,golang,golang/light,write-tests,24,1,2,3,4,5,6,7,8
		openrouter/openai/gpt-4,GPT 4,0.005,golang,golang/plain,write-tests,24,1,2,3,4,5,6,7,8
		openrouter/openai/gpt-4,GPT 4,0.005,java,java/light,write-tests,24,1,2,3,4,5,6,7,8
		openrouter/openai/gpt-4,GPT 4,0.005,java,java/plain,write-tests,24,1,2,3,4,5,6,7,8
	`)
	require.NoError(t, os.WriteFile(evaluationFilePath, []byte(evaluationFileContent), 0644))

	err := WriteCSVs(resultPath)
	require.NoError(t, err)

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			summedFilePath := filepath.Join(resultPath, tc.FileName)

			_, err = os.Stat(summedFilePath)
			require.NoError(t, err)

			actualSummedFileContent, err := os.ReadFile(summedFilePath)
			require.NoError(t, err)

			assert.Equal(t, bytesutil.StringTrimIndentations(tc.ExpectedFileContent), string(actualSummedFileContent))
		})
	}

	validate(t, &testCase{
		Name: "Models summed",

		FileName: "models-summed.csv",

		ExpectedFileContent: `
			model-id,model-name,cost,score,coverage,files-executed,generate-tests-for-file-character-count,processing-time,response-character-count,response-no-error,response-no-excess,response-with-code
			openrouter/anthropic/claude-2.0,Claude 2.0,0.001,186,22,26,30,34,38,42,46,50
			openrouter/anthropic/claude-3-sonnet,Claude 3 Sonnet,0.003,180,36,36,36,36,36,36,36,36
			openrouter/openai/gpt-4,GPT 4,0.005,96,4,8,12,16,20,24,28,32
		`,
	})
	validate(t, &testCase{
		Name: "Golang summed",

		FileName: "golang-summed.csv",

		ExpectedFileContent: `
			model-id,model-name,cost,score,coverage,files-executed,generate-tests-for-file-character-count,processing-time,response-character-count,response-no-error,response-no-excess,response-with-code
			openrouter/anthropic/claude-2.0,Claude 2.0,0.001,48,2,4,6,8,10,12,14,16
			openrouter/anthropic/claude-3-sonnet,Claude 3 Sonnet,0.003,42,16,14,12,10,8,6,4,2
			openrouter/openai/gpt-4,GPT 4,0.005,48,2,4,6,8,10,12,14,16
		`,
	})
	validate(t, &testCase{
		Name: "Java summed",

		FileName: "java-summed.csv",

		ExpectedFileContent: `
			model-id,model-name,cost,score,coverage,files-executed,generate-tests-for-file-character-count,processing-time,response-character-count,response-no-error,response-no-excess,response-with-code
			openrouter/anthropic/claude-2.0,Claude 2.0,0.001,138,20,22,24,26,28,30,32,34
			openrouter/anthropic/claude-3-sonnet,Claude 3 Sonnet,0.003,138,20,22,24,26,28,30,32,34
			openrouter/openai/gpt-4,GPT 4,0.005,48,2,4,6,8,10,12,14,16
		`,
	})
}
