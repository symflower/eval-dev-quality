package report

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zimmski/osutil/bytesutil"

	"github.com/symflower/eval-dev-quality/evaluate/metrics"
)

func TestFormatCSV(t *testing.T) {
	type testCase struct {
		Name string

		AssessmentPerModel map[string]metrics.Assessments

		ExpectedString string
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			actualString, err := FormatCSV(tc.AssessmentPerModel)
			assert.NoError(t, err)

			assert.Equal(t, bytesutil.StringTrimIndentations(tc.ExpectedString), actualString)
		})
	}

	validate(t, &testCase{
		Name: "Single Empty Model",

		AssessmentPerModel: map[string]metrics.Assessments{
			"Model": metrics.Assessments{},
		},

		ExpectedString: `
			model,score,coverage-statement,files-executed,response-no-error,response-no-excess,response-not-empty,response-with-code
			Model,0,0,0,0,0,0,0
		`,
	})
	validate(t, &testCase{
		Name: "Multiple Models",

		AssessmentPerModel: map[string]metrics.Assessments{
			"ModelA": metrics.Assessments{
				metrics.AssessmentKeyCoverageStatement: 1,
				metrics.AssessmentKeyFilesExecuted:     2,
				metrics.AssessmentKeyResponseNoError:   3,
				metrics.AssessmentKeyResponseNoExcess:  4,
				metrics.AssessmentKeyResponseNotEmpty:  5,
				metrics.AssessmentKeyResponseWithCode:  6,
			},
			"ModelB": metrics.Assessments{
				metrics.AssessmentKeyCoverageStatement: 2,
				metrics.AssessmentKeyFilesExecuted:     3,
				metrics.AssessmentKeyResponseNoError:   4,
				metrics.AssessmentKeyResponseNoExcess:  5,
				metrics.AssessmentKeyResponseNotEmpty:  6,
				metrics.AssessmentKeyResponseWithCode:  7,
			},
		},

		ExpectedString: `
			model,score,coverage-statement,files-executed,response-no-error,response-no-excess,response-not-empty,response-with-code
			ModelA,21,1,2,3,4,5,6
			ModelB,27,2,3,4,5,6,7
		`,
	})
}
