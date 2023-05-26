//	Copyright 2023 Dremio Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// simplelog package provides a simple logger
package simplelog

import (
	"fmt"
	"io"
	"log"
	"os"
	"path"
)

const (
	LevelError = iota
	LevelWarning
	LevelInfo
	LevelDebug
)

var LOGGER *Logger
var ddcLog *os.File

type Logger struct {
	debugLogger   *log.Logger
	infoLogger    *log.Logger
	warningLogger *log.Logger
	errorLogger   *log.Logger
}

func init() {
	LOGGER = NewLogger(LevelError)
}

func InitLogger(level int) {
	if level > 3 {
		LOGGER = NewLogger(LevelDebug)
	} else {
		LOGGER = NewLogger(level)
	}
}

func Close() error {
	if err := ddcLog.Close(); err != nil {
		return fmt.Errorf("unable to close ddc.log with error %v", err)
	}
	return nil
}

func NewLogger(level int) *Logger {
	debugOut, infoOut, warningOut, errorOut := io.Discard, io.Discard, io.Discard, io.Discard
	ddcLoc, err := os.Executable()
	if err != nil {
		log.Fatalf("unable to to find ddc cannot copy it to hosts due to error '%v'", err)
	}
	ddcLog, err = os.OpenFile(path.Join(path.Dir(ddcLoc), "ddc.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		log.Fatal(err)
	}
	adjustedLevel := level
	if adjustedLevel > 3 {
		adjustedLevel = 3
	}
	switch adjustedLevel {
	case LevelDebug:
		debugOut = io.MultiWriter(os.Stdout, ddcLog)
		fallthrough
	case LevelInfo:
		infoOut = io.MultiWriter(os.Stdout, ddcLog)
		fallthrough
	case LevelWarning:
		warningOut = io.MultiWriter(os.Stdout, ddcLog)
		fallthrough
	case LevelError:
		errorOut = io.MultiWriter(os.Stdout, ddcLog)
	}

	return &Logger{
		debugLogger:   log.New(debugOut, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile),
		infoLogger:    log.New(infoOut, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile),
		warningLogger: log.New(warningOut, "WARNING: ", log.Ldate|log.Ltime|log.Lshortfile),
		errorLogger:   log.New(errorOut, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile),
	}
}

func (l *Logger) Debug(format string) {
	handleLogError(l.debugLogger.Output(2, format))
}

func (l *Logger) Info(format string) {
	handleLogError(l.infoLogger.Output(2, format))
}

func (l *Logger) Warning(format string) {
	handleLogError(l.warningLogger.Output(2, format))
}

func (l *Logger) Error(format string) {
	handleLogError(l.errorLogger.Output(2, format))
}

func (l *Logger) Debugf(format string, v ...interface{}) {
	handleLogError(l.debugLogger.Output(2, fmt.Sprintf(format, v...)))
}

func (l *Logger) Infof(format string, v ...interface{}) {
	handleLogError(l.infoLogger.Output(2, fmt.Sprintf(format, v...)))
}

func (l *Logger) Warningf(format string, v ...interface{}) {
	handleLogError(l.warningLogger.Output(2, fmt.Sprintf(format, v...)))
}

func (l *Logger) Errorf(format string, v ...interface{}) {
	handleLogError(l.errorLogger.Output(2, fmt.Sprintf(format, v...)))
}

// package functions

func Debug(format string) {
	handleLogError(LOGGER.debugLogger.Output(2, format))
}

func Info(format string) {
	handleLogError(LOGGER.infoLogger.Output(2, format))
}

func Warning(format string) {
	handleLogError(LOGGER.warningLogger.Output(2, format))
}

func Error(format string) {
	handleLogError(LOGGER.errorLogger.Output(2, format))
}

func Debugf(format string, v ...interface{}) {
	handleLogError(LOGGER.debugLogger.Output(2, fmt.Sprintf(format, v...)))
}

func Infof(format string, v ...interface{}) {
	handleLogError(LOGGER.infoLogger.Output(2, fmt.Sprintf(format, v...)))
}

func Warningf(format string, v ...interface{}) {
	handleLogError(LOGGER.warningLogger.Output(2, fmt.Sprintf(format, v...)))
}

func Errorf(format string, v ...interface{}) {
	handleLogError(LOGGER.errorLogger.Output(2, fmt.Sprintf(format, v...)))
}

func handleLogError(err error) {
	if err != nil {
		log.Fatalf("critical error and therefore there is no log output due to error '%v'", err)
	}
}
