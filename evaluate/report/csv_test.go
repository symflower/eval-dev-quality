package report

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zimmski/osutil/bytesutil"

	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	languagetesting "github.com/symflower/eval-dev-quality/language/testing"
	modeltesting "github.com/symflower/eval-dev-quality/model/testing"
)

func TestFormatCSV(t *testing.T) {
	type testCase struct {
		Name string

		Assessments AssessmentPerModelPerLanguagePerRepository

		ExpectedString string
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			actualString, err := FormatCSV(tc.Assessments)
			assert.NoError(t, err)

			assert.Equal(t, bytesutil.StringTrimIndentations(tc.ExpectedString), actualString)
		})
	}

	validate(t, &testCase{
		Name: "Single Empty Model",

		Assessments: AssessmentPerModelPerLanguagePerRepository{
			modeltesting.NewMockModelNamed("some-model"): {
				languagetesting.NewMockLanguageNamed("some-language"): {
					"some-repository": metrics.NewAssessments(),
				},
			},
		},

		ExpectedString: `
			model,language,repository,score,coverage-statement,files-executed,response-no-error,response-no-excess,response-not-empty,response-with-code
			some-model,some-language,some-repository,0,0,0,0,0,0,0
		`,
	})
	validate(t, &testCase{
		Name: "Multiple Models",

		Assessments: AssessmentPerModelPerLanguagePerRepository{
			modeltesting.NewMockModelNamed("some-model-a"): {
				languagetesting.NewMockLanguageNamed("some-language"): {
					"some-repository": metrics.Assessments{
						metrics.AssessmentKeyCoverageStatement: 1,
						metrics.AssessmentKeyFilesExecuted:     2,
						metrics.AssessmentKeyResponseNoError:   3,
						metrics.AssessmentKeyResponseNoExcess:  4,
						metrics.AssessmentKeyResponseNotEmpty:  5,
						metrics.AssessmentKeyResponseWithCode:  6,
					},
				},
			},
			modeltesting.NewMockModelNamed("some-model-b"): {
				languagetesting.NewMockLanguageNamed("some-language"): {
					"some-repository": metrics.Assessments{
						metrics.AssessmentKeyCoverageStatement: 1,
						metrics.AssessmentKeyFilesExecuted:     2,
						metrics.AssessmentKeyResponseNoError:   3,
						metrics.AssessmentKeyResponseNoExcess:  4,
						metrics.AssessmentKeyResponseNotEmpty:  5,
						metrics.AssessmentKeyResponseWithCode:  6,
					},
				},
			},
		},

		ExpectedString: `
			model,language,repository,score,coverage-statement,files-executed,response-no-error,response-no-excess,response-not-empty,response-with-code
			some-model-a,some-language,some-repository,21,1,2,3,4,5,6
			some-model-b,some-language,some-repository,21,1,2,3,4,5,6
		`,
	})
}
