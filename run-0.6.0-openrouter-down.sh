#!/bin/bash

make install

eval-dev-quality evaluate \
	--runtime kubernetes \
	--runtime-image ghcr.io/symflower/eval-dev-quality:bc41058 \
	--runs 5 \
	--parallel 20 \
	--model openrouter/qwen/qwen-110b-chat \
	--model openrouter/01-ai/yi-large \
	--model openrouter/recursal/eagle-7b \
	--model openrouter/rwkv/rwkv-5-world-3b \
	--model openrouter/togethercomputer/stripedhyena-hessian-7b \
	--result-path ./evaluation-0.6.0-openrouter-down
