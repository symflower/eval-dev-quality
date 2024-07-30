#!/bin/bash

make install

eval-dev-quality evaluate \
	--runtime kubernetes \
	--runtime-image ghcr.io/symflower/eval-dev-quality:bc41058 \
	--runs 5 \
	--parallel 3 \
	--model ollama/codeqwen:7b-code-v1.5-q8_0 \
	--model ollama/codeqwen:7b-chat-v1.5-q8_0 \
	--model ollama/aya:8b-23-q8_0 \
	--model ollama/aya:35b-23-q8_0 \
	--model ollama/starcoder2:3b-q8_0 \
	--model ollama/starcoder2:7b-q8_0 \
	--model ollama/starcoder2:15b-instruct-v0.1-q8_0 \
	--model ollama/granite-code:3b-instruct-q8_0 \
	--model ollama/granite-code:8b-instruct-q8_0 \
	--model ollama/granite-code:20b-instruct-q8_0 \
	--model ollama/granite-code:34b-instruct-q8_0 \
	--model ollama/falcon2:11b-q8_0 \
	--model ollama/qwen2:0.5b-instruct-q8_0 \
	--model ollama/qwen2:1.5b-instruct-q8_0 \
	--model ollama/deepseek-coder-v2:16b-lite-instruct-q8_0 \
	--model ollama/codegeex4:9b-all-q8_0 \
	--model ollama/codestral:22b-v0.1-q8_0 \
	--result-path ./evaluation-0.6.0-ollama-q8
