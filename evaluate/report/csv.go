package report

import (
	"encoding/csv"
	"strconv"
	"strings"

	pkgerrors "github.com/pkg/errors"

	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	"github.com/symflower/eval-dev-quality/language"
	"github.com/symflower/eval-dev-quality/model"
)

// csvHeader returns the header description as a CSV row.
func csvHeader() []string {
	return append([]string{"model", "language", "repository", "score"}, metrics.AllAssessmentKeysStrings...)
}

// FormatCSV formats the given assessment metrics as CSV.
func FormatCSV(assessments AssessmentPerModelPerLanguagePerRepository) (string, error) {
	var out strings.Builder
	csv := csv.NewWriter(&out)

	if err := csv.Write(csvHeader()); err != nil {
		return "", pkgerrors.WithStack(err)
	}

	if err := assessments.Walk(func(m model.Model, l language.Language, r string, a metrics.Assessments) error {
		row := a.StringCSV()
		score := a.Score()

		if err := csv.Write(append([]string{m.ID(), l.ID(), r, strconv.FormatUint(uint64(score), 10)}, row...)); err != nil {
			return pkgerrors.WithStack(err)
		}

		return nil
	}); err != nil {
		return "", err
	}
	csv.Flush()

	return out.String(), nil
}
