package metrics

// AssessmentCategory represents a categorical ranking of a model based on Assessments.
type AssessmentCategory string

var (
	// AssessmentCategoryUnknown indicates that it is not possible to compute a model's category.
	AssessmentCategoryUnknown = AssessmentCategory("category-unknown")
	// AssessmentCategoryResponseError indicates that a model has encountered an error trying to produce a response.
	AssessmentCategoryResponseError = AssessmentCategory("response-error")
	// AssessmentCategoryResponseEmpty indicates that a model has returned an empty response.
	AssessmentCategoryResponseEmpty = AssessmentCategory("response-empty")
	// AssessmentCategoryResponseNoCode indicates that a model's response did not contain any source code.
	AssessmentCategoryResponseNoCode = AssessmentCategory("response-no-code")
	// AssessmentCategoryCodeInvalid indicates that a model's generated code produced an error when executed.
	AssessmentCategoryCodeInvalid = AssessmentCategory("code-invalid")
	// AssessmentCategoryCodeExecuted indicates that a model's generated code could be executed without an error.
	AssessmentCategoryCodeExecuted = AssessmentCategory("code-executed")
	// AssessmentCategoryCodeCoverageStatementReached indicates that a model's generated code reached 100% statement coverage.
	AssessmentCategoryCodeCoverageStatementReached = AssessmentCategory("code-coverage-statement")
	// AssessmentCategoryCodeNoExcess indicates that a model's response did not contain more content than requested.
	AssessmentCategoryCodeNoExcess = AssessmentCategory("code-no-excess")
)

// Category infers a categorical ranking of a model based on assessment values.
// A models overall category corresponds to the criterion where the model was consistently able to receive "total" amount of points. I.e. if there were 3 tasks in total and a model was able to produce executing code for all tasks, but only in one case the coverage goal was reached, then the category is only "CodeExecuted" because the coverage goal was not reached consistently.
func (a Assessments) Category(total uint) AssessmentCategory {
	if total == 0 {
		return AssessmentCategoryUnknown
	}

	switch {
	case a[AssessmentKeyResponseNoError] != total:
		return AssessmentCategoryResponseError
	case a[AssessmentKeyResponseNotEmpty] != total:
		return AssessmentCategoryResponseEmpty
	case a[AssessmentKeyResponseWithCode] != total && a[AssessmentKeyFilesExecuted] != total: // TODO We cannot always detect yet if a model response contains source code, so ensure we don't categorize into "no code" if the code actually ran successfully all the time. https://github.com/symflower/eval-dev-quality/issues/43
		return AssessmentCategoryResponseNoCode
	case a[AssessmentKeyFilesExecuted] != total:
		return AssessmentCategoryCodeInvalid
	case a[AssessmentKeyCoverageStatement] != total:
		return AssessmentCategoryCodeExecuted
	case a[AssessmentKeyResponseNoExcess] != total:
		return AssessmentCategoryCodeCoverageStatementReached
	default:
		return AssessmentCategoryCodeNoExcess
	}
}
