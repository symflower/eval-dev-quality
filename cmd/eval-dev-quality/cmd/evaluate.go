package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"text/template"
	"time"

	pkgerrors "github.com/pkg/errors"
	"github.com/zimmski/osutil"
	"golang.org/x/exp/maps"

	"github.com/symflower/eval-dev-quality/evaluate"
	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	"github.com/symflower/eval-dev-quality/evaluate/report"
	"github.com/symflower/eval-dev-quality/language"
	_ "github.com/symflower/eval-dev-quality/language/golang" // Register language.
	_ "github.com/symflower/eval-dev-quality/language/java"   // Register language.
	"github.com/symflower/eval-dev-quality/log"
	"github.com/symflower/eval-dev-quality/model"
	"github.com/symflower/eval-dev-quality/model/llm"
	"github.com/symflower/eval-dev-quality/provider"
	_ "github.com/symflower/eval-dev-quality/provider/ollama" // Register provider.
	openaiapi "github.com/symflower/eval-dev-quality/provider/openai-api"
	_ "github.com/symflower/eval-dev-quality/provider/openrouter" // Register provider.
	_ "github.com/symflower/eval-dev-quality/provider/symflower"  // Register provider.
	"github.com/symflower/eval-dev-quality/tools"
	"github.com/symflower/eval-dev-quality/util"
)

// Evaluate holds the "evaluation" command.
type Evaluate struct {
	// InstallToolsPath determines where tools for the evaluation are installed.
	InstallToolsPath string `long:"install-tools-path" description:"Install tools for the evaluation into this path."`
	// OllamaBinaryPath overwrites the Ollama binary path.
	OllamaBinaryPath string `long:"ollama-binary-path" description:"Overwrite the Ollama binary with this specific path instead of using a global one." env:"OLLAMA_BINARY_PATH"`
	// OllamaURL overwrites the Ollama URL.
	OllamaURL string `long:"ollama-url" description:"Overwrite the URL of the Ollama service." env:"OLLAMA_URL"`
	// SymflowerBinaryPath overwrites the Symflower binary path.
	SymflowerBinaryPath string `long:"symflower-binary-path" description:"Overwrite the Symflower binary with this specific path instead of installing and using a global one." env:"SYMFLOWER_BINARY_PATH"`

	// Languages determines which language should be used for the evaluation, or empty if all languages should be used.
	Languages []string `long:"language" description:"Evaluate with this language. By default all languages are used."`
	// Models determines which models should be used for the evaluation, or empty if all models should be used.
	Models []string `long:"model" description:"Evaluate with this model. By default all models are used."`
	// ProviderTokens holds all API tokens for the providers.
	ProviderTokens map[string]string `long:"tokens" description:"API tokens for model providers (of the form '$provider:$token'). When using the environment variable, separate multiple definitions with ','." env:"PROVIDER_TOKEN" env-delim:","`
	// ProviderUrls holds all custom inference endpoint urls for the providers.
	ProviderUrls map[string]string `long:"urls" description:"Custom OpenAI API compatible inference endpoints (of the form '$provider:$url,...'). Use '$provider=custom-$name' to manually register a custom OpenAI API endpoint provider. Note that the models of a custom OpenAI API endpoint provider must be declared explicitly using the '--model' option. When using the environment variable, separate multiple definitions with ','." env:"PROVIDER_URL" env-delim:","`
	// QueryAttempts holds the number of query attempts to perform when a model request errors in the process of solving a task.
	QueryAttempts uint `long:"attempts" description:"Number of query attempts to perform when a model request errors in the process of solving a task." default:"3"`

	// Repositories determines which repository should be used for the evaluation, or empty if all repositories should be used.
	Repositories []string `long:"repository" description:"Evaluate with this repository. By default all repositories are used."`
	// ResultPath holds the directory path where results should be written to.
	ResultPath string `long:"result-path" description:"Directory path where results should be written to. The placeholder \"%datetime%\" can be used for the current date and time." default:"evaluation-%datetime%"`
	// TestdataPath determines the testdata path where all repositories reside grouped by languages.
	TestdataPath string `long:"testdata" description:"Path to the testdata directory where all repositories reside grouped by languages." default:"testdata/"`

	// ExecutionTimeout holds the timeout for an execution.
	ExecutionTimeout uint `long:"execution-timeout" description:"Execution timeout for compilation and tests in minutes." default:"5"`
	// Runs holds the number of runs to perform.
	Runs uint `long:"runs" description:"Number of runs to perform." default:"1"`
	// RunsSequential indicates that interleaved runs are disabled and runs are performed sequentially.
	RunsSequential bool `long:"runs-sequential" description:"By default, multiple runs are performed in an interleaved fashion to avoid caching of model responses. Changing this behavior to \"sequential\" queries the same model repeatedly instead."`
	// NoDisqualification indicates that models are not to be disqualified if they fail to solve basic language tasks.
	NoDisqualification bool `long:"no-disqualification" description:"By default, models that cannot solve basic language tasks are disqualified for more complex tasks. Overwriting this behavior runs all tasks regardless."`

	// Runtime indicates if the evaluation is run locally or inside a container.
	Runtime string `long:"runtime" description:"The runtime which will be used for the evaluation." default:"local" choice:"local" choice:"docker" choice:"kubernetes"`
	// RuntimeImage determines the container image used for any container runtime.
	RuntimeImage string `long:"runtime-image" description:"The container image to use for the evaluation." default:""`
	// Parallel holds the number of parallel executed runs.
	Parallel uint `long:"parallel" description:"Amount of parallel containerized executed runs." default:"1"`
	// Namespace the namespace under which the kubernetes resources should be created.
	Namespace string `long:"namespace" description:"The Namespace which should be used for kubernetes resources." default:"eval-dev-quality"`

	// logger holds the logger of the command.
	logger *log.Logger
	// timestamp holds the timestamp of the command execution.
	timestamp time.Time
}

var _ SetLogger = (*Evaluate)(nil)

// SetLogger sets the logger of the command.
func (command *Evaluate) SetLogger(logger *log.Logger) {
	command.logger = logger
}

// Initialize initializes the command according to the arguments.
func (command *Evaluate) Initialize(args []string) (evaluationContext *evaluate.Context, cleanup func()) {
	// Ensure the cleanup always runs in case there is a panic.
	defer func() {
		if r := recover(); r != nil {
			if cleanup != nil {
				cleanup()
			}
			panic(r)
		}
	}()
	evaluationContext = &evaluate.Context{}

	// Check and validate common options.
	{
		if command.InstallToolsPath == "" {
			installToolsPath, err := tools.InstallPathDefault()
			if err != nil {
				command.logger.Panicf("ERROR: %s", err)
			}
			command.InstallToolsPath = installToolsPath
		}

		if command.OllamaBinaryPath != "" {
			tools.OllamaPath = command.OllamaBinaryPath
		}
		if command.OllamaURL != "" {
			tools.OllamaURL = command.OllamaURL
		}

		if command.SymflowerBinaryPath != "" {
			tools.SymflowerPath = command.SymflowerBinaryPath
		}

		if command.QueryAttempts == 0 {
			command.logger.Panicf("number of configured query attempts must be greater than zero")
		}
		evaluationContext.QueryAttempts = command.QueryAttempts

		if command.ExecutionTimeout == 0 {
			command.logger.Panicf("execution timeout for compilation and tests must be greater than zero")
		} else {
			language.DefaultExecutionTimeout = time.Duration(command.ExecutionTimeout) * time.Minute
		}

		if command.Runs == 0 {
			command.logger.Panicf("number of configured runs must be greater than zero")
		}
		evaluationContext.Runs = command.Runs

		if command.Runtime == "docker" {
			if _, err := exec.LookPath("docker"); err != nil {
				command.logger.Panic("docker runtime could not be found")
			}
		}

		if command.Parallel != 1 && command.Runtime == "local" {
			command.logger.Panic("the 'parallel' parameter can't be used with local execution")
		}

		if command.Parallel == 0 {
			command.logger.Panic("the 'parallel' parameter has to be greater then zero")
		}

		if command.RuntimeImage == "" {
			command.RuntimeImage = "ghcr.io/symflower/eval-dev-quality:main"
		}

		if command.Runtime == "kubernetes" && command.Namespace == "" {
			command.logger.Panic("the namespace parameter can't be empty when using the kubernetes runtime")
		}

		evaluationContext.NoDisqualification = command.NoDisqualification
	}

	// Ensure the "testdata" path exists and make it absolute.
	{
		if err := osutil.DirExists(command.TestdataPath); err != nil {
			command.logger.Panicf("ERROR: testdata path %q cannot be accessed: %s", command.TestdataPath, err)
		}
		testdataPath, err := filepath.Abs(command.TestdataPath)
		if err != nil {
			command.logger.Panicf("ERROR: could not resolve testdata path %q to an absolute path: %s", command.TestdataPath, err)
		}
		command.TestdataPath = testdataPath
		evaluationContext.TestdataPath = testdataPath
	}

	// Setup evaluation result directory.
	{
		command.ResultPath = strings.ReplaceAll(command.ResultPath, "%datetime%", command.timestamp.Format("2006-01-02-15:04:05")) // REMARK Use a datetime format with a dash, so directories can be easily marked because they are only one group.
		uniqueResultPath, err := util.UniqueDirectory(command.ResultPath)
		if err != nil {
			command.logger.Panicf("ERROR: %s", err)
		}
		command.ResultPath = uniqueResultPath
		evaluationContext.ResultPath = uniqueResultPath
		command.logger.Printf("Writing results to %s", command.ResultPath)
	}

	// Initialize logging within result directory.
	{
		log, logClose, err := log.WithFile(command.logger, filepath.Join(command.ResultPath, "evaluation.log"))
		if err != nil {
			command.logger.Panicf("ERROR: %s", err)
		}
		cleanup = logClose
		command.logger = log
		evaluationContext.Log = log
	}

	// Register custom OpenAI API providers and models.
	{
		customProviders := map[string]*openaiapi.Provider{}
		for providerID, providerURL := range command.ProviderUrls {
			if !strings.HasPrefix(providerID, "custom-") {
				continue
			}

			p := openaiapi.NewProvider(providerID, providerURL)
			provider.Register(p)
			customProviders[providerID] = p
		}
		for _, model := range command.Models {
			if !strings.HasPrefix(model, "custom-") {
				continue
			}

			providerID, _, ok := strings.Cut(model, provider.ProviderModelSeparator)
			if !ok {
				command.logger.Panicf("ERROR: cannot split %q into provider and model name by %q", model, provider.ProviderModelSeparator)
			}
			modelProvider, ok := customProviders[providerID]
			if !ok {
				command.logger.Panicf("ERROR: unknown custom provider %q for model %q", providerID, model)
			}

			modelProvider.AddModel(llm.NewModel(modelProvider, model))
		}
	}

	// Gather languages.
	languagesSelected := map[string]language.Language{}
	{
		languages := map[string]language.Language{}
		if len(command.Languages) == 0 {
			command.Languages = maps.Keys(language.Languages)
			languages = language.Languages
		} else {
			for _, languageID := range command.Languages {
				l, ok := language.Languages[languageID]
				if !ok {
					ls := maps.Keys(language.Languages)
					sort.Strings(ls)

					command.logger.Panicf("ERROR: language %s does not exist. Valid languages are: %s", languageID, strings.Join(ls, ", "))
				}

				languages[languageID] = l
			}
		}

		sort.Strings(command.Languages)
		for _, languageID := range command.Languages {
			languagesSelected[languageID] = languages[languageID]
		}

	}

	// Gather repositories and update language selection accordingly.
	{
		if len(command.Repositories) == 0 {
			for _, l := range command.Languages {
				repositories, err := language.RepositoriesForLanguage(language.Languages[l], command.TestdataPath)
				if err != nil {
					command.logger.Panicf("ERROR: %s", err)
				}
				command.Repositories = append(command.Repositories, repositories...)
			}
			sort.Strings(command.Repositories)
		} else {
			commandRepositories := map[string]bool{}
			commandRepositoriesLanguages := map[string]bool{}
			for _, r := range command.Repositories {
				languageIDOfRepository := strings.SplitN(r, string(os.PathSeparator), 2)[0]
				if _, ok := languagesSelected[languageIDOfRepository]; ok {
					commandRepositories[r] = true
					commandRepositoriesLanguages[languageIDOfRepository] = true
				} else {
					command.logger.Printf("Excluded repository %s because its language %q is not enabled for this evaluation", r, languageIDOfRepository)
				}
			}
			for languageID := range languagesSelected {
				if commandRepositoriesLanguages[languageID] { // Also add the plain repository in case we already have repositories for this language.
					commandRepositories[filepath.Join(languageID, evaluate.RepositoryPlainName)] = true
				} else {
					command.Languages = slices.DeleteFunc(command.Languages, func(l string) bool {
						return l == languageID
					})
					delete(languagesSelected, languageID)
					command.logger.Printf("Excluded language %q because it is not part of the selected repositories", languageID)
				}
			}

			command.Repositories = maps.Keys(commandRepositories)
			sort.Strings(command.Repositories)
		}
		evaluationContext.RepositoryPaths = command.Repositories
	}

	// Make the resolved selected languages available in the command.
	evaluationContext.Languages = make([]language.Language, len(command.Languages))
	for i, languageID := range command.Languages {
		evaluationContext.Languages[i] = languagesSelected[languageID]
	}

	// Gather models.
	{
		models := map[string]model.Model{}
		modelsSelected := map[string]model.Model{}
		evaluationContext.ProviderForModel = map[model.Model]provider.Provider{}
		for _, p := range provider.Providers {
			command.logger.Printf("Checking provider %q for models", p.ID())

			if t, ok := p.(provider.InjectToken); ok {
				token, ok := command.ProviderTokens[p.ID()]
				if ok {
					t.SetToken(token)
				}
			}
			if err := p.Available(command.logger); err != nil {
				command.logger.Printf("Skipping unavailable provider %q cause: %s", p.ID(), err)

				continue
			}

			// Start services of providers.
			if service, ok := p.(provider.Service); ok {
				command.logger.Printf("Starting services for provider %q", p.ID())
				shutdown, err := service.Start(command.logger)
				if err != nil {
					command.logger.Panicf("ERROR: could not start services for provider %q: %s", p, err)
				}
				defer func() {
					if err := shutdown(); err != nil {
						command.logger.Panicf("ERROR: could not shutdown services of provider %q: %s", p, err)
					}
				}()
			}

			ms, err := p.Models()
			if err != nil {
				command.logger.Panicf("ERROR: could not query models for provider %q: %s", p.ID(), err)
			}

			for _, m := range ms {
				models[m.ID()] = m
				evaluationContext.ProviderForModel[m] = p
			}
		}
		modelIDs := maps.Keys(models)
		sort.Strings(modelIDs)
		if len(command.Models) == 0 {
			command.Models = modelIDs
		} else {
			for _, modelID := range command.Models {
				if _, ok := models[modelID]; !ok {
					command.logger.Panicf("ERROR: model %s does not exist. Valid models are: %s", modelID, strings.Join(modelIDs, ", "))
				}
			}
		}
		sort.Strings(command.Models)
		for _, modelID := range command.Models {
			modelsSelected[modelID] = models[modelID]
		}

		// Make the resolved selected models available in the command.
		evaluationContext.Models = make([]model.Model, len(command.Models))
		for i, modelID := range command.Models {
			evaluationContext.Models[i] = modelsSelected[modelID]
		}
	}

	return evaluationContext, cleanup
}

// Execute executes the command.
func (command *Evaluate) Execute(args []string) (err error) {
	command.timestamp = time.Now()

	evaluationContext, cleanup := command.Initialize(args)
	defer cleanup()
	if evaluationContext == nil {
		command.logger.Panic("ERROR: empty evaluation context")
	}

	switch command.Runtime {
	case "local":
		return command.evaluateLocal(evaluationContext)
	case "docker":
		return command.evaluateDocker(evaluationContext)
	case "kubernetes":
		return command.evaluateKubernetes(evaluationContext)
	default:
		command.logger.Panicf("ERROR: unknown runtime")
	}

	return nil
}

// evaluateLocal executes the evaluation on the current system.
func (command *Evaluate) evaluateLocal(evaluationContext *evaluate.Context) (err error) {
	// Install required tools for the basic evaluation.
	if err := tools.InstallEvaluation(command.logger, command.InstallToolsPath); err != nil {
		command.logger.Panicf("ERROR: %s", err)
	}

	assessments, totalScore := evaluate.Evaluate(evaluationContext)

	assessmentsPerModel := assessments.CollapseByModel()
	if err := (report.Markdown{
		DateTime: command.timestamp,
		Version:  evaluate.Version,
		Revision: evaluate.Revision,

		CSVPath:       "./evaluation.csv",
		LogPath:       "./evaluation.log",
		ModelLogsPath: ".",
		SVGPath:       "./categories.svg",

		AssessmentPerModel: assessmentsPerModel,
		TotalScore:         totalScore,
	}).WriteToFile(filepath.Join(command.ResultPath, "README.md")); err != nil {
		command.logger.Panicf("ERROR: %s", err)
	}

	_ = assessmentsPerModel.WalkByScore(func(model model.Model, assessment metrics.Assessments, score uint64) (err error) {
		command.logger.Printf("Evaluation score for %q (%q): %s", model.ID(), assessment.Category(totalScore).ID, assessment)

		return nil
	})

	return nil
}

// evaluateDocker executes the evaluation for each model inside a docker container.
func (command *Evaluate) evaluateDocker(ctx *evaluate.Context) (err error) {
	availableFlags := util.Flags(command)
	ignoredFlags := []string{
		"model",
		"parallel",
		"result-path",
		"runtime",
	}

	// Filter all the args to only contain flags which can be used.
	args := util.FilterArgsKeep(os.Args[2:], availableFlags)
	// Filter the args to remove all flags unsuited for running the container.
	args = util.FilterArgsRemove(args, ignoredFlags)

	parallel := util.NewParallel(command.Parallel)

	// Iterate over each model and start the container.
	for _, model := range ctx.Models {
		// We are skipping ollama models until we fully support pulling. https://github.com/symflower/eval-dev-quality/issues/100.
		if ctx.ProviderForModel[model].ID() == "ollama" {
			command.logger.Print("Skipping unsupported ollama model with docker runtime")

			continue
		}

		// Create for each model a dedicated subfolder inside the results path.
		resultPath, err := filepath.Abs(command.ResultPath)
		if err != nil {
			return err
		}
		// Set permission 777 so the non-root docker image is able to store its results inside the result path.
		if err := os.Chmod(resultPath, 0777); err != nil {
			return err
		}

		// Commands regarding the docker runtime.
		dockerCommand := []string{
			"docker",
			"run",
			"-e", "PROVIDER_TOKEN",
			"-v", // bind volume
			resultPath + ":/home/ubuntu/evaluation",
			"--rm", // automatically remove container after it finished
			command.RuntimeImage,
		}

		// Commands for the evaluation to run inside the container.
		evaluationCommand := []string{
			"eval-dev-quality",
			"evaluate",
			"--model",
			model.ID(),
			"--result-path",
			"/home/ubuntu/evaluation/" + model.ID(),
		}

		cmd := append(dockerCommand, evaluationCommand...)
		cmd = append(cmd, args...)

		parallel.Execute(func() {
			commandOutput, err := util.CommandWithResult(context.Background(), command.logger, &util.Command{
				Command: cmd,
			})
			if err != nil {
				command.logger.Printf("ERROR: %s", pkgerrors.WithMessage(pkgerrors.WithStack(err), commandOutput))

				return
			}
		})
	}
	parallel.Wait()

	return nil
}

// evaluateKubernetes executes the evaluation for each model inside a kubernetes run container.
func (command *Evaluate) evaluateKubernetes(ctx *evaluate.Context) (err error) {
	tmpl, err := template.ParseFiles(filepath.Join("conf", "kubernetes-job.yml"))
	if err != nil {
		return pkgerrors.Wrap(err, "could not create kubernetes job template")
	}

	availableFlags := util.Flags(command)
	ignoredFlags := []string{
		"model",
		"parallel",
		"result-path",
		"runtime",
	}

	// Filter all the args to only contain flags which can be used.
	args := util.FilterArgsKeep(os.Args[2:], availableFlags)
	// Filter the args to remove all flags unsuited for running the container.
	args = util.FilterArgsRemove(args, ignoredFlags)

	parallel := util.NewParallel(command.Parallel)

	// Iterate over each model and start the container.
	for i, model := range ctx.Models {
		// We are skipping ollama models until we fully support pulling. https://github.com/symflower/eval-dev-quality/issues/100.
		if ctx.ProviderForModel[model].ID() == "ollama" {
			command.logger.Print("Skipping unsupported ollama model with kubernetes runtime")

			continue
		}

		// Commands regarding the docker runtime.
		kubeCommand := []string{
			"kubectl",
			"apply",
			"-f",
			"-", // apply STDIN
		}

		// Commands for the evaluation to run inside the container.
		evaluationCommand := []string{
			"eval-dev-quality",
			"evaluate",
			"--model",
			model.ID(),
			"--result-path",
			"/var/evaluations/" + model.ID(),
		}
		cmd := append(evaluationCommand, args...)

		// Template data
		nameReplacer := strings.NewReplacer(
			"/", "-",
			"\\", "-",
			":", "-",
		)
		jobName := fmt.Sprintf("%s-%v", nameReplacer.Replace(model.ID()), i)
		data := map[string]string{
			"name":      jobName,
			"namespace": command.Namespace,
			"image":     command.RuntimeImage,
			"command":   `["` + strings.Join(cmd, `","`) + `"]`,
		}

		parallel.Execute(func() {
			var tmplData bytes.Buffer
			tmpl.Execute(&tmplData, data)

			commandOutput, err := util.CommandWithResult(context.Background(), command.logger, &util.Command{
				Command: kubeCommand,
				Stdin:   tmplData.String(),
			})
			if err != nil {
				command.logger.Printf("ERROR: %s", pkgerrors.WithMessage(pkgerrors.WithStack(err), commandOutput))

				return
			}

			// This should block until the job is completed.
			commandOutput, err = util.CommandWithResult(context.Background(), command.logger, &util.Command{
				Command: []string{
					"kubectl",
					"wait",
					"--for=condition=complete",
					"--namespace",
					command.Namespace,
					"jobs/" + jobName,
				},
			})
			if err != nil {
				command.logger.Printf("ERROR: %s", pkgerrors.WithMessage(pkgerrors.WithStack(err), commandOutput))

				return
			}

			// Remove the job from the cluster.
			commandOutput, err = util.CommandWithResult(context.Background(), command.logger, &util.Command{
				Command: []string{
					"kubectl",
					"--namespace",
					command.Namespace,
					"delete",
					"job",
					jobName,
				},
			})
			if err != nil {
				command.logger.Printf("ERROR: %s", pkgerrors.WithMessage(pkgerrors.WithStack(err), commandOutput))

				return
			}
		})
	}
	parallel.Wait()

	return nil
}
