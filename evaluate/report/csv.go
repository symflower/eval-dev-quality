package report

import (
	"cmp"
	"encoding/csv"
	"io"
	"slices"
	"sort"
	"strconv"

	pkgerrors "github.com/pkg/errors"
	"golang.org/x/exp/maps"

	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	"github.com/symflower/eval-dev-quality/language"
	"github.com/symflower/eval-dev-quality/model"
	"github.com/symflower/eval-dev-quality/task"
)

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

	csv := csv.NewWriter(writer)

	if err := csv.Write(EvaluationHeader()); err != nil {
		return nil, pkgerrors.WithStack(err)
	}
	csv.Flush()

	return evaluationFile, nil
}

// WriteEvaluationRecord writes the assessments of a task into the evaluation CSV.
func (e *EvaluationFile) WriteEvaluationRecord(model model.Model, language language.Language, repositoryName string, run uint, assessmentsPerCasePerTask map[string]map[task.Identifier]metrics.Assessments) (err error) {
	allRecords := [][]string{}

	cases := maps.Keys(assessmentsPerCasePerTask)
	sort.Strings(cases)
	for _, caseName := range cases {
		assessmentsPerTask := assessmentsPerCasePerTask[caseName]

		tasks := maps.Keys(assessmentsPerTask)
		slices.SortStableFunc(tasks, func(a, b task.Identifier) int {
			return cmp.Compare(a, b)
		})

		for _, task := range tasks {
			assessment := assessmentsPerTask[task]
			row := append([]string{model.ID(), language.ID(), repositoryName, caseName, string(task), strconv.FormatUint(uint64(run), 10)}, assessment.StringCSV()...)
			allRecords = append(allRecords, row)
		}
	}

	return e.WriteLines(allRecords)
}

// WriteLines takes a slice of raw records and writes them into the evaluation file.
func (e *EvaluationFile) WriteLines(records [][]string) (err error) {
	if len(records) == 0 {
		return nil
	}

	csv := csv.NewWriter(e.Writer)
	if err := csv.WriteAll(records); err != nil {
		return pkgerrors.WithStack(err)
	}
	csv.Flush()

	return nil
}

// EvaluationHeader returns the CSV header for the evaluation CSV.
func EvaluationHeader() (header []string) {
	return append([]string{"model-id", "language", "repository", "case", "task", "run"}, metrics.AllAssessmentKeysStrings...)
}
