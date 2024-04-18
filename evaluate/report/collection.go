package report

import (
	"cmp"
	"slices"
	"sort"

	"golang.org/x/exp/maps"

	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	"github.com/symflower/eval-dev-quality/language"
	"github.com/symflower/eval-dev-quality/model"
)

// AssessmentPerModelPerLanguagePerRepository holds a collection of assessments per model per language and per repository.
type AssessmentPerModelPerLanguagePerRepository map[model.Model]map[language.Language]map[string]metrics.Assessments

// NewAssessmentPerModelPerLanguagePerRepository returns a new AssessmentPerModelPerLanguagePerRepository initialized with an empty assessment for each combination.
func NewAssessmentPerModelPerLanguagePerRepository(models []model.Model, languages []language.Language, repositories []string) AssessmentPerModelPerLanguagePerRepository {
	a := AssessmentPerModelPerLanguagePerRepository{}
	for _, m := range models {
		if _, ok := a[m]; !ok {
			a[m] = map[language.Language]map[string]metrics.Assessments{}
		}
		for _, l := range languages {
			if _, ok := a[m][l]; !ok {
				a[m][l] = map[string]metrics.Assessments{}
			}
			for _, r := range repositories {
				a[m][l][r] = metrics.NewAssessments()
			}
		}
	}

	return a
}

// Walk walks over all entries.
func (a AssessmentPerModelPerLanguagePerRepository) Walk(function func(m model.Model, l language.Language, r string, a metrics.Assessments) error) error {
	models := maps.Keys(a)
	slices.SortStableFunc(models, func(a, b model.Model) int {
		return cmp.Compare(a.ID(), b.ID())
	})
	for _, m := range models {
		languages := maps.Keys(a[m])
		slices.SortStableFunc(languages, func(a, b language.Language) int {
			return cmp.Compare(a.ID(), b.ID())
		})
		for _, l := range languages {
			repositories := maps.Keys(a[m][l])
			sort.Strings(repositories)
			for _, r := range repositories {
				if err := function(m, l, r, a[m][l][r]); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// Collapse returns all assessments aggregated per model ID.
func (a AssessmentPerModelPerLanguagePerRepository) Collapse() map[string]metrics.Assessments {
	perModel := make(map[string]metrics.Assessments, len(a))
	for _, m := range maps.Keys(a) {
		perModel[m.ID()] = metrics.NewAssessments()
	}
	_ = a.Walk(func(m model.Model, l language.Language, r string, a metrics.Assessments) error {
		perModel[m.ID()].Add(a)

		return nil
	})

	return perModel
}
