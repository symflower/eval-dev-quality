package evaluate

import (
	"os"
	"path/filepath"

	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	"github.com/symflower/eval-dev-quality/evaluate/report"
	evaluatetask "github.com/symflower/eval-dev-quality/evaluate/task"
	evallanguage "github.com/symflower/eval-dev-quality/language"
	"github.com/symflower/eval-dev-quality/log"
	evalmodel "github.com/symflower/eval-dev-quality/model"
	"github.com/symflower/eval-dev-quality/provider"
	evaltask "github.com/symflower/eval-dev-quality/task"
	"github.com/symflower/eval-dev-quality/util"
	"golang.org/x/exp/maps"
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
	// APIReqestAttempts holds the number of allowed API requests per LLM query.
	APIReqestAttempts uint
	// APIRequestTimeout holds the timeout for API requests in seconds.
	APIRequestTimeout uint

	// RepositoryPaths determines which relative repository paths should be used for the evaluation, or empty if all repositories should be used.
	RepositoryPaths []string
	// ResultPath holds the directory path where results should be written to.
	ResultPath string
	// TestdataPath determines the testdata path where all repositories reside grouped by languages.
	TestdataPath string

	// RunIDStartsAt holds the run ID starting index created when running a evaluation multiple times.
	RunIDStartsAt uint
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
func Evaluate(ctx *Context) {
	// Check that models and languages can be evaluated by executing the "plain" repositories.
	modelSucceededBasicChecksOfLanguage := map[evalmodel.Model]map[evallanguage.Language]bool{}
	ctx.Log.Info("checking that models and languages can be used for evaluation")

	problemsPerModel := map[string][]error{}
	// Write the evaluation CSV header so it's only written once.
	evaluationCSVFile, err := os.OpenFile(filepath.Join(ctx.ResultPath, "evaluation.csv"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		ctx.Log.Panicf("ERROR: unable to create evaluation CSV file: %+v", err)
	}
	defer func() {
		if err := evaluationCSVFile.Close(); err != nil {
			ctx.Log.Panicf("ERROR: cannot close CSV file: %s", err)
		}
	}()
	evaluationFile, err := report.NewEvaluationFile(evaluationCSVFile)
	if err != nil {
		ctx.Log.Panicf("ERROR: %+v", err)
	}

	{
		// Create temporary repositories for each language so the repository is copied only once per language.
		temporaryRepositories := map[string]evaltask.Repository{}
		for _, language := range ctx.Languages {
			repositoryPath := filepath.Join(language.ID(), RepositoryPlainName)
			temporaryRepository, cleanup, err := evaluatetask.TemporaryRepository(ctx.Log, ctx.TestdataPath, repositoryPath)
			if err != nil {
				ctx.Log.Panicf("ERROR: unable to create temporary repository path: %+v", err)
			} else if err = temporaryRepository.Validate(ctx.Log, language); err != nil {
				ctx.Log.Panicf("ERROR: malformed repository %q: %+v", temporaryRepository.Name(), err)
			}

			defer cleanup()

			temporaryRepositories[repositoryPath] = temporaryRepository
		}

		logger := ctx.Log
		for rl := uint(0); rl < ctx.runsAtLanguageLevel(); rl++ {
			if ctx.Runs > 1 && !ctx.RunsSequential {
				logger.Info("starting run", "count", rl+1, "total", ctx.Runs)
			}

			for _, language := range ctx.Languages {
				logger := logger.With(log.AttributeKeyLanguage, language.ID())

				languageID := language.ID()
				repositoryPath := filepath.Join(language.ID(), RepositoryPlainName)
				temporaryRepository := temporaryRepositories[repositoryPath]

				logger = logger.With(log.AttributeKeyRepository, temporaryRepository.Name())
				for _, model := range ctx.Models {
					modelID := model.ID()
					logger := logger.With(log.AttributeKeyModel, modelID)

					if modelSucceededBasicChecksOfLanguage[model] == nil {
						modelSucceededBasicChecksOfLanguage[model] = map[evallanguage.Language]bool{}
					}

					if r, ok := model.(evalmodel.ConfigureAPIRequestHandling); ok {
						r.SetAPIRequestAttempts(ctx.APIReqestAttempts)
						r.SetAPIRequestTimeout(ctx.APIRequestTimeout)
					}

					for _, taskIdentifier := range temporaryRepository.Configuration().Tasks {
						task, err := evaluatetask.ForIdentifier(taskIdentifier)
						if err != nil {
							logger.Fatalf(err)
						}

						logger := logger.With(log.AttributeKeyTask, taskIdentifier)
						withLoadedModel(logger, model, ctx.ProviderForModel[model], func() {
							for rm := uint(0); rm < ctx.runsAtModelLevel(); rm++ {
								var runCount uint
								if ctx.Runs > 1 && ctx.RunsSequential {
									logger.Info("starting run", "count", rm+1, "total", ctx.Runs, "model", modelID)
									runCount = rm + 1
								} else {
									runCount = rl + 1
								}
								runID := ctx.RunIDStartsAt + runCount - 1

								if err := temporaryRepository.Reset(logger); err != nil {
									logger.Panicf("ERROR: unable to reset temporary repository path: %s", err)
								}

								taskContext := evaltask.Context{
									Language:   language,
									Repository: temporaryRepository,
									Model:      model,

									ResultPath: ctx.ResultPath,

									Logger: logger,
								}
								assessment, ps, err := task.Run(taskContext)
								if err != nil {
									ps = append(ps, err)
								}
								if len(ps) > 0 {
									problemsPerModel[modelID] = append(problemsPerModel[modelID], ps...)
								}

								if succeededPlain(assessment) {
									modelSucceededBasicChecksOfLanguage[model][language] = true
								} else {
									logger.Info("model was not able to solve repository", "model", modelID, "repository", repositoryPath, "language", languageID, "problems", ps)
								}

								// Write the task assessment to the evaluation CSV file.
								if err := evaluationFile.WriteEvaluationRecord(model, language, temporaryRepository.Name(), runID, assessment); err != nil {
									logger.Panicf("ERROR: cannot write evaluation record: %s", err)
								}
							}
						})
					}
				}
			}
		}
	}

	repositoriesLookup := util.Set(ctx.RepositoryPaths)

	// Evaluating models and languages.
	ctx.Log.Info("evaluating models and languages")
	// Create temporary repositories for each language so the repository is copied only once per language.
	temporaryRepositories := map[string]*evaluatetask.Repository{}
	for _, l := range ctx.Languages {
		relativeRepositoryPaths, err := evallanguage.RepositoriesForLanguage(l, ctx.TestdataPath)
		if err != nil {
			ctx.Log.Panicf("ERROR: %s", err)
		}
		for _, repositoryPath := range relativeRepositoryPaths {
			// Do not include "plain" repositories in this step of the evaluation, because they have been checked with the common check before.
			if !repositoriesLookup[repositoryPath] || filepath.Base(repositoryPath) == RepositoryPlainName {
				continue
			}

			temporaryRepository, cleanup, err := evaluatetask.TemporaryRepository(ctx.Log, ctx.TestdataPath, repositoryPath)
			if err != nil {
				ctx.Log.Panicf("ERROR: unable to create temporary repository path: %s", err)
			} else if err = temporaryRepository.Validate(ctx.Log, l); err != nil {
				ctx.Log.Panicf("ERROR: malformed repository %q: %+v", temporaryRepository.Name(), err)
			}

			defer cleanup()

			temporaryRepositories[repositoryPath] = temporaryRepository
		}
	}
	logger := ctx.Log
	for rl := uint(0); rl < ctx.runsAtLanguageLevel(); rl++ {
		if ctx.Runs > 1 && !ctx.RunsSequential {
			logger.Info("starting run", "count", rl+1, "total", ctx.Runs)
		}

		for _, language := range ctx.Languages {
			languageID := language.ID()
			logger := logger.With(log.AttributeKeyLanguage, languageID)

			languagePath := filepath.Join(ctx.TestdataPath, languageID)
			repositories, err := os.ReadDir(languagePath)
			if err != nil {
				logger.Panicf("ERROR: language path %q cannot be accessed: %s", languagePath, err)
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

				logger = logger.With(log.AttributeKeyRepository, repositoryPath)
				for _, model := range ctx.Models {
					modelID := model.ID()
					logger := logger.With(log.AttributeKeyModel, modelID)

					if !ctx.NoDisqualification && !modelSucceededBasicChecksOfLanguage[model][language] {
						logger.Warn("excluding model cause it did not succeed basic checks", "model", model.ID(), "language", language.ID())

						continue
					}
					for _, taskIdentifier := range temporaryRepository.Tasks {
						task, err := evaluatetask.ForIdentifier(taskIdentifier)
						if err != nil {
							logger.Fatalf(err)
						}
						logger := logger.With(log.AttributeKeyTask, taskIdentifier)
						withLoadedModel(logger, model, ctx.ProviderForModel[model], func() {
							for rm := uint(0); rm < ctx.runsAtModelLevel(); rm++ {
								var runCount uint
								if ctx.Runs > 1 && ctx.RunsSequential {
									logger.Info("starting run", "count", rm+1, "total", ctx.Runs, "model", modelID)
									runCount = rm + 1
								} else {
									runCount = rl + 1
								}
								runID := ctx.RunIDStartsAt + runCount - 1

								if err := temporaryRepository.Reset(logger); err != nil {
									logger.Panicf("ERROR: unable to reset temporary repository path: %s", err)
								}

								taskContext := evaltask.Context{
									Language:   language,
									Repository: temporaryRepository,
									Model:      model,

									ResultPath: ctx.ResultPath,

									Logger: logger,
								}
								assessment, ps, err := task.Run(taskContext)
								problemsPerModel[modelID] = append(problemsPerModel[modelID], ps...)
								if err != nil {
									logger.Error("model encountered a hard error", "model", modelID, "language", languageID, "repository", repositoryPath, "error", err)
								}

								// Write the task assessment to the evaluation CSV file.
								if err := evaluationFile.WriteEvaluationRecord(model, language, temporaryRepository.Name(), runID, assessment); err != nil {
									logger.Panicf("ERROR: cannot write evaluation record: %s", err)
								}
							}
						})
					}
				}
			}
		}
	}
}

// withLoadedModel loads the model for the duration of the given task if supported by the model's provider.
func withLoadedModel(logger *log.Logger, model evalmodel.Model, modelProvider provider.Provider, task func()) {
	if loader, ok := modelProvider.(provider.Loader); ok {
		logger.Info("preloading model", "model", model.ID())
		if err := loader.Load(model.ID()); err != nil {
			logger.Panicf("ERROR: could not load model %q with provider %q", model.ID(), modelProvider.ID())
		}
		defer func() {
			logger.Info("unloading model", "model", model.ID())
			if err := loader.Unload(model.ID()); err != nil {
				logger.Panicf("ERROR: could not unload model %q with provider %q", model.ID(), modelProvider.ID())
			}
		}()
	}

	task()
}

// succeededPlain checks if the assessments attest that the "plain" repository was successfully solved.
func succeededPlain(assessmentPerCase map[string]map[evaltask.Identifier]metrics.Assessments) bool {
	if len(assessmentPerCase) != 1 { // The "plain" repository only has one case.
		return false
	}
	assessment := assessmentPerCase[maps.Keys(assessmentPerCase)[0]]

	if withoutTemplate, ok := assessment[evaluatetask.IdentifierWriteTests]; ok && withoutTemplate[metrics.AssessmentKeyFilesExecuted] > 0 {
		return true
	} else if withTemplate, ok := assessment[evaluatetask.IdentifierWriteTestsSymflowerTemplate]; ok && withTemplate[metrics.AssessmentKeyFilesExecuted] > 0 {
		return true
	}

	return false
}
