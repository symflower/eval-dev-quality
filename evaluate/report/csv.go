package report

import (
	"cmp"
	"encoding/csv"
	"io"
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

// EvaluationFile holds the evaluation CSV file writer.
type EvaluationFile struct {
	// Holds the writer where the evaluation CSV is written to.
	io.Writer
}

// NewEvaluationFile initializes an evaluation file and writes the corresponding CSV header.
func NewEvaluationFile(writer io.Writer) (evaluationFile *EvaluationFile, err error) {
	evaluationFile = &EvaluationFile{
		Writer: writer,
	}

	var out strings.Builder
	csv := csv.NewWriter(&out)

	if err := csv.Write(evaluationHeader()); err != nil {
		return nil, pkgerrors.WithStack(err)
	}
	csv.Flush()

	if _, err = evaluationFile.Writer.Write([]byte(out.String())); err != nil {
		return nil, pkgerrors.WithStack(err)
	}

	return evaluationFile, nil
}

// WriteEvaluationRecord writes the assessments of a task into the evaluation CSV.
func (e *EvaluationFile) WriteEvaluationRecord(model model.Model, language language.Language, repositoryName string, assessmentsPerTask map[task.Identifier]metrics.Assessments) (err error) {
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

	if _, err := e.Writer.Write([]byte(out.String())); err != nil {
		return pkgerrors.WithStack(err)
	}

	return nil
}

// evaluationHeader returns the CSV header for the evaluation CSV.
func evaluationHeader() (header []string) {
	return append([]string{"model-id", "model-name", "cost", "language", "repository", "task", "score"}, metrics.AllAssessmentKeysStrings...)
}
