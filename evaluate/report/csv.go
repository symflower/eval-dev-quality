package report

import (
	"encoding/csv"
	"strconv"
	"strings"

	pkgerrors "github.com/pkg/errors"

	"github.com/symflower/eval-dev-quality/evaluate/metrics"
)

// csvHeader returns the header description as a CSV row.
func csvHeader() []string {
	return append([]string{"model", "score"}, metrics.AllAssessmentKeysStrings...)
}

// FormatCSV formats the given assessment metrics as CSV.
func FormatCSV(assessmentsPerModel map[string]metrics.Assessments) (string, error) {
	var out strings.Builder
	csv := csv.NewWriter(&out)

	if err := csv.Write(csvHeader()); err != nil {
		return "", pkgerrors.WithStack(err)
	}

	if err := metrics.WalkByScore(assessmentsPerModel, func(model string, assessment metrics.Assessments, score uint) error {
		row := assessment.StringCSV()

		if err := csv.Write(append([]string{model, strconv.FormatUint(uint64(score), 10)}, row...)); err != nil {
			return pkgerrors.WithStack(err)
		}

		return nil
	}); err != nil {
		return "", err
	}
	csv.Flush()

	return out.String(), nil
}
