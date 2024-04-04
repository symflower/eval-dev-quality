export ROOT_DIR := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))

export PACKAGE_BASE := github.com/symflower/eval-dev-quality
export UNIT_TEST_TIMEOUT := 480

ARGS := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
$(eval $(ARGS):;@:) # turn arguments into do-nothing targets
export ARGS

ifdef ARGS
	HAS_ARGS := "1"
else
	HAS_ARGS :=
	PACKAGE := $(PACKAGE_BASE)/...
endif

.DEFAULT_GOAL := help

clean: # Clean up artifacts of the development environment to allow for untainted builds, installations and updates.
	go clean -i $(PACKAGE)
	go clean -i -race $(PACKAGE)
.PHONY: clean

install: # [<Go package] - # Build and install everything, or only the specified package.
	go install -v $(PACKAGE)
.PHONY: install

install-all: install-tools-testing install # Install everything for and of this repository.
.PHONY: install-all

install-tools-testing: # Install tools that are used for testing.
	go install -v gotest.tools/gotestsum@v1.11.0
.PHONY: install-tools-testing

test: # [<Go package] - # Test everything, or only the specified package.
	gotestsum --format standard-verbose --hide-summary skipped -- -race -test.timeout $(UNIT_TEST_TIMEOUT)s -v $(PACKAGE)
.PHONY: test
