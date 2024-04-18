package log

import (
	"bytes"
	"io"
	"log"
	"os"
	"path/filepath"

	pkgerrors "github.com/pkg/errors"
)

// Buffer returns a logger that writes to a buffer.
func Buffer() (buffer *bytes.Buffer, logger *log.Logger) {
	buffer = new(bytes.Buffer)
	logger = log.New(buffer, "", log.LstdFlags)

	return buffer, logger
}

// File returns a logger that writes to a file.
func File(path string) (logger *log.Logger, loggerClose func(), err error) {
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

	logger = log.New(file, "", log.LstdFlags)

	return logger, loggerClose, nil
}

// FileAndSTDOUT returns a logger that writes to a file and STDOUT at the same time.
func FileAndSTDOUT(filePath string) (logger *log.Logger, loggerClose func(), err error) {
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

	writer := io.MultiWriter(os.Stdout, file)
	logger = log.New(writer, "", log.LstdFlags)

	return logger, loggerClose, nil
}

// STDOUT returns a logger that writes to STDOUT.
func STDOUT() (logger *log.Logger) {
	return log.New(os.Stdout, "", log.LstdFlags)
}
