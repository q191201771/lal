package log

import "github.com/cihub/seelog"

var log seelog.LoggerInterface

func Initial(configFileName string) error {
	var err error
	log, err = seelog.LoggerFromConfigAsFile(configFileName)
	if err != nil {
		return err
	}
	err = log.SetAdditionalStackDepth(1)
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
