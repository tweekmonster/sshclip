package sshclip

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/fatih/color"
)

// Debug enables/disables debug logging.  This can be set at runtime.
var Debug = true
var LogPrefix = ""
var logger = log.New(color.Output, "", log.Ldate|log.Ltime|log.Lshortfile)
var logFunc = func(message string) {
	logger.Output(4, message)
}

func init() {
	f, _ := os.OpenFile("/tmp/sshclip.log", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0755)
	logger.SetOutput(f)
}

func combine(v ...interface{}) string {
	out := ""
	for _, val := range v {
		if s, ok := val.(fmt.Stringer); ok {
			out += s.String() + " "
		} else {
			out += fmt.Sprintf("%v ", val)
		}
	}
	return out[:len(out)-1]
}

func formatMessage(v ...interface{}) string {
	var message string

	switch first := v[0].(type) {
	case string:
		if strings.Contains(first, "%") {
			message = fmt.Sprintf(first, v[1:]...)
		}
	}

	if message == "" {
		message = combine(v...)
	}

	if i := strings.LastIndex(message, "\x1b[0m"); i != -1 {
		message = message[:i+4] + strings.Replace(message[i+4:], "\r", "\\r", 0)
	}
	return LogPrefix + ": " + message
}

func SetLogFunc(l func(string)) {
	logFunc = l
}

func SetLogOutput(w io.Writer) {
	logger.SetOutput(w)
}

func _log(args ...interface{}) {
	logFunc(formatMessage(args...))
}

// Log is an info log.
func Log(args ...interface{}) {
	_log(args...)
}

// Dlog is the debug log.
func Dlog(args ...interface{}) {
	if Debug {
		var margs []interface{}
		margs = append(margs, combine(color.CyanString("[DEBUG]"), args[0]))
		if len(args) > 1 {
			margs = append(margs, args[1:]...)
		}
		_log(margs...)
	}
}

// Elog is the error log.
func Elog(args ...interface{}) {
	var margs []interface{}
	margs = append(margs, combine(color.RedString("[ERROR]"), args[0]))
	if len(args) > 1 {
		margs = append(margs, args[1:]...)
	}
	_log(margs...)
}
