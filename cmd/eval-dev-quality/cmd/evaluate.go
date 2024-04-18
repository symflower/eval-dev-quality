package cmd

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/zimmski/osutil"
	"golang.org/x/exp/maps"

	"github.com/symflower/eval-dev-quality/evaluate"
	"github.com/symflower/eval-dev-quality/evaluate/metrics"
	"github.com/symflower/eval-dev-quality/language"
	"github.com/symflower/eval-dev-quality/log"
	"github.com/symflower/eval-dev-quality/model"
	"github.com/symflower/eval-dev-quality/provider"
	_ "github.com/symflower/eval-dev-quality/provider/openrouter"
	_ "github.com/symflower/eval-dev-quality/provider/symflower"
)

// Evaluate holds the "evaluation" command.
type Evaluate struct {
	// Languages determines which language should be used for the evaluation, or empty if all languages should be used.
	Languages []string `long:"language" description:"Evaluate with this language. By default all languages are used."`
	// Models determines which models should be used for the evaluation, or empty if all models should be used.
	Models []string `long:"model" description:"Evaluate with this model. By default all models are used."`
	// Repositories determines which repository should be used for the evaluation, or empty if all repositories should be used.
	Repositories []string `long:"repository" description:"Evaluate with this repository. By default all repositories are used."`
	// ResultPath holds the directory path where results should be written to.
	ResultPath string `long:"result-path" description:"Directory path where results should be written to. The placeholder \"%datetime%\" can be used for the current date and time." default:"evaluation-%datetime%"`
	// TestdataPath determines the testdata path where all repositories reside grouped by languages.
	TestdataPath string `long:"testdata" description:"Path to the testdata directory where all repositories reside grouped by languages." default:"testdata/"`

	// ProviderTokens holds all API tokens for the providers.
	ProviderTokens map[string]string `long:"tokens" description:"API tokens for model providers (of the form '$provider:$token,...')." env:"PROVIDER_TOKEN"`
}

func (command *Evaluate) Execute(args []string) (err error) {
	command.ResultPath = strings.ReplaceAll(command.ResultPath, "%datetime%", time.Now().Format("2006-01-02-15:04:05")) // REMARK Use a datetime format with a dash, so directories can be easily marked because they are only one group.

	log, logClose, err := log.FileAndSTDOUT(filepath.Join(command.ResultPath, "evaluation.log"))
	if err != nil {
		return err
	}
	defer logClose()

	// Gather languages.
	if len(command.Languages) == 0 {
		command.Languages = maps.Keys(language.Languages)
	} else {
		for _, languageID := range command.Languages {
			if _, ok := language.Languages[languageID]; !ok {
				ls := maps.Keys(language.Languages)
				sort.Strings(ls)

				log.Fatalf("ERROR: language %s does not exist. Valid languages are: %s", languageID, strings.Join(ls, ", "))
			}
		}
	}
	sort.Strings(command.Languages)

	commandRepositories := map[string]bool{}
	for _, r := range command.Repositories {
		commandRepositories[r] = true
	}

	// Gather models.
	models := map[string]model.Model{}
	{
		for _, p := range provider.Providers {
			ms, err := p.Models()
			if err != nil {
				log.Fatalf("ERROR: could not query models for provider %q: %s", p.ID(), err)
			}
			for _, m := range ms {
				if t, ok := p.(provider.InjectToken); ok {
					token, ok := command.ProviderTokens[p.ID()]
					if ok {
						t.SetToken(token)
					}
				}

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
					log.Fatalf("ERROR: model %s does not exist. Valid models are: %s", modelID, strings.Join(modelIDs, ", "))
				}
			}
		}
		sort.Strings(command.Models)
	}

	if err := osutil.DirExists(command.TestdataPath); err != nil {
		log.Fatalf("ERROR: testdata path %q cannot be accessed: %s", command.TestdataPath, err)
	}
	command.TestdataPath, err = filepath.Abs(command.TestdataPath)
	if err != nil {
		log.Fatalf("ERROR: could not resolve testdata path %q to an absolute path: %s", command.TestdataPath, err)
	}

	// Check that models and languages can be evaluated by executing the "plain" repositories.
	log.Printf("Checking that models and languages can be used for evaluation")
	assessmentsPerModel := map[string]metrics.Assessments{}
	problemsPerModel := map[string][]error{}
	{
		// Ensure we report metrics for every model even if they are excluded.
		for _, modelID := range command.Models {
			assessmentsPerModel[modelID] = metrics.NewAssessments()
		}

		for _, languageID := range command.Languages {
			for _, modelID := range command.Models {
				model := models[modelID]
				language := language.Languages[languageID]

				assessment, ps, err := evaluate.EvaluateRepository(command.ResultPath, model, language, command.TestdataPath, filepath.Join(language.ID(), "plain"))
				assessmentsPerModel[modelID].Add(assessment)
				if err != nil {
					ps = append(ps, err)
				}
				if len(ps) > 0 {
					log.Printf("Excluding model %q since it was not able to solve the \"plain\" repository for language %q: %+v", modelID, languageID, ps)
					problemsPerModel[modelID] = append(problemsPerModel[modelID], ps...)
				}
			}
		}
	}

	// Evaluating models and languages.
	log.Printf("Evaluating models and languages")
	for _, languageID := range command.Languages {
		languagePath := filepath.Join(command.TestdataPath, languageID)
		repositories, err := os.ReadDir(languagePath)
		if err != nil {
			log.Fatalf("ERROR: language path %q cannot be accessed: %s", languagePath, err)
		}

		for _, repository := range repositories {
			if !repository.IsDir() || (len(commandRepositories) > 0 && !commandRepositories[repository.Name()]) {
				continue
			}

			// Do not include "plain" repositories in this step of the evaluation, because they have been checked with the common check before.
			if filepath.Base(repository.Name()) == "plain" {
				continue
			}

			for _, modelID := range command.Models {
				if len(problemsPerModel[modelID]) > 0 {
					continue
				}

				model := models[modelID]
				language := language.Languages[languageID]

				assessment, ps, err := evaluate.EvaluateRepository(command.ResultPath, model, language, command.TestdataPath, filepath.Join(languageID, repository.Name()))
				assessmentsPerModel[model.ID()].Add(assessment)
				problemsPerModel[modelID] = append(problemsPerModel[modelID], ps...)
				if err != nil {
					log.Printf("ERROR: Model %q encountered a hard error for language %q, repository %q: %+v", modelID, languageID, repository.Name(), err)
				}
			}
		}
	}

	_ = metrics.WalkByScore(assessmentsPerModel, func(model string, assessment metrics.Assessments, score uint) error {
		log.Printf("Evaluation score for %q: %s", model, assessment)

		return nil
	})

	csv, err := metrics.FormatStringCSV(assessmentsPerModel)
	if err != nil {
		log.Fatalf("ERROR: could not create result summary: %s", err)
	}
	if err := os.WriteFile(filepath.Join(command.ResultPath, "evaluation.csv"), []byte(csv), 0644); err != nil {
		log.Fatalf("ERROR: could not write result summary: %s", err)
	}

	return nil
}
