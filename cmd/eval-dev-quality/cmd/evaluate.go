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
	"github.com/symflower/eval-dev-quality/provider"
	_ "github.com/symflower/eval-dev-quality/provider/ollama"     // Register provider.
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
	// Repositories determines which repository should be used for the evaluation, or empty if all repositories should be used.
	Repositories []string `long:"repository" description:"Evaluate with this repository. By default all repositories are used."`
	// ResultPath holds the directory path where results should be written to.
	ResultPath string `long:"result-path" description:"Directory path where results should be written to. The placeholder \"%datetime%\" can be used for the current date and time." default:"evaluation-%datetime%"`
	// Runs holds the number of runs to perform.
	Runs uint `long:"runs" description:"Number of runs to perform." default:"1"`
	// TestdataPath determines the testdata path where all repositories reside grouped by languages.
	TestdataPath string `long:"testdata" description:"Path to the testdata directory where all repositories reside grouped by languages." default:"testdata/"`

	// ProviderTokens holds all API tokens for the providers.
	ProviderTokens map[string]string `long:"tokens" description:"API tokens for model providers (of the form '$provider:$token,...')." env:"PROVIDER_TOKEN"`

	// logger holds the logger of the command.
	logger *log.Logger
}

var _ SetLogger = (*Evaluate)(nil)

// SetLogger sets the logger of the command.
func (command *Evaluate) SetLogger(logger *log.Logger) {
	command.logger = logger
}

// repositoryPlainName holds the name of the plain repository.
const repositoryPlainName = "plain"

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

		if command.Runs == 0 {
			log.Panicf("number of configured runs is 0")
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
			commandRepositories[filepath.Join(languageID, repositoryPlainName)] = true
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

	// Check that models and languages can be evaluated by executing the "plain" repositories.
	log.Printf("Checking that models and languages can be used for evaluation")
	// Ensure we report metrics for every model even if they are excluded.
	assessments := report.NewAssessmentPerModelPerLanguagePerRepository(maps.Values(modelsSelected), maps.Values(languagesSelected), command.Repositories)
	problemsPerModel := map[string][]error{}
	{
		for r := uint(0); r < command.Runs; r++ {
			if command.Runs > 1 {
				log.Printf("Run %d/%d", r+1, command.Runs)
			}

			for _, languageID := range command.Languages {
				for _, modelID := range command.Models {
					model := modelsSelected[modelID]
					language := languagesSelected[languageID]

					repositoryPath := filepath.Join(languageID, repositoryPlainName)

					assessment, ps, err := evaluate.Repository(command.logger, command.ResultPath, model, language, command.TestdataPath, repositoryPath)
					assessments[model][language][repositoryPath].Add(assessment)
					if err != nil {
						ps = append(ps, err)
					}
					if len(ps) > 0 {
						log.Printf("Excluding model %q since it was not able to solve the %q repository for language %q: %+v", modelID, repositoryPath, languageID, ps)
						problemsPerModel[modelID] = append(problemsPerModel[modelID], ps...)
					}
				}
			}
		}
	}

	// Evaluating models and languages.
	log.Printf("Evaluating models and languages")
	for r := uint(0); r < command.Runs; r++ {
		if command.Runs > 1 {
			log.Printf("Run %d/%d", r+1, command.Runs)
		}

		for _, languageID := range command.Languages {
			languagePath := filepath.Join(command.TestdataPath, languageID)
			repositories, err := os.ReadDir(languagePath)
			if err != nil {
				log.Panicf("ERROR: language path %q cannot be accessed: %s", languagePath, err)
			}

			for _, repository := range repositories {
				repositoryPath := filepath.Join(languageID, repository.Name())

				if !repository.IsDir() || (len(commandRepositories) > 0 && !commandRepositories[repositoryPath]) {
					continue
				}

				// Do not include "plain" repositories in this step of the evaluation, because they have been checked with the common check before.
				if repository.Name() == repositoryPlainName {
					continue
				}

				for _, modelID := range command.Models {
					if len(problemsPerModel[modelID]) > 0 {
						continue
					}

					model := modelsSelected[modelID]
					language := languagesSelected[languageID]

					assessment, ps, err := evaluate.Repository(command.logger, command.ResultPath, model, language, command.TestdataPath, repositoryPath)
					assessments[model][language][repositoryPath].Add(assessment)
					problemsPerModel[modelID] = append(problemsPerModel[modelID], ps...)
					if err != nil {
						log.Printf("ERROR: Model %q encountered a hard error for language %q, repository %q: %+v", modelID, languageID, repositoryPath, err)
					}
				}
			}
		}
	}

	totalScore := uint(0)
	// Set the total score to the number of evaluated languages if we are just checking the "plain" repositories since there is only one task to solve per language.
	isOnlyPlainRepositories := true
	for repository := range commandRepositories {
		if filepath.Base(repository) != repositoryPlainName {
			isOnlyPlainRepositories = false

			break
		}
	}
	if isOnlyPlainRepositories {
		totalScore = uint(len(languagesSelected)) * command.Runs
	}

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

	_ = assessmentsPerModel.WalkByScore(func(model model.Model, assessment metrics.Assessments, score uint) (err error) {
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
