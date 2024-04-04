package cmd

import (
	"log"
	"os"

	"github.com/jessevdk/go-flags"
)

// Command holds the root command.
type Command struct {
	Evaluate `command:"evaluate" description:"Run an evaluation, by default with all defined models, repositories and tasks."`
}

// Execute executes the root command.
func Execute() {
	var parser = flags.NewNamedParser("eval-dev-quality", flags.Default)
	parser.LongDescription = "Command to manage, update and actually execute the `eval-dev-quality` evaluation benchmark."
	if _, err := parser.AddGroup("Common command options", "", &Command{}); err != nil {
		log.Fatalf("Could not add arguments group: %+v", err)
	}

	// Print the help, when there is no active command.
	parser.SubcommandsOptional = true

	if _, err := parser.Parse(); err != nil {
		if e, ok := err.(*flags.Error); ok && e.Type == flags.ErrHelp {
			return
		}

		log.Fatalf("Could not parse arguments: %+v", err)
	}
	if parser.Active == nil {
		parser.WriteHelp(os.Stdout)
	}
}
