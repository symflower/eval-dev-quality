package log

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	pkgerrors "github.com/pkg/errors"
	"github.com/zimmski/osutil/bytesutil"
)

// AttributeKey defines a key for attributes handed to the structural logger.
type AttributeKey string

const (
	// AttributeKeyArtifact holds the key for the "Artifact" attribute.
	AttributeKeyArtifact = AttributeKey("Artifact")
	// AttributeKeyLanguage holds the key for the "Language" attribute.
	AttributeKeyLanguage = AttributeKey("Language")
	// AttributeKeyModel holds the key for the "Model" attribute.
	AttributeKeyModel = AttributeKey("Model")
	// AttributeKeyRepository holds the key for the "Repository" attribute.
	AttributeKeyRepository = AttributeKey("Repository")
	// AttributeKeyResultPath holds the key for the "ResultPath" attribute.
	AttributeKeyResultPath = AttributeKey("ResultPath")
	// AttributeKeyRun holds the key for the "Run" attribute.
	AttributeKeyRun = AttributeKey("Run")
	// AttributeKeyTask holds the key for the "Task" attribute.
	AttributeKeyTask = AttributeKey("Task")
)

// Attribute returns a logging attribute.
func Attribute(key AttributeKey, value any) (attribute slog.Attr) {
	return slog.Any(string(key), value)
}

var (
	// openLogFiles holds the files that were opened by some logger.
	openLogFiles      []*os.File
	openLogFilesMutex sync.Mutex
)

func addOpenLogFile(file *os.File) {
	openLogFilesMutex.Lock()
	defer openLogFilesMutex.Unlock()

	openLogFiles = append(openLogFiles, file)
}

// CloseOpenLogFiles closes the files that were opened by some logger.
func CloseOpenLogFiles() {
	openLogFilesMutex.Lock()
	defer openLogFilesMutex.Unlock()

	for _, logFile := range openLogFiles {
		if err := logFile.Close(); err != nil {
			panic(err)
		}
	}

	openLogFiles = nil
}

// openLogFile opens the given file and creates it if necessary.
func openLogFile(filePath string) (file *os.File, err error) {
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return nil, pkgerrors.WithStack(err)
	}

	file, err = os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return nil, pkgerrors.WithStack(err)
	}

	addOpenLogFile(file)

	return file, nil
}

// Logger holds a logger to log to.
type Logger struct {
	*slog.Logger
}

func newLoggerWithHandler(handler slog.Handler) *Logger {
	return &Logger{
		Logger: slog.New(newSpawningHandler(handler, defaultLogFileSpawners)),
	}
}

// With returns a Logger that includes the given attributes in each output operation.
func (l *Logger) With(key AttributeKey, value any) *Logger {
	return &Logger{
		Logger: l.Logger.With(string(key), value),
	}
}

// PrintfWithoutMeta prints a message without any timestamp, log level or origin program counter.
func (l *Logger) PrintfWithoutMeta(message string, args ...any) {
	// If time, level and PC use default values any Handler should ignore these fields (https://pkg.go.dev/log/slog#Handler).
	record := slog.NewRecord(time.Time{}, slog.LevelInfo, fmt.Sprintf(message, args...), 0)
	_ = l.Logger.Handler().Handle(context.Background(), record)
}

// Panicf is equivalent to "Printf" followed by a panic.
func (l *Logger) Panicf(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	l.Logger.Info(message)

	panic(message)
}

// Fatalf is equivalent to "Print" followed by a "os.Exit(1)".
func (l *Logger) Fatalf(v ...any) {
	l.Logger.Info(fmt.Sprint(v...))

	//revive:disable:deep-exit
	os.Exit(1)
	//revive:enable:deep-exit
}

// Buffer returns a logger that writes to a buffer.
func Buffer() (buffer *bytesutil.SynchronizedBuffer, logger *Logger) {
	buffer = new(bytesutil.SynchronizedBuffer)
	logger = newLoggerWithHandler(newPlainTextHandler(buffer))

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

	logger = newLoggerWithHandler(slog.NewJSONHandler(file, nil))

	return logger, loggerClose, nil
}

// STDOUT returns a logger that writes to STDOUT.
func STDOUT() (logger *Logger) {
	return newLoggerWithHandler(newPlainTextHandler(os.Stdout))
}

// spawningHandler is a structural logging handler which spawns a new handler on "WithAttrs" if one of the given spawners triggers.
type spawningHandler struct {
	// handler holds the handler to forward the records to.
	handler slog.Handler

	// attributes holds the attributes already handed to the logger.
	attributes map[AttributeKey]string

	// logFileSpawners holds the spawners responsible for spawning a new log file.
	logFileSpawners []handlerSpawner
}

// newSpawningHandler returns a new spawning handler.
func newSpawningHandler(handler slog.Handler, spawners []handlerSpawner) *spawningHandler {
	return &spawningHandler{
		handler: handler,

		attributes: map[AttributeKey]string{},

		logFileSpawners: spawners,
	}
}

// Clone returns a copy of the object.
func (h *spawningHandler) Clone() (clone *spawningHandler) {
	return &spawningHandler{
		handler: h.handler,

		attributes: maps.Clone(h.attributes),

		logFileSpawners: slices.Clone(h.logFileSpawners),
	}
}

var _ slog.Handler = (*spawningHandler)(nil)

// Enabled reports whether the handler handles records at the given level.
func (h *spawningHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return true
}

// Handle handles the Record.
func (h *spawningHandler) Handle(ctx context.Context, record slog.Record) (err error) {
	// The record might contain temporary attributes so merge them with the static attributes of the handler.
	temporaryAttributes := maps.Clone(h.attributes)
	record.Attrs(func(attribute slog.Attr) bool {
		temporaryAttributes[AttributeKey(attribute.Key)] = attribute.Value.String()

		return true
	})

	if newHandler := h.spawnIfNecessary(temporaryAttributes); newHandler != nil {
		return newHandler.Handle(ctx, record)
	}

	return h.handler.Handle(ctx, record)
}

// WithAttrs returns a new Handler whose attributes consist of both the receiver's attributes and the arguments.
// The Handler owns the slice: it may retain, modify or discard it.
func (h *spawningHandler) WithAttrs(attributes []slog.Attr) slog.Handler {
	// Collect new attributes and merge them with the already existing attributes of the handler.
	combinedAttributes := maps.Clone(h.attributes)
	for _, attribute := range attributes {
		combinedAttributes[AttributeKey(attribute.Key)] = attribute.Value.String()
	}

	if newHandler := h.spawnIfNecessary(combinedAttributes); newHandler != nil {
		return newHandler
	}

	// We cannot modify the handler itself but must return a new instance.
	newHandler := h.Clone()
	newHandler.attributes = combinedAttributes

	return newHandler
}

// spawnIfNecessary checks if the given attributes trigger the creation of a new handler and returns it with the given attributes set.
func (h *spawningHandler) spawnIfNecessary(attributes map[AttributeKey]string) slog.Handler {
	for i, spawner := range h.logFileSpawners {
		if !spawner.NeedsSpawn(attributes) {
			continue
		}

		child, err := spawner.Spawn(attributes)
		if err != nil {
			logToHandler(h.handler, slog.LevelError, "ERROR: cannot create new handler: %s", err)

			continue
		}

		newHandler := h.Clone()
		newHandler.handler = child
		newHandler.attributes = attributes
		newHandler.logFileSpawners = slices.Delete(newHandler.logFileSpawners, i, i+1) // The currently triggered spawner must not be part of the new handler as it would trigger again and again.

		return newHandler
	}

	return nil
}

// WithGroup returns a new Handler with the given group appended to the receiver's existing groups.
func (h *spawningHandler) WithGroup(group string) slog.Handler {
	logToHandler(h.handler, slog.LevelWarn, "Groups are unsupported %q", group)

	return h
}

var defaultLogFileSpawners = []handlerSpawner{
	handlerSpawner{
		NeededAttributes: []AttributeKey{
			AttributeKeyResultPath,
		},
		Spawn: func(attributes map[AttributeKey]string) (slog.Handler, error) {
			file, err := openLogFile(filepath.Join(attributes[AttributeKeyResultPath], "evaluation.log"))
			if err != nil {
				return nil, err
			}

			return slog.NewJSONHandler(file, nil), nil
		},
	},
	handlerSpawner{
		NeededAttributes: []AttributeKey{
			AttributeKeyResultPath,

			AttributeKeyLanguage,
			AttributeKeyModel,
			AttributeKeyRepository,
			AttributeKeyTask,
		},
		Spawn: func(attributes map[AttributeKey]string) (slog.Handler, error) {
			resultPath := attributes[AttributeKeyResultPath]
			modelID := attributes[AttributeKeyModel]
			languageID := attributes[AttributeKeyLanguage]
			repositoryName := attributes[AttributeKeyRepository]
			taskIdentifier := attributes[AttributeKeyTask]

			file, err := openLogFile(filepath.Join(resultPath, taskIdentifier, CleanModelNameForFileSystem(modelID), languageID, repositoryName, "evaluation.log"))
			if err != nil {
				return nil, err
			}

			return slog.NewJSONHandler(file, nil), nil
		},
	},
	handlerSpawner{
		NeededAttributes: []AttributeKey{
			AttributeKeyResultPath,

			AttributeKeyArtifact,
			AttributeKeyLanguage,
			AttributeKeyModel,
			AttributeKeyRepository,
			AttributeKeyRun,
			AttributeKeyTask,
		},
		Spawn: func(attributes map[AttributeKey]string) (slog.Handler, error) {
			resultPath := attributes[AttributeKeyResultPath]
			modelID := attributes[AttributeKeyModel]
			languageID := attributes[AttributeKeyLanguage]
			repositoryName := attributes[AttributeKeyRepository]
			taskIdentifier := attributes[AttributeKeyTask]
			run := attributes[AttributeKeyRun]
			artifact := attributes[AttributeKeyArtifact]

			file, err := openLogFile(filepath.Join(resultPath, taskIdentifier, CleanModelNameForFileSystem(modelID), languageID, repositoryName, fmt.Sprintf("%s-%s.log", artifact, run)))
			if err != nil {
				return nil, err
			}

			return slog.NewJSONHandler(file, nil), nil
		},
	},
}

// handlerSpawner defines when a new handler is spawned.
type handlerSpawner struct {
	// NeededAttributes holds the list of attributes that need to be set in order to spawn a new handler.
	NeededAttributes []AttributeKey
	// Spawn is called if all needed attributes are set and returns the new handler.
	Spawn func(attributes map[AttributeKey]string) (slog.Handler, error)
}

// NeedsSpawn returns if a new log file has to be spawned.
func (s handlerSpawner) NeedsSpawn(attributes map[AttributeKey]string) bool {
	for _, attributeKey := range s.NeededAttributes {
		if value := attributes[attributeKey]; value == "" {
			return false
		}
	}

	return true
}

// plainTextHandler wraps a normal TextHandler with the ability to print plain text if no timestamp, log level and program counter are provided.
type plainTextHandler struct {
	handler slog.Handler
	writer  io.Writer
}

func newPlainTextHandler(writer io.Writer) slog.Handler {
	return &plainTextHandler{
		handler: slog.NewTextHandler(writer, nil),
		writer:  writer,
	}
}

// Enabled implements slog.Handler.
func (p *plainTextHandler) Enabled(context.Context, slog.Level) bool {
	return true
}

// Handle implements slog.Handler.
func (p *plainTextHandler) Handle(ctx context.Context, record slog.Record) error {
	// The default "TextHandler" would still print `msg=...` but we just print the message as a whole.
	if record.Time.IsZero() && record.Level == slog.LevelInfo && record.PC == 0 {
		_, err := p.writer.Write([]byte(record.Message))

		return err
	}

	return p.handler.Handle(ctx, record)
}

// WithAttrs implements slog.Handler.
func (p *plainTextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &plainTextHandler{
		handler: p.handler.WithAttrs(attrs),
		writer:  p.writer,
	}
}

// WithGroup implements slog.Handler.
func (p *plainTextHandler) WithGroup(name string) slog.Handler {
	return &plainTextHandler{
		handler: p.handler.WithGroup(name),
		writer:  p.writer,
	}
}

var _ slog.Handler = (*plainTextHandler)(nil)

// logToHandler logs directly to a handler to communicate logging-internal events.
func logToHandler(handler slog.Handler, level slog.Level, message string, args ...any) {
	_ = handler.Handle(context.Background(), slog.NewRecord(time.Now(), level, fmt.Sprintf(message, args...), 0))
}

var cleanModelNameForFileSystemReplacer = strings.NewReplacer(
	"/", "_",
	"\\", "_",
	":", "_",
)

// CleanModelNameForFileSystem cleans a model name to be useable for directory and file names on the file system.
func CleanModelNameForFileSystem(modelName string) (modelNameCleaned string) {
	return cleanModelNameForFileSystemReplacer.Replace(modelName)
}
