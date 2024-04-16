package prompt

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	"github.com/zimmski/osutil/bytesutil"
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

			assert.Equal(t, tc.ExpectedAssessment, actualAssessment)
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
		Name: "Only Code",

		Response: code,

		ExpectedAssessment: metrics.Assessments{
			metrics.AssessmentKeyNoExcessResponse: 1,
		},
		ExpectedCode: code,
	})

	t.Run("Formatted Code", func(t *testing.T) {
		validate(t, &testCase{
			Name: "No Prose",

			Response: "```\n" + code + "\n```\n",

			ExpectedAssessment: metrics.Assessments{
				metrics.AssessmentKeyNoExcessResponse: 1,
			},
			ExpectedCode: code,
		})

		validate(t, &testCase{
			Name: "With Prose",

			Response: "Some text...\n\n```\n" + code + "\n```\n\nSome more text...",

			ExpectedAssessment: metrics.Assessments{
				metrics.AssessmentKeyNoExcessResponse: 0,
			},
			ExpectedCode: code,
		})
	})

	validate(t, &testCase{
		Name: "Language Specified",

		Response: "```go\n" + code + "\n```\n",

		ExpectedAssessment: metrics.Assessments{
			metrics.AssessmentKeyNoExcessResponse: 1,
		},
		ExpectedCode: code,
	})

	validate(t, &testCase{
		Name: "Whitespace before Code Block Guards",

		Response: " ```\n" + code + "\n\t```\n",
		ExpectedAssessment: metrics.Assessments{
			metrics.AssessmentKeyNoExcessResponse: 1,
		},
		ExpectedCode: code,
	})

	validate(t, &testCase{
		Name: "Duplicated Code Block Guards",

		Response: "```\n```\n" + code + "\n```\n```\n",
		ExpectedAssessment: metrics.Assessments{
			metrics.AssessmentKeyNoExcessResponse: 1,
		},
		ExpectedCode: code,
	})
}
