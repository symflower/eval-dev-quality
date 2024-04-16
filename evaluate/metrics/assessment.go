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

// Assessments holds numerical assessment metrics.
type Assessments map[AssessmentKey]uint

// Merge combines two assessments into a new assessment and returns it.
func (a Assessments) Merge(o Assessments) Assessments {
	if a == nil {
		a = Assessments{}
	}
	if o == nil {
		o = Assessments{}
	}

	assessments := map[AssessmentKey]uint{}

	for _, k := range allAssessmentKeys {
		assessments[k] = a[k]
	}
	for _, k := range allAssessmentKeys {
		assessments[k] = o[k]
	}

	return Assessments(assessments)
}
