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
func (p *openRouterProvider) client() (client *openai.Client) {
	config := openai.DefaultConfig(p.token)
	config.BaseURL = p.baseURL

	return openai.NewClientWithConfig(config)
}

// Query queries the LLM.
func (p *openRouterProvider) Query(ctx context.Context, modelIdentifier string, promptText string) (response string, err error) {
	client := p.client()
	modelIdentifier = strings.TrimPrefix(modelIdentifier, p.ID()+provider.ProviderModelSeparator)

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
func (p *openRouterProvider) SetToken(token string) {
	p.token = token
}

// ID returns the unique ID of this provider.
func (p *openRouterProvider) ID() (id string) {
	return "openrouter"
}

// Models returns which models are available to be queried via this provider.
func (p *openRouterProvider) Models() (models []model.Model, err error) {
	client := p.client()
	ms, err := client.ListModels(context.Background())
	if err != nil {
		return nil, pkgerrors.WithStack(err)
	}

	models = make([]model.Model, len(ms.Models))
	for i, model := range ms.Models {
		models[i] = llm.NewLLMModel(p, p.ID()+provider.ProviderModelSeparator+model.ID)
	}

	return models, nil
}
