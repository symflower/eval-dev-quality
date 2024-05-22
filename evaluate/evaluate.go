package evaluate

import (
	"log"
	"os"
	"path/filepath"

	"github.com/symflower/eval-dev-quality/evaluate/report"
	evallanguage "github.com/symflower/eval-dev-quality/language"
	evalmodel "github.com/symflower/eval-dev-quality/model"
)

// Context holds an evaluation context.
type Context struct {
	// Log holds the logger of the context.
	Log *log.Logger

	// Languages determines which language should be used for the evaluation, or empty if all languages should be used.
	Languages []evallanguage.Language

	// Models determines which models should be used for the evaluation, or empty if all models should be used.
	Models []evalmodel.Model
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
func Evaluate(ctx *Context) (assessments report.AssessmentPerModelPerLanguagePerRepository, totalScore uint64) {
	// Check that models and languages can be evaluated by executing the "plain" repositories.
	modelSucceededBasicChecksOfLanguage := map[evalmodel.Model]map[evallanguage.Language]bool{}
	ctx.Log.Printf("Checking that models and languages can be used for evaluation")
	// Ensure we report metrics for every model even if they are excluded.
	assessments = report.NewAssessmentPerModelPerLanguagePerRepository(ctx.Models, ctx.Languages, ctx.RepositoryPaths)
	problemsPerModel := map[string][]error{}

	{
		// Create temporary repositories for each language so the repository is copied only once per language.
		temporaryRepositories := map[string]string{}
		for _, language := range ctx.Languages {
			repositoryPath := filepath.Join(language.ID(), RepositoryPlainName)
			temporaryRepositoryPath, cleanup, err := TemporaryRepository(ctx.Log, filepath.Join(ctx.TestdataPath, repositoryPath))
			if err != nil {
				ctx.Log.Panicf("ERROR: unable to create temporary repository path: %s", err)
			}

			defer cleanup()

			temporaryRepositories[repositoryPath] = temporaryRepositoryPath
		}
		for rl := uint(0); rl < ctx.runsAtLanguageLevel(); rl++ {
			if ctx.Runs > 1 && !ctx.RunsSequential {
				ctx.Log.Printf("Run %d/%d", rl+1, ctx.Runs)
			}

			for _, language := range ctx.Languages {
				languageID := language.ID()
				repositoryPath := filepath.Join(language.ID(), RepositoryPlainName)
				temporaryRepositoryPath := temporaryRepositories[repositoryPath]

				for _, model := range ctx.Models {
					modelID := model.ID()

					if modelSucceededBasicChecksOfLanguage[model] == nil {
						modelSucceededBasicChecksOfLanguage[model] = map[evallanguage.Language]bool{}
					}

					if r, ok := model.(evalmodel.SetQueryAttempts); ok {
						r.SetQueryAttempts(ctx.QueryAttempts)
					}

					for rm := uint(0); rm < ctx.runsAtModelLevel(); rm++ {
						if err := ResetTemporaryRepository(ctx.Log, temporaryRepositoryPath); err != nil {
							ctx.Log.Panicf("ERROR: unable to reset temporary repository path: %s", err)
						}

						if ctx.Runs > 1 && ctx.RunsSequential {
							ctx.Log.Printf("Run %d/%d for model %q", rm+1, ctx.Runs, modelID)
						}

						assessment, ps, err := Repository(ctx.Log, ctx.ResultPath, model, language, temporaryRepositoryPath, repositoryPath)
						assessments[model][language][repositoryPath].Add(assessment)
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
	temporaryRepositories := map[string]string{}
	for _, language := range ctx.Languages {
		languagePath := filepath.Join(ctx.TestdataPath, language.ID())
		repositories, err := os.ReadDir(languagePath)
		if err != nil {
			ctx.Log.Panicf("ERROR: language path %q cannot be accessed: %s", languagePath, err)
		}
		for _, repository := range repositories {
			repositoryPath := filepath.Join(language.ID(), repository.Name())
			temporaryRepositoryPath, cleanup, err := TemporaryRepository(ctx.Log, filepath.Join(ctx.TestdataPath, repositoryPath))
			if err != nil {
				ctx.Log.Panicf("ERROR: unable to create temporary repository path: %s", err)
			}

			defer cleanup()

			temporaryRepositories[repositoryPath] = temporaryRepositoryPath
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
				temporaryRepositoryPath := temporaryRepositories[repositoryPath]

				if !repository.IsDir() || (len(ctx.RepositoryPaths) > 0 && !repositoriesLookup[repositoryPath]) {
					continue
				}

				// Do not include "plain" repositories in this step of the evaluation, because they have been checked with the common check before.
				if repository.Name() == RepositoryPlainName {
					continue
				}

				for _, model := range ctx.Models {
					modelID := model.ID()

					if !modelSucceededBasicChecksOfLanguage[model][language] {
						log.Printf("Excluding model %q for language %q cause it did not succeed basic checks", model.ID(), language.ID())

						continue
					}

					for rm := uint(0); rm < ctx.runsAtModelLevel(); rm++ {
						if ctx.Runs > 1 && ctx.RunsSequential {
							ctx.Log.Printf("Run %d/%d for model %q", rm+1, ctx.Runs, modelID)
						}

						if err := ResetTemporaryRepository(ctx.Log, temporaryRepositoryPath); err != nil {
							ctx.Log.Panicf("ERROR: unable to reset temporary repository path: %s", err)
						}

						assessment, ps, err := Repository(ctx.Log, ctx.ResultPath, model, language, temporaryRepositoryPath, repositoryPath)
						assessments[model][language][repositoryPath].Add(assessment)
						problemsPerModel[modelID] = append(problemsPerModel[modelID], ps...)
						if err != nil {
							ctx.Log.Printf("ERROR: Model %q encountered a hard error for language %q, repository %q: %+v", modelID, languageID, repositoryPath, err)
						}
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
