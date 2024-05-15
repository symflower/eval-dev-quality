package tools

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/zimmski/osutil"

	"github.com/symflower/eval-dev-quality/log"
)

func TestOllamaInstall(t *testing.T) {
	if !osutil.IsLinux() {
		t.Skipf("Installation of Ollama is not supported on this OS")
	}

	ValidateInstallTool(t, NewOllama())
}

func TestStartOllama(t *testing.T) {
	if !osutil.IsLinux() {
		t.Skipf("Installation of Ollama is not supported on this OS")
	}

	buffer, logger := log.Buffer()
	defer func() {
		if t.Failed() {
			t.Log(buffer.String())
		}
	}()

	shutdown, err := OllamaStart(logger, OllamaPath, OllamaURL)
	assert.NoError(t, err)

	_, err = OllamaModels(OllamaURL)
	assert.NoError(t, err)

	time.Sleep(3 * time.Second)
	assert.NoError(t, shutdown())
}
