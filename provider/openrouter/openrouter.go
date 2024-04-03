package openrouter

import (
	"context"
	"fmt"
	"strings"

	pkgerrors "github.com/pkg/errors"
	"github.com/sashabaranov/go-openai"

	"github.com/symflower/eval-symflower-codegen-testing/model"
	"github.com/symflower/eval-symflower-codegen-testing/model/llm"
	"github.com/symflower/eval-symflower-codegen-testing/provider"
)

type openRouterProvider struct {
	baseURL string
	token   string
}

func init() {
	provider.Register(newOpenRouterProvider())
}

// newOpenRouterProvider returns an "openrouter.ai" provider.
func newOpenRouterProvider() provider.Provider {
	return &openRouterProvider{
		baseURL: "https://openrouter.ai/api/v1",
	}
}

// client returns a new client with the current configuration.
func (o *openRouterProvider) client() (client *openai.Client) {
	config := openai.DefaultConfig(o.token)
	config.BaseURL = o.baseURL

	return openai.NewClientWithConfig(config)
}

// Query queries the LLM.
func (o *openRouterProvider) Query(ctx context.Context, modelIdentifier string, promptText string) (response string, err error) {
	client := o.client()
	modelIdentifier = strings.TrimPrefix(modelIdentifier, o.ID()+provider.ProviderModelSeparator)

	apiResponse, err := client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: modelIdentifier,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: promptText,
				},
			},
		},
	)
	if err != nil {
		return "", pkgerrors.WithStack(err)
	} else if len(apiResponse.Choices) == 0 {
		return "", fmt.Errorf("empty LLM %q response: %+v", modelIdentifier, apiResponse)
	}

	return apiResponse.Choices[0].Message.Content, nil
}

// SetToken sets a potential token to be used in case the provider needs to authenticate a remote API.
func (o *openRouterProvider) SetToken(token string) {
	o.token = token
}

// ID returns the unique ID of this provider.
func (o *openRouterProvider) ID() (id string) {
	return "openrouter"
}

// Models returns which models are available to be queried via this provider.
func (o *openRouterProvider) Models() (models []model.Model) {
	// TODO Query this from the API directly. https://pkg.go.dev/github.com/sashabaranov/go-openai#ModelsList
	modelIDs := []string{
		"google/gemma-7b-it:free",
		"gryphe/mythomist-7b:free",
		"mistralai/mistral-7b-instruct:free",
		"nousresearch/nous-capybara-7b:free",
		"openrouter/cinematika-7b:free",
		"undi95/toppy-m-7b:free",
	}

	models = make([]model.Model, len(modelIDs))
	for i, id := range modelIDs {
		models[i] = llm.NewLLMModel(o, o.ID()+provider.ProviderModelSeparator+id)
	}

	return models
}
