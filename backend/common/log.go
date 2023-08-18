package common

import (
	"bytes"
	"fmt"
	"io"
	"log"
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

func InitLogger(name string) {
	logger := &lumberjack.Logger{
		Filename:   filepath.Join(Config.Log.Basedir, name),
		MaxSize:    1,
		MaxBackups: 100,
		MaxAge:     100,
		LocalTime:  true,
	}

	if Config.Log.Stdout {
		logrus.SetOutput(io.MultiWriter(os.Stdout, logger))
	} else {
		logrus.SetOutput(logger)
	}

	level, err := logrus.ParseLevel(Config.Log.Level)
	if err != nil {
		log.Printf("parse log level=%s err=%v", level, err)
		level = logrus.InfoLevel
	}

	logrus.SetLevel(level)
	logrus.SetReportCaller(true)
	logrus.SetFormatter(&logFormatter{})
}
