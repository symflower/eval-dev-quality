package testing

import (
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	metricstesting "github.com/symflower/eval-dev-quality/evaluate/metrics/testing"
	"github.com/symflower/eval-dev-quality/task"
)

func parseFloat64(t *testing.T, s string) float64 {
	value, err := strconv.ParseFloat(s, 64)
	assert.NoErrorf(t, err, "parsing unsigned integer from: %q", s)

	return value
}

// ParseMetrics extracts multiple assessment metrics from the given string.
func ParseMetrics(t *testing.T, data string) (assessments metricstesting.AssessmentTuples) {
	lines := strings.Split(strings.TrimSpace(data), "\n")
	if len(lines) < 2 {
		return assessments
	}

	for _, line := range lines[1:] {
		cells := strings.Split(line, ",")

		tuple := &metricstesting.AssessmentTuple{
			Model:          cells[0],
			Language:       cells[1],
			RepositoryPath: cells[2],
			Case:           cells[3],
			Task:           task.Identifier(cells[4]),
			Assessment:     metrics.Assessments{},
		}
		for i, key := range metrics.AllAssessmentKeys {
			tuple.Assessment[key] = parseFloat64(t, cells[i+6])
		}

		assessments = append(assessments, tuple)
	}

	return assessments
}
