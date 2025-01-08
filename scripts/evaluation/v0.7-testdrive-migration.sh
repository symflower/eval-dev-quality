#!/bin/bash

RESULTS="./evaluation-testdrive-migration"

make install
docker build . -t testdrive-migration

eval-dev-quality evaluate \
	--runtime docker \
	--runtime-image testdrive-migration \
	--no-disqualification \
	--result-path $RESULTS \
	--runs 2 \
	--parallel 3 \
	--repository java/migrate-symflower-curated \
	--model openrouter/meta-llama/llama-3.3-70b-instruct \
	--model openrouter/cohere/command-r7b-12-2024 \
	--model openrouter/google/gemini-2.0-flash-exp:free
