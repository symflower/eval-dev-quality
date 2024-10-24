package main

import (
	"os"

	"github.com/symflower/eval-dev-quality/cmd/eval-dev-quality/cmd"
	"github.com/symflower/eval-dev-quality/cmd/eval-dev-quality/cmd/cleanup"
	"github.com/symflower/eval-dev-quality/log"
)

func main() {
	cleanup.Init()
	defer cleanup.Trigger()

	cmd.Execute(log.STDOUT(), os.Args[1:])
}
