package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zimmski/osutil/bytesutil"
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

func TestAssessmentString(t *testing.T) {
	type testCase struct {
		Name string

		Assessment Assessments

		ExpectedString string
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			actualString := tc.Assessment.String()

			assert.Equal(t, tc.ExpectedString, actualString)
		})
	}

	validate(t, &testCase{
		Name: "Initial Metrics",

		Assessment: NewAssessments(),

		ExpectedString: "files-executed=0, files-problems=0, coverage-statement=0, no-excess-response=0",
	})

	validate(t, &testCase{
		Name: "Empty Metrics",

		Assessment: Assessments{
			AssessmentKeyCoverageStatement: 1,
			AssessmentKeyFilesExecuted:     2,
			AssessmentKeyFilesProblems:     3,
			AssessmentKeyNoExcessResponse:  4,
		},

		ExpectedString: "files-executed=2, files-problems=3, coverage-statement=1, no-excess-response=4",
	})
}

func TestFormatStringCSV(t *testing.T) {
	type testCase struct {
		Name string

		AssessmentPerModel map[string]Assessments

		ExpectedString string
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			actualString, err := FormatStringCSV(tc.AssessmentPerModel)
			assert.NoError(t, err)

			assert.Equal(t, bytesutil.StringTrimIndentations(tc.ExpectedString), actualString)
		})
	}

	validate(t, &testCase{
		Name: "Single Empty Model",

		AssessmentPerModel: map[string]Assessments{
			"Model": Assessments{},
		},

		ExpectedString: `
			model,files-executed,files-problems,coverage-statement,no-excess-response
			Model,0,0,0,0
		`,
	})
	validate(t, &testCase{
		Name: "Multiple Models",

		AssessmentPerModel: map[string]Assessments{
			"ModelA": Assessments{
				AssessmentKeyCoverageStatement: 1,
				AssessmentKeyFilesExecuted:     2,
				AssessmentKeyFilesProblems:     3,
				AssessmentKeyNoExcessResponse:  4,
			},
			"ModelB": Assessments{
				AssessmentKeyCoverageStatement: 1,
				AssessmentKeyFilesExecuted:     2,
				AssessmentKeyFilesProblems:     3,
				AssessmentKeyNoExcessResponse:  4,
			},
		},

		ExpectedString: `
			model,files-executed,files-problems,coverage-statement,no-excess-response
			ModelA,2,3,1,4
			ModelB,2,3,1,4
		`,
	})
}
