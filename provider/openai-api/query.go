package openaiapi

import (
	"context"
	"fmt"

	pkgerrors "github.com/pkg/errors"
	"github.com/sashabaranov/go-openai"
)

// QueryOpenAIAPIModel queries an OpenAI API model.
func QueryOpenAIAPIModel(ctx context.Context, client *openai.Client, modelIdentifier string, attributes map[string]string, promptText string) (response string, err error) {
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
		return "", pkgerrors.WithStack(err)
	} else if len(apiResponse.Choices) == 0 {
		return "", pkgerrors.WithStack(fmt.Errorf("empty LLM %q response: %+v", modelIdentifier, apiResponse))
	}

	return apiResponse.Choices[0].Message.Content, nil
}
