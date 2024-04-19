package cmd

import (
	"github.com/symflower/eval-dev-quality/log"
	"github.com/symflower/eval-dev-quality/tools"
)

// InstallTools holds the "install-tools" command.
type InstallTools struct {
	// InstallToolsPath determines where tools for the evaluation are installed.
	InstallToolsPath string `long:"install-tools-path" description:"Install tools for the evaluation into this path."`
}

// Execute executes the command.
func (command *InstallTools) Execute(args []string) (err error) {
	log := log.STDOUT()

	if command.InstallToolsPath == "" {
		command.InstallToolsPath, err = tools.InstallPathDefault()
		if err != nil {
			log.Fatalf("ERROR: %s", err)
		}
	}

	if err := tools.Install(log, command.InstallToolsPath); err != nil {
		log.Fatalf("ERROR: %s", err)
	}

	return nil
}
