package log

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/GPTx-global/guru/oracle/config"
)

var customLog logger

type logger struct {
	debug *log.Logger
	info  *log.Logger
	err   *log.Logger
	dir   string
}

func InitLogger() {
	customLog.dir = *config.DaemonDir

	format := log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile
	name := fmt.Sprintf("%s.%d.log", filepath.Base(os.Args[0]), os.Getpid())
	file, err := os.Create(filepath.Join(customLog.dir, name))
	if err != nil {
		log.Fatal(err)
	}

	customLog.debug = log.New(file, "[DEBUG] ", format)
	customLog.info = log.New(file, "[INFO] ", format)
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
