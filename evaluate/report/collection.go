package report

import (
	"cmp"
	"slices"
	"sort"

	"golang.org/x/exp/maps"

	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	"github.com/symflower/eval-dev-quality/language"
	"github.com/symflower/eval-dev-quality/model"
	"github.com/symflower/eval-dev-quality/task"
)

// AssessmentPerLanguagePerModel holds a collection of assessments per language and model.
type AssessmentPerLanguagePerModel map[language.Language]AssessmentPerModel

// AssessmentPerModel holds a collection of assessments per model.
type AssessmentPerModel map[model.Model]metrics.Assessments

// WalkByScore walks the given assessment metrics by their score.
func (a AssessmentPerModel) WalkByScore(function func(model model.Model, assessment metrics.Assessments, score uint64) error) (err error) {
	models := maps.Keys(a)
	slices.SortStableFunc(models, func(a, b model.Model) int {
		return cmp.Compare(a.ID(), b.ID())
	})

	scores := make(map[model.Model]uint64, len(models))
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

// AssessmentPerModelPerLanguagePerRepositoryPerTask holds a collection of assessments per model per language and per repository.
type AssessmentPerModelPerLanguagePerRepositoryPerTask map[model.Model]map[language.Language]map[string]map[task.Identifier]metrics.Assessments

// NewAssessmentPerModelPerLanguagePerRepositoryPerTask returns a new AssessmentPerModelPerLanguagePerRepository initialized with an empty assessment for each combination.
func NewAssessmentPerModelPerLanguagePerRepositoryPerTask() (assessments AssessmentPerModelPerLanguagePerRepositoryPerTask) {
	return AssessmentPerModelPerLanguagePerRepositoryPerTask{}
}

// Add adds a new assessment.
func (a AssessmentPerModelPerLanguagePerRepositoryPerTask) Add(model model.Model, l language.Language, repositoryPath string, taskIdentifier task.Identifier, assessment metrics.Assessments) {
	perLanguageLookup, ok := a[model]
	if !ok {
		perLanguageLookup = map[language.Language]map[string]map[task.Identifier]metrics.Assessments{}
		a[model] = perLanguageLookup
	}

	perRepositoryLookup, ok := perLanguageLookup[l]
	if !ok {
		perRepositoryLookup = map[string]map[task.Identifier]metrics.Assessments{}
		perLanguageLookup[l] = perRepositoryLookup
	}

	perTaskLookup, ok := a[model][l][repositoryPath]
	if !ok {
		perTaskLookup = map[task.Identifier]metrics.Assessments{}
		a[model][l][repositoryPath] = perTaskLookup
	}

	assessments, ok := perTaskLookup[taskIdentifier]
	if !ok {
		assessments = metrics.NewAssessments()
		a[model][l][repositoryPath][taskIdentifier] = assessments
	}

	assessments.Add(assessment)
}

// Walk walks over all entries.
func (a AssessmentPerModelPerLanguagePerRepositoryPerTask) Walk(function func(m model.Model, l language.Language, r string, t task.Identifier, a metrics.Assessments) error) (err error) {
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
				taskIdentifiers := maps.Keys(a[m][l][r])
				for _, t := range taskIdentifiers {
					if err := function(m, l, r, t, a[m][l][r][t]); err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

// CollapseByModel returns all assessments aggregated per model ID.
func (a AssessmentPerModelPerLanguagePerRepositoryPerTask) CollapseByModel() AssessmentPerModel {
	perModel := make(AssessmentPerModel, len(a))
	for _, m := range maps.Keys(a) {
		perModel[m] = metrics.NewAssessments()
	}
	_ = a.Walk(func(m model.Model, l language.Language, r string, t task.Identifier, a metrics.Assessments) (err error) {
		perModel[m].Add(a)

		return nil
	})

	return perModel
}

// CollapseByLanguage returns all assessments aggregated per language and model.
func (a AssessmentPerModelPerLanguagePerRepositoryPerTask) CollapseByLanguage() AssessmentPerLanguagePerModel {
	assessments := AssessmentPerLanguagePerModel{}
	_ = a.Walk(func(m model.Model, l language.Language, r string, t task.Identifier, a metrics.Assessments) (err error) {
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
