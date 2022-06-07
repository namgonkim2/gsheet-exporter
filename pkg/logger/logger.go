package logger

import (
	"io"
	"log"
	"os"
	"sync"
)

type Logger struct {
	Info  *log.Logger
	Warn  *log.Logger
	Error *log.Logger
}

var (
	gsheetExporterLogger *Logger
	once                 sync.Once
)

func GetInstance() *Logger {
	once.Do(func() {
		gsheetExporterLogger = New(os.Stdout, os.Stdout, os.Stderr)
	})
	return gsheetExporterLogger

}

func New(infoHandle io.Writer, warnHandle io.Writer, errorHandle io.Writer) *Logger {

	return &Logger{
		log.New(infoHandle, "[INFO] ", log.Ldate|log.Ltime|log.Lshortfile),
		log.New(warnHandle, "[WARN] ", log.Ldate|log.Ltime|log.Lshortfile),
		log.New(errorHandle, "[ERROR] ", log.Ldate|log.Ltime|log.Lshortfile),
	}
}
