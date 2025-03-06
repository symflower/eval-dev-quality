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
		X: map[AssessmentKey]float64{
			AssessmentKeyResponseNoExcess: 1,
		},

		ExpectedAssessments: map[AssessmentKey]float64{
			AssessmentKeyResponseNoExcess: 1,
		},
	})

	validate(t, &testCase{
		Name: "Existing key",

		Assessments: map[AssessmentKey]float64{
			AssessmentKeyResponseNoExcess: 1,
		},
		X: map[AssessmentKey]float64{
			AssessmentKeyResponseNoExcess: 1,
		},

		ExpectedAssessments: map[AssessmentKey]float64{
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

		ExpectedString: "costs-total-actual=0, coverage=0, files-executed=0, files-executed-maximum-reachable=0, generate-tests-for-file-character-count=0, native-token-input=0, native-token-output=0, processing-time=0, response-character-count=0, response-no-error=0, response-no-excess=0, response-with-code=0, tests-passing=0, token-input=0, token-output=0",
	})

	validate(t, &testCase{
		Name: "Non-empty Metrics",

		Assessment: Assessments{
			AssessmentKeyGenerateTestsForFileCharacterCount: 50,
			AssessmentKeyResponseCharacterCount:             100,
			AssessmentKeyCoverage:                           1,
			AssessmentKeyFilesExecuted:                      2,
			AssessmentKeyFilesExecutedMaximumReachable:      2,
			AssessmentKeyResponseNoError:                    3,
			AssessmentKeyResponseNoExcess:                   4,
			AssessmentKeyResponseWithCode:                   5,
			AssessmentKeyProcessingTime:                     200,
			AssessmentKeyTestsPassing:                       7,
			AssessmentKeyTokenInput:                         123,
			AssessmentKeyTokenOutput:                        456,
		},

		ExpectedString: "costs-total-actual=0, coverage=1, files-executed=2, files-executed-maximum-reachable=2, generate-tests-for-file-character-count=50, native-token-input=0, native-token-output=0, processing-time=200, response-character-count=100, response-no-error=3, response-no-excess=4, response-with-code=5, tests-passing=7, token-input=123, token-output=456",
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

func TestCombineModelAndSymflowerFixAssessments(t *testing.T) {
	type testCase struct {
		Name string

		ModelAssessment         Assessments
		SymflowerFixAssessments Assessments

		ExpectedAssessments Assessments
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			actualAssessments := CombineWithSymflowerFixAssessments(tc.ModelAssessment, tc.SymflowerFixAssessments)

			assert.Equal(t, tc.ExpectedAssessments, actualAssessments)
		})
	}

	validate(t, &testCase{
		Name: "Simple",

		ModelAssessment: Assessments{
			AssessmentKeyFilesExecuted:                      1,
			AssessmentKeyProcessingTime:                     float64(200),
			AssessmentKeyCoverage:                           0,
			AssessmentKeyResponseCharacterCount:             100,
			AssessmentKeyGenerateTestsForFileCharacterCount: 50,
			AssessmentKeyResponseNoError:                    0,
			AssessmentKeyResponseWithCode:                   1,
			AssessmentKeyResponseNoExcess:                   1,
		},
		SymflowerFixAssessments: Assessments{
			AssessmentKeyFilesExecuted:   1,
			AssessmentKeyProcessingTime:  float64(100),
			AssessmentKeyCoverage:        1,
			AssessmentKeyResponseNoError: 1,
			AssessmentKeyTestsPassing:    10,
		},

		ExpectedAssessments: Assessments{
			AssessmentKeyFilesExecuted:                      1,
			AssessmentKeyProcessingTime:                     float64(300),
			AssessmentKeyCoverage:                           1,
			AssessmentKeyResponseCharacterCount:             100,
			AssessmentKeyGenerateTestsForFileCharacterCount: 50,
			AssessmentKeyResponseNoError:                    0,
			AssessmentKeyResponseWithCode:                   1,
			AssessmentKeyResponseNoExcess:                   1,
			AssessmentKeyTestsPassing:                       10,
		},
	})
}
