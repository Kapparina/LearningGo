package main

import (
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"syscall"

	"LearningGo/nonblockingWriter"
	"LearningGo/socketServer"
)

func main() {
	logFile, err := os.OpenFile(
		filepath.Join(os.TempDir(), "LearningLog.log"),
		syscall.O_CREAT|syscall.O_WRONLY|os.O_APPEND,
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
	nbLogFileWriter := nonblockingWriter.NewBufWriter(logFile, 20000)
	nbStdErrWriter := nonblockingWriter.NewBufWriter(os.Stderr, 20000)
	log.SetOutput(io.MultiWriter(nbLogFileWriter, nbStdErrWriter))
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	defer func() {
		if r := recover(); r != nil {
			slog.Error("Recovered in f", "r", r)
		}
	}()
	socketServer.Example()
}
