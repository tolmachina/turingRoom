package logger

import (
	"log"
	"os"
	"sync"
)

var (
	instance *log.Logger
	once     sync.Once
)

// Initialize creates a new logger instance
func Initialize(filename string) {
	once.Do(func() {
		file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}
		instance = log.New(file, "", log.Ldate|log.Ltime|log.Lshortfile)
	})
}

// Info logs an info message
func Info(v ...interface{}) {
	instance.Println(v...)
}

// Error logs an error message
func Error(v ...interface{}) {
	instance.Println("ERROR:", v)
}

// Fatal logs a fatal message and exits
func Fatal(v ...interface{}) {
	instance.Fatal(v...)
}
