package metrics

import (
	"encoding/csv"
	"fmt"
	"sort"
	"strings"

	pkgerrors "github.com/pkg/errors"
	"golang.org/x/exp/maps"
)

// AssessmentKey defines a key for a numerical key-value assessment pair.
type AssessmentKey string

var (
	// allAssessmentKeys holds all registered assessment keys.
	allAssessmentKeys []AssessmentKey
	// allAssessmentKeysStrings returns all registered assessment keys as strings.
	allAssessmentKeysStrings []string
)

// RegisterAssessmentKey registers a new assessment key.
func RegisterAssessmentKey(key string) AssessmentKey {
	assessment := AssessmentKey(key)
	allAssessmentKeys = append(allAssessmentKeys, assessment)
	allAssessmentKeysStrings = append(allAssessmentKeysStrings, key)

	return assessment
}

var (
	// AssessmentKeyFilesExecutes holds the successfully executed files.
	AssessmentKeyFilesExecuted = RegisterAssessmentKey("files-executed")
	// AssessmentKeyFilesProblems holds the files with problems.
	AssessmentKeyFilesProblems = RegisterAssessmentKey("files-problems")

	// AssessmentKeyCoverageStatement counts the cases where 100% coverage was reached.
	AssessmentKeyCoverageStatement = RegisterAssessmentKey("coverage-statement")

	// AssessmentKeyNoExcessResponse indicates that a model did not produce more content as requested.
	AssessmentKeyNoExcessResponse = RegisterAssessmentKey("no-excess-response")
)

// Assessments holds a collection of numerical assessment metrics.
type Assessments map[AssessmentKey]uint

// NewAssessments creates a new assessment collection.
func NewAssessments() Assessments {
	return map[AssessmentKey]uint{}
}

// Add adds the given assessment collection to the current one.
func (a Assessments) Add(x Assessments) {
	for k, v := range x {
		a[k] += v
	}
}

// Merge combines two assessment collections into a new assessment collection and returns the new assessment collection.
func Merge(a Assessments, b Assessments) (c Assessments) {
	c = NewAssessments()
	if a != nil {
		c.Add(a)
	}
	if b != nil {
		c.Add(b)
	}

	return c
}

// String returns a string representation of the metrics.
func (a Assessments) String() string {
	if a == nil {
		a = NewAssessments()
	}
	metrics := make([]string, len(allAssessmentKeys))

	for i, key := range allAssessmentKeys {
		metrics[i] = fmt.Sprintf("%s=%d", key, a[key])
	}

	return strings.Join(metrics, ", ")
}

// StringCSV returns a CSV row string representation of the metrics.
func (a Assessments) StringCSV() (row []string) {
	if a == nil {
		a = NewAssessments()
	}

	row = make([]string, len(allAssessmentKeys))
	for i, key := range allAssessmentKeys {
		row[i] = fmt.Sprintf("%d", a[key])
	}

	return row
}

func csvHeader() []string {
	return append([]string{"model"}, allAssessmentKeysStrings...)
}

// FormatStringCSV formats the given metrics as CSV.
func FormatStringCSV(metricsPerModel map[string]Assessments) (string, error) {
	var out strings.Builder
	csv := csv.NewWriter(&out)

	if err := csv.Write(csvHeader()); err != nil {
		return "", err
	}
	models := maps.Keys(metricsPerModel)
	sort.Strings(models)
	for _, model := range models {
		row := metricsPerModel[model].StringCSV()

		if err := csv.Write(append([]string{model}, row...)); err != nil {
			return "", pkgerrors.WithStack(err)
		}
	}
	csv.Flush()

	return out.String(), nil
}
