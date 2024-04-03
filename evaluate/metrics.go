package evaluate

import (
	"encoding/csv"
	"fmt"
	"math"
	"sort"
	"strings"

	pkgerrors "github.com/pkg/errors"
	"golang.org/x/exp/maps"
	"gonum.org/v1/gonum/stat"
)

// Metrics holds numerical benchmarking metrics.
type Metrics struct {
	// Total is the total number of benchmarking candidates.
	Total uint
	// Executed is the number of benchmarking candidates with successful execution.
	Executed uint

	// Coverage holds the coverage of the benchmarking candidates.
	Coverage []float64
}

// Add sums two metrics objects.
func (m Metrics) Add(o Metrics) Metrics {
	return Metrics{
		Total:    m.Total + o.Total,
		Executed: m.Executed + o.Executed,

		Coverage: append(m.Coverage, o.Coverage...),
	}
}

// AverageCoverage returns the average coverage.
func (m Metrics) AverageCoverage() float64 {
	averageCoverage := stat.Mean(m.Coverage, nil)
	if math.IsNaN(averageCoverage) {
		averageCoverage = 0
	}

	return averageCoverage
}

// String returns a string representation of the metrics.
func (m Metrics) String() string {
	executedPercentage := float64(m.Executed) / float64(m.Total) * 100.0
	if math.IsNaN(executedPercentage) {
		executedPercentage = 0
	}
	return fmt.Sprintf("#executed=%3.1f%%(%d/%d), average coverage=%3.1f", executedPercentage, m.Executed, m.Total, m.AverageCoverage())
}

// StringCSV returns a CSV row string representation of the metrics.
func (m Metrics) StringCSV() []string {
	return []string{
		fmt.Sprintf("%d", m.Total),
		fmt.Sprintf("%d", m.Executed),
		fmt.Sprintf("%.0f", m.AverageCoverage()),
	}
}

// FormatStringCSV formats the given metrics as CSV.
func FormatStringCSV(metricsPerModel map[string]Metrics) (string, error) {
	var out strings.Builder
	csv := csv.NewWriter(&out)

	if err := csv.Write([]string{"model", "total", "executed", "coverage"}); err != nil {
		return "", err
	}
	categories := maps.Keys(metricsPerModel)
	sort.Strings(categories)
	for _, category := range categories {
		row := metricsPerModel[category].StringCSV()

		if err := csv.Write(append([]string{category}, row...)); err != nil {
			return "", pkgerrors.WithStack(err)
		}
	}
	csv.Flush()

	return out.String(), nil
}
