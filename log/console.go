package log

import (
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

var _defaultLogger = logrus.New()

const defaultTimestampFormat = time.RFC3339

type Formatter struct {
	TimestampFormat string
}

// func (f *Formatter) Format(entry *logrus.Logger) ([]byte, error) {

// }

func DefaultLogger() *logrus.Logger {
	return _defaultLogger
}
func init() {
	_defaultLogger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339Nano,
	})
	_defaultLogger.SetOutput(os.Stdout)
	_defaultLogger.SetLevel(logrus.TraceLevel)
}

type Fields = logrus.Fields

var WithFields = _defaultLogger.WithFields
var WithContext = _defaultLogger.WithContext
var Traceln = _defaultLogger.Traceln
var Tracef = _defaultLogger.Tracef
var Debugf = _defaultLogger.Debugf
var Debugln = _defaultLogger.Debugln
var Printf = _defaultLogger.Printf
var Println = _defaultLogger.Println
var Infof = _defaultLogger.Infof
var Infoln = _defaultLogger.Infoln
var Warnf = _defaultLogger.Warnf
var Warnln = _defaultLogger.Warnln
var Errorf = _defaultLogger.Errorf
var Errorln = _defaultLogger.Errorln
var Panicf = _defaultLogger.Panicf
var Paincln = _defaultLogger.Panicln
var Fatalf = _defaultLogger.Fatalf
var Fatalln = _defaultLogger.Fatalln
