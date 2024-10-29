package log

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

var config Config
var fileWriter *os.File
var fileSize int64
var mutex sync.Mutex

func rotateLogFile() {
	now := time.Now()
	newFileName := filepath.Join(config.LogFilePath, fmt.Sprintf("%s_%s.log", now.Format("20060102150405"), now.Format("01-02-2006")))
	if fileWriter != nil {
		fileWriter.Close()
	}
	fileWriter, _ = os.OpenFile(newFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	fileSize = 0
}

func writeToFile(entry *logrus.Entry) error {
	mutex.Lock()
	defer mutex.Unlock()
	if fileWriter == nil || fileSize > config.MaxSize {
		rotateLogFile()
	}
	message, err := json.Marshal(entry.Data)
	if err != nil {
		return err
	}
	n, err := fileWriter.Write(message)
	if err == nil {
		fileSize += int64(n)
	}
	return err
}
