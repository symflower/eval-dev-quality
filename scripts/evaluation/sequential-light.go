package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	pkgerrors "github.com/pkg/errors"
	"github.com/zimmski/osutil"

	"github.com/symflower/eval-dev-quality/log"
	"github.com/symflower/eval-dev-quality/util"
)

var models = []string{
	"openrouter/01-ai/yi-34b-chat",
	"openrouter/anthropic/claude-2.1",
	"openrouter/anthropic/claude-3-haiku",
	"openrouter/anthropic/claude-3-sonnet",
	"openrouter/cohere/command-r-plus",
	"openrouter/cohere/command-r",
	"openrouter/databricks/dbrx-instruct",
	"openrouter/google/gemini-flash-1.5",
	"openrouter/google/gemini-pro-1.5",
	"openrouter/google/gemma-7b-it",
	"openrouter/meta-llama/llama-2-70b-chat",
	"openrouter/meta-llama/llama-3-70b-instruct",
	"openrouter/meta-llama/llama-3-8b-instruct",
	"openrouter/microsoft/wizardlm-2-7b",
	"openrouter/microsoft/wizardlm-2-8x22b",
	"openrouter/mistralai/mistral-7b-instruct",
	"openrouter/mistralai/mistral-medium",
	"openrouter/mistralai/mixtral-8x22b-instruct",
	"openrouter/mistralai/mixtral-8x7b-instruct",
	"openrouter/openai/gpt-3.5-turbo",
	"openrouter/openai/gpt-4-turbo",
	"openrouter/openai/gpt-4o",
	"openrouter/qwen/qwen-2-72b-instruct",
}

func main() {
	logger := log.STDOUT()

	output, err := util.CommandWithResult(context.Background(), logger, &util.Command{
		Command: []string{
			"make",
			"install-all",
		},
	})
	if err != nil {
		panic(pkgerrors.WithStack(pkgerrors.WithMessage(err, output)))
	}

	progress := osutil.ProgressBar(os.Stdout, len(models), "Benchmarking")
	defer progress.Exit()

	for _, model := range models {
		progress.Describe("Benchmarking: " + model)

		err := benchmarkModel(logger, "evaluation-sequential-light-", "light", model, 150*time.Minute)
		if err != nil {
			fmt.Printf("Model %q misbehaved", model)
		}

		progress.Add(1)
	}
}

func benchmarkModel(logger *log.Logger, resultPrefix string, repository string, id string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	output, err := util.CommandWithResult(ctx, logger, &util.Command{
		Command: []string{
			"eval-dev-quality",
			"evaluate",
			"--runs", "5",
			"--result-path", resultPrefix + log.CleanModelNameForFileSystem(id),
			"--repository", filepath.Join("java", repository),
			"--repository", filepath.Join("golang", repository),
			"--model", id,
		},
	})
	if err != nil {
		return pkgerrors.WithStack(pkgerrors.WithMessage(err, output))
	}

	return nil
}
