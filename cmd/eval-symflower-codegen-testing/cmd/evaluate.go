package cmd

import (
	"log"

	"github.com/spf13/cobra"

	"github.com/symflower/eval-symflower-codegen-testing/evaluate"
	"github.com/symflower/eval-symflower-codegen-testing/language"
	"github.com/symflower/eval-symflower-codegen-testing/model"
)

var commandEvalute = &cobra.Command{
	Use:   "evaluate",
	Short: "Run an evaluation, by default with all defined models and benchmarks.",
	Run: func(command *cobra.Command, arguments []string) {
		model := &model.ModelSymflower{}
		language := &language.LanguageGolang{}

		if err := evaluate.EvaluateRepository("testdata/golang/plain", model, language); err != nil {
			log.Fatalf("%+v", err)
		}
	},
}

func init() {
	commandRoot.AddCommand(commandEvalute)
}
