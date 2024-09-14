package main

import (
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"

	"LearningGo/socketServer"
)

func main() {
	logFile, err := os.OpenFile(
		filepath.Join(os.TempDir(), "LearningLog.log"),
		os.O_RDWR|os.O_CREATE|os.O_APPEND,
		0666,
	)
	if err != nil {
		slog.Error(err.Error())
	} else {
		slog.Info("Log file opened successfully", "file", logFile.Name())
	}
	defer func(logFile *os.File) {
		_ = logFile.Close()
	}(logFile)
	log.SetOutput(io.MultiWriter(os.Stderr, logFile))
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	defer func() {
		if r := recover(); r != nil {
			slog.Error("Recovered in f", "r", r)
		}
	}()
	socketServer.StartServer()
}
