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
eval-dev-quality evaluate --model=openrouter/meta-llama/llama-3-70b-instruct
```

Should return an evaluation log similar to this:

<details>
<summary>Log for the above command.</summary>

````plain
2024/05/02 10:01:58 Writing results to evaluation-2024-05-02-10:01:58
2024/05/02 10:01:58 Checking that models and languages can be used for evaluation
2024/05/02 10:01:58 Evaluating model "openrouter/meta-llama/llama-3-70b-instruct" using language "golang" and repository "golang/plain"
2024/05/02 10:01:58 Querying model "openrouter/meta-llama/llama-3-70b-instruct" with:
        Given the following Go code file "plain.go" with package "plain", provide a test file for this code.
        The tests should produce 100 percent code coverage and must compile.
        The response must contain only the test code and nothing else.

        ```golang
        package plain

        func plain() {
                return // This does not do anything but it gives us a line to cover.
        }
        ```
2024/05/02 10:02:00 Model "openrouter/meta-llama/llama-3-70b-instruct" responded with:
        ```go
        package plain

        import "testing"

        func TestPlain(t *testing.T) {
                plain()
        }
        ```
2024/05/02 10:02:00 $ symflower test --language golang --workspace /tmp/eval-dev-quality2330727502/plain
Checking for updates
There is a new version of symflower available! Please run `symflower update`.
=== RUN   TestPlain
--- PASS: TestPlain (0.00s)
PASS
coverage: 100.0% of statements
ok      plain   0.002s  coverage: 100.0% of statements

DONE 1 tests in 0.348s
2024/05/02 10:02:01 Evaluated model "openrouter/meta-llama/llama-3-70b-instruct" using language "golang" and repository "golang/plain": encountered 0 problems: []
2024/05/02 10:02:01 Evaluating model "openrouter/meta-llama/llama-3-70b-instruct" using language "java" and repository "java/plain"
2024/05/02 10:02:01 Querying model "openrouter/meta-llama/llama-3-70b-instruct" with:
        Given the following Java code file "src/main/java/com/eval/Plain.java" with package "com.eval", provide a test file for this code with JUnit 5 as a test framework.
        The tests should produce 100 percent code coverage and must compile.
        The response must contain only the test code and nothing else.

        ```java
        package com.eval;

        class Plain {
            static void plain() {
            }
        }
        ```
2024/05/02 10:02:02 Model "openrouter/meta-llama/llama-3-70b-instruct" responded with:
        ```java
        package com.eval;

        import org.junit.jupiter.api.Test;

        import static org.junit.jupiter.api.Assertions.assertFalse;
        import static org.junit.jupiter.api.Assertions.assertTrue;

        public class PlainTest {
            @Test
            void testPlain() {
                Plain.plain();
                // Since the method is empty, we can only assert that it doesn't throw an exception
                assertTrue(true);
            }
        }
        ```
2024/05/02 10:02:02 $ symflower test --language java --workspace /tmp/eval-dev-quality1094965069/plain
Total coverage 100.000000%
2024/05/02 10:02:09 Evaluated model "openrouter/meta-llama/llama-3-70b-instruct" using language "java" and repository "java/plain": encountered 0 problems: []
2024/05/02 10:02:09 Evaluating models and languages
2024/05/02 10:02:09 Evaluation score for "openrouter/meta-llama/llama-3-70b-instruct" ("code-no-excess"): score=12, coverage-statement=2, files-executed=2, response-no-error=2, response-no-excess=2, response-not-empty=2, response-with-code=2
````

</details>

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
