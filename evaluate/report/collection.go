package report

import (
	"cmp"
	"os"
	"slices"
	"sort"
	"strings"

	"golang.org/x/exp/maps"

	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	"github.com/symflower/eval-dev-quality/language"
	"github.com/symflower/eval-dev-quality/model"
)

// AssessmentPerLanguagePerModel holds a collection of assessments per language and model.
type AssessmentPerLanguagePerModel map[language.Language]AssessmentPerModel

// AssessmentPerModel holds a collection of assessments per model.
type AssessmentPerModel map[model.Model]metrics.Assessments

// WalkByScore walks the given assessment metrics by their score.
func (a AssessmentPerModel) WalkByScore(function func(model model.Model, assessment metrics.Assessments, score uint) error) error {
	models := maps.Keys(a)
	slices.SortStableFunc(models, func(a, b model.Model) int {
		return cmp.Compare(a.ID(), b.ID())
	})

	scores := make(map[model.Model]uint, len(models))
	for _, model := range models {
		scores[model] = a[model].Score()
	}
	sort.SliceStable(models, func(i, j int) bool {
		return scores[models[i]] < scores[models[j]]
	})

	for _, model := range models {
		if err := function(model, a[model], scores[model]); err != nil {
			return err
		}
	}

	return nil
}

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
				// Ensure the repository path matches the language.
				if !strings.HasPrefix(r, l.ID()+string(os.PathSeparator)) {
					continue
				}

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

// CollapseByModel returns all assessments aggregated per model ID.
func (a AssessmentPerModelPerLanguagePerRepository) CollapseByModel() AssessmentPerModel {
	perModel := make(AssessmentPerModel, len(a))
	for _, m := range maps.Keys(a) {
		perModel[m] = metrics.NewAssessments()
	}
	_ = a.Walk(func(m model.Model, l language.Language, r string, a metrics.Assessments) error {
		perModel[m].Add(a)

		return nil
	})

	return perModel
}

// CollapseByLanguage returns all assessments aggregated per language and model.
func (a AssessmentPerModelPerLanguagePerRepository) CollapseByLanguage() AssessmentPerLanguagePerModel {
	assessments := AssessmentPerLanguagePerModel{}
	_ = a.Walk(func(m model.Model, l language.Language, r string, a metrics.Assessments) error {
		if _, ok := assessments[l]; !ok {
			assessments[l] = map[model.Model]metrics.Assessments{}
		}

		if _, ok := assessments[l][m]; !ok {
			assessments[l][m] = metrics.NewAssessments()
		}

		assessments[l][m].Add(a)

		return nil
	})

	return assessments
}
