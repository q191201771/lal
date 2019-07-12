package log

import (
	"github.com/cihub/seelog"
	"os"
)

var log seelog.LoggerInterface

func Initial(configFileName string) error {
	var err error
	log, err = seelog.LoggerFromConfigAsFile(configFileName)
	if err != nil {
		return err
	}
	err = log.SetAdditionalStackDepth(2)
	return err
}

func Debugf(format string, params ...interface{}) {
	log.Debugf(format, params...)
}

func Infof(format string, params ...interface{}) {
	log.Infof(format, params...)
}

func Warnf(format string, params ...interface{}) {
	log.Warnf(format, params...)
}

func Errorf(format string, params ...interface{}) {
	log.Errorf(format, params...)
}

func Debug(v ...interface{}) {
	log.Debug(v...)
}

func Info(v ...interface{}) {
	log.Info(v...)
}

func Warn(v ...interface{}) {
	log.Warn(v...)
}

func Error(v ...interface{}) {
	log.Error(v...)
}

var defaultLogConfig = `
<seelog minlevel="trace">
<outputs formatid="file_common">
<filter levels="trace">
<console formatid="console_trace" />
</filter>
<filter levels="debug">
<console formatid="console_debug" />
</filter>
<filter levels="info">
<console formatid="console_info" />
</filter>
<filter levels="warn">
<console formatid="console_warn" />
</filter>
<filter levels="error">
<console formatid="console_error" />
</filter>
<filter levels="critical">
<console formatid="console_critical" />
</filter>
</outputs>
<formats>
<format id="file_common" format="%Date(2006-01-02 15:04:05.000) %LEV %Msg - %File:%Line%n" />
<format id="console_trace" format="%Date(2006-01-02 15:04:05.000) %EscM(37)%LEV%EscM(49)%EscM(0) %Msg - %File:%Line%n" />
<format id="console_debug" format="%Date(2006-01-02 15:04:05.000) %EscM(37)%LEV%EscM(49)%EscM(0) %Msg - %File:%Line%n" />
<format id="console_info" format="%Date(2006-01-02 15:04:05.000) %EscM(36)%LEV%EscM(49)%EscM(0) %Msg - %File:%Line%n" />
<format id="console_warn" format="%Date(2006-01-02 15:04:05.000) %EscM(33)%LEV%EscM(49)%EscM(0) %Msg - %File:%Line%n" />
<format id="console_error" format="%Date(2006-01-02 15:04:05.000) %EscM(31)%LEV%EscM(49)%EscM(0) %Msg - %File:%Line%n" />
<format id="console_critical" format="%Date(2006-01-02 15:04:05.000) %EscM(31)%LEV%EscM(49)%EscM(0) %Msg - %File:%Line%n" />
</formats>
</seelog>
`

func init() {
	var err error
	log, err = seelog.LoggerFromConfigAsString(defaultLogConfig)
	if err != nil {
		os.Exit(1)
	}
	err = log.SetAdditionalStackDepth(1)
	if err != nil {
		os.Exit(1)
	}
}
