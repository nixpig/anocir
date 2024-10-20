package logging

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog"
)

func CreateLogger(logfile string, level zerolog.Level) (*zerolog.Logger, error) {
	dir, _ := filepath.Split(logfile)

	if err := os.MkdirAll(dir, 0666); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}

	logFile, err := os.OpenFile(
		logfile,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0644,
	)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}

	log := zerolog.New(logFile).
		With().
		Timestamp().
		Logger().
		Level(level)

	return &log, nil
}
