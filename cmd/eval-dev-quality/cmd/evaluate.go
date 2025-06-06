package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	pkgerrors "github.com/pkg/errors"
	"github.com/zimmski/osutil"
	"golang.org/x/exp/maps"

	"github.com/symflower/eval-dev-quality/evaluate"
	"github.com/symflower/eval-dev-quality/evaluate/report"
	evaltask "github.com/symflower/eval-dev-quality/evaluate/task"
	"github.com/symflower/eval-dev-quality/language"
	_ "github.com/symflower/eval-dev-quality/language/golang" // Register language.
	_ "github.com/symflower/eval-dev-quality/language/java"   // Register language.
	_ "github.com/symflower/eval-dev-quality/language/ruby"   // Register language.
	_ "github.com/symflower/eval-dev-quality/language/rust"   // Register language.
	"github.com/symflower/eval-dev-quality/log"
	"github.com/symflower/eval-dev-quality/model"
	"github.com/symflower/eval-dev-quality/model/llm"
	"github.com/symflower/eval-dev-quality/provider"
	_ "github.com/symflower/eval-dev-quality/provider/ollama" // Register provider.
	openaiapi "github.com/symflower/eval-dev-quality/provider/openai-api"
	_ "github.com/symflower/eval-dev-quality/provider/openrouter" // Register provider.
	_ "github.com/symflower/eval-dev-quality/provider/symflower"  // Register provider.
	"github.com/symflower/eval-dev-quality/task"
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
	// ModelIDsWithProviderAndAttributes determines which models should be used for the evaluation, or empty if all models should be used.
	ModelIDsWithProviderAndAttributes []string `long:"model" description:"Evaluate with this model. By default all models are used."`
	// ProviderTokens holds all API tokens for the providers.
	ProviderTokens map[string]string `long:"tokens" description:"API tokens for model providers (of the form '$provider:$token'). When using the environment variable, separate multiple definitions with ','." env:"PROVIDER_TOKEN" env-delim:","`
	// ProviderUrls holds all custom inference endpoint urls for the providers.
	ProviderUrls map[string]string `long:"urls" description:"Custom OpenAI API compatible inference endpoints (of the form '$provider:$url,...'). Use '$provider=custom-$name' to manually register a custom OpenAI API endpoint provider. Note that the models of a custom OpenAI API endpoint provider must be declared explicitly using the '--model' option. When using the environment variable, separate multiple definitions with ','." env:"PROVIDER_URL" env-delim:","`
	// APIRequestAttempts holds the number of allowed API requests per LLM query.
	APIRequestAttempts uint `long:"api-request-attempts" description:"Number of allowed API requests per LLM query." default:"3"`
	// APIRequestTimeout holds the timeout for API requests in seconds.
	APIRequestTimeout uint `long:"api-request-timeout" description:"Timeout of API requests in seconds. ('0' to disable)" default:"1200"`

	// Repositories determines which repository should be used for the evaluation, or empty if all repositories should be used.
	Repositories []string `long:"repository" description:"Evaluate with this repository. By default all repositories are used."`
	// ResultPath holds the directory path where results should be written to.
	ResultPath string `long:"result-path" description:"Directory path where results should be written to. The placeholder \"%datetime%\" can be used for the current date and time." default:"evaluation-%datetime%"`
	// TestdataPath determines the testdata path where all repositories reside grouped by languages.
	TestdataPath string `long:"testdata" description:"Path to the testdata directory where all repositories reside grouped by languages." default:"testdata/"`

	// Configuration holds the file path of an evaluation configuration file.
	Configuration string `long:"configuration" description:"Configuration file to set up an evaluation run."`
	// ExecutionTimeout holds the timeout for an execution.
	ExecutionTimeout uint `long:"execution-timeout" description:"Execution timeout for compilation and tests in minutes." default:"5"`
	// RunIDStartsAt holds the offset increment for the run id used in creating the result folders.
	RunIDStartsAt uint `long:"run-id-starts-at" description:"Sets the starting index for the run ID." default:"1"`
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

	// args holds a list of all the passed arguments.
	args []string
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

var _ SetArguments = (*Evaluate)(nil)

// SetArguments sets the commands arguments.
func (command *Evaluate) SetArguments(args []string) {
	availableFlags := util.Flags(command)
	command.args = util.FilterArgsKeep(args, availableFlags)
}

// Initialize initializes the command according to the arguments.
func (command *Evaluate) Initialize(args []string) (evaluationContext *evaluate.Context, evaluationConfiguration *EvaluationConfiguration, cleanup func()) {
	evaluationContext = &evaluate.Context{}
	evaluationConfiguration = NewEvaluationConfiguration()

	// Load the provided configuration file, if any.
	if command.Configuration != "" {
		if command.Runtime != "local" {
			command.logger.Panicf("the configuration file is not supported in containerized runtimes")
		}

		if len(command.ModelIDsWithProviderAndAttributes) > 0 || len(command.Repositories) > 0 {
			command.logger.Panicf("do not provide models and repositories when loading a configuration file")
		}

		configurationFile, err := os.Open(command.Configuration)
		if err != nil {
			command.logger.Panicf("ERROR: %s", err)
		}
		configuration, err := ReadEvaluationConfiguration(configurationFile)
		if err != nil {
			command.logger.Panicf("ERROR: %s", err)
		}
		if err := configurationFile.Close(); err != nil {
			command.logger.Panicf("ERROR: %s", err)
		}

		command.ModelIDsWithProviderAndAttributes = configuration.Models.Selected
		command.Repositories = configuration.Repositories.Selected
		command.APIRequestAttempts = uint(configuration.APIRequestAttempts)
	}

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

		if command.APIRequestAttempts == 0 {
			command.logger.Panicf("number of configured API request attempts must be greater than zero")
		}
		evaluationContext.APIReqestAttempts = command.APIRequestAttempts
		evaluationContext.APIRequestTimeout = command.APIRequestTimeout
		evaluationConfiguration.APIRequestAttempts = int(command.APIRequestAttempts)

		if command.ExecutionTimeout == 0 {
			command.logger.Panicf("execution timeout for compilation and tests must be greater than zero")
		} else {
			language.DefaultExecutionTimeout = time.Duration(command.ExecutionTimeout) * time.Minute
		}

		evaluationContext.RunIDStartsAt = command.RunIDStartsAt

		if command.Runs == 0 {
			command.logger.Panicf("number of configured runs must be greater than zero")
		}
		evaluationContext.Runs = command.Runs

		if command.Runtime == "docker" {
			if _, err := exec.LookPath("docker"); err != nil {
				command.logger.Panicf("docker runtime could not be found")
			}
		}

		if command.Parallel != 1 && command.Runtime == "local" {
			command.logger.Panicf("the 'parallel' parameter can't be used with local execution")
		}

		if command.Parallel == 0 {
			command.logger.Panicf("the 'parallel' parameter has to be greater then zero")
		}

		if command.RuntimeImage == "" {
			command.RuntimeImage = "ghcr.io/symflower/eval-dev-quality:main"
		}

		if command.Runtime == "kubernetes" && command.Namespace == "" {
			command.logger.Panicf("the namespace parameter can't be empty when using the kubernetes runtime")
		}

		evaluationContext.NoDisqualification = command.NoDisqualification
	}

	// Setup evaluation result directory.
	{
		command.ResultPath = strings.ReplaceAll(command.ResultPath, "%datetime%", command.timestamp.Format("2006-01-02-15:04:05")) // REMARK Use a datetime format with a dash, so directories can be easily marked because they are only one group.
		uniqueResultPath, err := util.UniqueDirectory(command.ResultPath)
		if err != nil {
			command.logger.Panicf("ERROR: %s", err)
		}
		// Ensure that the directory really exists.
		if err := osutil.MkdirAll(uniqueResultPath); err != nil {
			command.logger.Panicf("ERROR: %s", err)
		}
		command.ResultPath = uniqueResultPath
		evaluationContext.ResultPath = uniqueResultPath
		command.logger.Info("configured results directory", "path", command.ResultPath)
	}

	// Initialize logging within result directory.
	{
		log := command.logger.With(log.AttributeKeyResultPath, command.ResultPath)
		command.logger = log
		evaluationContext.Log = log
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

	// In a containerized runtime we check the availability of the testdata, repositories and models/providers inside the container.
	if command.Runtime != "local" {
		// Copy the models over.
		for _, modelID := range command.ModelIDsWithProviderAndAttributes {
			evaluationContext.Models = append(evaluationContext.Models, llm.NewModel(nil, modelID))
		}

		return evaluationContext, evaluationConfiguration, func() {}
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

	// Gather repositories and update language selection accordingly.
	{
		// First gather all available repositories to store in the configuration report.
		for _, l := range language.Languages {
			repositories, err := language.RepositoriesForLanguage(l, command.TestdataPath)
			if err != nil {
				command.logger.Panicf("ERROR: %s", err)
			}
			for _, r := range repositories {
				config, err := task.LoadRepositoryConfiguration(filepath.Join(command.TestdataPath, r), evaltask.AllIdentifiers)
				if err != nil {
					command.logger.Panicf("ERROR: %s", err)
				}
				evaluationConfiguration.Repositories.Available[r] = config.Tasks
			}
		}

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
					command.logger.Info("excluded repository because its language is not enabled for this evaluation", "repository", r, "language", languageIDOfRepository)
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
					command.logger.Info("excluded language because it is not part of the selected repositories", "language", languageID)
				}
			}

			command.Repositories = maps.Keys(commandRepositories)
			sort.Strings(command.Repositories)
		}
		evaluationContext.RepositoryPaths = command.Repositories
		evaluationConfiguration.Repositories.Selected = append(evaluationConfiguration.Repositories.Selected, command.Repositories...)
	}

	// Make the resolved selected languages available in the command.
	evaluationContext.Languages = make([]language.Language, len(command.Languages))
	for i, languageID := range command.Languages {
		evaluationContext.Languages[i] = languagesSelected[languageID]
	}

	// Gather models and initialize providers.
	var serviceShutdown []func() (err error)
	{
		// Gather providers.
		providers := map[string]provider.Provider{}
		if len(command.ModelIDsWithProviderAndAttributes) == 0 {
			for providerID, provider := range provider.Providers {
				providers[providerID] = provider
				command.logger.Info("selected provider", "provider", providerID)
			}
		} else {
			// Register custom providers.
			for providerID, providerURL := range command.ProviderUrls {
				if !strings.HasPrefix(providerID, "custom-") {
					command.logger.Panicf("ERROR: cannot set URL of %q because it is not a custom provider", providerID)
				}

				p := openaiapi.NewProvider(providerID, providerURL)
				provider.Register(p)
				providers[providerID] = p
				command.logger.Info("selected provider", "provider", providerID)
			}

			// Add remaining providers from models.
			for _, modelIDsWithProviderAndAttributes := range command.ModelIDsWithProviderAndAttributes {
				providerID, _, ok := strings.Cut(modelIDsWithProviderAndAttributes, provider.ProviderModelSeparator)
				if !ok {
					command.logger.Panicf("ERROR: cannot split %q into provider and model name by %q", modelIDsWithProviderAndAttributes, provider.ProviderModelSeparator)
				}

				p, ok := provider.Providers[providerID]
				if !ok {
					command.logger.Panicf("ERROR: unknown provider %q for model %q", providerID, modelIDsWithProviderAndAttributes)
				}
				if _, ok := providers[providerID]; !ok {
					providers[providerID] = p
					command.logger.Info("selected provider", "provider", providerID)
				}
			}
		}

		// Initialize providers.
		{
			providerIDsSorted := maps.Keys(providers)
			sort.Strings(providerIDsSorted)
			for _, providerID := range providerIDsSorted {
				p := providers[providerID]

				command.logger.Info("initializing provider", "provider", providerID)
				if t, ok := p.(provider.InjectToken); ok {
					if token, ok := command.ProviderTokens[p.ID()]; ok {
						command.logger.Info("set token of provider", "provider", providerID)
						t.SetToken(token)
					}
				}
				command.logger.Info("checking availability for provider", "provider", providerID)
				if err := p.Available(command.logger); err != nil {
					command.logger.Info("skipping provider because it is not available", "error", err, "provider", providerID)
					delete(providers, providerID)

					continue
				}
				if service, ok := p.(provider.Service); ok {
					command.logger.Info("starting services for provider", "provider", p.ID())
					shutdown, err := service.Start(command.logger)
					if err != nil {
						command.logger.Panicf("ERROR: could not start services for provider %q: %s", p, err)
					}
					serviceShutdown = append(serviceShutdown, shutdown)
				}
			}
		}

		// Gather models.
		models := map[string]model.Model{}
		{
			addAllModels := len(command.ModelIDsWithProviderAndAttributes) == 0
			for _, p := range providers {
				ms, err := p.Models()
				if err != nil {
					command.logger.Panicf("ERROR: could not query models for provider %q: %s", p.ID(), err)
				}
				for _, m := range ms {
					models[m.ID()] = m
					evaluationConfiguration.Models.Available = append(evaluationConfiguration.Models.Available, m.ID())

					if addAllModels {
						command.ModelIDsWithProviderAndAttributes = append(command.ModelIDsWithProviderAndAttributes, m.ID())
					}
				}
			}
		}
		modelIDs := maps.Keys(models)
		sort.Strings(modelIDs)
		sort.Strings(command.ModelIDsWithProviderAndAttributes)

		// Check and initialize models.
		evaluationContext.ProviderForModel = map[model.Model]provider.Provider{}
		for _, modelIDsWithProviderAndAttributes := range command.ModelIDsWithProviderAndAttributes {
			command.logger.Info("selecting model", "model", modelIDsWithProviderAndAttributes)

			providerID, modelIDsWithAttributes, ok := strings.Cut(modelIDsWithProviderAndAttributes, provider.ProviderModelSeparator)
			if !ok {
				command.logger.Panicf("ERROR: cannot split %q into provider and model name by %q", modelIDsWithProviderAndAttributes, provider.ProviderModelSeparator)
			}

			modelID, attributes := model.ParseModelID(modelIDsWithAttributes)
			modelIDWithProvider := providerID + provider.ProviderModelSeparator + modelID

			p, ok := providers[providerID]
			if !ok {
				command.logger.Panicf("ERROR: cannot find provider %q", providerID)
			}
			if puller, ok := p.(provider.Puller); ok {
				command.logger.Info("pulling model", "model", modelID)
				if err := puller.Pull(command.logger, modelID); err != nil {
					command.logger.Panicf("ERROR: could not pull model %q: %s", modelID, err)
				}

				// TODO If a model has not been pulled before, it was not available for at least the "Ollama" provider. Make this cleaner, we should not rebuild every time.
				if _, ok := models[modelIDWithProvider]; !ok {
					ms, err := p.Models()
					if err != nil {
						command.logger.Panicf("ERROR: could not query models for provider %q: %s", p.ID(), err)
					}
					for _, m := range ms {
						if _, ok := models[m.ModelID()]; ok {
							continue
						}

						models[m.ModelID()] = m
						evaluationConfiguration.Models.Available = append(evaluationConfiguration.Models.Available, m.ModelID())
					}
					modelIDs = maps.Keys(models)
					sort.Strings(modelIDs)
				}
			}

			var m model.Model
			if strings.HasPrefix(providerID, "custom-") {
				pc, ok := p.(*openaiapi.Provider)
				if !ok {
					command.logger.Panicf("ERROR: %q is not a custom provider", providerID)
				}

				m = llm.NewModel(pc, modelIDsWithProviderAndAttributes)
				pc.AddModel(m)
			} else {
				var ok bool
				m, ok = models[modelIDWithProvider]
				if !ok {
					command.logger.Panicf("ERROR: model %q does not exist for provider %q. Valid models are: %s", modelIDsWithProviderAndAttributes, providerID, strings.Join(modelIDs, ", "))
				}

				// If a model with attributes is requested, we add the base model plus attributes as new model to our list.
				if len(attributes) > 0 {
					modelWithAttributes := m.Clone()
					modelWithAttributes.SetAttributes(attributes)
					models[modelWithAttributes.ID()] = modelWithAttributes
					m = modelWithAttributes
				}
			}
			evaluationContext.Models = append(evaluationContext.Models, m)
			evaluationContext.ProviderForModel[m] = p
			evaluationConfiguration.Models.Selected = append(evaluationConfiguration.Models.Selected, modelIDsWithProviderAndAttributes)
		}
	}

	return evaluationContext, evaluationConfiguration, func() {
		for _, shutdown := range serviceShutdown {
			if err := shutdown(); err != nil {
				command.logger.Error(err.Error())
			}
		}
	}
}

// Execute executes the command.
func (command *Evaluate) Execute(args []string) (err error) {
	command.timestamp = time.Now()

	evaluationContext, evaluationConfiguration, cleanup := command.Initialize(args)
	defer cleanup()
	if evaluationContext == nil {
		command.logger.Panicf("ERROR: empty evaluation context")
	} else if evaluationConfiguration == nil {
		command.logger.Panicf("ERROR: empty evaluation configuration")
	}

	configurationFile, err := os.Create(filepath.Join(evaluationContext.ResultPath, "config.json"))
	if err != nil {
		command.logger.Panicf("ERROR: cannot create configuration file: %s", err)
	}
	defer func() {
		if err := configurationFile.Close(); err != nil {
			command.logger.Panicf("ERROR: %s", err)
		}
	}()
	if err := evaluationConfiguration.Write(configurationFile); err != nil {
		command.logger.Panicf("ERROR: %s", err)
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

	evaluate.Evaluate(evaluationContext)

	if err := (report.Markdown{
		DateTime: command.timestamp,
		Version:  evaluate.Version,
		Revision: evaluate.Revision,
	}).WriteToFile(filepath.Join(command.ResultPath, "README.md")); err != nil {
		command.logger.Panicf("ERROR: %s", err)
	}

	return nil
}

// evaluateDocker executes the evaluation for each model inside a docker container.
func (command *Evaluate) evaluateDocker(ctx *evaluate.Context) (err error) {
	ignoredFlags := []string{
		"model",
		"parallel",
		"result-path",
		"runtime-image",
		"runtime",
	}

	// Filter the args to remove all flags unsuited for running the container.
	args := util.FilterArgsRemove(command.args, ignoredFlags)

	parallel := util.NewParallel(command.Parallel)

	volumeName := "evaluation-volume"

	// Create data volume.
	{
		// Create the volume where all the data of the evaluations is stored.
		output, err := util.CommandWithResult(context.Background(), command.logger, &util.Command{
			Command: []string{
				"docker",
				"volume",
				"create",
				volumeName,
			},
		})
		if err != nil {
			return pkgerrors.WithMessage(pkgerrors.WithStack(err), output)
		}

		// Cleanup volume.
		defer func() {
			output, deferErr := util.CommandWithResult(context.Background(), command.logger, &util.Command{
				Command: []string{
					"docker",
					"volume",
					"rm",
					volumeName,
				},
			})
			if deferErr != nil {
				err = errors.Join(err, pkgerrors.WithMessage(pkgerrors.WithStack(deferErr), output))
			}
		}()
	}

	// Pull the image to ensure using the latest version
	{
		ctx.Log.Info("pulling latest version", "image", command.RuntimeImage)
		cmd := []string{
			"docker",
			"pull",
			command.RuntimeImage,
		}

		commandOutput, err := util.CommandWithResult(context.Background(), command.logger, &util.Command{
			Command: cmd,
		})
		if err != nil {
			ctx.Log.Error(
				"ERROR: unable to pull image",
				"error", pkgerrors.WithMessage(pkgerrors.WithStack(err), commandOutput),
			)
		}
	}

	// Iterate over each model and start the container.
	models := map[string]bool{}
	for i, model := range ctx.Models {
		// Commands regarding the docker runtime.
		dockerCommand := []string{
			"docker",
			"run",
			"-e", "PROVIDER_TOKEN",
			"-e", "SYMFLOWER_INTERNAL_LICENSE_FILE",
			"-e", "SYMFLOWER_LICENSE_KEY",
			"-v", volumeName + ":/app/evaluation",
			"--rm", // Automatically remove container after it finished.
			command.RuntimeImage,
		}

		// Commands for the evaluation to run inside the container.
		resultPath := "/app/evaluation/" + log.CleanModelNameForFileSystem(model.ID())
		if models[model.ID()] {
			resultPath += "_" + strconv.Itoa(i)
		} else {
			models[model.ID()] = true
		}

		evaluationCommand := []string{
			"eval-dev-quality",
			"evaluate",
			"--model", model.ID(),
			"--result-path", resultPath,
		}

		cmd := append(dockerCommand, evaluationCommand...)
		cmd = append(cmd, args...)

		parallel.Execute(func() {
			commandOutput, err := util.CommandWithResult(context.Background(), command.logger, &util.Command{
				Command: cmd,
			})
			if err != nil {
				command.logger.Error("docker evaluation failed", "error", pkgerrors.WithMessage(pkgerrors.WithStack(err), commandOutput))

				return
			}
		})
	}
	parallel.Wait()

	// Copy data from volume back to host.
	{
		// Run a container mounting the volume.
		output, err := util.CommandWithResult(context.Background(), command.logger, &util.Command{
			Command: []string{
				"docker",
				"run",
				"-d",
				"--name", "volume-fetch",
				"-v", volumeName + ":/data",
				"busybox",
				"true",
			},
		})
		if err != nil {
			return pkgerrors.WithMessage(pkgerrors.WithStack(err), output)
		}

		// Cleanup volume mount container.
		defer func() {
			output, deferErr := util.CommandWithResult(context.Background(), command.logger, &util.Command{
				Command: []string{
					"docker",
					"rm",
					"volume-fetch",
				},
			})
			if deferErr != nil {
				err = errors.Join(err, pkgerrors.WithMessage(pkgerrors.WithStack(deferErr), output))
			}
		}()

		// Copy data from volume to filesystem.
		resultPath, err := filepath.Abs(command.ResultPath)
		if err != nil {
			return pkgerrors.Wrap(err, "could not create an absolute path")
		}
		output, err = util.CommandWithResult(context.Background(), command.logger, &util.Command{
			Command: []string{
				"docker",
				"cp",
				"volume-fetch:/data/.",
				resultPath,
			},
		})
		if err != nil {
			return pkgerrors.WithMessage(pkgerrors.WithStack(err), output)
		}
	}

	return nil
}

// evaluateKubernetes executes the evaluation for each model inside a kubernetes run container.
func (command *Evaluate) evaluateKubernetes(ctx *evaluate.Context) (err error) {
	// Define a regex to replace all non alphanumeric characters and "-".
	kubeNameRegex := regexp.MustCompile(`[^a-zA-Z0-9-]+`)

	kubernetesJobTemplate, err := template.ParseFiles(filepath.Join("conf", "kube", "job.yml"))
	if err != nil {
		return pkgerrors.Wrap(err, "could not create kubernetes job template")
	}

	availableFlags := util.Flags(command)
	ignoredFlags := []string{
		"model",
		"parallel",
		"result-path",
		"runtime-image",
		"runtime",
	}

	// Filter all the args to only contain flags which can be used.
	args := util.FilterArgsKeep(os.Args[2:], availableFlags)
	// Filter the args to remove all flags unsuited for running the container.
	args = util.FilterArgsRemove(args, ignoredFlags)

	parallel := util.NewParallel(command.Parallel)

	// Iterate over each model and start the container.
	models := map[string]bool{}
	for i, model := range ctx.Models {
		// Commands regarding the docker runtime.
		kubeCommand := []string{
			"kubectl",
			"apply",
			"-f",
			"-", // Apply STDIN.
		}

		// Commands for the evaluation to run inside the container.
		resultPath := "/var/evaluations/" + log.CleanModelNameForFileSystem(model.ID())
		if models[model.ID()] {
			resultPath += "_" + strconv.Itoa(i)
		} else {
			models[model.ID()] = true
		}
		evaluationCommand := []string{
			"eval-dev-quality",
			"evaluate",
			"--model", model.ID(),
			"--result-path", resultPath,
		}
		cmd := append(evaluationCommand, args...)

		// Template data
		jobName := fmt.Sprintf("%s-%v", kubeNameRegex.ReplaceAllString(model.ID(), "-"), i)
		data := map[string]string{
			"name":      jobName,
			"namespace": command.Namespace,
			"image":     command.RuntimeImage,
			"command":   `["` + strings.Join(cmd, `","`) + `"]`,
		}

		parallel.Execute(func() {
			var kubernetesJobData bytes.Buffer
			if err := kubernetesJobTemplate.Execute(&kubernetesJobData, data); err != nil {
				command.logger.Panicf("ERROR: %s", err)
			}

			commandOutput, err := util.CommandWithResult(context.Background(), command.logger, &util.Command{
				Command: kubeCommand,
				Stdin:   kubernetesJobData.String(),
			})
			if err != nil {
				command.logger.Error("kubernetes evaluation failed", "error", pkgerrors.WithMessage(pkgerrors.WithStack(err), commandOutput))

				return
			}

			// This should block until the job is completed.
			commandOutput, err = util.CommandWithResult(context.Background(), command.logger, &util.Command{
				Command: []string{
					"kubectl",
					"wait",
					"--timeout", "24h",
					"--for=condition=complete",
					"--namespace",
					command.Namespace,
					"jobs/" + jobName,
				},
			})
			if err != nil {
				command.logger.Error("kubernetes evaluation failed", "error", pkgerrors.WithMessage(pkgerrors.WithStack(err), commandOutput))

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
				command.logger.Error("kubernetes evaluation failed", "error", pkgerrors.WithMessage(pkgerrors.WithStack(err), commandOutput))

				return
			}
		})
	}
	parallel.Wait()

	// Copy data from volume back to host.
	{
		storageTemplate, err := template.ParseFiles(filepath.Join("conf", "kube", "storage-access.yml"))
		if err != nil {
			return pkgerrors.Wrap(err, "could not create Kubernetes storage access template")
		}

		data := map[string]string{
			"name":      "eval-storage-access",
			"namespace": command.Namespace,
		}

		var storageTemplateData bytes.Buffer
		if err := storageTemplate.Execute(&storageTemplateData, data); err != nil {
			return pkgerrors.Wrap(err, "could not execute storage template")
		}

		// Create the storage access pod.
		output, err := util.CommandWithResult(context.Background(), command.logger, &util.Command{
			Command: []string{
				"kubectl",
				"apply",
				"-f",
				"-", // Apply STDIN.
			},
			Stdin: storageTemplateData.String(),
		})
		if err != nil {
			return pkgerrors.WithMessage(pkgerrors.WithStack(err), output)
		}

		// Fetch the container name.
		output, err = util.CommandWithResult(context.Background(), command.logger, &util.Command{
			Command: []string{
				"kubectl",
				"get",
				"pods",
				"--namespace", command.Namespace,
				"-l", "app=eval-storage-access",
				"-o", "custom-columns=:metadata.name",
			},
			Stdin: storageTemplateData.String(),
		})
		if err != nil {
			return pkgerrors.WithMessage(pkgerrors.WithStack(err), output)
		}
		podName := strings.TrimSpace(output)

		// Copy data from volume to filesystem.
		output, err = util.CommandWithResult(context.Background(), command.logger, &util.Command{
			Command: []string{
				"kubectl",
				"cp",
				"--namespace", command.Namespace,
				command.Namespace + "/" + podName + ":/var/evaluations/.",
				command.ResultPath,
			},
		})
		if err != nil {
			return pkgerrors.WithMessage(pkgerrors.WithStack(err), output)
		}

		// Remove the data from the cluster volume
		output, err = util.CommandWithResult(context.Background(), command.logger, &util.Command{
			Command: []string{
				"kubectl",
				"exec",
				"--namespace", command.Namespace,
				podName,
				"--",
				"sh", "-c",
				"rm -rf /var/evaluations/*",
			},
		})
		if err != nil {
			return pkgerrors.WithMessage(pkgerrors.WithStack(err), output)
		}
	}

	return nil
}
