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
	_VER string = "1.0.0"
)

type LEVEL int32

type Logger struct {
	logLevel        LEVEL
	maxFileSize     int64
	maxFileCount    int32
	dailyRolling    bool
	consoleAppender bool
	RollingFile     bool
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
	mu       *sync.RWMutex
	logfile  *os.File
	lg       *log.Logger
}

var defaultLogger = Logger{logLevel: 1, dailyRolling: true, consoleAppender: true}

func SetConsole(isConsole bool) {
	defaultLogger.consoleAppender = isConsole
}

func SetLevel(_level LEVEL) {
	defaultLogger.logLevel = _level
}

func (logger *Logger) SetRollingFile(fileDir, fileName string, maxNumber int32, maxSize int64, _unit UNIT) {
	logger.maxFileCount = maxNumber
	logger.maxFileSize = maxSize * int64(_unit)
	logger.RollingFile = true
	logger.dailyRolling = false
	logger.logObj = &_FILE{dir: fileDir, filename: fileName, isCover: false, mu: new(sync.RWMutex)}
	logger.logObj.mu.Lock()
	defer logger.logObj.mu.Unlock()
	for i := 1; i <= int(maxNumber); i++ {
		if isExist(fileDir + "/" + fileName + "." + strconv.Itoa(i)) {
			logger.logObj._suffix = i
		} else {
			break
		}
	}
	if !logger.logObj.isMustRename(logger) {
		defaultLogger.logObj.logfile, _ = os.OpenFile(fileDir+"/"+fileName, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0)
		logger.logObj.lg = log.New(logger.logObj.logfile, "\n", log.Ldate|log.Ltime|log.Lshortfile)
	} else {
		logger.logObj.rename(logger)
	}
	go logger.fileMonitor()
}

func SetRollingFile(fileDir, fileName string, maxNumber int32, maxSize int64, _unit UNIT) {
	(&defaultLogger).SetRollingFile(fileDir, fileName, maxNumber, maxSize, _unit)
}

func SetRollingDaily(fileDir, fileName string) {
	(&defaultLogger).SetRollingDaily(fileDir, fileName)
}

func (logger *Logger) SetRollingDaily(fileDir, fileName string) {
	logger.RollingFile = false
	logger.dailyRolling = true
	t, _ := time.Parse(DATEFORMAT, time.Now().Format(DATEFORMAT))
	logger.logObj = &_FILE{dir: fileDir, filename: fileName, _date: &t, isCover: false, mu: new(sync.RWMutex)}
	logger.logObj.mu.Lock()
	defer logger.logObj.mu.Unlock()

	if !logger.logObj.isMustRename(logger) {
		logger.logObj.logfile, _ = os.OpenFile(fileDir+"/"+fileName, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0)
		logger.logObj.lg = log.New(logger.logObj.logfile, "\n", log.Ldate|log.Ltime|log.Lshortfile)
	} else {
		logger.logObj.rename(logger)
	}
}

func (logger *Logger) console(s ...interface{}) {
	if logger.consoleAppender {
		_, file, line, _ := runtime.Caller(2)
		short := file
		for i := len(file) - 1; i > 0; i-- {
			if file[i] == '/' {
				short = file[i+1:]
				break
			}
		}
		file = short
		log.Println(file+":"+strconv.Itoa(line), s)
	}
}

func catchError() {
	if err := recover(); err != nil {
		log.Println("err", err)
	}
}

func Debug(v ...interface{}) {
	(&defaultLogger).Debug(v...)
}

func (logger *Logger) Debug(v ...interface{}) {
	if logger.dailyRolling {
		logger.fileCheck()
	}
	defer catchError()
	logger.logObj.mu.RLock()
	defer logger.logObj.mu.RUnlock()

	if logger.logLevel <= DEBUG {
		logger.logObj.lg.Output(2, fmt.Sprintln("debug", v))
		logger.console("debug", v)
	}
}

func Info(v ...interface{}) {
	(&defaultLogger).Info(v)
}

func (logger *Logger) Info(v ...interface{}) {
	if logger.dailyRolling {
		logger.fileCheck()
	}
	defer catchError()
	logger.logObj.mu.RLock()
	defer logger.logObj.mu.RUnlock()
	if logger.logLevel <= INFO {
		logger.logObj.lg.Output(2, fmt.Sprintln("info", v))
		logger.console("info", v)
	}
}
func (logger *Logger) Warn(v ...interface{}) {
	if logger.dailyRolling {
		logger.fileCheck()
	}
	defer catchError()
	logger.logObj.mu.RLock()
	defer logger.logObj.mu.RUnlock()
	if logger.logLevel <= WARN {
		logger.logObj.lg.Output(2, fmt.Sprintln("warn", v))
		logger.console("warn", v)
	}
}

func Error(v ...interface{}) {
	(&defaultLogger).Error(v...)
}

func (logger *Logger) Error(v ...interface{}) {
	if logger.dailyRolling {
		logger.fileCheck()
	}
	defer catchError()
	logger.logObj.mu.RLock()
	defer logger.logObj.mu.RUnlock()
	if logger.logLevel <= ERROR {
		logger.logObj.lg.Output(2, fmt.Sprintln("error", v))
		logger.console("error", v)
	}
}

func Fatal(v ...interface{}) {
	(&defaultLogger).Fatal(v...)
}

func (logger *Logger) Fatal(v ...interface{}) {
	if logger.dailyRolling {
		logger.fileCheck()
	}
	defer catchError()
	logger.logObj.mu.RLock()
	defer logger.logObj.mu.RUnlock()
	if logger.logLevel <= FATAL {
		logger.logObj.lg.Output(2, fmt.Sprintln("fatal", v))
		logger.console("fatal", v)
	}
}

func (f *_FILE) isMustRename(logger *Logger) bool {
	if logger.dailyRolling {
		t, _ := time.Parse(DATEFORMAT, time.Now().Format(DATEFORMAT))
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
			f.lg = log.New(logger.logObj.logfile, "\n", log.Ldate|log.Ltime|log.Lshortfile)
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
	f.lg = log.New(logger.logObj.logfile, "\n", log.Ldate|log.Ltime|log.Lshortfile)
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
		logger.logObj.mu.Lock()
		defer logger.logObj.mu.Unlock()
		logger.logObj.rename(logger)
	}
}
