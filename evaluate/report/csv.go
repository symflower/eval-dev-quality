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

// CSVFormatter defines a formatter for CSV data.
type CSVFormatter interface {
	// Header returns the header description as a CSV row.
	Header() (header []string)
	// Rows returns all data as CSV rows.
	Rows() (rows [][]string)
}

// GenerateCSV returns the whole CSV as string.
func GenerateCSV(formatter CSVFormatter) (csvData string, err error) {
	var out strings.Builder
	csv := csv.NewWriter(&out)

	if err := csv.Write(formatter.Header()); err != nil {
		return "", pkgerrors.WithStack(err)
	}

	for _, row := range formatter.Rows() {
		if err := csv.Write(row); err != nil {
			return "", pkgerrors.WithStack(err)
		}
	}

	csv.Flush()

	return out.String(), nil
}

// Header returns the header description as a CSV row.
func (a AssessmentPerModelPerLanguagePerRepository) Header() (header []string) {
	return append([]string{"model", "language", "repository", "score"}, metrics.AllAssessmentKeysStrings...)
}

// Rows returns all data as CSV rows.
func (a AssessmentPerModelPerLanguagePerRepository) Rows() (rows [][]string) {
	_ = a.Walk(func(m model.Model, l language.Language, r string, a metrics.Assessments) error {
		metrics := a.StringCSV()
		score := a.Score()

		row := append([]string{m.ID(), l.ID(), r, strconv.FormatUint(uint64(score), 10)}, metrics...)
		rows = append(rows, row)

		return nil
	})

	return rows
}

// Header returns the header description as a CSV row.
func (a AssessmentPerModel) Header() (header []string) {
	return append([]string{"model", "score"}, metrics.AllAssessmentKeysStrings...)
}

// Rows returns all data as CSV rows.
func (a AssessmentPerModel) Rows() (rows [][]string) {
	for model, assessment := range a {
		metrics := assessment.StringCSV()
		score := assessment.Score()

		row := append([]string{model.ID(), strconv.FormatUint(uint64(score), 10)}, metrics...)
		rows = append(rows, row)
	}

	return rows
}
