package logging

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

type ErrorWriter struct {
	logger *slog.Logger
}

func (ew *ErrorWriter) Write(p []byte) (int, error) {
	ew.logger.Error(string(bytes.TrimSpace(p)))
	return len(p), nil
}

func NewErrorWriter(logger *slog.Logger) *ErrorWriter {
	return &ErrorWriter{logger}
}

func Initialise(logfile string, debug bool) (*slog.Logger, error) {
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
