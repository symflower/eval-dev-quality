package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		Name: "Empty Metrics",

		Assessment: NewAssessments(),

		ExpectedString: "score=0, coverage-statement=0, files-executed=0, response-no-error=0, response-no-excess=0, response-not-empty=0, response-with-code=0",
	})

	validate(t, &testCase{
		Name: "Non-empty Metrics",

		Assessment: Assessments{
			AssessmentKeyCoverageStatement: 1,
			AssessmentKeyFilesExecuted:     2,
			AssessmentKeyResponseNoError:   3,
			AssessmentKeyResponseNoExcess:  4,
			AssessmentKeyResponseNotEmpty:  5,
			AssessmentKeyResponseWithCode:  6,
		},

		ExpectedString: "score=21, coverage-statement=1, files-executed=2, response-no-error=3, response-no-excess=4, response-not-empty=5, response-with-code=6",
	})
}

func TestFormatCSV(t *testing.T) {
	type testCase struct {
		Name string

		AssessmentPerModel map[string]Assessments

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

		AssessmentPerModel: map[string]Assessments{
			"Model": Assessments{},
		},

		ExpectedString: `
			model,score,coverage-statement,files-executed,response-no-error,response-no-excess,response-not-empty,response-with-code
			Model,0,0,0,0,0,0,0
		`,
	})
	validate(t, &testCase{
		Name: "Multiple Models",

		AssessmentPerModel: map[string]Assessments{
			"ModelA": Assessments{
				AssessmentKeyCoverageStatement: 1,
				AssessmentKeyFilesExecuted:     2,
				AssessmentKeyResponseNoError:   3,
				AssessmentKeyResponseNoExcess:  4,
				AssessmentKeyResponseNotEmpty:  5,
				AssessmentKeyResponseWithCode:  6,
			},
			"ModelB": Assessments{
				AssessmentKeyCoverageStatement: 2,
				AssessmentKeyFilesExecuted:     3,
				AssessmentKeyResponseNoError:   4,
				AssessmentKeyResponseNoExcess:  5,
				AssessmentKeyResponseNotEmpty:  6,
				AssessmentKeyResponseWithCode:  7,
			},
		},

		ExpectedString: `
			model,score,coverage-statement,files-executed,response-no-error,response-no-excess,response-not-empty,response-with-code
			ModelA,21,1,2,3,4,5,6
			ModelB,27,2,3,4,5,6,7
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

func TestAssessmentsScore(t *testing.T) {
	type testCase struct {
		Name string

		Assessments Assessments

		ExpectedScore uint
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			actualScore := tc.Assessments.Score()

			assert.Equal(t, tc.ExpectedScore, actualScore)
		})
	}

	validate(t, &testCase{
		Name: "Empty Assessment",

		Assessments: NewAssessments(),

		ExpectedScore: 0,
	})

	validate(t, &testCase{
		Name: "Values Assessment",

		Assessments: Assessments{
			AssessmentKeyFilesExecuted:     5,
			AssessmentKeyCoverageStatement: 4,
		},

		ExpectedScore: 9,
	})
}

func TestWalkByScore(t *testing.T) {
	type testCase struct {
		Name string

		AssessmentPerModel map[string]Assessments

		ExpectedModelOrder []string
		ExpectedScoreOrder []uint
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			require.Equal(t, len(tc.ExpectedModelOrder), len(tc.ExpectedScoreOrder), "expected order needs equal lengths")

			actualModelOrder := make([]string, 0, len(tc.ExpectedModelOrder))
			actualAssessmentOrder := make([]Assessments, 0, len(tc.ExpectedModelOrder))
			actualScoreOrder := make([]uint, 0, len(tc.ExpectedScoreOrder))
			assert.NoError(t, WalkByScore(tc.AssessmentPerModel, func(model string, assessment Assessments, score uint) error {
				actualModelOrder = append(actualModelOrder, model)
				actualAssessmentOrder = append(actualAssessmentOrder, assessment)
				actualScoreOrder = append(actualScoreOrder, score)

				return nil
			}))

			assert.Equal(t, tc.ExpectedModelOrder, actualModelOrder)
			assert.Equal(t, tc.ExpectedScoreOrder, actualScoreOrder)
			for i, model := range tc.ExpectedModelOrder {
				assert.Equal(t, tc.AssessmentPerModel[model], actualAssessmentOrder[i])
			}
		})
	}

	validate(t, &testCase{
		Name: "No Assessment",

		AssessmentPerModel: map[string]Assessments{},

		ExpectedModelOrder: []string{},
		ExpectedScoreOrder: []uint{},
	})

	validate(t, &testCase{
		Name: "Single Assessment",

		AssessmentPerModel: map[string]Assessments{
			"Model": Assessments{
				AssessmentKeyFilesExecuted: 1,
			},
		},

		ExpectedModelOrder: []string{
			"Model",
		},
		ExpectedScoreOrder: []uint{
			1,
		},
	})

	validate(t, &testCase{
		Name: "Multiple Assessments",

		AssessmentPerModel: map[string]Assessments{
			"ModelA": Assessments{
				AssessmentKeyFilesExecuted: 1,
			},
			"ModelB": Assessments{
				AssessmentKeyFilesExecuted: 2,
			},
			"ModelC": Assessments{
				AssessmentKeyFilesExecuted: 3,
			},
		},

		ExpectedModelOrder: []string{
			"ModelA",
			"ModelB",
			"ModelC",
		},
		ExpectedScoreOrder: []uint{
			1,
			2,
			3,
		},
	})
}
