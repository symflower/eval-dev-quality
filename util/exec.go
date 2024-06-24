package util

import (
	"context"
	"io"
	"os/exec"
	"strings"
	"time"

	pkgerrors "github.com/pkg/errors"
	"github.com/zimmski/osutil"
	"github.com/zimmski/osutil/bytesutil"

	"github.com/symflower/eval-dev-quality/log"
)

// Command defines a command that should be executed.
type Command struct {
	// Command holds the command with its optional arguments.
	Command []string

	// Directory defines the directory the execution should run in, without changing the working directory of the caller.
	Directory string
	// Env overwrites the environment variables of the executed command.
	Env map[string]string
}

// CommandWithResult executes a command and returns its output, while printing the same output to the given logger.
func CommandWithResult(ctx context.Context, logger *log.Logger, command *Command) (output string, err error) {
	logger.Printf("$ %s", strings.Join(command.Command, " "))

	var writer bytesutil.SynchronizedBuffer
	c := exec.CommandContext(ctx, command.Command[0], command.Command[1:]...)
	if command.Directory != "" {
		c.Dir = command.Directory
	}
	if command.Env != nil {
		envs := osutil.EnvironMap()
		for k, v := range command.Env {
			envs[k] = v
		}
		for k, v := range envs {
			c.Env = append(c.Env, k+"="+v)
		}
	}
	c.Stdout = io.MultiWriter(logger.Writer(), &writer)
	c.Stderr = c.Stdout

	c.WaitDelay = 3 * time.Second // Some binaries do not like to be killed, e.g. "ollama", so we kill them after some time automatically.

	if err := c.Run(); err != nil {
		return writer.String(), pkgerrors.WithStack(pkgerrors.WithMessage(err, writer.String()))
	}

	return writer.String(), nil
}

// FilterArgs parses args and removes the ignored ones.
func FilterArgs(args []string, ignored []string) (filtered []string) {
	filterMap := map[string]bool{}
	for _, v := range ignored {
		filterMap[v] = true
	}

	// Resolve args with equals sign.
	var resolvedArgs []string
	for _, v := range args {
		if strings.HasPrefix(v, "--") && strings.Contains(v, "=") {
			resolvedArgs = append(resolvedArgs, strings.SplitN(v, "=", 2)...)
		} else {
			resolvedArgs = append(resolvedArgs, v)
		}
	}

	skip := false
	for _, v := range resolvedArgs {
		if skip && strings.HasPrefix(v, "--") {
			skip = false
		}
		if filterMap[v] {
			skip = true
		}

		if skip {
			continue
		}

		filtered = append(filtered, v)
	}

	return filtered
}
