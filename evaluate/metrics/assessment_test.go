package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAssessmentsAdd(t *testing.T) {
	type testCase struct {
		Name string

		Assessments Assessments
		X           Assessments

		ExpectedAssessments Assessments
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			tc.Assessments.Add(tc.X)

			assert.Equal(t, tc.Assessments, tc.ExpectedAssessments)
		})
	}

	validate(t, &testCase{
		Name: "Empty",

		Assessments: NewAssessments(),
		X:           NewAssessments(),

		ExpectedAssessments: NewAssessments(),
	})

	validate(t, &testCase{
		Name: "Non existing key",

		Assessments: NewAssessments(),
		X: map[AssessmentKey]uint{
			AssessmentKeyNoExcessResponse: 1,
		},

		ExpectedAssessments: map[AssessmentKey]uint{
			AssessmentKeyNoExcessResponse: 1,
		},
	})

	validate(t, &testCase{
		Name: "Existing key",

		Assessments: map[AssessmentKey]uint{
			AssessmentKeyNoExcessResponse: 1,
		},
		X: map[AssessmentKey]uint{
			AssessmentKeyNoExcessResponse: 1,
		},

		ExpectedAssessments: map[AssessmentKey]uint{
			AssessmentKeyNoExcessResponse: 2,
		},
	})
}

func TestMerge(t *testing.T) {
	type testCase struct {
		Name string

		A Assessments
		B Assessments

		ExpectedC Assessments
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			actualC := Merge(tc.A, tc.B)

			assert.Equal(t, tc.ExpectedC, actualC)
		})
	}

	validate(t, &testCase{
		Name: "Empty",

		ExpectedC: NewAssessments(),
	})

	validate(t, &testCase{
		Name: "Non existing key",

		A: NewAssessments(),
		B: map[AssessmentKey]uint{
			AssessmentKeyNoExcessResponse: 1,
		},

		ExpectedC: map[AssessmentKey]uint{
			AssessmentKeyNoExcessResponse: 1,
		},
	})

	validate(t, &testCase{
		Name: "Existing key",

		A: map[AssessmentKey]uint{
			AssessmentKeyNoExcessResponse: 1,
		},
		B: map[AssessmentKey]uint{
			AssessmentKeyNoExcessResponse: 1,
		},

		ExpectedC: map[AssessmentKey]uint{
			AssessmentKeyNoExcessResponse: 2,
		},
	})
}
