package report

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zimmski/osutil/bytesutil"

	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	metricstesting "github.com/symflower/eval-dev-quality/evaluate/metrics/testing"
	evaluatetask "github.com/symflower/eval-dev-quality/evaluate/task"
	languagetesting "github.com/symflower/eval-dev-quality/language/testing"
	modeltesting "github.com/symflower/eval-dev-quality/model/testing"
)

func TestGenerateCSVForAssessmentPerModelPerLanguagePerRepository(t *testing.T) {
	type testCase struct {
		Name string

		Assessments metricstesting.AssessmentTuples

		ExpectedString string
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			assessmentStore := assessmentTuplesToStore(tc.Assessments)

			actualString, err := GenerateCSV(assessmentStore)
			assert.NoError(t, err)

			assert.Equal(t, bytesutil.StringTrimIndentations(tc.ExpectedString), actualString)
		})
	}

	validate(t, &testCase{
		Name: "Single Empty Model",

		Assessments: metricstesting.AssessmentTuples{
			&metricstesting.AssessmentTuple{
				Model:          modeltesting.NewMockModelNamedWithCosts(t, "some-model", 0),
				Language:       languagetesting.NewMockLanguageNamed(t, "some-language"),
				RepositoryPath: "some-repository",
				Task:           evaluatetask.IdentifierWriteTests,
				Assessment:     metrics.NewAssessments(),
			},
		},

		ExpectedString: `
			model,cost,language,repository,task,score,coverage,files-executed,generate-tests-for-file-character-count,processing-time,response-character-count,response-no-error,response-no-excess,response-with-code
			some-model,0,some-language,some-repository,write-tests,0,0,0,0,0,0,0,0,0
		`,
	})
	validate(t, &testCase{
		Name: "Multiple Models",

		Assessments: metricstesting.AssessmentTuples{
			&metricstesting.AssessmentTuple{
				Model:          modeltesting.NewMockModelNamedWithCosts(t, "some-model-a", 0.0001),
				Language:       languagetesting.NewMockLanguageNamed(t, "some-language"),
				RepositoryPath: "some-repository",
				Task:           evaluatetask.IdentifierWriteTests,
				Assessment: metrics.Assessments{
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
			&metricstesting.AssessmentTuple{
				Model:          modeltesting.NewMockModelNamedWithCosts(t, "some-model-b", 0.0005),
				Language:       languagetesting.NewMockLanguageNamed(t, "some-language"),
				RepositoryPath: "some-repository",
				Task:           evaluatetask.IdentifierWriteTests,
				Assessment: metrics.Assessments{
					metrics.AssessmentKeyGenerateTestsForFileCharacterCount: 100,
					metrics.AssessmentKeyResponseCharacterCount:             200,
					metrics.AssessmentKeyCoverage:                           1,
					metrics.AssessmentKeyFilesExecuted:                      2,
					metrics.AssessmentKeyResponseNoError:                    3,
					metrics.AssessmentKeyResponseNoExcess:                   4,
					metrics.AssessmentKeyResponseWithCode:                   5,
					metrics.AssessmentKeyProcessingTime:                     300,
				},
			},
		},

		ExpectedString: `
			model,cost,language,repository,task,score,coverage,files-executed,generate-tests-for-file-character-count,processing-time,response-character-count,response-no-error,response-no-excess,response-with-code
			some-model-a,0.0001,some-language,some-repository,write-tests,15,1,2,50,200,100,3,4,5
			some-model-b,0.0005,some-language,some-repository,write-tests,15,1,2,100,300,200,3,4,5
		`,
	})
}

func TestGenerateCSVForAssessmentPerModel(t *testing.T) {
	type testCase struct {
		Name string

		Assessments AssessmentPerModel

		ExpectedString string
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			actualString, err := GenerateCSV(tc.Assessments)
			assert.NoError(t, err)

			assert.Equal(t, bytesutil.StringTrimIndentations(tc.ExpectedString), actualString)
		})
	}

	validate(t, &testCase{
		Name: "Single Empty Model",

		Assessments: AssessmentPerModel{
			modeltesting.NewMockModelNamedWithCosts(t, "some-model", 0): {},
		},

		ExpectedString: `
			model,cost,score,coverage,files-executed,generate-tests-for-file-character-count,processing-time,response-character-count,response-no-error,response-no-excess,response-with-code
			some-model,0,0,0,0,0,0,0,0,0,0
		`,
	})
	validate(t, &testCase{
		Name: "Multiple Models",

		Assessments: AssessmentPerModel{
			modeltesting.NewMockModelNamedWithCosts(t, "some-model-a", 0.0001): {
				metrics.AssessmentKeyGenerateTestsForFileCharacterCount: 50,
				metrics.AssessmentKeyResponseCharacterCount:             100,
				metrics.AssessmentKeyCoverage:                           1,
				metrics.AssessmentKeyFilesExecuted:                      2,
				metrics.AssessmentKeyResponseNoError:                    3,
				metrics.AssessmentKeyResponseNoExcess:                   4,
				metrics.AssessmentKeyResponseWithCode:                   5,
				metrics.AssessmentKeyProcessingTime:                     200,
			},
			modeltesting.NewMockModelNamedWithCosts(t, "some-model-b", 0.0005): {
				metrics.AssessmentKeyGenerateTestsForFileCharacterCount: 100,
				metrics.AssessmentKeyResponseCharacterCount:             200,
				metrics.AssessmentKeyCoverage:                           1,
				metrics.AssessmentKeyFilesExecuted:                      2,
				metrics.AssessmentKeyResponseNoError:                    3,
				metrics.AssessmentKeyResponseNoExcess:                   4,
				metrics.AssessmentKeyResponseWithCode:                   5,
				metrics.AssessmentKeyProcessingTime:                     300,
			},
		},

		ExpectedString: `
			model,cost,score,coverage,files-executed,generate-tests-for-file-character-count,processing-time,response-character-count,response-no-error,response-no-excess,response-with-code
			some-model-a,0.0001,15,1,2,50,200,100,3,4,5
			some-model-b,0.0005,15,1,2,100,300,200,3,4,5
		`,
	})
}
