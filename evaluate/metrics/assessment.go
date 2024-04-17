package metrics

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
	// AssessmentKeyNoExcessResponse indicates that a model did not produce more content as requested.
	AssessmentKeyNoExcessResponse = RegisterAssessmentKey("no-excess-response")
)

// Assessments holds a collection of numerical assessment metrics.
type Assessments map[AssessmentKey]uint

// NewAssessments create a new assessment collection.
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
