package logging

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
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
func NewLogger(logfile string, debug bool) (*slog.Logger, error) {
	if err := os.MkdirAll(filepath.Dir(logfile), 0o755); err != nil {
		return nil, fmt.Errorf("create log directory: %w", err)
	}

	f, err := os.OpenFile(
		logfile,
		os.O_CREATE|os.O_APPEND|os.O_WRONLY,
		0o644,
	)
	if err != nil {
		return nil, fmt.Errorf("open log file %s: %w", logfile, err)
	}

	level := slog.LevelInfo
	addSource := false
	if debug {
		level = slog.LevelDebug
		addSource = true
	}

	logger := slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{
		Level:     level,
		AddSource: addSource,
	}))

	return logger, nil
}
