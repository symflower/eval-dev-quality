package report

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	metricstesting "github.com/symflower/eval-dev-quality/evaluate/metrics/testing"
	"github.com/symflower/eval-dev-quality/language"
	languagetesting "github.com/symflower/eval-dev-quality/language/testing"
	"github.com/symflower/eval-dev-quality/model"
	modeltesting "github.com/symflower/eval-dev-quality/model/testing"
)

func TestAssessmentPerModelPerLanguagePerRepositoryWalk(t *testing.T) {
	type testCase struct {
		Name string

		Assessments AssessmentPerModelPerLanguagePerRepository

		ExpectedOrder []metrics.Assessments
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			actualOrder := []metrics.Assessments{}
			assert.NoError(t, tc.Assessments.Walk(func(m model.Model, l language.Language, r string, a metrics.Assessments) error {
				actualOrder = append(actualOrder, a)
				metricstesting.AssertAssessmentsEqual(t, tc.Assessments[m][l][r], a)

				return nil
			}))

			if assert.Equal(t, len(tc.ExpectedOrder), len(actualOrder)) {
				for i := range tc.ExpectedOrder {
					metricstesting.AssertAssessmentsEqual(t, tc.ExpectedOrder[i], actualOrder[i])
				}
			}
		})
	}

	validate(t, &testCase{
		Name: "Single Group",

		Assessments: AssessmentPerModelPerLanguagePerRepository{
			modeltesting.NewMockModelNamed(t, "some-model"): {
				languagetesting.NewMockLanguageNamed(t, "some-language"): {
					"some-repository": metrics.Assessments{
						metrics.AssessmentKeyResponseNoExcess: 1,
					},
				},
			},
		},

		ExpectedOrder: []metrics.Assessments{
			metrics.Assessments{
				metrics.AssessmentKeyResponseNoExcess: 1,
			},
		},
	})

	validate(t, &testCase{
		Name: "Multiple Groups",

		Assessments: AssessmentPerModelPerLanguagePerRepository{
			modeltesting.NewMockModelNamed(t, "some-model-a"): {
				languagetesting.NewMockLanguageNamed(t, "some-language-a"): {
					"some-repository-a": metrics.Assessments{
						metrics.AssessmentKeyResponseNoExcess: 1,
					},
					"some-repository-b": metrics.Assessments{
						metrics.AssessmentKeyResponseNoExcess: 2,
					},
				},
				languagetesting.NewMockLanguageNamed(t, "some-language-b"): {
					"some-repository-a": metrics.Assessments{
						metrics.AssessmentKeyResponseNoExcess: 3,
					},
					"some-repository-b": metrics.Assessments{
						metrics.AssessmentKeyResponseNoExcess: 4,
					},
				},
			},
			modeltesting.NewMockModelNamed(t, "some-model-b"): {
				languagetesting.NewMockLanguageNamed(t, "some-language-a"): {
					"some-repository-a": metrics.Assessments{
						metrics.AssessmentKeyResponseNoExcess: 5,
					},
					"some-repository-b": metrics.Assessments{
						metrics.AssessmentKeyResponseNoExcess: 6,
					},
				},
				languagetesting.NewMockLanguageNamed(t, "some-language-b"): {
					"some-repository-a": metrics.Assessments{
						metrics.AssessmentKeyResponseNoExcess: 7,
					},
					"some-repository-b": metrics.Assessments{
						metrics.AssessmentKeyResponseNoExcess: 8,
					},
				},
			},
		},

		ExpectedOrder: []metrics.Assessments{
			metrics.Assessments{
				metrics.AssessmentKeyResponseNoExcess: 1,
			},
			metrics.Assessments{
				metrics.AssessmentKeyResponseNoExcess: 2,
			},
			metrics.Assessments{
				metrics.AssessmentKeyResponseNoExcess: 3,
			},
			metrics.Assessments{
				metrics.AssessmentKeyResponseNoExcess: 4,
			},
			metrics.Assessments{
				metrics.AssessmentKeyResponseNoExcess: 5,
			},
			metrics.Assessments{
				metrics.AssessmentKeyResponseNoExcess: 6,
			},
			metrics.Assessments{
				metrics.AssessmentKeyResponseNoExcess: 7,
			},
			metrics.Assessments{
				metrics.AssessmentKeyResponseNoExcess: 8,
			},
		},
	})
}
