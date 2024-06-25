package report

import (
	"cmp"
	"encoding/csv"
	"slices"
	"strconv"
	"strings"

	pkgerrors "github.com/pkg/errors"
	"golang.org/x/exp/maps"

	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	"github.com/symflower/eval-dev-quality/language"
	"github.com/symflower/eval-dev-quality/model"
	"github.com/symflower/eval-dev-quality/task"
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
func (a *AssessmentStore) Header() (header []string) {
	return append([]string{"model", "cost", "language", "repository", "task", "score"}, metrics.AllAssessmentKeysStrings...)
}

// Rows returns all data as CSV rows.
func (a *AssessmentStore) Rows() (rows [][]string) {
	_ = a.Walk(func(m model.Model, l language.Language, r string, t task.Identifier, a metrics.Assessments) (err error) {
		metrics := a.StringCSV()
		score := a.Score()
		cost := m.Cost()

		row := append([]string{m.ID(), strconv.FormatFloat(cost, 'f', -1, 64), l.ID(), r, string(t), strconv.FormatUint(uint64(score), 10)}, metrics...)
		rows = append(rows, row)

		return nil
	})

	return rows
}

// Header returns the header description as a CSV row.
func (a AssessmentPerModel) Header() (header []string) {
	return append([]string{"model", "cost", "score"}, metrics.AllAssessmentKeysStrings...)
}

// Rows returns all data as CSV rows.
func (a AssessmentPerModel) Rows() (rows [][]string) {
	models := maps.Keys(a)
	slices.SortStableFunc(models, func(a, b model.Model) int {
		return cmp.Compare(a.ID(), b.ID())
	})

	for _, model := range models {
		metrics := a[model].StringCSV()
		score := a[model].Score()
		cost := model.Cost()

		row := append([]string{model.ID(), strconv.FormatFloat(cost, 'f', -1, 64), strconv.FormatUint(uint64(score), 10)}, metrics...)
		rows = append(rows, row)
	}

	return rows
}
