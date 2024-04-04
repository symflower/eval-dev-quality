# DevQualityEval

An evaluation benchmark 📈 and framework for LLMs and friends to compare and evolve the quality of code generation.

In recent years AI made great advances in all areas, from art to enhancing productivity in real-world tasks. One of the most notable technologies that brought these advancements is of course the [Large Language model (LLM)](https://en.wikipedia.org/wiki/Large_language_model). LLMs are useful in many areas but it seems that they are especially useful for the generation of source code. However, benchmarks for this area have been lacking, while benchmarks for general software development tasks are almost non-existent.

This repository gives developers of LLMs (and other code generation tools) a standardized benchmark and framework to improve real-world usage in the software development domain and provides users of LLMs with metrics to check if a given LLM is useful for their tasks.

## Installation

[Install Git](https://git-scm.com/downloads), [install Go](https://go.dev/doc/install), and then execute the following commands:

```bash
git clone https://github.com/symflower/eval-dev-quality.git
cd eval-dev-quality
go install -v github.com/symflower/eval-dev-quality/cmd/eval-dev-quality
```

You can now use the `eval-dev-quality` binary to [execute the benchmark](#usage).

## Usage

> REMARK This project does not currently implement a sandbox for executing code. Make sure that you are running benchmarks only inside of a sandbox, e.g. at least a container.

At the moment, the only LLM provider implemented is [openrouter.ai](https://openrouter.ai/). You need to create an [access key](https://openrouter.ai/keys) and save it in an environment variable:

```bash
export PROVIDER_TOKEN=openrouter:${your-key}
```

Then you can run all benchmark tasks on all models and repositories:

```bash
eval-dev-quality evaluate
```

The output of the commands is a detailed log of all the requests and responses to the models and of all the commands executed. After the execution, you can find the final result saved to the file `evaluation.csv`.

See `eval-dev-quality --help` and especially `eval-dev-quality evaluate --help` for options.

### Example usage: Evaluate only one or more models

In the case you only want to evaluate only one or more models you can use the `--model` option to define a model you want to use. You can use this option with as many models as you want.

Executing the following output:

```bash
eval-dev-quality evaluate --model=openrouter/anthropic/claude-3-opus
```

Should return an evaluation log like this:

````plain
2024/04/04 13:16:12 Checking that models and languages can be used for evaluation
2024/04/04 13:16:12 Evaluating model "openrouter/anthropic/claude-3-opus" using language "golang" and repository "golang/plain"
2024/04/04 13:16:15 Model "openrouter/anthropic/claude-3-opus" responded to query       Given the following Go code file "plain.go" with package "plain", provide a test file for this code.
        The tests should produce 100 percent code coverage and must compile.
        The response must contain only the test code and nothing else.

        ```golang
        package plain

        func plain() {
                return // This does not do anything but it gives us a line to cover.
        }
        ```
 with:  Here's the test file for the given Go code that provides 100 percent code coverage:

        ```golang
        package plain

        import "testing"

        func Test_plain(t *testing.T) {
                plain()
        }
        ```
2024/04/04 13:16:15 $ gotestsum --format standard-verbose --hide-summary skipped -- -cover -v -vet=off ./...
=== RUN   Test_plain
--- PASS: Test_plain (0.00s)
PASS
coverage: 100.0% of statements
ok      plain   0.001s  coverage: 100.0% of statements

DONE 1 tests in 0.169s
2024/04/04 13:16:15 Evaluated model "openrouter/anthropic/claude-3-opus" using language "golang" and repository "golang/plain": encountered 0 problems
2024/04/04 13:16:15 Evaluating models and languages
2024/04/04 13:16:15 Evaluation score for "openrouter/anthropic/claude-3-opus": #executed=100.0%(1/1), #problems=0.0%(0/1), average statement coverage=100.0%
````

The execution by default also creates an evaluation file `evaluation.csv` that contains:

```
model,files-total,files-executed,files-problems,coverage-statement
openrouter/anthropic/claude-3-opus,1,1,0,100
```

# How to extend the benchmark?

If you want to add new files to existing language repositories or new repositories to existing languages, [install the evaluation binary](#installation) of this repository and you are good to go.

To add new tasks to the benchmark, add features, or fix bugs, you'll need a development environment. The development environment comes with this repository and can be installed by executing `make install-all`. Then you can run `make` to see the documentation for all the available commands.

# How to contribute?

First of all, thank you for thinking about contributing! There are multiple ways to contribute:

-   Add more files to existing language repositories.
-   Add more repositories to languages.
-   Implement another language and add repositories for it.
-   Implement new tasks for existing languages and repositories.
-   Add more features and fix bugs in the evaluation, development environment, or CI: [best to have a look at the list of issues](https://github.com/symflower/eval-dev-quality/issues).

If you want to contribute but are unsure how: [create a discussion](https://github.com/symflower/eval-dev-quality/discussions) or write us directly at [markus.zimmermann@symflower.com](mailto:markus.zimmermann@symflower.com).
