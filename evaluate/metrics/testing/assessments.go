package metricstesting

import (
	"maps"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/symflower/eval-dev-quality/evaluate/metrics"
)

// AssertAssessmentsEqual checks if the given assessments are equal ignoring default and nondeterministic values.
func AssertAssessmentsEqual(t *testing.T, expected metrics.Assessments, actual metrics.Assessments) {
	expected = maps.Clone(expected)
	actual = maps.Clone(actual)

	expected[metrics.AssessmentKeyProcessingTime] = 0
	actual[metrics.AssessmentKeyProcessingTime] = 0

	assert.Truef(t, expected.Equal(actual), "expected:%s\nactual:%s", expected, actual)
}
