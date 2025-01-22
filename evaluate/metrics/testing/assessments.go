package metricstesting

import (
	"maps"

	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	"github.com/symflower/eval-dev-quality/task"
)

// Clean deletes all empty and nondeterministic keys from the assessment.
func Clean(assessment metrics.Assessments) metrics.Assessments {
	c := metrics.Assessments{}
	maps.Copy(c, assessment)

	delete(c, metrics.AssessmentKeyProcessingTime)

	for _, key := range metrics.AllAssessmentKeysStrings {
		if c[metrics.AssessmentKey(key)] == 0 {
			delete(c, metrics.AssessmentKey(key))
		}
	}

	return c
}

// CleanSlice deletes all empty and nondeterministic keys from the assessments.
func CleanSlice(assessments []metrics.Assessments) []metrics.Assessments {
	c := make([]metrics.Assessments, len(assessments))

	for i, assessment := range assessments {
		c[i] = Clean(assessment)
	}

	return c
}

// CleanMap deletes all empty and nondeterministic keys from the assessments.
func CleanMap[E comparable](assessments map[E]metrics.Assessments) map[E]metrics.Assessments {
	c := map[E]metrics.Assessments{}

	for key, assessment := range assessments {
		c[key] = Clean(assessment)
	}

	return c
}

// AssessmentsWithProcessingTime is an empty assessment collection with positive processing time.
var AssessmentsWithProcessingTime = metrics.Assessments{
	metrics.AssessmentKeyProcessingTime: 1,
}

// AssessmentTuple holds all parameters uniquely defining to which run an assessment belongs to.
type AssessmentTuple struct {
	Model          string
	Language       string
	RepositoryPath string
	Case           string
	Task           task.Identifier
	Assessment     metrics.Assessments
}

// AssessmentTuples holds a list of all parameters uniquely defining to which run an assessment belongs to.
type AssessmentTuples []*AssessmentTuple
