package model

import (
	"strings"

	"github.com/symflower/eval-dev-quality/language"
	"github.com/symflower/eval-dev-quality/log"
)

// Model defines a model that can be queried for generations.
type Model interface {
	// ID returns full identifier, including the provider and attributes.
	ID() (id string)
	// ModelID returns the unique identifier of this model with its provider.
	ModelID() (modelID string)
	// ModelIDWithoutProvider returns the unique identifier of this model without its provider.
	ModelIDWithoutProvider() (modelID string)

	// Attributes returns query attributes.
	Attributes() (attributes map[string]string)
	// SetAttributes sets the given attributes.
	SetAttributes(attributes map[string]string)

	// MetaInformation returns the meta information of a model.
	MetaInformation() *MetaInformation

	// Clone returns a copy of the model.
	Clone() (clone Model)
}

// ParseModelID takes a packaged model ID with optional attributes and converts it into its model ID and optional attributes.
func ParseModelID(modelIDWithAttributes string) (modelID string, attributes map[string]string) {
	ms := strings.Split(modelIDWithAttributes, "@")
	if len(ms) > 1 {
		attributes = map[string]string{}
		for i := 1; i < len(ms); i++ {
			as := strings.Split(ms[i], "=")
			attributes[as[0]] = as[1]
		}
	}

	return ms[0], attributes
}

// MetaInformation holds a model.
type MetaInformation struct {
	// ID holds the model ID.
	ID string `json:"id"`
	// Name holds the model name.
	Name string `json:"name"`

	// Pricing holds the pricing information of a model.
	Pricing Pricing `json:"pricing"`
}

// Pricing holds the pricing information of a model.
type Pricing struct {
	// Prompt holds the price for a prompt in dollars per token.
	Prompt float64 `json:"prompt,string"`
	// Completion holds the price for a completion in dollars per token.
	Completion float64 `json:"completion,string"`
	// Request holds the price for a request in dollars per request.
	Request float64 `json:"request,string"`
	// Image holds the price for an image in dollars per token.
	Image float64 `json:"image,string"`
}

// Context holds the data needed by a model for running a task.
type Context struct {
	// Language holds the language for which the task should be evaluated.
	Language language.Language

	// RepositoryPath holds the absolute path to the repository.
	RepositoryPath string
	// FilePath holds the path to the file the model should act on.
	FilePath string

	// Arguments holds extra data that can be used in a query prompt.
	Arguments any

	// Logger is used for logging during evaluation.
	Logger *log.Logger
}

// SetQueryAttempts defines a model that can set the number of query attempts when a model request errors in the process of solving a task.
type SetQueryAttempts interface {
	// SetQueryAttempts sets the number of query attempts to perform when a model request errors in the process of solving a task.
	SetQueryAttempts(attempts uint)
}
