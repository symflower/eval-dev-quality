package report

import (
	"cmp"
	"encoding/csv"
	"io"
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

	if err := csv.Write(evaluationHeader()); err != nil {
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
		row := append([]string{model.ID(), model.Name(), strconv.FormatFloat(model.Cost(), 'f', -1, 64), language.ID(), repositoryName, string(task), strconv.FormatUint(uint64(assessment.Score()), 10)}, assessment.StringCSV()...)
		csv.Write(row)
	}
	csv.Flush()

	return nil
}

// evaluationHeader returns the CSV header for the evaluation CSV.
func evaluationHeader() (header []string) {
	return append([]string{"model-id", "model-name", "cost", "language", "repository", "task", "score"}, metrics.AllAssessmentKeysStrings...)
}

// EvaluationRecord holds a line of the evaluation CSV.
type EvaluationRecord struct {
	// ModelID holds the model id.
	ModelID string
	// ModelName holds the model name.
	ModelName string
	// ModelCost holds the model cost.
	ModelCost float64

	// LanguageID holds the language id.
	LanguageID string

	// RepositoryName holds the name of a repository .
	RepositoryName string
	// Task holds the task identifier.
	Task string

	// Assessments holds the assessments of an entry.
	Assessments metrics.Assessments
}

// Clone clones an evaluation record.
func (e *EvaluationRecord) Clone() (new *EvaluationRecord) {
	new = &EvaluationRecord{}

	new.ModelID = e.ModelID
	new.ModelName = e.ModelName
	new.ModelCost = e.ModelCost
	new.LanguageID = e.LanguageID
	new.Assessments = metrics.Merge(e.Assessments, nil)

	return new
}

// EvaluationRecords holds all the evaluation records.
type EvaluationRecords []*EvaluationRecord

// EvaluationRecordsPerModel holds the collection of evaluation records per model.
type EvaluationRecordsPerModel map[string]*EvaluationRecord

// GroupByModel groups the evaluation records by model.
func (e EvaluationRecords) GroupByModel() EvaluationRecordsPerModel {
	perModel := map[string]*EvaluationRecord{}

	for _, record := range e {
		_, ok := perModel[record.ModelID]
		if !ok {
			perModel[record.ModelID] = record.Clone()
		} else {
			r := perModel[record.ModelID]
			r.Assessments = metrics.Merge(r.Assessments, record.Assessments)
		}
	}

	return perModel
}

// Header returns the header description as a CSV row.
func (EvaluationRecordsPerModel) Header() (header []string) {
	return append([]string{"model-id", "model-name", "cost", "score"}, metrics.AllAssessmentKeysStrings...)
}

// Rows returns all data as CSV rows.
func (e EvaluationRecordsPerModel) Rows() (rows [][]string) {
	models := maps.Keys(e)
	slices.SortStableFunc(models, func(a, b string) int {
		return cmp.Compare(a, b)
	})

	for _, model := range models {
		record := e[model]
		metrics := record.Assessments.StringCSV()
		score := record.Assessments.Score()
		modelCost := record.ModelCost

		row := append([]string{record.ModelID, record.ModelName, strconv.FormatFloat(modelCost, 'f', -1, 64), strconv.FormatUint(uint64(score), 10)}, metrics...)
		rows = append(rows, row)
	}

	return rows
}

// EvaluationRecordsPerModel holds the collection of evaluation records per model.
type EvaluationRecordsPerLanguagePerModel map[string]EvaluationRecordsPerModel

// GroupByLanguageAndModel groups the evaluation records by language and model.
func (e EvaluationRecords) GroupByLanguageAndModel() EvaluationRecordsPerLanguagePerModel {
	perLanguageAndModel := map[string]EvaluationRecordsPerModel{}

	for _, record := range e {
		perModel, ok := perLanguageAndModel[record.LanguageID]
		if !ok {
			perLanguageAndModel[record.LanguageID] = EvaluationRecordsPerModel{
				record.ModelID: record,
			}
		} else {
			_, ok := perModel[record.ModelID]
			if !ok {
				perModel[record.ModelID] = record.Clone()
			} else {
				perModel[record.ModelID].Assessments = metrics.Merge(perModel[record.ModelID].Assessments, record.Assessments)
			}
		}
	}

	return perLanguageAndModel
}

// loadEvaluationRecords reads and returns the evaluation records from the evaluation CSV file.
func loadEvaluationRecords(evaluationFilePath string) (evaluationRecords EvaluationRecords, err error) {
	evaluationFile, err := os.Open(evaluationFilePath)
	if err != nil {
		return nil, pkgerrors.WithStack(err)
	}
	defer evaluationFile.Close()

	reader := csv.NewReader(evaluationFile)

	// Check if the evaluation CSV header is correct.
	if header, err := reader.Read(); err != nil {
		return nil, pkgerrors.Wrap(err, "found error while reading evaluation file")
	} else if strings.Join(header, ",") != strings.Join(evaluationHeader(), ",") {
		return nil, pkgerrors.WithStack(pkgerrors.Errorf("expected header %+v\nfound header %+v", evaluationHeader(), header))
	}

	// Read the raw records from the evaluation CSV file.
	records, err := reader.ReadAll()
	if err != nil {
		return nil, pkgerrors.WithStack(err)
	}

	// Convert the raw records into assessments that can be easily manipulated.
	evaluationRecords = EvaluationRecords{}
	for _, record := range records {
		evaluationRecord, err := convertRawRecordToEvaluationRecord(record)
		if err != nil {
			return nil, err
		}
		evaluationRecords = append(evaluationRecords, evaluationRecord)
	}

	return evaluationRecords, nil
}

// convertRawRecordToEvaluationRecord converts a raw CSV record into an evaluation record.
func convertRawRecordToEvaluationRecord(raw []string) (record *EvaluationRecord, err error) {
	assessments := metrics.NewAssessments()

	modelID := raw[0]
	modelName := raw[1]
	modelCost, err := strconv.ParseFloat(raw[2], 64)
	if err != nil {
		return nil, pkgerrors.WithStack(err)
	}

	languageID := raw[3]

	repositoryName := raw[4]
	task := raw[5]

	rawMetrics := raw[7:]
	for i, assessementKey := range metrics.AllAssessmentKeysStrings {
		metric, err := strconv.ParseUint(rawMetrics[i], 10, 64)
		if err != nil {
			return nil, pkgerrors.WithStack(err)
		}

		assessments[metrics.AssessmentKey(assessementKey)] = metric
	}

	return &EvaluationRecord{
		ModelID:   modelID,
		ModelName: modelName,
		ModelCost: modelCost,

		LanguageID: languageID,

		RepositoryName: repositoryName,
		Task:           task,

		Assessments: assessments,
	}, nil
}

// generateCSV returns the whole CSV as string.
func generateCSV(formatter CSVFormatter) (csvData string, err error) {
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

// WriteCSVs writes the various CSV reports to disk.
func WriteCSVs(resultPath string) (err error) {
	evaluationRecords, err := loadEvaluationRecords(filepath.Join(resultPath, "evaluation.csv"))
	if err != nil {
		return err
	}

	// Write the "models-summed.csv" containing the summary per model.
	perModel := evaluationRecords.GroupByModel()
	csvByModel, err := generateCSV(perModel)
	if err != nil {
		return pkgerrors.Wrap(err, "could not create models-summed.csv summary")
	}
	if err := os.WriteFile(filepath.Join(resultPath, "models-summed.csv"), []byte(csvByModel), 0644); err != nil {
		return pkgerrors.Wrap(err, "could not write models-summed.csv summary")
	}

	// Write the individual "language-summed.csv" containing the summary per model per language.
	perLanguage := evaluationRecords.GroupByLanguageAndModel()
	for language, modelsByLanguage := range perLanguage {
		csvByLanguage, err := generateCSV(modelsByLanguage)
		if err != nil {
			return pkgerrors.Wrap(err, "could not create "+language+"-summed.csv summary")
		}
		if err := os.WriteFile(filepath.Join(resultPath, language+"-summed.csv"), []byte(csvByLanguage), 0644); err != nil {
			return pkgerrors.Wrap(err, "could not write "+language+"-summed.csv summary")
		}
	}

	return nil
}
