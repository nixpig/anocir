package logging

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

func Initialise(logfile string, debug bool) {
	if err := os.MkdirAll(filepath.Dir(logfile), 0o755); err != nil {
		fmt.Printf("Warning: failed to create log directory: %v\n", err)
	}

	if f, err := os.OpenFile(
		logfile,
		os.O_CREATE|os.O_APPEND|os.O_WRONLY,
		0o644,
	); err != nil {
		fmt.Printf("Warning: failed to open log file: %v\n", err)
		fmt.Println("Logging to stdout...")
	} else {
		logrus.SetOutput(f)
	}

	if debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	logrus.SetFormatter(&logrus.TextFormatter{
		DisableColors: false,
		FullTimestamp: true,
	})
}
