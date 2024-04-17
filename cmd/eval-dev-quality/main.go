package main

import (
	"os"

	"github.com/symflower/eval-dev-quality/cmd/eval-dev-quality/cmd"
)

func main() {
	cmd.Execute(os.Args[1:])
}
