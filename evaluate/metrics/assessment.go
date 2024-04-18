package metrics

import (
	"encoding/csv"
	"fmt"
	"sort"
	"strconv"
	"strings"

	pkgerrors "github.com/pkg/errors"
	"golang.org/x/exp/maps"

	"github.com/symflower/eval-dev-quality/util"
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

	allAssessmentKeys = util.InsertToSortedSlice(allAssessmentKeys, assessment)
	allAssessmentKeysStrings = util.InsertToSortedSlice(allAssessmentKeysStrings, key)

	return assessment
}

var (
	// AssessmentKeyFilesExecutes holds the successfully executed files.
	AssessmentKeyFilesExecuted = RegisterAssessmentKey("files-executed")

	// AssessmentKeyCoverageStatement counts the cases where 100% coverage was reached.
	AssessmentKeyCoverageStatement = RegisterAssessmentKey("coverage-statement")

	// AssessmentKeyResponseNoError indicates that a model responded without error.
	AssessmentKeyResponseNoError = RegisterAssessmentKey("response-no-error")
	// AssessmentKeyResponseNotEmpty indicates that a model response was not empty.
	AssessmentKeyResponseNotEmpty = RegisterAssessmentKey("response-not-empty")
	// AssessmentKeyResponseWithCode indicates that a model responded with code.
	AssessmentKeyResponseWithCode = RegisterAssessmentKey("response-with-code")
	// AssessmentKeyResponseNoExcess indicates that a model did not produce more content as requested.
	// TODO Infer if a model produced "too much" code. https://github.com/symflower/eval-dev-quality/issues/44
	AssessmentKeyResponseNoExcess = RegisterAssessmentKey("response-no-excess")
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

// Equal checks if both assessment collections are equal.
func (a Assessments) Equal(x Assessments) bool {
	if a == nil || x == nil {
		return a == nil && x == nil
	}

	for _, key := range allAssessmentKeys {
		if a[key] != x[key] {
			return false
		}
	}

	return true
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

// Score computes the score over all assessments in the collection.
func (a Assessments) Score() (score uint) {
	if len(a) == 0 {
		return 0
	}

	for _, value := range maps.Values(a) {
		score += value
	}

	return score
}

// String returns a string representation of the metrics.
func (a Assessments) String() string {
	if a == nil {
		a = NewAssessments()
	}
	entries := make([]string, len(allAssessmentKeys))

	for i, key := range allAssessmentKeys {
		entries[i] = fmt.Sprintf("%s=%d", key, a[key])
	}
	entries = append([]string{fmt.Sprintf("score=%d", a.Score())}, entries...)

	return strings.Join(entries, ", ")
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
	return append([]string{"model", "score"}, allAssessmentKeysStrings...)
}

// FormatStringCSV formats the given assessment metrics as CSV.
func FormatStringCSV(assessmentsPerModel map[string]Assessments) (string, error) {
	var out strings.Builder
	csv := csv.NewWriter(&out)

	if err := csv.Write(csvHeader()); err != nil {
		return "", pkgerrors.WithStack(err)
	}

	if err := WalkByScore(assessmentsPerModel, func(model string, assessment Assessments, score uint) error {
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

// WalkByScore walks the given assessment metrics by their score.
func WalkByScore(assessmentsPerModel map[string]Assessments, function func(model string, assessment Assessments, score uint) error) error {
	models := maps.Keys(assessmentsPerModel)
	sort.Strings(models)
	scores := make(map[string]uint, len(models))
	for _, model := range models {
		scores[model] = assessmentsPerModel[model].Score()
	}
	sort.SliceStable(models, func(i, j int) bool {
		return scores[models[i]] < scores[models[j]]
	})

	for _, model := range models {
		if err := function(model, assessmentsPerModel[model], scores[model]); err != nil {
			return err
		}
	}

	return nil
}
