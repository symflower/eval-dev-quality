package cmd

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"

	"github.com/jessevdk/go-flags"
	"github.com/symflower/eval-dev-quality/log"
)

// Command holds the root command.
type Command struct {
	Evaluate     `command:"evaluate" description:"Run an evaluation, by default with all defined models, repositories and tasks."`
	InstallTools `command:"install-tools" description:"Checks and installs all tools required for the evaluation benchmark."`
	Version      `command:"version" description:"Display the version information of the binary."`
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

	// Cache Symflower's license file if we receive its data and the license file path is not yet set. This is helpful in a container environment where most likely only a environment variable is set.
	licenseData := os.Getenv("SYMFLOWER_INTERNAL_LICENSE_FILE")
	if licenseData != "" && os.Getenv("SYMFLOWER_INTERNAL_LICENSE_FILE_PATH") == "" {
		homePath, err := os.UserHomeDir()
		if err != nil {
			panic(err)
		}
		licensePath := filepath.Join(homePath, ".symflower-license")
		logger.Info("write license to", "path", licensePath)
		if decoded, err := base64.StdEncoding.DecodeString(licenseData); err == nil {
			licenseData = string(decoded)
		}
		if err := os.WriteFile(licensePath, []byte(licenseData), 0600); err != nil {
			panic(err)
		}

		// Forward the path of the license file for future steps of the job.
		if err := os.Setenv("SYMFLOWER_INTERNAL_LICENSE_FILE_PATH", licensePath); err != nil {
			panic(err)
		}
	}

	parser.CommandHandler = func(command flags.Commander, args []string) (err error) {
		if command == nil {
			return nil
		}

		if c, ok := command.(SetLogger); ok {
			c.SetLogger(logger)
		}

		if c, ok := command.(SetArguments); ok {
			c.SetArguments(arguments)
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
		var sb strings.Builder
		parser.WriteHelp(&sb)
		logger.PrintfWithoutMeta(sb.String())
	}
}

// SetLogger defines a command that allows to set its logger.
type SetLogger interface {
	// SetLogger sets the logger of the command.
	SetLogger(logger *log.Logger)
}

// SetArguments defines a command that allows to set its arguments.
type SetArguments interface {
	// SetArguments sets the commands arguments.
	SetArguments(args []string)
}
