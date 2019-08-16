package log_test

import (
	"github.com/q191201771/lal/pkg/util/log"
	"testing"
)

func TestLogger(t *testing.T) {
	c := log.Config{
		Level:       log.LevelInfo,
		Filename:    "/tmp/lallogtest/aaa.log",
		IsToStdout:  true,
		RotateMByte: 10,
	}
	l, err := log.New(c)
	if err != nil {
		panic(err)
	}
	l.Debugf("test msg by Debug%s", "f")
	l.Infof("test msg by Info%s", "f")
	l.Warnf("test msg by Warn%s", "f")
	l.Errorf("test msg by Error%s", "f")
	l.Debug("test msg by Debug")
	l.Info("test msg by Info")
	l.Warn("test msg by Warn")
	l.Error("test msg by Error")
}

func TestGlobal(t *testing.T) {
	log.Debugf("test msg by Debug%s", "f")
	log.Infof("test msg by Info%s", "f")
	log.Warnf("test msg by Warn%s", "f")
	log.Errorf("test msg by Error%s", "f")
	log.Debug("test msg by Debug")
	log.Info("test msg by Info")
	log.Warn("test msg by Warn")
	log.Error("test msg by Error")

	c := log.Config{
		Level:       log.LevelInfo,
		Filename:    "/tmp/lallogtest/bbb.log",
		IsToStdout:  true,
		RotateMByte: 10,
	}
	err := log.Init(c)
	if err != nil {
		panic(err)
	}
	log.Debugf("test msg by Debug%s", "f")
	log.Infof("test msg by Info%s", "f")
	log.Warnf("test msg by Warn%s", "f")
	log.Errorf("test msg by Error%s", "f")
	log.Debug("test msg by Debug")
	log.Info("test msg by Info")
	log.Warn("test msg by Warn")
	log.Error("test msg by Error")
}

func BenchmarkStdout(b *testing.B) {
	c := log.Config{
		Level:       log.LevelInfo,
		Filename:    "/tmp/lallogtest/ccc.log",
		IsToStdout:  true,
		RotateMByte: 10,
	}
	err := log.Init(c)
	if err != nil {
		panic(err)
	}
	for i := 0; i < b.N; i++ {
		log.Infof("hello %s %d", "world", i)
	}
}
