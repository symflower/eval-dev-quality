package metrics

import "fmt"

// AssessmentCategory represents a categorical ranking of a model based on Assessments.
type AssessmentCategory struct {
	// Name holds a unique short name of the category.
	Name string
	// Description holds the description of a category.
	Description string
}

// AllAssessmentCategories holds all assessment categories.
var AllAssessmentCategories []*AssessmentCategory

// registerAssessmentCategory registers a new assessment category.
func registerAssessmentCategory(c AssessmentCategory) *AssessmentCategory {
	for _, category := range AllAssessmentCategories {
		if c.Name == category.Name {
			panic(fmt.Sprintf("duplicated category name %q", c.Name))
		}
	}

	AllAssessmentCategories = append(AllAssessmentCategories, &c)

	return &c
}

var (
	// AssessmentCategoryUnknown indicates that it is not possible to compute a model's category.
	AssessmentCategoryUnknown = registerAssessmentCategory(AssessmentCategory{
		Name:        "Category Unknown",
		Description: "Models in this category could not be categorized.",
	})
	// AssessmentCategoryResponseError indicates that a model has encountered an error trying to produce a response.
	AssessmentCategoryResponseError = registerAssessmentCategory(AssessmentCategory{
		Name:        "Response Error",
		Description: "Models in this category encountered an error.",
	})
	// AssessmentCategoryResponseEmpty indicates that a model has returned an empty response.
	AssessmentCategoryResponseEmpty = registerAssessmentCategory(AssessmentCategory{
		Name:        "Response Empty",
		Description: "Models in this category produced an empty response.",
	})
	// AssessmentCategoryResponseNoCode indicates that a model's response did not contain any source code.
	AssessmentCategoryResponseNoCode = registerAssessmentCategory(AssessmentCategory{
		Name:        "No Code",
		Description: "Models in this category produced no code.",
	})
	// AssessmentCategoryCodeInvalid indicates that a model's generated code produced an error when executed.
	AssessmentCategoryCodeInvalid = registerAssessmentCategory(AssessmentCategory{
		Name:        "Invalid Code",
		Description: "Models in this category produced invalid code.",
	})
	// AssessmentCategoryCodeExecuted indicates that a model's generated code could be executed without an error.
	AssessmentCategoryCodeExecuted = registerAssessmentCategory(AssessmentCategory{
		Name:        "Executable Code",
		Description: "Models in this category produced executable code.",
	})
	// AssessmentCategoryCodeCoverageStatementReached indicates that a model's generated code reached 100% statement coverage.
	AssessmentCategoryCodeCoverageStatementReached = registerAssessmentCategory(AssessmentCategory{
		Name:        "Statement Coverage Reached",
		Description: "Models in this category produced code that reached full statement coverage.",
	})
	// AssessmentCategoryCodeNoExcess indicates that a model's response did not contain more content than requested.
	AssessmentCategoryCodeNoExcess = registerAssessmentCategory(AssessmentCategory{
		Name:        "No Excess Response",
		Description: "Models in this category did not respond with more content than requested.",
	})
)

// Category infers a categorical ranking of a model based on assessment values.
// A models overall category corresponds to the criterion where the model was consistently able to receive "total" amount of points. I.e. if there were 3 tasks in total and a model was able to produce executing code for all tasks, but only in one case the coverage goal was reached, then the category is only "CodeExecuted" because the coverage goal was not reached consistently.
// The returned category is never "nil".
func (a Assessments) Category(total uint) *AssessmentCategory {
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
