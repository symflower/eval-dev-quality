package report

import (
	"cmp"
	"encoding/csv"
	"os"
	"path/filepath"
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
func (a AssessmentPerModel) Header() (header []string) {
	return append([]string{"model-id", "model-name", "cost", "score"}, metrics.AllAssessmentKeysStrings...)
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

		row := append([]string{model.ID(), model.Name(), strconv.FormatFloat(cost, 'f', -1, 64), strconv.FormatUint(uint64(score), 10)}, metrics...)
		rows = append(rows, row)
	}

	return rows
}

// Evaluation header returns the CSV header for the evaluation CSV.
func EvaluationHeader() (header []string) {
	return append([]string{"model-id", "model-name", "cost", "language", "repository", "task", "score"}, metrics.AllAssessmentKeysStrings...)
}

// WriteHeader writes the header to the evaluation CSV file.
func WriteEvaluationHeader(resultPath string) (err error) {
	var out strings.Builder
	csv := csv.NewWriter(&out)

	if err := csv.Write(EvaluationHeader()); err != nil {
		return pkgerrors.WithStack(err)
	}
	csv.Flush()

	if err = os.WriteFile(filepath.Join(resultPath, "evaluation.csv"), []byte(out.String()), 0644); err != nil {
		return pkgerrors.WithStack(err)
	}

	return nil
}

// WriteEvaluationRecord writes the assessments of a task into the evaluation CSV.
func WriteEvaluationRecord(resultPath string, model model.Model, language language.Language, repositoryName string, assessmentsPerTask map[task.Identifier]metrics.Assessments) (err error) {
	var out strings.Builder
	csv := csv.NewWriter(&out)

	tasks := maps.Keys(assessmentsPerTask)
	slices.SortStableFunc(tasks, func(a, b task.Identifier) int {
		return cmp.Compare(a, b)
	})

	for _, task := range tasks {
		assessment := assessmentsPerTask[task]
		row := append([]string{model.ID(), model.Name(), strconv.FormatFloat(model.Cost(), 'f', -1, 64), language.ID(), repositoryName, string(task), strconv.FormatUint(uint64(assessment.Score()), 10)}, assessment.StringCSV()...)
		csv.Write(row)
	}
	csv.Flush()

	evaluationFile, err := os.OpenFile(filepath.Join(resultPath, "evaluation.csv"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return pkgerrors.WithStack(err)
	}
	defer evaluationFile.Close()

	if _, err := evaluationFile.WriteString(out.String()); err != nil {
		return pkgerrors.WithStack(err)
	}

	return nil
}
