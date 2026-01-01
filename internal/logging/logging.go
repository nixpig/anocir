package logging

import (
	"bytes"
	"io"
	"log/slog"
)

// ErrorWriter wraps a Logger and implements the Writer interface.
type ErrorWriter struct {
	logger *slog.Logger
}

// Write implements the Writer interface and writes the given bytes to the
// error logger.
func (ew *ErrorWriter) Write(p []byte) (int, error) {
	ew.logger.Error(string(bytes.TrimSpace(p)))
	return len(p), nil
}

// NewErrorWriter creates a ErrorWriter for the given logger.
func NewErrorWriter(logger *slog.Logger) *ErrorWriter {
	return &ErrorWriter{logger}
}

// NewLogger creates a Logger, outputting to the given logfile. If debug is
// true then the log level is set to DEBUG, else it's INFO.
func NewLogger(w io.Writer, debug bool, format string) *slog.Logger {
	level := slog.LevelInfo
	addSource := false
	if debug {
		level = slog.LevelDebug
		addSource = true
	}

	options := &slog.HandlerOptions{
		Level:     level,
		AddSource: addSource,
	}

	var handler slog.Handler

	if format == "json" {
		handler = slog.NewJSONHandler(w, options)
	} else {
		handler = slog.NewTextHandler(w, options)
	}

	logger := slog.New(handler)

	return logger
}
