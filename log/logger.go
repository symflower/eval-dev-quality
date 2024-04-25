package log

import (
	"io"
	"log"
	"os"
	"path/filepath"

	pkgerrors "github.com/pkg/errors"
	"github.com/zimmski/osutil/bytesutil"
)

// Logger holds a logger to log to.
type Logger = log.Logger

// newLoggerWithWriter instantiate a logger with a writer.
func newLoggerWithWriter(writer io.Writer) *Logger {
	return log.New(writer, "", log.LstdFlags)
}

// Buffer returns a logger that writes to a buffer.
func Buffer() (buffer *bytesutil.SynchronizedBuffer, logger *Logger) {
	buffer = new(bytesutil.SynchronizedBuffer)
	logger = newLoggerWithWriter(buffer)

	return buffer, logger
}

// File returns a logger that writes to a file.
func File(path string) (logger *Logger, loggerClose func(), err error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, nil, pkgerrors.WithStack(err)
	}

	file, err := os.OpenFile(path, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, nil, pkgerrors.WithStack(err)
	}
	loggerClose = func() {
		if err := file.Close(); err != nil {
			panic(err)
		}
	}

	logger = newLoggerWithWriter(file)

	return logger, loggerClose, nil
}

// STDOUT returns a logger that writes to STDOUT.
func STDOUT() (logger *Logger) {
	return newLoggerWithWriter(os.Stdout)
}

// WithFile returns a logger that writes to a file and to the parent logger at the same time.
func WithFile(parent *Logger, filePath string) (logger *Logger, loggerClose func(), err error) {
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return nil, nil, pkgerrors.WithStack(err)
	}

	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, nil, pkgerrors.WithStack(err)
	}
	loggerClose = func() {
		if err := file.Close(); err != nil {
			panic(err)
		}
	}

	writer := io.MultiWriter(parent.Writer(), file)
	logger = newLoggerWithWriter(writer)

	return logger, loggerClose, nil
}
