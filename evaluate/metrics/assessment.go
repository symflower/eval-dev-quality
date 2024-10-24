package metrics

import (
	"fmt"
	"slices"
	"sort"
	"strings"
)

// AssessmentKey defines a key for a numerical key-value assessment pair.
type AssessmentKey string

var (
	// allAssessmentKeys holds all registered assessment keys.
	allAssessmentKeys []AssessmentKey
	// AllAssessmentKeysStrings returns all registered assessment keys as strings.
	AllAssessmentKeysStrings []string
)

// RegisterAssessmentKey registers a new assessment key.
func RegisterAssessmentKey(key string) AssessmentKey {
	assessment := AssessmentKey(key)
	i := sort.SearchStrings(AllAssessmentKeysStrings, key)

	allAssessmentKeys = slices.Insert(allAssessmentKeys, i, assessment)
	AllAssessmentKeysStrings = slices.Insert(AllAssessmentKeysStrings, i, key)

	return assessment
}

var (
	// AssessmentKeyFilesExecuted holds the successfully executed files.
	AssessmentKeyFilesExecuted = RegisterAssessmentKey("files-executed")
	// AssessmentKeyFilesExecutedMaximumReachable holds the maximum theoretically reachable executed files.
	AssessmentKeyFilesExecutedMaximumReachable = RegisterAssessmentKey("files-executed-maximum-reachable")
	// AssessmentKeyProcessingTime holds the time in milliseconds that it took to complete the task.
	AssessmentKeyProcessingTime = RegisterAssessmentKey("processing-time")

	// AssessmentKeyCoverage counts execution coverage objects.
	AssessmentKeyCoverage = RegisterAssessmentKey("coverage")

	// AssessmentKeyTestsPassing holds the percentage of passing tests.
	AssessmentKeyTestsPassing = RegisterAssessmentKey("tests-passing")

	// AssessmentKeyResponseCharacterCount counts the number of characters of a response.
	AssessmentKeyResponseCharacterCount = RegisterAssessmentKey("response-character-count")
	// AssessmentKeyGenerateTestsForFileCharacterCount counts the number of characters of a generated test file.
	AssessmentKeyGenerateTestsForFileCharacterCount = RegisterAssessmentKey("generate-tests-for-file-character-count")

	// AssessmentKeyResponseNoError indicates that a model responded without error.
	AssessmentKeyResponseNoError = RegisterAssessmentKey("response-no-error")
	// AssessmentKeyResponseWithCode indicates that a model responded with code.
	AssessmentKeyResponseWithCode = RegisterAssessmentKey("response-with-code")
	// AssessmentKeyResponseNoExcess indicates that a model did not produce more content as requested.
	// TODO Infer if a model produced "too much" code. https://github.com/symflower/eval-dev-quality/issues/44
	AssessmentKeyResponseNoExcess = RegisterAssessmentKey("response-no-excess")
)

// Assessments holds a collection of numerical assessment metrics.
type Assessments map[AssessmentKey]uint64

// NewAssessments creates a new assessment collection.
func NewAssessments() Assessments {
	return map[AssessmentKey]uint64{}
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

// Award yields a score point.
func (a Assessments) Award(key AssessmentKey) {
	a[key]++
}

// AwardMultiple yields multiple score points.
func (a Assessments) AwardMultiple(key AssessmentKey, count uint64) {
	a[key] += count
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

// CombineWithSymflowerFixAssessments combines the model assessments with the ones from "symflower fix".
func CombineWithSymflowerFixAssessments(model Assessments, fixed Assessments) (combined Assessments) {
	combined = NewAssessments()

	combined[AssessmentKeyCoverage] = fixed[AssessmentKeyCoverage]
	combined[AssessmentKeyFilesExecuted] = fixed[AssessmentKeyFilesExecuted]
	combined[AssessmentKeyGenerateTestsForFileCharacterCount] = model[AssessmentKeyGenerateTestsForFileCharacterCount]
	combined[AssessmentKeyProcessingTime] = model[AssessmentKeyProcessingTime] + fixed[AssessmentKeyProcessingTime]
	combined[AssessmentKeyResponseCharacterCount] = model[AssessmentKeyResponseCharacterCount]
	combined[AssessmentKeyResponseNoError] = model[AssessmentKeyResponseNoError]
	combined[AssessmentKeyResponseNoExcess] = model[AssessmentKeyResponseNoExcess]
	combined[AssessmentKeyResponseWithCode] = model[AssessmentKeyResponseWithCode]
	combined[AssessmentKeyTestsPassing] = fixed[AssessmentKeyTestsPassing]

	return combined
}
