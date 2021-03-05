package log

import (
	"fmt"
	"os"
	"path"

	"github.com/go-logr/logr"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	k8szap "sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var Logger tpaasLogger

type tpaasLogger interface {
	logr.Logger
	Warn(msg string, keyAndValue ...interface{})
	Debug(msg string, keyAndValue ...interface{})
}

type logger struct {
	logr.Logger
}

func NewLogger(log logr.Logger) *logger {
	return &logger{Logger: log}
}

func (l *logger) Info(msg string, keyAndValue ...interface{}) {
	l.Logger.Info(msg, keyAndValue...)
}

func (l *logger) V(level int) logr.InfoLogger {
	return l.Logger.V(level)
}

func (l *logger) Debug(msg string, keyAndValue ...interface{}) {
	l.Logger.V(int(zapcore.DebugLevel)).Info(msg, keyAndValue...)
}

func (l *logger) Error(err error, msg string, keyAndValue ...interface{}) {
	l.Logger.Error(err, msg, keyAndValue...)
}

func (l *logger) Warn(msg string, keyAndValue ...interface{}) {
	l.Logger.V(int(zapcore.WarnLevel)).Info(msg, keyAndValue...)
}

func (l *logger) WithValues(keyAndValue ...interface{}) logr.Logger {
	return l.Logger.WithValues(keyAndValue...)
}

func (l *logger) WithName(name string) logr.Logger {
	return l.Logger.WithName(name)
}

//InitLog init system log config options zapr config options logFile the log file path
func InitLog(options *k8szap.Options, logFile string) {
	_, err := os.Stat(logFile)
	if os.IsNotExist(err) {
		dir := path.Dir(logFile)
		err = os.MkdirAll(dir, 0777)
		if err != nil {
			fmt.Println(err)
		}
	}
	w := zapcore.AddSync(&lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    50, // megabytes
		MaxBackups: 3,
		MaxAge:     7, // days
	})
	destWritter := zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), w)
	var opts []k8szap.Opts
	opts = append(opts, k8szap.WriteTo(destWritter))
	if options != nil {
		if options.Development {
			opts = append(opts, k8szap.UseDevMode(true))
		}

		if options.Level != nil {
			opts = append(opts, k8szap.Level(options.Level))
		}

		if options.StacktraceLevel != nil {
			opts = append(opts, k8szap.StacktraceLevel(options.StacktraceLevel))
		}

		if options.Encoder != nil {
			opts = append(opts, k8szap.Encoder(options.Encoder))
		}
	}
	localLogger := k8szap.New(opts...)
	Logger = NewLogger(localLogger)
}
