package testing

import (
	"regexp"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	metricstesting "github.com/symflower/eval-dev-quality/evaluate/metrics/testing"
	"github.com/symflower/eval-dev-quality/task"
)

func atoiUint64(t *testing.T, s string) uint64 {
	value, err := strconv.ParseUint(s, 10, 64)
	assert.NoErrorf(t, err, "parsing unsigned integer from: %q", s)

	return uint64(value)
}

// extractMetricsCSVMatch is a regular expression to extract metrics from CSV rows.
var extractMetricsCSVMatch = regexp.MustCompile(`(\S+),(\S+),(\S+),(\S+),\d+,(\d+),(\d+),(\d+),(\d+),(\d+),(\d+),(\d+),(\d+),(\d+),(\d+)`)

// ParseMetrics extracts multiple assessment metrics from the given string.
func ParseMetrics(t *testing.T, data string) (assessments metricstesting.AssessmentTuples) {
	matches := extractMetricsCSVMatch.FindAllStringSubmatch(data, -1)

	for _, match := range matches {
		assessments = append(assessments, &metricstesting.AssessmentTuple{
			Model:          match[1],
			Language:       match[2],
			RepositoryPath: match[3],
			Task:           task.Identifier(match[4]),
			Assessment: metrics.Assessments{
				metrics.AssessmentKeyCoverage:                           atoiUint64(t, match[5]),
				metrics.AssessmentKeyFilesExecuted:                      atoiUint64(t, match[6]),
				metrics.AssessmentKeyFilesExecutedMaximumReachable:      atoiUint64(t, match[7]),
				metrics.AssessmentKeyGenerateTestsForFileCharacterCount: atoiUint64(t, match[8]),
				metrics.AssessmentKeyProcessingTime:                     atoiUint64(t, match[9]),
				metrics.AssessmentKeyResponseCharacterCount:             atoiUint64(t, match[10]),
				metrics.AssessmentKeyResponseNoError:                    atoiUint64(t, match[11]),
				metrics.AssessmentKeyResponseNoExcess:                   atoiUint64(t, match[12]),
				metrics.AssessmentKeyResponseWithCode:                   atoiUint64(t, match[13]),
			},
		})
	}

	return assessments
}
