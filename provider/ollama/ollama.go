package ollama

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	pkgerrors "github.com/pkg/errors"
	"github.com/sashabaranov/go-openai"

	"github.com/symflower/eval-dev-quality/log"
	"github.com/symflower/eval-dev-quality/model"
	"github.com/symflower/eval-dev-quality/model/llm"
	"github.com/symflower/eval-dev-quality/provider"
	"github.com/symflower/eval-dev-quality/tools"
)

// Provider holds an "Ollama" provider.
type Provider struct {
	// binaryPath holds the binary file path of the Ollama binary.
	binaryPath string
	// url holds the URL to the Ollama service.
	url string
}

func init() {
	provider.Register(NewProvider())
}

// NewProvider returns an "Ollama" provider.
func NewProvider() (provider provider.Provider) {
	return &Provider{
		binaryPath: tools.OllamaPath,
		url:        tools.OllamaURL,
	}
}

// SetBinaryPath sets the binary file path of the Ollama binary.
func (p *Provider) SetBinaryPath(binaryPath string) {
	p.binaryPath = binaryPath
}

// SetURL sets the URL to the Ollama service.
func (p *Provider) SetURL(url string) {
	p.url = url
}

var _ provider.Provider = (*Provider)(nil)

// Available checks if the provider ready to be used and returns a potential error explaining why not.
// This might include checking for an installation or making sure an API access token is valid.
func (p *Provider) Available(logger *log.Logger) (err error) {
	if err := tools.NewOllama().CheckVersion(logger, p.binaryPath); err != nil {
		return pkgerrors.WithStack(pkgerrors.WithMessage(err, `"Ollama" version check unsuccessful`))
	}

	return nil
}

// ID returns the unique ID of this provider.
func (p *Provider) ID() (id string) {
	return "ollama"
}

// Models returns which models are available to be queried via this provider.
func (p *Provider) Models() (models []model.Model, err error) {
	ms, err := tools.OllamaModels(p.url)
	if err != nil {
		return nil, err
	}

	models = make([]model.Model, len(ms))
	for i, modelName := range ms {
		models[i] = llm.NewModel(p, p.ID()+provider.ProviderModelSeparator+modelName)
	}

	return models, nil
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
	config := openai.DefaultConfig("")

	var err error
	config.BaseURL, err = url.JoinPath(p.url, "v1")
	if err != nil {
		panic(err)
	}

	return openai.NewClientWithConfig(config)
}

var _ provider.Service = (*Provider)(nil)

// Start starts necessary background services to use this provider and returns a shutdown function.
func (p *Provider) Start(logger *log.Logger) (shutdown func() (err error), err error) {
	return tools.OllamaStart(logger, p.binaryPath, p.url)
}
