package log

import (
	"testing"
	"github.com/q191201771/lal/pkg/util/assert"
)

func TestLogger(t *testing.T) {
	c := Config{
		Level:       LevelInfo,
		Filename:    "/tmp/lallogtest/aaa.log",
		IsToStdout:  true,
		RotateMByte: 10,
	}
	l, err := New(c)
	assert.Equal(t, nil, err)
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
	Debugf("test msg by Debug%s", "f")
	Infof("test msg by Info%s", "f")
	Warnf("test msg by Warn%s", "f")
	Errorf("test msg by Error%s", "f")
	Debug("test msg by Debug")
	Info("test msg by Info")
	Warn("test msg by Warn")
	Error("test msg by Error")

	c := Config{
		Level:       LevelInfo,
		Filename:    "/tmp/lallogtest/bbb.log",
		IsToStdout:  true,
		RotateMByte: 10,
	}
	err := Init(c)
	assert.Equal(t, nil, err)
	Debugf("test msg by Debug%s", "f")
	Infof("test msg by Info%s", "f")
	Warnf("test msg by Warn%s", "f")
	Errorf("test msg by Error%s", "f")
	Debug("test msg by Debug")
	Info("test msg by Info")
	Warn("test msg by Warn")
	Error("test msg by Error")
	Output(LevelInfo, 3, "test msg by Output")
	Outputf(LevelInfo, 3, "test msg by Output%s", "f")
}

func TestNew(t *testing.T) {
	l, err := New(Config{Level:LevelError+1})
	assert.Equal(t, nil, l)
	assert.Equal(t, logErr, err)
}

func TestRotate(t *testing.T) {
	c := Config{
		Level:       LevelInfo,
		Filename:    "/tmp/lallogtest/ccc.log",
		IsToStdout:  false,
		RotateMByte: 1,
	}
	err := Init(c)
	assert.Equal(t, nil, err)
	b := make([]byte, 1024)
	for i := 0; i < 2 * 1024; i++ {
		Info(b)
	}
	for i := 0; i < 2 * 1024; i++ {
		Infof("%+v", b)
	}
}

func BenchmarkStdout(b *testing.B) {
	c := Config{
		Level:       LevelInfo,
		Filename:    "/tmp/lallogtest/ccc.log",
		IsToStdout:  true,
		RotateMByte: 10,
	}
	err := Init(c)
	assert.Equal(b, nil, err)
	for i := 0; i < b.N; i++ {
		Infof("hello %s %d", "world", i)
	}
}
