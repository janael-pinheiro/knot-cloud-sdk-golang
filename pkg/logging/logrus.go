package logging

import (
	"io"

	"github.com/sirupsen/logrus"
)

// Logrus represents the logrus logger
type Logrus struct {
	level  string
	output io.Writer
}

// NewLogrus creates a new logrus instance
func NewLogrus(level string, output io.Writer) *Logrus {
	return &Logrus{level: level, output: output}
}

// Get returns a logrus instance based on the specific context
func (l *Logrus) Get(context string) *logrus.Entry {
	log := logrus.New()
	//log.Out = os.Stderr
	level, _ := logrus.ParseLevel(l.level)
	log.SetLevel(level)
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	log.SetOutput(l.output)
	logger := log.WithFields(logrus.Fields{
		"Context": context,
	})

	return logger
}
