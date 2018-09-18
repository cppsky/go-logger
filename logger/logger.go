package logger

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"
)

const (
	_VER              string = "1.0.0"
	_defaultCallDepth        = 4
)

type LEVEL int32

type Logger struct {
	logLevel        LEVEL
	maxFileSize     int64
	maxFileCount    int32
	dailyRolling    bool
	consoleAppender bool
	RollingFile     bool
	mu              *sync.RWMutex
	logObj          *_FILE
}

const DATEFORMAT = "2006-01-02"

type UNIT int64

const (
	_       = iota
	KB UNIT = 1 << (iota * 10)
	MB
	GB
	TB
)

const (
	ALL LEVEL = iota
	DEBUG
	INFO
	WARN
	ERROR
	FATAL
	OFF
)

type _FILE struct {
	dir      string
	filename string
	_suffix  int
	isCover  bool
	_date    *time.Time

	logfile *os.File
	lg      *log.Logger
}

var DefaultLogger = &Logger{logLevel: 1, dailyRolling: true, consoleAppender: true, mu: new(sync.RWMutex)}

func New() *Logger {
	return &Logger{logLevel: 1, dailyRolling: true, consoleAppender: true, mu: new(sync.RWMutex)}
}

func SetConsole(isConsole bool) {
	DefaultLogger.SetConsole(isConsole)
}

func SetLevel(_level LEVEL) {
	DefaultLogger.SetLevel(_level)
}

func (logger *Logger) SetLevel(_level LEVEL) {
	logger.logLevel = _level
}

func (logger *Logger) Write(p []byte) (n int, err error) {
	logger.Info(string(p))
	return 0, nil
}

func (logger *Logger) SetConsole(isConsole bool) {
	logger.consoleAppender = isConsole
}

func (logger *Logger) SetRollingFile(fileDir, fileName string, maxNumber int32, maxSize int64, _unit UNIT) {
	logger.maxFileCount = maxNumber
	logger.maxFileSize = maxSize * int64(_unit)
	logger.RollingFile = true
	logger.dailyRolling = false
	logger.logObj = &_FILE{dir: fileDir, filename: fileName, isCover: false}
	logger.mu.Lock()
	defer logger.mu.Unlock()
	for i := 1; i <= int(maxNumber); i++ {
		if isExist(fileDir + "/" + fileName + "." + strconv.Itoa(i)) {
			logger.logObj._suffix = i
		} else {
			break
		}
	}
	if !logger.logObj.isMustRename(logger) {
		DefaultLogger.logObj.logfile, _ = os.OpenFile(fileDir+"/"+fileName, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
		logger.logObj.lg = log.New(logger.logObj.logfile, "", log.Ldate|log.Ltime|log.Lshortfile)
	} else {
		logger.logObj.rename(logger)
	}
	go logger.fileMonitor()
}

func SetRollingFile(fileDir, fileName string, maxNumber int32, maxSize int64, _unit UNIT) {
	DefaultLogger.SetRollingFile(fileDir, fileName, maxNumber, maxSize, _unit)
}

func SetRollingDaily(fileDir, fileName string) {
	DefaultLogger.SetRollingDaily(fileDir, fileName)
}

func (logger *Logger) SetRollingDaily(fileDir, fileName string) {
	var err error
	logger.RollingFile = false
	logger.dailyRolling = true
	t, _ := time.Parse(DATEFORMAT, time.Now().Format(DATEFORMAT))
	logger.logObj = &_FILE{dir: fileDir, filename: fileName, _date: &t, isCover: false}
	logger.mu.Lock()
	defer logger.mu.Unlock()

	if !logger.logObj.isMustRename(logger) {
		logger.logObj.logfile, err = os.OpenFile(fileDir+"/"+fileName, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
		if err != nil {
			log.Println(err)
			logger.logObj = nil
			return
		}
		logger.logObj.lg = log.New(logger.logObj.logfile, "", log.Ldate|log.Ltime|log.Lshortfile)
	} else {
		logger.logObj.rename(logger)
	}
}

func (logger *Logger) console(calldepth int, level string, s ...interface{}) {
	if logger.logObj != nil && logger.logObj.lg != nil {
		logger.logObj.lg.Output(calldepth, fmt.Sprintln(level, s))
	}

	if logger.consoleAppender {
		_, file, line, _ := runtime.Caller(calldepth - 1)
		short := file
		for i := len(file) - 1; i > 0; i-- {
			if file[i] == '/' {
				short = file[i+1:]
				break
			}
		}
		file = short
		fmt.Println(time.Now().String()+":"+file+":"+strconv.Itoa(line), level, s)
	}
}

func catchError() {
	if err := recover(); err != nil {
		log.Println("err", err)
	}
}

func (logger *Logger) checkLogObj() {
	if logger.logObj == nil {
		logger.logObj = &_FILE{dir: "", filename: "", isCover: false}
	}
}

func (logger *Logger) innerDebug(calldepth int, v ...interface{}) {
	if logger.logLevel <= DEBUG {
		if logger.dailyRolling {
			logger.fileCheck()
		}
		defer catchError()
		logger.checkLogObj()
		logger.mu.RLock()
		defer logger.mu.RUnlock()
		logger.console(calldepth, "debug", v)
	}
}

func (logger *Logger) Printf(format string, v ...interface{}) {
	logger.innerDebug(_defaultCallDepth, fmt.Sprintf(format, v...))
}

func (logger *Logger) Debug(v ...interface{}) {
	logger.innerDebug(_defaultCallDepth, v...)
}

func DebugN(calldepth int, v ...interface{}) {
	DefaultLogger.innerDebug(_defaultCallDepth+calldepth, v...)
}

func Debug(v ...interface{}) {
	DefaultLogger.innerDebug(_defaultCallDepth, v...)
}

func Debugf(format string, v ...interface{}) {
	DefaultLogger.innerDebug(4, fmt.Sprintf(format, v...))
}

func (logger *Logger) innerInfo(calldepth int, v ...interface{}) {
	if logger.logLevel <= INFO {
		if logger.dailyRolling {
			logger.fileCheck()
		}

		defer catchError()
		logger.mu.RLock()
		defer logger.mu.RUnlock()
		logger.console(calldepth, "info", v)

	}
}

func (logger *Logger) Info(v ...interface{}) {
	logger.innerInfo(_defaultCallDepth, v...)
}

func Info(v ...interface{}) {
	DefaultLogger.innerInfo(_defaultCallDepth, v...)
}

func Infof(format string, v ...interface{}) {
	DefaultLogger.innerInfo(_defaultCallDepth, fmt.Sprintf(format, v...))
}

func (logger *Logger) innerWarn(calldepth int, v ...interface{}) {
	if logger.logLevel <= WARN {
		if logger.dailyRolling {
			logger.fileCheck()
		}
		defer catchError()
		logger.mu.RLock()
		defer logger.mu.RUnlock()

		logger.console(calldepth, "warn", v)
	}
}

func (logger *Logger) Warn(v ...interface{}) {
	logger.innerWarn(_defaultCallDepth, v...)
}

func (logger *Logger) innerError(calldepth int, v ...interface{}) {
	if logger.logLevel <= ERROR {
		if logger.dailyRolling {
			logger.fileCheck()
		}
		defer catchError()
		logger.mu.RLock()
		defer logger.mu.RUnlock()

		logger.console(calldepth, "error", v)
	}
}

func (logger *Logger) Error(v ...interface{}) {
	logger.innerError(_defaultCallDepth, v...)
}

func Error(v ...interface{}) {
	DefaultLogger.innerError(_defaultCallDepth, v...)
}

func Errorf(format string, v ...interface{}) {
	DefaultLogger.innerError(_defaultCallDepth, fmt.Sprintf(format, v...))
}

func ErrorN(callDepth int, v ...interface{}) {
	DefaultLogger.innerError(callDepth+3, v...)
}

func (logger *Logger) innerFatal(calldepth int, v ...interface{}) {
	if logger.logLevel <= FATAL {
		if logger.dailyRolling {
			logger.fileCheck()
		}
		defer catchError()
		logger.mu.RLock()
		defer logger.mu.RUnlock()
		logger.console(calldepth, "fatal", v)
	}
}

func (logger *Logger) Fatal(v ...interface{}) {
	logger.innerFatal(_defaultCallDepth, v...)
}

func Fatal(v ...interface{}) {
	DefaultLogger.innerFatal(_defaultCallDepth, v...)
}

func Fatalf(format string, v ...interface{}) {
	DefaultLogger.innerFatal(_defaultCallDepth, fmt.Sprintf(format, v...))
}

func (f *_FILE) isMustRename(logger *Logger) bool {

	if logger.dailyRolling {

		if f._date == nil {
			return false
		}

		t, err := time.Parse(DATEFORMAT, time.Now().Format(DATEFORMAT))
		if err != nil {
			return false
		}

		if t.After(*f._date) {
			return true
		}
	} else {
		if logger.maxFileCount > 1 {
			if fileSize(f.dir+"/"+f.filename) >= logger.maxFileSize {
				return true
			}
		}
	}
	return false
}

func (f *_FILE) rename(logger *Logger) {
	if logger.dailyRolling {
		fn := f.dir + "/" + f.filename + "." + f._date.Format(DATEFORMAT)
		if !isExist(fn) && f.isMustRename(logger) {
			if f.logfile != nil {
				f.logfile.Close()
			}
			err := os.Rename(f.dir+"/"+f.filename, fn)
			if err != nil {
				f.lg.Println("rename err", err.Error())
			}
			t, _ := time.Parse(DATEFORMAT, time.Now().Format(DATEFORMAT))
			f._date = &t
			f.logfile, _ = os.Create(f.dir + "/" + f.filename)
			f.lg = log.New(logger.logObj.logfile, "", log.Ldate|log.Ltime|log.Lshortfile)
		}
	} else {
		f.coverNextOne(logger)
	}
}

func (f *_FILE) nextSuffix(logger *Logger) int {
	return int(f._suffix%int(logger.maxFileCount) + 1)
}

func (f *_FILE) coverNextOne(logger *Logger) {
	f._suffix = f.nextSuffix(logger)
	if f.logfile != nil {
		f.logfile.Close()
	}
	if isExist(f.dir + "/" + f.filename + "." + strconv.Itoa(int(f._suffix))) {
		os.Remove(f.dir + "/" + f.filename + "." + strconv.Itoa(int(f._suffix)))
	}
	os.Rename(f.dir+"/"+f.filename, f.dir+"/"+f.filename+"."+strconv.Itoa(int(f._suffix)))
	f.logfile, _ = os.Create(f.dir + "/" + f.filename)
	f.lg = log.New(logger.logObj.logfile, "", log.Ldate|log.Ltime|log.Lshortfile)
}

func fileSize(file string) int64 {
	fmt.Println("fileSize", file)
	f, e := os.Stat(file)
	if e != nil {
		fmt.Println(e.Error())
		return 0
	}
	return f.Size()
}

func isExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

func (logger *Logger) fileMonitor() {
	timer := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-timer.C:
			logger.fileCheck()
		}
	}
}

func (logger *Logger) fileCheck() {
	defer func() {
		if err := recover(); err != nil {
			log.Println(err)
		}
	}()

	if logger.logObj != nil && logger.logObj.isMustRename(logger) {
		logger.mu.Lock()
		defer logger.mu.Unlock()
		logger.logObj.rename(logger)
	}
}
