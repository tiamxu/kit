package log

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

var _defaultLogger = logrus.New()

const defaultTimestampFormat = time.RFC3339

type Formatter struct {
	TimestampFormat string
}

func (f *Formatter) Format(entry *logrus.Entry) ([]byte, error) {
	data := make(Fields)
	for k, v := range entry.Data {
		data[k] = v
	}
	var b *bytes.Buffer
	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}

	timestampFormat := f.TimestampFormat
	if timestampFormat == "" {
		timestampFormat = defaultTimestampFormat
	}

	var (
		levelStr = strings.ToUpper(entry.Level.String())
		timeStr  = entry.Time.Format(timestampFormat)
	)

	_, err := fmt.Fprintf(b, "%s %s %s\n", levelStr, timeStr, entry.Message)
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func DefaultLogger() *logrus.Logger {
	return _defaultLogger
}
func init() {
	_defaultLogger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339Nano,
	})
	_defaultLogger.SetOutput(os.Stdout)
	_defaultLogger.SetLevel(logrus.TraceLevel)
	_defaultLogger.SetReportCaller(true)
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
