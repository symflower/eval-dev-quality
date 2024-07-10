package openrouter

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/avast/retry-go"
	pkgerrors "github.com/pkg/errors"
	"github.com/sashabaranov/go-openai"

	"github.com/symflower/eval-dev-quality/log"
	"github.com/symflower/eval-dev-quality/model"
	"github.com/symflower/eval-dev-quality/model/llm"
	"github.com/symflower/eval-dev-quality/provider"
	openaiapi "github.com/symflower/eval-dev-quality/provider/openai-api"
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

// Available checks if the provider is ready to be used.
// This might include checking for an installation or making sure an API access token is valid.
func (p *Provider) Available(logger *log.Logger) (err error) {
	if p.token == "" {
		return pkgerrors.WithStack(errors.New("missing access token"))
	}

	return nil
}

// ID returns the unique ID of this provider.
func (p *Provider) ID() (id string) {
	return "openrouter"
}

// ModelsList holds a list of models.
type ModelsList struct {
	Models []Model `json:"data"`
}

// Model holds a model.
type Model struct {
	// ID holds the model id.
	ID string `json:"id"`
	// Name holds the model name.
	Name string `json:"name"`

	// Pricing holds the pricing information of a model.
	Pricing Pricing `json:"pricing"`
}

// Pricing holds the pricing information of a model.
type Pricing struct {
	// Prompt holds the price for a prompt in dollars per token.
	Prompt string `json:"prompt"`
	// Completion holds the price for a completion in dollars per token.
	Completion string `json:"completion"`
	// Request holds the price for a request in dollars per request.
	Request string `json:"request"`
	// Image holds the price for an image in dollars per token.
	Image string `json:"image"`
}

// Models returns which models are available to be queried via this provider.
func (p *Provider) Models() (models []model.Model, err error) {
	client := p.client()

	var responseModels openai.ModelsList
	if err := retry.Do( // Query available models with a retry logic cause "openrouter.ai" has failed us in the past.
		func() error {
			ms, err := client.ListModels(context.Background())
			if err != nil {
				return pkgerrors.WithStack(err)
			}
			responseModels = ms

			return nil
		},
		retry.Attempts(3),
		retry.Delay(5*time.Second),
		retry.DelayType(retry.BackOffDelay),
		retry.LastErrorOnly(true),
	); err != nil {
		return nil, err
	}

	models = make([]model.Model, len(responseModels.Models))
	for i, model := range responseModels.Models {
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

	return openaiapi.QueryOpenAIAPIModel(ctx, client, modelIdentifier, promptText)
}

// client returns a new client with the current configuration.
func (p *Provider) client() (client *openai.Client) {
	config := openai.DefaultConfig(p.token)
	config.BaseURL = p.baseURL

	return openai.NewClientWithConfig(config)
}
