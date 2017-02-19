package treeloader

import (
	"log"
	"runtime"
)

type loggerFunc func(string, ...interface{})

func detailedLog(format string, args ...interface{}) {
	// var pcs [512]uintptr
	// n := runtime.Callers(3, pcs[:])
	// cs := make([]uintptr, n)
	// copy(cs, pcs[:n])
	// caller := cs[0]
	// fn := runtime.FuncForPC(caller)
	// file, line := fn.FileLine(caller)
	// format = fmt.Sprintf("%s %s:%d %s", fn.Name(), file, line, format)
	_, file, line, ok := runtime.Caller(1)
	if ok {
		log.Printf("%s: %d", file, line)
	}
	log.Printf(format, args...)
}

func makeLogger(verboseLogging bool) loggerFunc {
	if verboseLogging {
		return detailedLog
	}
	return func(fmt string, args ...interface{}) {}
}
