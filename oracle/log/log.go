package log

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

var customLog logger

type logger struct {
	debug *log.Logger
	info  *log.Logger
	err   *log.Logger
	dir   string
}

func InitLogger() {
	customLog = logger{
		debug: log.New(os.Stdout, "[DEBUG] ", 0),
		info:  log.New(os.Stdout, "[INFOM] ", 0),
		err:   log.New(os.Stderr, "[ERROR] ", 0),
		dir:   "",
	}
}

func ResetLogger(oracleHome string) {
	if oracleHome == "" {
		osHome, err := os.UserHomeDir()
		if err != nil {
			Fatalf("Failed to get user home directory: %v", err)
		}
		customLog.dir = filepath.Join(osHome, ".oracled", "logs")
	} else {
		customLog.dir = filepath.Join(oracleHome, "logs")
	}

	if err := os.MkdirAll(customLog.dir, 0755); err != nil {
		Fatalf("Failed to create log directory %s: %v", customLog.dir, err)
	}

	format := log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile
	name := fmt.Sprintf("%s.%d.log", filepath.Base(os.Args[0]), os.Getpid())
	path := filepath.Join(customLog.dir, name)
	file, err := os.Create(path)
	if err != nil {
		Fatalf("Failed to create log file: %v", err)
	}

	Infof("From now on, all logs will be written to %s", path)

	customLog.debug = log.New(file, "[DEBUG] ", format)
	customLog.info = log.New(file, "[INFOM] ", format)
	customLog.err = log.New(file, "[ERROR] ", format)
}

func Debug(v ...any) {
	_ = customLog.debug.Output(2, fmt.Sprint(v...))
}

func Debugf(format string, v ...any) {
	_ = customLog.debug.Output(2, fmt.Sprintf(format, v...))
}

func Info(v ...any) {
	_ = customLog.info.Output(2, fmt.Sprint(v...))
}

func Infof(format string, v ...any) {
	_ = customLog.info.Output(2, fmt.Sprintf(format, v...))
}

func Error(v ...any) {
	_ = customLog.err.Output(2, fmt.Sprint(v...))
}

func Errorf(format string, v ...any) {
	_ = customLog.err.Output(2, fmt.Sprintf(format, v...))
}

func Fatal(v ...any) {
	_ = customLog.err.Output(2, fmt.Sprint(v...))
	log.Fatal(v...)
}

func Fatalf(format string, v ...any) {
	_ = customLog.err.Output(2, fmt.Sprintf(format, v...))
	log.Fatalf(format, v...)
}
