package logging

import (
	"io"
	"log"
)

type Logger struct {
	*log.Logger
}

func New(w io.Writer) *Logger {
	return &Logger{Logger: log.New(w, "agswitch: ", log.LstdFlags)}
}
