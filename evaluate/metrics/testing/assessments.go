package metricstesting

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/symflower/eval-dev-quality/evaluate/metrics"
)

// AssertAssessmentsEqual checks if the given assessments are equal ignoring default values.
func AssertAssessmentsEqual(t *testing.T, expected metrics.Assessments, actual metrics.Assessments) {
	assert.Truef(t, expected.Equal(actual), "expected:%s\nactual:%s", expected, actual)
}
