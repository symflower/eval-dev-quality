package cmd

import (
	"github.com/jessevdk/go-flags"
	"github.com/symflower/eval-dev-quality/log"
)

// Command holds the root command.
type Command struct {
	Evaluate     `command:"evaluate" description:"Run an evaluation, by default with all defined models, repositories and tasks."`
	InstallTools `command:"install-tools" description:"Checks and installs all tools required for the evaluation benchmark."`
}

// Execute executes the root command.
func Execute(logger *log.Logger, arguments []string) {
	var parser = flags.NewNamedParser("eval-dev-quality", flags.Default)
	parser.LongDescription = "Command to manage, update and actually execute the `eval-dev-quality` evaluation benchmark."
	if _, err := parser.AddGroup("Common command options", "", &Command{}); err != nil {
		logger.Panicf("Could not add arguments group: %+v", err)
	}

	// Print the help, when there is no active command.
	parser.SubcommandsOptional = true

	parser.CommandHandler = func(command flags.Commander, args []string) error {
		if command == nil {
			return nil
		}

		if c, ok := command.(SetLogger); ok {
			c.SetLogger(logger)
		}

		return command.Execute(args)
	}

	if _, err := parser.ParseArgs(arguments); err != nil {
		if e, ok := err.(*flags.Error); ok && e.Type == flags.ErrHelp {
			return
		}

		logger.Panicf("Could not parse arguments: %+v", err)
	}
	if parser.Active == nil {
		parser.WriteHelp(logger.Writer())
	}
}

// SetLogger defines a command that allows to set its logger.
type SetLogger interface {
	// SetLogger sets the logger of the command.
	SetLogger(logger *log.Logger)
}
