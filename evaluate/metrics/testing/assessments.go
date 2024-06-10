package metricstesting

import (
	"maps"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	"github.com/symflower/eval-dev-quality/language"
	"github.com/symflower/eval-dev-quality/model"
	"github.com/symflower/eval-dev-quality/task"
)

// AssertAssessmentsEqual checks if the given assessments are equal ignoring default and nondeterministic values.
func AssertAssessmentsEqual(t *testing.T, expected metrics.Assessments, actual metrics.Assessments) {
	expected = maps.Clone(expected)
	actual = maps.Clone(actual)

	expected[metrics.AssessmentKeyProcessingTime] = 0
	actual[metrics.AssessmentKeyProcessingTime] = 0

	expected[metrics.AssessmentKeyGenerateTestsForFileCharacterCount] = 0
	actual[metrics.AssessmentKeyGenerateTestsForFileCharacterCount] = 0
	expected[metrics.AssessmentKeyResponseCharacterCount] = 0
	actual[metrics.AssessmentKeyResponseCharacterCount] = 0

	assert.Truef(t, expected.Equal(actual), "expected:%s\nactual:%s", expected, actual)
}

// AssessmentsWithProcessingTime is an empty assessment collection with positive processing time.
var AssessmentsWithProcessingTime = metrics.Assessments{
	metrics.AssessmentKeyProcessingTime: 1,
}

// AssessmentTuple holds all parameters uniquely defining to which run an assessment belongs to.
type AssessmentTuple struct {
	Model          model.Model
	Language       language.Language
	RepositoryPath string
	Task           task.Identifier
	Assessment     metrics.Assessments
}

type AssessmentTuples []*AssessmentTuple

func (at AssessmentTuples) ToMap() (lookup map[model.Model]map[language.Language]map[string]map[task.Identifier]metrics.Assessments) {
	lookup = map[model.Model]map[language.Language]map[string]map[task.Identifier]metrics.Assessments{}
	for _, t := range at {
		perLanguageLookup, ok := lookup[t.Model]
		if !ok {
			perLanguageLookup = map[language.Language]map[string]map[task.Identifier]metrics.Assessments{}
			lookup[t.Model] = perLanguageLookup
		}

		perRepositoryLookup, ok := perLanguageLookup[t.Language]
		if !ok {
			perRepositoryLookup = map[string]map[task.Identifier]metrics.Assessments{}
			perLanguageLookup[t.Language] = perRepositoryLookup
		}

		perTaskLookup, ok := perRepositoryLookup[t.RepositoryPath]
		if !ok {
			perTaskLookup = map[task.Identifier]metrics.Assessments{}
			perRepositoryLookup[t.RepositoryPath] = perTaskLookup
		}

		assessments, ok := perTaskLookup[t.Task]
		if !ok {
			assessments = metrics.NewAssessments()
			perTaskLookup[t.Task] = assessments
		}

		assessments.Add(t.Assessment)
	}

	return lookup
}
