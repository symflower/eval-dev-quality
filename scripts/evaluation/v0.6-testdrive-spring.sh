#!/bin/bash

RESULTS="./evaluation-testdrive-spring"

make install
docker build . -t testdrive-spring

eval-dev-quality evaluate \
	--runtime docker \
	--runtime-image testdrive-spring \
	--result-path $RESULTS \
	--runs 3 \
	--parallel 4 \
	--repository java/spring-symflower-curated \
	--model openrouter/meta-llama/llama-3.2-1b-instruct \
	--model openrouter/meta-llama/llama-3.2-3b-instruct \
	--model openrouter/meta-llama/llama-3.1-8b-instruct \
	--model openrouter/meta-llama/llama-3.1-405b-instruct \
	--model openrouter/anthropic/claude-3.5-haiku \
	--model openrouter/anthropic/claude-3.5-sonnet \
	--model openrouter/cohere/command-r-08-2024 \
	--model openrouter/cohere/command-r-plus-08-2024 \
	--model openrouter/google/gemma-2-27b-it \
	--model openrouter/google/gemini-flash-1.5-8b \
	--model openrouter/google/gemini-flash-1.5 \
	--model openrouter/google/gemini-pro-1.5 \
	--model openrouter/mistralai/ministral-3b \
	--model openrouter/mistralai/ministral-8b \
	--model openrouter/mistralai/mistral-7b-instruct-v0.3 \
	--model openrouter/mistralai/mistral-large \
	--model openrouter/qwen/qwen-2.5-7b-instruct \
	--model openrouter/qwen/qwen-2.5-72b-instruct
