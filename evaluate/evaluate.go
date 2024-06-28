package evaluate

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/symflower/eval-dev-quality/evaluate/report"
	evaluatetask "github.com/symflower/eval-dev-quality/evaluate/task"
	"github.com/symflower/eval-dev-quality/language"
	evallanguage "github.com/symflower/eval-dev-quality/language"
	"github.com/symflower/eval-dev-quality/log"
	evalmodel "github.com/symflower/eval-dev-quality/model"
	"github.com/symflower/eval-dev-quality/provider"
	"github.com/symflower/eval-dev-quality/task"
)

// Context holds an evaluation context.
type Context struct {
	// Log holds the logger of the context.
	Log *log.Logger

	// Languages determines which language should be used for the evaluation, or empty if all languages should be used.
	Languages []evallanguage.Language

	// Models determines which models should be used for the evaluation, or empty if all models should be used.
	Models []evalmodel.Model
	// ProviderForModel holds the models and their associated provider.
	ProviderForModel map[evalmodel.Model]provider.Provider
	// QueryAttempts holds the number of query attempts to perform when a model request errors in the process of solving a task.
	QueryAttempts uint

	// RepositoryPaths determines which relative repository paths should be used for the evaluation, or empty if all repositories should be used.
	RepositoryPaths []string
	// ResultPath holds the directory path where results should be written to.
	ResultPath string
	// TestdataPath determines the testdata path where all repositories reside grouped by languages.
	TestdataPath string

	// Runs holds the number of runs to perform.
	Runs uint
	// RunsSequential indicates that interleaved runs are disabled and runs are performed sequentially.
	RunsSequential bool
	// NoDisqualification indicates that models are not to be disqualified if they fail to solve basic language tasks.
	NoDisqualification bool
}

// runsAtLanguageLevel returns how many runs to perform on language level.
func (ctx *Context) runsAtLanguageLevel() uint {
	if ctx.RunsSequential {
		return 1
	}

	return ctx.Runs
}

// runsAtModelLevel returns how many runs to perform on model level.
func (ctx *Context) runsAtModelLevel() uint {
	if ctx.RunsSequential {
		return ctx.Runs
	}

	return 1
}

// RepositoryPlainName holds the name of the plain repository.
const RepositoryPlainName = "plain"

// Evaluate runs an evaluation on the given context and returns its results.
func Evaluate(ctx *Context) (assessments *report.AssessmentStore, totalScore uint64) {
	// Check that models and languages can be evaluated by executing the "plain" repositories.
	modelSucceededBasicChecksOfLanguage := map[evalmodel.Model]map[evallanguage.Language]bool{}
	ctx.Log.Printf("Checking that models and languages can be used for evaluation")
	// Ensure we report metrics for every model even if they are excluded.
	assessments = report.NewAssessmentStore()
	problemsPerModel := map[string][]error{}

	{
		// Create temporary repositories for each language so the repository is copied only once per language.
		temporaryRepositories := map[string]task.Repository{}
		for _, language := range ctx.Languages {
			repositoryPath := filepath.Join(language.ID(), RepositoryPlainName)
			temporaryRepository, cleanup, err := evaluatetask.TemporaryRepository(ctx.Log, ctx.TestdataPath, repositoryPath)
			if err != nil {
				ctx.Log.Panicf("ERROR: unable to create temporary repository path: %+v", err)
			}

			defer cleanup()

			temporaryRepositories[repositoryPath] = temporaryRepository
		}
		for rl := uint(0); rl < ctx.runsAtLanguageLevel(); rl++ {
			if ctx.Runs > 1 && !ctx.RunsSequential {
				ctx.Log.Printf("Run %d/%d", rl+1, ctx.Runs)
			}

			for _, language := range ctx.Languages {
				languageID := language.ID()
				repositoryPath := filepath.Join(language.ID(), RepositoryPlainName)
				temporaryRepository := temporaryRepositories[repositoryPath]

				for _, model := range ctx.Models {
					modelID := model.ID()

					if modelSucceededBasicChecksOfLanguage[model] == nil {
						modelSucceededBasicChecksOfLanguage[model] = map[evallanguage.Language]bool{}
					}

					if r, ok := model.(evalmodel.SetQueryAttempts); ok {
						r.SetQueryAttempts(ctx.QueryAttempts)
					}

					for _, taskIdentifier := range temporaryRepository.SupportedTasks() {
						task, err := evaluatetask.TaskForIdentifier(taskIdentifier, ctx.Log, ctx.ResultPath, model, language)
						if err != nil {
							ctx.Log.Fatal(err)
						}
						withLoadedModel(ctx.Log, model, ctx.ProviderForModel[model], func() {
							for rm := uint(0); rm < ctx.runsAtModelLevel(); rm++ {
								if ctx.Runs > 1 && ctx.RunsSequential {
									ctx.Log.Printf("Run %d/%d for model %q", rm+1, ctx.Runs, modelID)
								}

								if err := temporaryRepository.Reset(ctx.Log); err != nil {
									ctx.Log.Panicf("ERROR: unable to reset temporary repository path: %s", err)
								}

								assessment, ps, err := task.Run(temporaryRepository)
								assessments.AddAssessmentPerTask(model, language, repositoryPath, assessment)
								if err != nil {
									ps = append(ps, err)
								}
								if len(ps) > 0 {
									ctx.Log.Printf("Model %q was not able to solve the %q repository for language %q: %+v", modelID, repositoryPath, languageID, ps)
									problemsPerModel[modelID] = append(problemsPerModel[modelID], ps...)
								} else {
									modelSucceededBasicChecksOfLanguage[model][language] = true
								}
							}
						})
					}
				}
			}
		}
	}

	repositoriesLookup := make(map[string]bool, len(ctx.RepositoryPaths))
	for _, repositoryPath := range ctx.RepositoryPaths {
		repositoriesLookup[repositoryPath] = true
	}

	// Evaluating models and languages.
	ctx.Log.Printf("Evaluating models and languages")
	// Create temporary repositories for each language so the repository is copied only once per language.
	temporaryRepositories := map[string]*evaluatetask.Repository{}
	for _, l := range ctx.Languages {
		relativeRepositoryPaths, err := language.RepositoriesForLanguage(l, ctx.TestdataPath)
		if err != nil {
			ctx.Log.Panicf("ERROR: %s", err)
		}
		for _, repositoryPath := range relativeRepositoryPaths {

			// Do not include "plain" repositories in this step of the evaluation, because they have been checked with the common check before.
			if !repositoriesLookup[repositoryPath] || strings.HasSuffix(repositoryPath, RepositoryPlainName) {
				continue
			}

			temporaryRepository, cleanup, err := evaluatetask.TemporaryRepository(ctx.Log, ctx.TestdataPath, repositoryPath)
			if err != nil {
				ctx.Log.Panicf("ERROR: unable to create temporary repository path: %s", err)
			}

			defer cleanup()

			temporaryRepositories[repositoryPath] = temporaryRepository
		}
	}
	for rl := uint(0); rl < ctx.runsAtLanguageLevel(); rl++ {
		if ctx.Runs > 1 && !ctx.RunsSequential {
			ctx.Log.Printf("Run %d/%d", rl+1, ctx.Runs)
		}

		for _, language := range ctx.Languages {
			languageID := language.ID()

			languagePath := filepath.Join(ctx.TestdataPath, languageID)
			repositories, err := os.ReadDir(languagePath)
			if err != nil {
				ctx.Log.Panicf("ERROR: language path %q cannot be accessed: %s", languagePath, err)
			}

			for _, repository := range repositories {
				repositoryPath := filepath.Join(languageID, repository.Name())
				temporaryRepository := temporaryRepositories[repositoryPath]

				if !repository.IsDir() || (len(ctx.RepositoryPaths) > 0 && !repositoriesLookup[repositoryPath]) {
					continue
				}

				// Do not include "plain" repositories in this step of the evaluation, because they have been checked with the common check before.
				if repository.Name() == RepositoryPlainName {
					continue
				}

				for _, model := range ctx.Models {
					modelID := model.ID()

					if !ctx.NoDisqualification && !modelSucceededBasicChecksOfLanguage[model][language] {
						ctx.Log.Printf("Excluding model %q for language %q cause it did not succeed basic checks", model.ID(), language.ID())

						continue
					}
					for _, taskIdentifier := range temporaryRepository.Tasks {
						task, err := evaluatetask.TaskForIdentifier(taskIdentifier, ctx.Log, ctx.ResultPath, model, language)
						if err != nil {
							ctx.Log.Fatal(err)
						}
						withLoadedModel(ctx.Log, model, ctx.ProviderForModel[model], func() {
							for rm := uint(0); rm < ctx.runsAtModelLevel(); rm++ {
								if ctx.Runs > 1 && ctx.RunsSequential {
									ctx.Log.Printf("Run %d/%d for model %q", rm+1, ctx.Runs, modelID)
								}

								if err := temporaryRepository.Reset(ctx.Log); err != nil {
									ctx.Log.Panicf("ERROR: unable to reset temporary repository path: %s", err)
								}

								assessment, ps, err := task.Run(temporaryRepository)
								assessments.AddAssessmentPerTask(model, language, repositoryPath, assessment)
								problemsPerModel[modelID] = append(problemsPerModel[modelID], ps...)
								if err != nil {
									ctx.Log.Printf("ERROR: Model %q encountered a hard error for language %q, repository %q: %+v", modelID, languageID, repositoryPath, err)
								}
							}
						})
					}
				}
			}
		}
	}

	// Set the total score to the number of evaluated languages if we are just checking the "plain" repositories since there is only one task to solve per language.
	isOnlyPlainRepositories := true
	for _, repositoryPath := range ctx.RepositoryPaths {
		if filepath.Base(repositoryPath) != RepositoryPlainName {
			isOnlyPlainRepositories = false

			break
		}
	}
	if isOnlyPlainRepositories {
		totalScore = uint64(len(ctx.Languages)) * uint64(ctx.Runs)
	}

	return assessments, totalScore
}

// withLoadedModel loads the model for the duration of the given task if supported by the model's provider.
func withLoadedModel(log *log.Logger, model evalmodel.Model, modelProvider provider.Provider, task func()) {
	if loader, ok := modelProvider.(provider.Loader); ok {
		log.Printf("preloading model %q", model.ID())
		if err := loader.Load(model.ID()); err != nil {
			log.Panicf("ERROR: could not load model %q with provider %q", model.ID(), modelProvider.ID())
		}
		defer func() {
			log.Printf("unloading model %q", model.ID())
			if err := loader.Unload(model.ID()); err != nil {
				log.Panicf("ERROR: could not unload model %q with provider %q", model.ID(), modelProvider.ID())
			}
		}()
	}

	task()
}
