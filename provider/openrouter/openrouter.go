package openrouter

import (
	"context"
	"fmt"
	"strings"

	pkgerrors "github.com/pkg/errors"
	"github.com/sashabaranov/go-openai"

	"github.com/symflower/eval-dev-quality/model"
	"github.com/symflower/eval-dev-quality/model/llm"
	"github.com/symflower/eval-dev-quality/provider"
)

// Provider holds an "openrouter.ai" provider using its public REST API.
type Provider struct {
	baseURL string
	token   string
}

func init() {
	provider.Register(NewProvider())
}

// NewProvider returns an "openrouter.ai" provider.
func NewProvider() (provider provider.Provider) {
	return &Provider{
		baseURL: "https://openrouter.ai/api/v1",
	}
}

var _ provider.Provider = (*Provider)(nil)

// ID returns the unique ID of this provider.
func (p *Provider) ID() (id string) {
	return "openrouter"
}

// Models returns which models are available to be queried via this provider.
func (p *Provider) Models() (models []model.Model, err error) {
	client := p.client()
	ms, err := client.ListModels(context.Background())
	if err != nil {
		return nil, pkgerrors.WithStack(err)
	}

	models = make([]model.Model, len(ms.Models))
	for i, model := range ms.Models {
		models[i] = llm.NewModel(p, p.ID()+provider.ProviderModelSeparator+model.ID)
	}

	return models, nil
}

var _ provider.InjectToken = (*Provider)(nil)

// SetToken sets a potential token to be used in case the provider needs to authenticate a remote API.
func (p *Provider) SetToken(token string) {
	p.token = token
}

var _ provider.Query = (*Provider)(nil)

// Query queries the provider with the given model name.
func (p *Provider) Query(ctx context.Context, modelIdentifier string, promptText string) (response string, err error) {
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
		return "", pkgerrors.WithStack(fmt.Errorf("empty LLM %q response: %+v", modelIdentifier, apiResponse))
	}

	return apiResponse.Choices[0].Message.Content, nil
}

// client returns a new client with the current configuration.
func (p *Provider) client() (client *openai.Client) {
	config := openai.DefaultConfig(p.token)
	config.BaseURL = p.baseURL

	return openai.NewClientWithConfig(config)
}
