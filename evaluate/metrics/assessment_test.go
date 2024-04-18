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
			AssessmentKeyResponseNoExcess: 1,
		},

		ExpectedAssessments: map[AssessmentKey]uint{
			AssessmentKeyResponseNoExcess: 1,
		},
	})

	validate(t, &testCase{
		Name: "Existing key",

		Assessments: map[AssessmentKey]uint{
			AssessmentKeyResponseNoExcess: 1,
		},
		X: map[AssessmentKey]uint{
			AssessmentKeyResponseNoExcess: 1,
		},

		ExpectedAssessments: map[AssessmentKey]uint{
			AssessmentKeyResponseNoExcess: 2,
		},
	})
}

func TestAssessmentsMerge(t *testing.T) {
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
			AssessmentKeyResponseNoExcess: 1,
		},

		ExpectedC: map[AssessmentKey]uint{
			AssessmentKeyResponseNoExcess: 1,
		},
	})

	validate(t, &testCase{
		Name: "Existing key",

		A: map[AssessmentKey]uint{
			AssessmentKeyResponseNoExcess: 1,
		},
		B: map[AssessmentKey]uint{
			AssessmentKeyResponseNoExcess: 1,
		},

		ExpectedC: map[AssessmentKey]uint{
			AssessmentKeyResponseNoExcess: 2,
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

		ExpectedString: "coverage-statement=0, files-executed=0, response-no-excess=0",
	})

	validate(t, &testCase{
		Name: "Empty Metrics",

		Assessment: Assessments{
			AssessmentKeyCoverageStatement: 1,
			AssessmentKeyFilesExecuted:     2,
			AssessmentKeyResponseNoExcess:  4,
		},

		ExpectedString: "coverage-statement=1, files-executed=2, response-no-excess=4",
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
			model,coverage-statement,files-executed,response-no-excess
			Model,0,0,0
		`,
	})
	validate(t, &testCase{
		Name: "Multiple Models",

		AssessmentPerModel: map[string]Assessments{
			"ModelA": Assessments{
				AssessmentKeyCoverageStatement: 1,
				AssessmentKeyFilesExecuted:     2,
				AssessmentKeyResponseNoExcess:  4,
			},
			"ModelB": Assessments{
				AssessmentKeyCoverageStatement: 1,
				AssessmentKeyFilesExecuted:     2,
				AssessmentKeyResponseNoExcess:  4,
			},
		},

		ExpectedString: `
			model,coverage-statement,files-executed,response-no-excess
			ModelA,1,2,4
			ModelB,1,2,4
		`,
	})
}

func TestAssessmentsEqual(t *testing.T) {
	type testCase struct {
		Name string

		Assessments Assessments
		X           Assessments

		ExpectedBool bool
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			actualBool := tc.Assessments.Equal(tc.X)

			assert.Equal(t, tc.ExpectedBool, actualBool)
		})
	}

	validate(t, &testCase{
		Name: "Empty",

		Assessments: NewAssessments(),
		X:           NewAssessments(),

		ExpectedBool: true,
	})

	validate(t, &testCase{
		Name: "Nil",

		Assessments: nil,
		X:           nil,

		ExpectedBool: true,
	})

	validate(t, &testCase{
		Name: "Equal Values",

		Assessments: Assessments{
			AssessmentKeyResponseWithCode: 2,
		},
		X: Assessments{
			AssessmentKeyResponseWithCode: 2,
		},

		ExpectedBool: true,
	})

	validate(t, &testCase{
		Name: "Default Value",

		Assessments: Assessments{
			AssessmentKeyResponseWithCode: 2,
			AssessmentKeyResponseNoError:  0,
		},
		X: Assessments{
			AssessmentKeyResponseWithCode: 2,
		},

		ExpectedBool: true,
	})

	validate(t, &testCase{
		Name: "Different Values",

		Assessments: Assessments{
			AssessmentKeyResponseWithCode: 3,
		},
		X: Assessments{
			AssessmentKeyResponseWithCode: 2,
		},

		ExpectedBool: false,
	})
}
