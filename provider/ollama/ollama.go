package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/symflower/eval-dev-quality/model"
	"github.com/symflower/eval-dev-quality/model/llm"
	"github.com/symflower/eval-dev-quality/provider"
	"io/ioutil"
	"net/http"
	"strings"
)

// ollamaQueryProvider is a mocked QueryProvider.
type ollamaQueryProvider struct {
	baseURL string
}

func init() {
	provider.Register(newOllamaQueryProvider())
}

// newOllamaQueryProvider returns an "ollama" provider.
func newOllamaQueryProvider() provider.Provider {
	return &ollamaQueryProvider{
		baseURL: "http://localhost:11434/api",
	}
}

var _ provider.QueryProvider = &ollamaQueryProvider{}

// Assuming openRouterProvider is defined in the same package,
// and has baseURL and token as its attributes.

// Query queries the LLM using the Ollama API's `/api/generate` endpoint.
func (p *ollamaQueryProvider) Query(ctx context.Context, modelIdentifier string, promptText string) (response string, err error) {
	requestBody, err := json.Marshal(map[string]interface{}{
		"model":  modelIdentifier[strings.Index(modelIdentifier, provider.ProviderModelSeparator)+1:],
		"prompt": promptText,
		"stream": false,
	})
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal request body")
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/generate", bytes.NewBuffer(requestBody))
	if err != nil {
		return "", errors.Wrap(err, "failed to create request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "ollama-client")

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "failed to send request to ollama service")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama service returned non-200 status code: %d", resp.StatusCode)
	}

	var result struct {
		Response string `json:"response"`
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "failed to read response body")
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return "", errors.Wrap(err, "failed to unmarshal response body")
	}

	return result.Response, nil
}

// ID returns the unique ID of this provider.
func (p *ollamaQueryProvider) ID() (id string) {
	return "ollama"
}

// Models returns which models are available to be queried via this provider.
func (p *ollamaQueryProvider) Models() (models []model.Model, err error) {
	req, err := http.NewRequestWithContext(context.Background(), "GET", "http://localhost:11434/api/tags", nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to send request to ollama service")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama service returned non-200 status code: %d", resp.StatusCode)
	}

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body")
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal response body")
	}

	models = make([]model.Model, len(result.Models))
	for i, model := range result.Models {
		models[i] = llm.NewLLMModel(p, p.ID()+provider.ProviderModelSeparator+model.Name)
	}

	return models, nil
}
