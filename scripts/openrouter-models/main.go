package main

import (
	"strings"

	"github.com/symflower/eval-dev-quality/model"
	"github.com/symflower/eval-dev-quality/provider/openrouter"
	"github.com/symflower/eval-dev-quality/util"
)

func main() {
	ignoredModels := map[string]bool{
		"openrouter/openrouter/auto": true,
	}

	provider := openrouter.NewProvider()
	allModels, err := provider.Models()
	if err != nil {
		panic(err)
	}

	var models []model.Model
	for _, model := range allModels {
		_, isIgnored := ignoredModels[model.ID()]
		isFree := strings.HasSuffix(model.ID(), ":free")

		if isIgnored || isFree {
			continue
		}

		models = append(models, model)
	}

	util.PrettyPrint(models)
}
