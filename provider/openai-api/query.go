package openaiapi

import (
	"context"
	"fmt"

	pkgerrors "github.com/pkg/errors"
	"github.com/sashabaranov/go-openai"
	"github.com/symflower/eval-dev-quality/provider"
)

// QueryOpenAIAPIModel queries an OpenAI API model.
func QueryOpenAIAPIModel(ctx context.Context, client *openai.Client, modelIdentifier string, attributes map[string]string, promptText string) (result *provider.QueryResult, err error) {
	apiRequest := openai.ChatCompletionRequest{
		Model: modelIdentifier,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: promptText,
			},
		},
	}

	if attributes != nil {
		if reasoningEffort, ok := attributes["reasoning_effort"]; ok {
			apiRequest.ReasoningEffort = reasoningEffort
		}
	}

	apiResponse, err := client.CreateChatCompletion(
		ctx,
		apiRequest,
	)
	if err != nil {
		return nil, pkgerrors.WithStack(err)
	} else if len(apiResponse.Choices) == 0 {
		return nil, pkgerrors.WithStack(fmt.Errorf("empty LLM %q response: %+v", modelIdentifier, apiResponse))
	}

	return &provider.QueryResult{
		Message: apiResponse.Choices[0].Message.Content,
	}, nil
}
