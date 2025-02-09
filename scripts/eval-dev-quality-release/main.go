package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"regexp"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/jessevdk/go-flags"
	"github.com/symflower/eval-dev-quality/log"
	"github.com/symflower/eval-dev-quality/util"
	"github.com/zimmski/osutil"
)

// Command holds the root command.
type Command struct {
	// Positional holds positional arguments.
	Positional struct {
		// Version holds the version tag that should be released, e.g. "1.2.3".
		Version string `positional-arg-name:"version" description:"Tag that should be released, e.g. \"1.2.3\"." required:"true"`
	} `positional-args:"true"`
}

func main() {
	logger := log.STDOUT()
	options := &Command{}

	var parser = flags.NewNamedParser("eval-dev-quality-release", flags.Default)
	parser.LongDescription = "Command to do releases for DevQualityEval."
	if _, err := parser.AddGroup("Common command options", "", options); err != nil {
		logger.Panicf("could not add arguments group: %+v", err)
	}

	if _, err := parser.ParseArgs(os.Args[1:]); err != nil {
		if e, ok := err.(*flags.Error); ok && e.Type == flags.ErrHelp {
			return
		}

		logger.Panicf("could not parse arguments: %+v", err)
	}

	version, err := semver.NewVersion(options.Positional.Version)
	if err != nil {
		logger.Panicf("cannot parse version: %+v", err)
	}

	logger.Info("update repository to the latest commits")
	_, err = util.CommandWithResult(context.Background(), logger, &util.Command{
		Command: []string{
			"git-update-branch",
		},
	})
	if err != nil {
		logger.Warn(
			"cannot update repository to the latest commits:",
			"error", err,
		)
	}

	logger.Info("check that version is newer than current version")
	versionCurrent := "v0.6.2" // TODO
	versionCurrent = strings.TrimPrefix(versionCurrent, "v")
	vc, err := semver.NewVersion(versionCurrent)
	if err != nil {
		logger.Panicf("cannot parse current version: %+v", err)
	}
	if !version.GreaterThan(vc) {
		logger.Panicf("version %q must be greater than current version %q", version, vc)
	}

	versionTag := "v" + version.String()

	logger.Info("check if tag already exists")
	if _, err := util.CommandWithResult(context.Background(), logger, &util.Command{
		Command: []string{
			"git",
			"rev-parse",
			versionTag,
		},
	}); err == nil {
		logger.Info("version tag already exists")
	}

	logger.Info("change version in Go code")
	if err := osutil.FileChange("evaluate/version.go", func(data []byte) (changed []byte, err error) {
		changed = regexp.MustCompile(`var Version = ".+?"`).ReplaceAll(data, []byte(`var Version = "`+version.String()+`"`))

		if bytes.Equal(data, changed) {
			return nil, errors.New("could not change version")
		}

		return changed, nil
	}); err != nil {
		logger.Panicf("cannot change version in Go code: %+v", err)
	}

	logger.Info("create release commit")
	if _, err := util.CommandWithResult(context.Background(), logger, &util.Command{
		Command: []string{
			"git",
			"add",
			"evaluate/version.go",
		},
	}); err != nil {
		logger.Panicf(err.Error())
	}
	if _, err := util.CommandWithResult(context.Background(), logger, &util.Command{
		Command: []string{
			"git",
			"commit",
			"--message=Release version " + versionTag,
		},
	}); err != nil {
		logger.Panicf(err.Error())
	}

	logger.Info("tag release commit")
	if _, err := util.CommandWithResult(context.Background(), logger, &util.Command{
		Command: []string{
			"git",
			"tag",
			"--annotate",
			"--message=Version " + versionTag,
			"--sign",
			versionTag,
		},
	}); err != nil {
		logger.Panicf(err.Error())
	}

	logger.Info("push the branch and tag")
	if _, err := util.CommandWithResult(context.Background(), logger, &util.Command{
		Command: []string{
			"git",
			"push",
			"origin",
		},
	}); err != nil {
		logger.Panicf(err.Error())
	}
	if _, err := util.CommandWithResult(context.Background(), logger, &util.Command{
		Command: []string{
			"git",
			"push",
			"origin",
			versionTag,
		},
	}); err != nil {
		logger.Panicf(err.Error())
	}
}
