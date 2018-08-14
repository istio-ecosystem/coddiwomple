package main

import (
	log "github.com/sirupsen/logrus"

	"os"

	"github.com/tetratelabs/mcc/cmd"
)

func init() {
	log.SetFormatter(&log.TextFormatter{})
	// log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.WarnLevel)
}

func main() {
	cmd.Execute()
}
