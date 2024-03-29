package cmd

import (
	"log"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"

	"github.com/symflower/eval-symflower-codegen-testing/evaluate"
	"github.com/symflower/eval-symflower-codegen-testing/language"
	"github.com/symflower/eval-symflower-codegen-testing/model"
)

var commandEvalute = &cobra.Command{
	Use:   "evaluate",
	Short: "Run an evaluation, by default with all defined models, repositories and tasks.",
	Run: func(command *cobra.Command, arguments []string) {
		// Gather languages.
		languageIDs := maps.Keys(language.Languages)
		sort.Strings(languageIDs)

		// Gather models.
		modelIDs := maps.Keys(model.Models)
		sort.Strings(modelIDs)

		// Check that models and languages can be evaluated by executing the "plain" repositories.
		log.Printf("Checking that models and languages can used for evaluation")
		for _, languageID := range languageIDs {
			for _, modelID := range modelIDs {
				model := model.Models[modelID]
				language := language.Languages[languageID]

				if err := evaluate.EvaluateRepository(model, language, filepath.Join("testdata", language.ID(), "plain")); err != nil {
					log.Fatalf("%+v", err)
				}
			}
		}
	},
}

func init() {
	commandRoot.AddCommand(commandEvalute)
}
