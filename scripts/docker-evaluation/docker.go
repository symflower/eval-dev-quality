package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/symflower/eval-dev-quality/provider"
)

var models = []string{
	"openrouter/01-ai/yi-34b-chat",
	"openrouter/01-ai/yi-6b",
	"openrouter/allenai/olmo-7b-instruct",
	"openrouter/alpindale/goliath-120b",
	"openrouter/anthropic/claude-3-haiku",
	"openrouter/anthropic/claude-3-haiku:beta",
	"openrouter/anthropic/claude-3-opus",
	"openrouter/anthropic/claude-3-opus:beta",
	"openrouter/anthropic/claude-3-sonnet",
	"openrouter/anthropic/claude-3-sonnet:beta",
	"openrouter/anthropic/claude-instant-1.1",
	"openrouter/austism/chronos-hermes-13b",
	"openrouter/bigcode/starcoder2-15b-instruct",
	"openrouter/cognitivecomputations/dolphin-mixtral-8x22b",
	"openrouter/cognitivecomputations/dolphin-mixtral-8x7b",
	"openrouter/cohere/command-r-plus",
	"openrouter/databricks/dbrx-instruct",
	"openrouter/deepseek/deepseek-chat",
	"openrouter/deepseek/deepseek-coder",
	"openrouter/fireworks/firellava-13b",
	"openrouter/google/gemini-flash-1.5",
	"openrouter/google/gemini-pro-1.5",
	"openrouter/google/gemma-7b-it",
	"openrouter/google/palm-2-chat-bison",
	"openrouter/google/palm-2-codechat-bison",
	"openrouter/huggingfaceh4/zephyr-7b-beta",
	"openrouter/intel/neural-chat-7b",
	"openrouter/jondurbin/airoboros-l2-70b",
	"openrouter/liuhaotian/llava-13b",
	"openrouter/liuhaotian/llava-yi-34b",
	"openrouter/meta-llama/codellama-34b-instruct",
	"openrouter/meta-llama/codellama-70b-instruct",
	"openrouter/meta-llama/llama-3-70b-instruct",
	"openrouter/meta-llama/llama-3-8b-instruct",
	"openrouter/microsoft/phi-3-medium-128k-instruct",
	"openrouter/microsoft/phi-3-medium-4k-instruct",
	"openrouter/microsoft/phi-3-mini-128k-instruct",
	"openrouter/microsoft/wizardlm-2-7b",
	"openrouter/microsoft/wizardlm-2-8x22b",
	"openrouter/mistralai/mistral-7b-instruct-v0.3",
	"openrouter/mistralai/mistral-large",
	"openrouter/mistralai/mistral-medium",
	"openrouter/mistralai/mistral-small",
	"openrouter/mistralai/mistral-tiny",
	"openrouter/mistralai/mixtral-8x22b-instruct",
	"openrouter/mistralai/mixtral-8x7b-instruct",
	"openrouter/neversleep/llama-3-lumimaid-70b",
	"openrouter/neversleep/llama-3-lumimaid-8b",
	"openrouter/nousresearch/hermes-2-pro-llama-3-8b",
	"openrouter/nousresearch/nous-capybara-34b",
	"openrouter/nousresearch/nous-capybara-7b",
	"openrouter/nousresearch/nous-hermes-2-mistral-7b-dpo",
	"openrouter/nousresearch/nous-hermes-2-mixtral-8x7b-dpo",
	"openrouter/nousresearch/nous-hermes-2-mixtral-8x7b-sft",
	"openrouter/nousresearch/nous-hermes-yi-34b",
	"openrouter/open-orca/mistral-7b-openorca",
	"openrouter/openai/gpt-4",
	"openrouter/openai/gpt-4-turbo",
	"openrouter/openai/gpt-4o",
	"openrouter/openchat/openchat-8b",
	"openrouter/perplexity/llama-3-sonar-large-32k-chat",
	"openrouter/perplexity/llama-3-sonar-small-32k-chat",
	"openrouter/phind/phind-codellama-34b",
	"openrouter/qwen/qwen-110b-chat",
	"openrouter/qwen/qwen-14b-chat",
	"openrouter/qwen/qwen-2-72b-instruct",
	"openrouter/qwen/qwen-32b-chat",
	"openrouter/qwen/qwen-4b-chat",
	"openrouter/qwen/qwen-72b-chat",
	"openrouter/qwen/qwen-7b-chat",
	"openrouter/recursal/eagle-7b",
	"openrouter/recursal/rwkv-5-3b-ai-town",
	"openrouter/rwkv/rwkv-5-world-3b",
	"openrouter/snowflake/snowflake-arctic-instruct",
	"openrouter/teknium/openhermes-2-mistral-7b",
	"openrouter/teknium/openhermes-2.5-mistral-7b",
	"openrouter/togethercomputer/stripedhyena-hessian-7b",
	"openrouter/togethercomputer/stripedhyena-nous-7b",
	"openrouter/undi95/toppy-m-7b",
	"openrouter/xwin-lm/xwin-lm-70b",
}

func main() {

	for _, model := range models {

		modelIdentifier := strings.Split(model, provider.ProviderModelSeparator)[2]
		repo := "light"
		fmt.Printf("screen -dmS %s docker run -v ./:/home/ubuntu/evaluation -e PROVIDER_TOKEN ghcr.io/symflower/eval-dev-quality:docker-image eval-dev-quality evaluate --runs 5 --repository %s --repository %s --model %s --result-path /home/ubuntu/evaluation/%s\n",
			modelIdentifier,
			filepath.Join("golang", repo),
			filepath.Join("java", repo),
			model, modelIdentifier,
		)
	}

}
