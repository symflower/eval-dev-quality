package util

import (
	"bytes"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
)

// Command defines a command that should be executed.
type Command struct {
	// Command holds the command with its optional arguments.
	Command []string

	// Directory defines the directory the execution should run in, without changing the working directory of the caller.
	Directory string
}

// CommandWithResult executes a command, and prints and returns STDERR/STDOUT.
func CommandWithResult(command *Command) (stdout string, stderr string, err error) {
	log.Printf("$ %s", strings.Join(command.Command, " "))

	var stdoutWriter bytes.Buffer
	var stderrWriter bytes.Buffer
	c := exec.Command(command.Command[0], command.Command[1:]...)
	if command.Directory != "" {
		c.Dir = command.Directory
	}
	c.Stdout = io.MultiWriter(os.Stdout, &stdoutWriter)
	c.Stderr = io.MultiWriter(os.Stderr, &stderrWriter)

	if err := c.Run(); err != nil {
		return stdoutWriter.String(), stderrWriter.String(), err
	}

	return stdoutWriter.String(), stderrWriter.String(), nil
}
