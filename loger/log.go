package loger

import (
	"fmt"
	"github.com/peakedshout/go-CFC/complexRW"
	"io"
	"net"
	"os"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var logLevelLock sync.RWMutex

var logLevel uint8 = LogLevelInfo
var needStack atomic.Bool

var needColor atomic.Bool

var broadcast *complexRW.RWBroadcast

func init() {
	needColor.Store(true)
	needStack.Store(false)
	broadcast = complexRW.NewRWBroadcast(nil, 64)
	broadcast.SetListener(os.Stdout)
}

func SetLoggerCopy(w io.Writer) complexRW.RWId {
	return broadcast.SetListener(w)
}
func DelLoggerCopy(id complexRW.RWId) {
	broadcast.DelListener(id)
}

func SetLoggerLevel(l uint8) {
	logLevelLock.Lock()
	defer logLevelLock.Unlock()
	logLevel = l
}
func SetLoggerStack(need bool) {
	needStack.Store(need)
}

func SetLoggerColor(need bool) {
	needColor.Store(need)
}

func getLoggerLevel() uint8 {
	logLevelLock.RLock()
	defer logLevelLock.RUnlock()
	return logLevel
}

const (
	LogLevelAll = iota
	LogLevelTrace
	LogLevelDebug
	LogLevelInfo
	LogLevelWarn
	LogLevelXY
	LogLevelError
	LogLevelFatal
	LogLevelOff
	LogLevelMust
)

var logShow = []string{"ALL", "TRACE", "DEBUG", "INFO", "WARN", "Log", "ERROR", "FATAL", "OFF", "Must"}

func SetLogAll(a ...any) {
	setLog(LogLevelAll, a...)
}
func SetLogTrace(a ...any) {
	setLog(LogLevelTrace, a...)
}
func SetLogDebug(a ...any) {
	setLog(LogLevelDebug, a...)
}
func SetLogInfo(a ...any) {
	setLog(LogLevelInfo, a...)
}
func SetLogWarn(a ...any) {
	setLog(LogLevelWarn, a...)
}
func SetLogXY(a ...any) {
	setLog(LogLevelXY, a...)
}
func SetLogError(a ...any) {
	setLog(LogLevelError, a...)
}
func SetLogFatal(a ...any) {
	setLog(LogLevelFatal, a...)
}
func SetLogOff(a ...any) {
	setLog(LogLevelOff, a...)
}
func SetLogMust(a ...any) {
	setLog(LogLevelMust, a...)
}

func setLog(level uint8, a ...any) {
	ll := getLoggerLevel()
	if level < ll || ll == LogLevelOff {
		return
	} else {
		pre := getPreTag(level)
		now := time.Now().Format("2006/01/02 15:04:05")
		now = SprintColor(4, 1, 1, now)
		body := Sprint(a...)

		str := fmt.Sprintln(pre, now, body, addStack())
		broadcast.Write([]byte(str))
		switch level {
		case LogLevelAll, LogLevelTrace, LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelXY, LogLevelMust:
		case LogLevelError:
			panic(fmt.Sprintln(pre, now, body))
		case LogLevelFatal:
			os.Exit(1)
		}
	}
}

func Sprint(a ...any) string {
	return strings.TrimSuffix(fmt.Sprintln(a...), "\n")
}

func SprintConn(conn net.Conn, a ...any) string {
	if conn == nil {
		return Sprint("[", "No network", "] :", Sprint(a...))
	}
	return Sprint("[", conn.LocalAddr().String(), "->", conn.RemoteAddr().String(), "] :", Sprint(a...))
}

func getPreTag(logLevel uint8) (out string) {
	str := logShow[logLevel]
	switch logLevel {
	case LogLevelAll:
		out = SprintColor(7, 37, 47, "[", str, "]")
	case LogLevelTrace:
		out = SprintColor(7, 34, 44, "[", str, "]")
	case LogLevelDebug:
		out = SprintColor(7, 36, 46, "[", str, "]")
	case LogLevelInfo:
		out = SprintColor(7, 32, 42, "[", str, "]")
	case LogLevelWarn:
		out = SprintColor(7, 33, 43, "[", str, "]")
	case LogLevelXY:
		out = SprintColor(7, 38, 43, "[", str, "]")
	case LogLevelError:
		out = SprintColor(7, 31, 41, "[", str, "]")
	case LogLevelFatal:
		out = SprintColor(7, 35, 45, "[", str, "]")
	case LogLevelMust:
		out = SprintColor(7, 30, 40, "[", str, "]")
	}
	return out
}

func SprintColor(t, f, b int, body ...any) string {
	if needColor.Load() {
		return fmt.Sprintf("\033[%d;%d;%dm%s\033[0m", t, f, b, fmt.Sprint(body...))
	} else {
		return fmt.Sprint(body...)
	}
}

func addStack() string {
	if needStack.Load() {
		return "\n[Stack View]:\n" + stack()
	}
	return ""
}
func stack() string {
	str := string(debug.Stack())
	sl := strings.Split(str, "\n")
	sl = sl[11:]
	return strings.Join(sl, "\n")
}
