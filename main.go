// main.go

package main

import (
	"fmt"
	"os"

	"github.com/nixpig/anocir/internal/cli"
	"github.com/sirupsen/logrus"
	"github.com/thediveo/gons"
)

func main() {
	logfile := "/var/log/anocir/log.txt"
	if f, err := os.OpenFile(logfile, os.O_APPEND|os.O_WRONLY, os.ModeAppend); err != nil {
		fmt.Printf("Warning: failed to open log file %s. Logging to stderr.\n", logfile)
		logrus.SetOutput(os.Stderr)
	} else {
		logrus.SetOutput(f)
	}

	logrus.Infof("🤪: %s", os.Args)

	if err := gons.Status(); err != nil {
		logrus.Errorf("failed to join namespaces: %s\n", err)
		os.Exit(1)
	}

	if err := cli.RootCmd().Execute(); err != nil {
		logrus.Errorf("failed to execute: %s\n", err)
		os.Exit(1)
	}
}
