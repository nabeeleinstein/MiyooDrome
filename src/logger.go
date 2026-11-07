package main

/*
#include <stdlib.h>
*/
import "C"
import (
	"os"
	"time"
)

const LOG_PATH = "./miyoodrome.log"

var logFile *os.File
var globalApp *MiyooPod // Global reference for checking LocalLogsEnabled

func init() {
	var err error
	logFile, err = os.OpenFile(LOG_PATH, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
}

func logMsg(message string) {
	if globalApp != nil && globalApp.LocalLogsEnabled {
		logFile.WriteString(time.Now().Format("2006-01-02 15:04:05.999") + " - " + message + "\n")
	}
}

//export GoLogMsg
func GoLogMsg(cMsg *C.char) {
	msg := C.GoString(cMsg)
	logMsg(msg)
}
