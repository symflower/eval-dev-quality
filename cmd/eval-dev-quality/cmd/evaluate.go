package cmd

import (
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
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

	// Runs holds the number of runs to perform.
	Runs uint `long:"runs" description:"Number of runs to perform." default:"1"`
	// RunsSequential indicates that interleaved runs are disabled and runs are performed sequentially.
	RunsSequential bool `long:"runs-sequential" description:"By default, multiple runs are performed in an interleaved fashion to avoid caching of model responses. Changing this behavior to \"sequential\" queries the same model repeatedly instead."`

	// logger holds the logger of the command.
	logger *log.Logger
}

var _ SetLogger = (*Evaluate)(nil)

// SetLogger sets the logger of the command.
func (command *Evaluate) SetLogger(logger *log.Logger) {
	command.logger = logger
}

// Execute executes the command.
func (command *Evaluate) Execute(args []string) (err error) {
	evaluationTimestamp := time.Now()
	command.ResultPath = strings.ReplaceAll(command.ResultPath, "%datetime%", evaluationTimestamp.Format("2006-01-02-15:04:05")) // REMARK Use a datetime format with a dash, so directories can be easily marked because they are only one group.
	command.logger.Printf("Writing results to %s", command.ResultPath)

	log, logClose, err := log.WithFile(command.logger, filepath.Join(command.ResultPath, "evaluation.log"))
	if err != nil {
		return err
	}
	defer logClose()

	// Check common options.
	{
		if command.InstallToolsPath == "" {
			command.InstallToolsPath, err = tools.InstallPathDefault()
			if err != nil {
				log.Panicf("ERROR: %s", err)
			}
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
			log.Panicf("number of configured query attempts must be greater than zero")
		}

		if command.Runs == 0 {
			log.Panicf("number of configured runs must be greater than zero")
		}
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
				log.Panicf("ERROR: cannot split %q into provider and model name by %q", model, provider.ProviderModelSeparator)
			}
			modelProvider, ok := customProviders[providerID]
			if !ok {
				log.Panicf("ERROR: unknown custom provider %q for model %q", providerID, model)
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

					log.Panicf("ERROR: language %s does not exist. Valid languages are: %s", languageID, strings.Join(ls, ", "))
				}

				languages[languageID] = l
			}
		}
		sort.Strings(command.Languages)
		for _, languageID := range command.Languages {
			languagesSelected[languageID] = languages[languageID]
		}
	}

	commandRepositories := map[string]bool{}
	commandRepositoriesLanguages := map[string]bool{}
	for _, r := range command.Repositories {
		languageIDOfRepository := strings.SplitN(r, string(os.PathSeparator), 2)[0]
		commandRepositoriesLanguages[languageIDOfRepository] = true

		if _, ok := languagesSelected[languageIDOfRepository]; ok {
			commandRepositories[r] = true
		} else {
			log.Printf("Excluded repository %s because its language %q is not enabled for this evaluation", r, languageIDOfRepository)
		}
	}
	for languageID := range languagesSelected {
		if len(command.Repositories) == 0 || commandRepositoriesLanguages[languageID] {
			commandRepositories[filepath.Join(languageID, evaluate.RepositoryPlainName)] = true
		} else {
			command.Languages = slices.DeleteFunc(command.Languages, func(l string) bool {
				return l == languageID
			})
			delete(languagesSelected, languageID)
			log.Printf("Excluded language %q because it is not part of the selected repositories", languageID)
		}
	}
	command.Repositories = maps.Keys(commandRepositories)
	sort.Strings(command.Repositories)

	// Gather models.
	modelsSelected := map[string]model.Model{}
	providerForModel := map[model.Model]provider.Provider{}
	{
		models := map[string]model.Model{}
		for _, p := range provider.Providers {
			log.Printf("Checking provider %q for models", p.ID())

			if t, ok := p.(provider.InjectToken); ok {
				token, ok := command.ProviderTokens[p.ID()]
				if ok {
					t.SetToken(token)
				}
			}
			if err := p.Available(log); err != nil {
				log.Printf("Skipping unavailable provider %q cause: %s", p.ID(), err)

				continue
			}

			// Start services of providers.
			if service, ok := p.(provider.Service); ok {
				log.Printf("Starting services for provider %q", p.ID())
				shutdown, err := service.Start(log)
				if err != nil {
					log.Panicf("ERROR: could not start services for provider %q: %s", p, err)
				}
				defer func() {
					if err := shutdown(); err != nil {
						log.Panicf("ERROR: could not shutdown services of provider %q: %s", p, err)
					}
				}()
			}

			ms, err := p.Models()
			if err != nil {
				log.Panicf("ERROR: could not query models for provider %q: %s", p.ID(), err)
			}

			for _, m := range ms {
				models[m.ID()] = m
				providerForModel[m] = p
			}
		}
		modelIDs := maps.Keys(models)
		sort.Strings(modelIDs)
		if len(command.Models) == 0 {
			command.Models = modelIDs
		} else {
			for _, modelID := range command.Models {
				if _, ok := models[modelID]; !ok {
					log.Panicf("ERROR: model %s does not exist. Valid models are: %s", modelID, strings.Join(modelIDs, ", "))
				}
			}
		}
		sort.Strings(command.Models)
		for _, modelID := range command.Models {
			modelsSelected[modelID] = models[modelID]
		}
	}

	if err := osutil.DirExists(command.TestdataPath); err != nil {
		log.Panicf("ERROR: testdata path %q cannot be accessed: %s", command.TestdataPath, err)
	}
	command.TestdataPath, err = filepath.Abs(command.TestdataPath)
	if err != nil {
		log.Panicf("ERROR: could not resolve testdata path %q to an absolute path: %s", command.TestdataPath, err)
	}

	// Install required tools for the basic evaluation.
	if err := tools.InstallEvaluation(log, command.InstallToolsPath); err != nil {
		log.Panicf("ERROR: %s", err)
	}

	ls := make([]language.Language, len(command.Languages))
	for i, languageID := range command.Languages {
		ls[i] = languagesSelected[languageID]
	}
	ms := make([]model.Model, len(command.Models))
	for i, modelID := range command.Models {
		ms[i] = modelsSelected[modelID]
	}
	assessments, totalScore := evaluate.Evaluate(&evaluate.Context{
		Log: log,

		Languages: ls,

		Models:           ms,
		ProviderForModel: providerForModel,
		QueryAttempts:    command.QueryAttempts,

		RepositoryPaths: command.Repositories,
		ResultPath:      command.ResultPath,
		TestdataPath:    command.TestdataPath,

		Runs:           command.Runs,
		RunsSequential: command.RunsSequential,
	})

	assessmentsPerModel := assessments.CollapseByModel()
	if err := (report.Markdown{
		DateTime: evaluationTimestamp,
		Version:  evaluate.Version,

		CSVPath:       "./evaluation.csv",
		LogPath:       "./evaluation.log",
		ModelLogsPath: ".",
		SVGPath:       "./categories.svg",

		AssessmentPerModel: assessmentsPerModel,
		TotalScore:         totalScore,
	}).WriteToFile(filepath.Join(command.ResultPath, "README.md")); err != nil {
		return err
	}

	_ = assessmentsPerModel.WalkByScore(func(model model.Model, assessment metrics.Assessments, score uint64) (err error) {
		log.Printf("Evaluation score for %q (%q): %s", model.ID(), assessment.Category(totalScore).ID, assessment)

		return nil
	})

	if err := writeCSVs(command.ResultPath, assessments); err != nil {
		log.Panicf("ERROR: %s", err)
	}

	return nil
}

// WriteCSVs writes the various CSV reports to disk.
func writeCSVs(resultPath string, assessments report.AssessmentPerModelPerLanguagePerRepository) (err error) {
	// Write the "evaluation.csv" containing all data.
	csv, err := report.GenerateCSV(assessments)
	if err != nil {
		return pkgerrors.Wrap(err, "could not create evaluation.csv summary")
	}
	if err := os.WriteFile(filepath.Join(resultPath, "evaluation.csv"), []byte(csv), 0644); err != nil {
		return pkgerrors.Wrap(err, "could not write evaluation.csv summary")
	}

	// Write the "models-summed.csv" containing the summary per model.
	byModel := assessments.CollapseByModel()
	csvByModel, err := report.GenerateCSV(byModel)
	if err != nil {
		return pkgerrors.Wrap(err, "could not create models-summed.csv summary")
	}
	if err := os.WriteFile(filepath.Join(resultPath, "models-summed.csv"), []byte(csvByModel), 0644); err != nil {
		return pkgerrors.Wrap(err, "could not write models-summed.csv summary")
	}

	// Write the individual "language-summed.csv" containing the summary per model per language.
	byLanguage := assessments.CollapseByLanguage()
	for language, modelsByLanguage := range byLanguage {
		csvByLanguage, err := report.GenerateCSV(modelsByLanguage)
		if err != nil {
			return pkgerrors.Wrap(err, "could not create "+language.ID()+"-summed.csv summary")
		}
		if err := os.WriteFile(filepath.Join(resultPath, language.ID()+"-summed.csv"), []byte(csvByLanguage), 0644); err != nil {
			return pkgerrors.Wrap(err, "could not write "+language.ID()+"-summed.csv summary")
		}
	}

	return nil
}
