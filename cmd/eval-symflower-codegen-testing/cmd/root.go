package cmd

import (
	"log"

	"github.com/spf13/cobra"
)

// commandRoot holds the root command.
var commandRoot = &cobra.Command{
	Use:   "eval-symflower-codegen-testing",
	Short: "Command to manage, update and actually execute the `eval-symflower-codegen-testing` evaluation benchmark.",
	Run: func(command *cobra.Command, arguments []string) {
		if err := command.Help(); err != nil {
			log.Fatalf("%+v", err)
		}
	},
}

// Execute executes the root command.
func Execute() {
	if err := commandRoot.Execute(); err != nil {
		log.Fatalf("%+v", err)
	}
}
