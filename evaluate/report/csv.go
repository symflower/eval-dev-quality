package report

import (
	"cmp"
	"encoding/csv"
	"io"
	"os"
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
func (e *EvaluationFile) WriteEvaluationRecord(model model.Model, language language.Language, repositoryName string, assessmentsPerTask map[task.Identifier]metrics.Assessments) (err error) {
	csv := csv.NewWriter(e.Writer)

	tasks := maps.Keys(assessmentsPerTask)
	slices.SortStableFunc(tasks, func(a, b task.Identifier) int {
		return cmp.Compare(a, b)
	})

	for _, task := range tasks {
		assessment := assessmentsPerTask[task]
		row := append([]string{model.ID(), language.ID(), repositoryName, string(task), strconv.FormatUint(uint64(assessment.Score()), 10)}, assessment.StringCSV()...)
		csv.Write(row)
	}
	csv.Flush()

	return nil
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

// evaluationHeader returns the CSV header for the evaluation CSV.
func EvaluationHeader() (header []string) {
	return append([]string{"model-id", "language", "repository", "task", "score"}, metrics.AllAssessmentKeysStrings...)
}

// RecordsFromEvaluationCSVFiles returns all the records from all the given evaluation CSV files.
func RecordsFromEvaluationCSVFiles(evaluationCSVFilePaths []string) (records [][]string, err error) {
	for _, evaluationCSVFilePath := range evaluationCSVFilePaths {
		file, err := os.Open(evaluationCSVFilePath)
		if err != nil {
			return nil, pkgerrors.WithStack(err)
		}
		defer file.Close()

		csv := csv.NewReader(file)

		// Ignore the CSV header.
		csv.Read()

		evaluationRecords, err := csv.ReadAll()
		if err != nil {
			return nil, pkgerrors.WithStack(err)
		}
		records = append(records, evaluationRecords...)
	}

	return records, nil
}

// SortEvaluationRecords sorts the evaluation records.
func SortEvaluationRecords(records [][]string) {
	sort.Slice(records, func(i, j int) bool {
		for x := range records[i] {
			if records[i][x] == records[j][x] {
				continue
			}

			return records[i][x] < records[j][x]
		}

		return false
	})
}
