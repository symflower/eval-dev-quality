package report

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
			assert.NoError(t, tc.Assessments.Walk(func(m model.Model, l language.Language, r string, a metrics.Assessments) (err error) {
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

func TestWalkByScore(t *testing.T) {
	type testCase struct {
		Name string

		AssessmentPerModel AssessmentPerModel

		ExpectedModelOrder []model.Model
		ExpectedScoreOrder []uint64
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			require.Equal(t, len(tc.ExpectedModelOrder), len(tc.ExpectedScoreOrder), "expected order needs equal lengths")

			actualModelOrder := make([]model.Model, 0, len(tc.ExpectedModelOrder))
			actualAssessmentOrder := make([]metrics.Assessments, 0, len(tc.ExpectedModelOrder))
			actualScoreOrder := make([]uint64, 0, len(tc.ExpectedScoreOrder))
			assert.NoError(t, tc.AssessmentPerModel.WalkByScore(func(model model.Model, assessment metrics.Assessments, score uint64) (err error) {
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

	modelA := modeltesting.NewMockModelNamed(t, "ModelA")
	modelB := modeltesting.NewMockModelNamed(t, "ModelB")
	modelC := modeltesting.NewMockModelNamed(t, "ModelC")

	validate(t, &testCase{
		Name: "No Assessment",

		AssessmentPerModel: AssessmentPerModel{},

		ExpectedModelOrder: []model.Model{},
		ExpectedScoreOrder: []uint64{},
	})

	validate(t, &testCase{
		Name: "Single Assessment",

		AssessmentPerModel: AssessmentPerModel{
			modelA: metrics.Assessments{
				metrics.AssessmentKeyFilesExecuted: 1,
			},
		},

		ExpectedModelOrder: []model.Model{
			modelA,
		},
		ExpectedScoreOrder: []uint64{
			1,
		},
	})

	validate(t, &testCase{
		Name: "Multiple Assessments",

		AssessmentPerModel: AssessmentPerModel{
			modelA: metrics.Assessments{
				metrics.AssessmentKeyFilesExecuted: 1,
			},
			modelB: metrics.Assessments{
				metrics.AssessmentKeyFilesExecuted: 2,
			},
			modelC: metrics.Assessments{
				metrics.AssessmentKeyFilesExecuted: 3,
			},
		},

		ExpectedModelOrder: []model.Model{
			modelA,
			modelB,
			modelC,
		},
		ExpectedScoreOrder: []uint64{
			1,
			2,
			3,
		},
	})
}

func TestAssessmentCollapseByModel(t *testing.T) {
	type testCase struct {
		Name string

		Assessments AssessmentPerModelPerLanguagePerRepository

		ExpectedAssessmentPerModel AssessmentPerModel
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			actualAssessmentPerModel := tc.Assessments.CollapseByModel()

			assert.Equal(t, tc.ExpectedAssessmentPerModel, actualAssessmentPerModel)
		})
	}

	modelA := modeltesting.NewMockModelNamed(t, "some-model-a")
	modelB := modeltesting.NewMockModelNamed(t, "some-model-b")

	validate(t, &testCase{
		Name: "Collapse",

		Assessments: AssessmentPerModelPerLanguagePerRepository{
			modelA: {
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
			modelB: {
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

		ExpectedAssessmentPerModel: AssessmentPerModel{
			modelA: metrics.Assessments{
				metrics.AssessmentKeyResponseNoExcess: 10,
			},
			modelB: metrics.Assessments{
				metrics.AssessmentKeyResponseNoExcess: 26,
			},
		},
	})
}

func TestAssessmentCollapseByLanguage(t *testing.T) {
	type testCase struct {
		Name string

		Assessments AssessmentPerModelPerLanguagePerRepository

		ExpectedAssessmentPerLanguagePerModel AssessmentPerLanguagePerModel
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			actualAssessmentPerLanguagePerModel := tc.Assessments.CollapseByLanguage()

			assert.Equal(t, tc.ExpectedAssessmentPerLanguagePerModel, actualAssessmentPerLanguagePerModel)
		})
	}

	modelA := modeltesting.NewMockModelNamed(t, "some-model-a")
	modelB := modeltesting.NewMockModelNamed(t, "some-model-b")

	languageA := languagetesting.NewMockLanguageNamed(t, "some-language-a")
	languageB := languagetesting.NewMockLanguageNamed(t, "some-language-b")

	validate(t, &testCase{
		Name: "Collapse",

		Assessments: AssessmentPerModelPerLanguagePerRepository{
			modelA: {
				languageA: {
					"some-repository-a": metrics.Assessments{
						metrics.AssessmentKeyResponseNoExcess: 1,
					},
					"some-repository-b": metrics.Assessments{
						metrics.AssessmentKeyResponseNoExcess: 2,
					},
				},
				languageB: {
					"some-repository-a": metrics.Assessments{
						metrics.AssessmentKeyResponseNoExcess: 3,
					},
					"some-repository-b": metrics.Assessments{
						metrics.AssessmentKeyResponseNoExcess: 4,
					},
				},
			},
			modelB: {
				languageA: {
					"some-repository-a": metrics.Assessments{
						metrics.AssessmentKeyResponseNoExcess: 5,
					},
					"some-repository-b": metrics.Assessments{
						metrics.AssessmentKeyResponseNoExcess: 6,
					},
				},
				languageB: {
					"some-repository-a": metrics.Assessments{
						metrics.AssessmentKeyResponseNoExcess: 7,
					},
					"some-repository-b": metrics.Assessments{
						metrics.AssessmentKeyResponseNoExcess: 8,
					},
				},
			},
		},

		ExpectedAssessmentPerLanguagePerModel: AssessmentPerLanguagePerModel{
			languageA: map[model.Model]metrics.Assessments{
				modelA: {
					metrics.AssessmentKeyResponseNoExcess: 3,
				},
				modelB: {
					metrics.AssessmentKeyResponseNoExcess: 11,
				},
			},
			languageB: map[model.Model]metrics.Assessments{
				modelA: {
					metrics.AssessmentKeyResponseNoExcess: 7,
				},
				modelB: {
					metrics.AssessmentKeyResponseNoExcess: 15,
				},
			},
		},
	})
}
