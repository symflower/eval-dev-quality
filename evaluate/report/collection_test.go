package report

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	metricstesting "github.com/symflower/eval-dev-quality/evaluate/metrics/testing"
	evaluatetask "github.com/symflower/eval-dev-quality/evaluate/task"
	"github.com/symflower/eval-dev-quality/language"
	languagetesting "github.com/symflower/eval-dev-quality/language/testing"
	"github.com/symflower/eval-dev-quality/model"
	modeltesting "github.com/symflower/eval-dev-quality/model/testing"
	"github.com/symflower/eval-dev-quality/task"
)

func TestAssessmentPerModelPerLanguagePerRepositoryWalk(t *testing.T) {
	type testCase struct {
		Name string

		Assessments metricstesting.AssessmentTuples

		ExpectedOrder []metrics.Assessments
	}

	validate := func(t *testing.T, tc *testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			assessmentStore := assessmentTuplesToStore(t, tc.Assessments)

			assessmentLookup := tc.Assessments.ToMap()
			actualOrder := []metrics.Assessments{}

			assert.NoError(t, assessmentStore.Walk(func(m model.Model, l language.Language, r string, ti task.Identifier, a metrics.Assessments) (err error) {
				actualOrder = append(actualOrder, a)
				assert.Equal(t, metricstesting.Clean(assessmentLookup[m.ID()][l.ID()][r][ti]), metricstesting.Clean(a))

				return nil
			}))

			assert.Equal(t, metricstesting.CleanSlice(tc.ExpectedOrder), metricstesting.CleanSlice(actualOrder))
		})
	}

	validate(t, &testCase{
		Name: "Single Group",

		Assessments: metricstesting.AssessmentTuples{
			&metricstesting.AssessmentTuple{
				Model:          "some-model",
				Language:       "some-language",
				RepositoryPath: "some-repository",
				Task:           evaluatetask.IdentifierWriteTests,
				Assessment: metrics.Assessments{
					metrics.AssessmentKeyResponseNoExcess: 1,
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

		Assessments: metricstesting.AssessmentTuples{
			&metricstesting.AssessmentTuple{
				Model:          "some-model-a",
				Language:       "some-language-a",
				RepositoryPath: "some-repository-a",
				Task:           evaluatetask.IdentifierWriteTests,
				Assessment: metrics.Assessments{
					metrics.AssessmentKeyResponseNoExcess: 1,
				},
			},
			&metricstesting.AssessmentTuple{
				Model:          "some-model-a",
				Language:       "some-language-a",
				RepositoryPath: "some-repository-b",
				Task:           evaluatetask.IdentifierWriteTests,
				Assessment: metrics.Assessments{
					metrics.AssessmentKeyResponseNoExcess: 2,
				},
			},
			&metricstesting.AssessmentTuple{
				Model:          "some-model-a",
				Language:       "some-language-b",
				RepositoryPath: "some-repository-a",
				Task:           evaluatetask.IdentifierWriteTests,
				Assessment: metrics.Assessments{
					metrics.AssessmentKeyResponseNoExcess: 3,
				},
			},
			&metricstesting.AssessmentTuple{
				Model:          "some-model-a",
				Language:       "some-language-b",
				RepositoryPath: "some-repository-b",
				Task:           evaluatetask.IdentifierWriteTests,
				Assessment: metrics.Assessments{
					metrics.AssessmentKeyResponseNoExcess: 4,
				},
			},
			&metricstesting.AssessmentTuple{
				Model:          "some-model-b",
				Language:       "some-language-a",
				RepositoryPath: "some-repository-a",
				Task:           evaluatetask.IdentifierWriteTests,
				Assessment: metrics.Assessments{
					metrics.AssessmentKeyResponseNoExcess: 5,
				},
			},
			&metricstesting.AssessmentTuple{
				Model:          "some-model-b",
				Language:       "some-language-a",
				RepositoryPath: "some-repository-b",
				Task:           evaluatetask.IdentifierWriteTests,
				Assessment: metrics.Assessments{
					metrics.AssessmentKeyResponseNoExcess: 6,
				},
			},
			&metricstesting.AssessmentTuple{
				Model:          "some-model-b",
				Language:       "some-language-b",
				RepositoryPath: "some-repository-a",
				Task:           evaluatetask.IdentifierWriteTests,
				Assessment: metrics.Assessments{
					metrics.AssessmentKeyResponseNoExcess: 7,
				},
			},
			&metricstesting.AssessmentTuple{
				Model:          "some-model-b",
				Language:       "some-language-b",
				RepositoryPath: "some-repository-b",
				Task:           evaluatetask.IdentifierWriteTests,
				Assessment: metrics.Assessments{
					metrics.AssessmentKeyResponseNoExcess: 8,
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

func assessmentTuplesToStore(t *testing.T, at metricstesting.AssessmentTuples) (store *AssessmentStore) {
	store = NewAssessmentStore()
	modelMocks := map[string]model.Model{}
	languageMocks := map[string]language.Language{}

	for _, a := range at {
		m := modelMocks[a.Model]
		if m == nil {
			m = modeltesting.NewMockModelNamed(t, a.Model)
			modelMocks[a.Model] = m
		}

		l := languageMocks[a.Language]
		if l == nil {
			l = languagetesting.NewMockLanguageNamed(t, a.Language)
			languageMocks[a.Language] = l
		}

		store.Add(m, l, a.RepositoryPath, a.Task, a.Assessment)
	}

	return store
}
