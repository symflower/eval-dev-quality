package log

import (
	"context"
	"errors"
	"log/slog"
	"slices"
)

// subHandler is a handler that delegates to sub-handlers.
type subHandler interface {
	// Handlers returns the sub-handlers of the handler.
	Handlers() []slog.Handler
}

// getHandlerFromChain walks through a chain of handlers to return any handlers matched by the filter.
func getHandlerFromChain(chain slog.Handler, filter func(slog.Handler) bool) []slog.Handler {
	if filter(chain) {
		return []slog.Handler{chain}
	}

	var matches []slog.Handler
	if handler, ok := chain.(subHandler); ok {
		for _, sub := range handler.Handlers() {
			if subMatches := getHandlerFromChain(sub, filter); subMatches != nil {
				matches = append(matches, subMatches...)
			}
		}
	}

	return matches
}

var _ slog.Handler = (*multiHandler)(nil)
var _ subHandler = (*multiHandler)(nil)

// multiHandler is a handler that just distributes to multiple handlers in parallel.
// Adapted from https://github.com/samber/slog-multi/blob/17df2e74690f9216b7773f771ce1b8d76d7206c7/multi.go.
type multiHandler struct {
	handlers []slog.Handler
}

// newMultiHandler distributes records to multiple slog.Handler in parallel.
func newMultiHandler(handlers ...slog.Handler) slog.Handler {
	return &multiHandler{
		handlers: handlers,
	}
}

// Implements slog.Handler
func (h *multiHandler) Enabled(ctx context.Context, l slog.Level) bool {
	return true
}

// Implements slog.Handler
func (h *multiHandler) Handle(ctx context.Context, r slog.Record) error {
	var errs []error
	for i := range h.handlers {
		if h.handlers[i].Enabled(ctx, r.Level) {
			if err := h.handlers[i].Handle(ctx, r.Clone()); err != nil {
				errs = append(errs, err)
			}
		}
	}

	return errors.Join(errs...)
}

// Implements slog.Handler
func (h *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		handlers[i] = handler.WithAttrs(slices.Clone(attrs))
	}

	return newMultiHandler(handlers...)
}

// Implements slog.Handler
func (h *multiHandler) WithGroup(name string) slog.Handler {
	// https://cs.opensource.google/go/x/exp/+/46b07846:slog/handler.go;l=247
	if name == "" {
		return h
	}

	handlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		handlers[i] = handler.WithGroup(name)
	}

	return newMultiHandler(handlers...)
}

// Handlers returns the sub-handlers of the handler.
func (h *multiHandler) Handlers() []slog.Handler {
	return h.handlers
}
