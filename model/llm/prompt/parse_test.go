package prompt

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	"github.com/zimmski/osutil/bytesutil"

	metricstesting "github.com/symflower/eval-dev-quality/evaluate/metrics/testing"
)

func TestParseResponse(t *testing.T) {
	type testCase struct {
		Name string

		Response string

		ExpectedAssessment metrics.Assessments
		ExpectedCode       string
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			actualAssessment, actualCode := ParseResponse(tc.Response)

			metricstesting.AssertAssessmentsEqual(t, tc.ExpectedAssessment, actualAssessment)
			assert.Equal(t, strings.TrimSpace(tc.ExpectedCode), actualCode)
		})
	}

	code := bytesutil.StringTrimIndentations(`
		package main

		import "testing"

		func TestPlain(t *testing.T) {
			plain()
		}
	`)

	validate(t, &testCase{
		Name: "Empty Response",

		ExpectedAssessment: metrics.Assessments{
			metrics.AssessmentKeyResponseNotEmpty: 0,
			metrics.AssessmentKeyResponseNoExcess: 0,
			metrics.AssessmentKeyResponseWithCode: 0,
		},
		ExpectedCode: "",
	})

	validate(t, &testCase{
		Name: "Only Code",

		Response: code,

		ExpectedAssessment: metrics.Assessments{
			metrics.AssessmentKeyResponseNotEmpty: 1,
			// If there are no code fences, we currently cannot determine what is code and what is (excessive) text.
			metrics.AssessmentKeyResponseNoExcess: 0,
			metrics.AssessmentKeyResponseWithCode: 0,
		},
		ExpectedCode: code,
	})

	t.Run("Formatted Code", func(t *testing.T) {
		validate(t, &testCase{
			Name: "No Prose",

			Response: "```\n" + code + "\n```\n",

			ExpectedAssessment: metrics.Assessments{
				metrics.AssessmentKeyResponseNotEmpty: 1,
				metrics.AssessmentKeyResponseNoExcess: 1,
				metrics.AssessmentKeyResponseWithCode: 1,
			},
			ExpectedCode: code,
		})

		validate(t, &testCase{
			Name: "With Prose",

			Response: "Some text...\n\n```\n" + code + "\n```\n\nSome more text...",

			ExpectedAssessment: metrics.Assessments{
				metrics.AssessmentKeyResponseNotEmpty: 1,
				metrics.AssessmentKeyResponseNoExcess: 0,
				metrics.AssessmentKeyResponseWithCode: 1,
			},
			ExpectedCode: code,
		})
	})

	validate(t, &testCase{
		Name: "Language Specified",

		Response: "```go\n" + code + "\n```\n",

		ExpectedAssessment: metrics.Assessments{
			metrics.AssessmentKeyResponseNotEmpty: 1,
			metrics.AssessmentKeyResponseNoExcess: 1,
			metrics.AssessmentKeyResponseWithCode: 1,
		},
		ExpectedCode: code,
	})

	validate(t, &testCase{
		Name: "Whitespace before Code Block Guards",

		Response: " ```\n" + code + "\n\t```\n",
		ExpectedAssessment: metrics.Assessments{
			metrics.AssessmentKeyResponseNotEmpty: 1,
			metrics.AssessmentKeyResponseNoExcess: 1,
			metrics.AssessmentKeyResponseWithCode: 1,
		},
		ExpectedCode: code,
	})

	validate(t, &testCase{
		Name: "Duplicated Code Block Guards",

		Response: "```\n```\n" + code + "\n```\n```\n",
		ExpectedAssessment: metrics.Assessments{
			metrics.AssessmentKeyResponseNotEmpty: 1,
			metrics.AssessmentKeyResponseNoExcess: 1,
			metrics.AssessmentKeyResponseWithCode: 1,
		},
		ExpectedCode: code,
	})
}
