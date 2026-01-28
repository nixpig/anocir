package logging

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

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

	switch format {
	case "json":
		handler = slog.NewJSONHandler(w, options)
	case "text":
		handler = slog.NewTextHandler(w, options)
	default:
		handler = slog.NewTextHandler(w, options)
	}

	logger := slog.New(handler)

	return logger
}

func OpenLogFile(logFile string) (io.Writer, error) {
	if err := os.MkdirAll(filepath.Dir(logFile), 0o755); err != nil {
		return nil, err
	}

	return os.OpenFile(
		logFile,
		os.O_CREATE|os.O_APPEND|os.O_WRONLY,
		0o644,
	)
}
