package common

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

type logFormatter struct{}

func (m *logFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	var b *bytes.Buffer
	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}

	timestamp := entry.Time.Format("2006-01-02 15:04:05")
	var newLog string

	if entry.HasCaller() {
		fileName := filepath.Base(entry.Caller.File)
		newLog = fmt.Sprintf("[%s] [%s] [%s:%d]  %s\n", timestamp, entry.Level, fileName, entry.Caller.Line, entry.Message)
	} else {
		newLog = fmt.Sprintf("[%s] [%s]: %s\n", timestamp, entry.Level, entry.Message)
	}

	b.WriteString(newLog)
	return b.Bytes(), nil
}

func InitLogger(verbose int, stdout bool, filename string) {
	logger := &lumberjack.Logger{
		Filename:   filename,
		MaxSize:    1,
		MaxBackups: 100,
		MaxAge:     100,
		LocalTime:  true,
	}

	if stdout {
		logrus.SetOutput(io.MultiWriter(os.Stdout, logger))
	} else {
		logrus.SetOutput(logger)
	}

	logrus.SetLevel(logrus.Level(verbose))
	logrus.SetReportCaller(true)
	logrus.SetFormatter(&logFormatter{})
}
