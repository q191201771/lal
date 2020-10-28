package hls

import (
	"fmt"
	"github.com/q191201771/lal/pkg/mpegts"
	"github.com/q191201771/naza/pkg/nazalog"
	"os"
)

type TsRecorder struct {
	fp            *os.File
	opened        bool
	firstKeyFrame bool
	outPath       string
	streamName    string
	UniqueKey     string
}

func NewTsRecorder(outPath string, streamName string, uk string) *TsRecorder {
	return &TsRecorder{
		fp:            nil,
		opened:        false,
		firstKeyFrame: false,
		outPath:       outPath,
		streamName:    streamName,
		UniqueKey:     uk,
	}
}

func (t *TsRecorder) FeedTsPacket(rawFrame []byte, boundary bool) {
	var err error
	if !t.opened {
		if err = t.openFile(); err != nil {
			nazalog.Errorf("[%s] open file fail. err=%+v", t.UniqueKey, err)
			return
		}

		nazalog.Infof("[%s] open file to write", t.UniqueKey)
		if _, err = t.fp.Write(mpegts.FixedFragmentHeader); err != nil {
			nazalog.Errorf("[%s]write fixed fragment header fail. err=%+v ", t.UniqueKey, err)
			return
		}
	}

	if t.firstKeyFrame {
		if _, err = t.fp.Write(rawFrame); err != nil {
			nazalog.Errorf("[%s]write ts raw frame fail, err =%+v ", t.UniqueKey, err)
			return
		}
	} else if boundary {
		if _, err = t.fp.Write(rawFrame); err != nil {
			nazalog.Error("write ts raw frame fail, err = ", err.Error())
			return
		}
		t.firstKeyFrame = true
	}

	return
}

func (t *TsRecorder) openFile() (err error) {

	filePathPre := fmt.Sprintf("%s%s", t.outPath, t.streamName)
	t.fp, err = os.OpenFile(fmt.Sprintf("%s.ts", filePathPre), os.O_WRONLY|os.O_EXCL|os.O_CREATE, 0666)
	var i int
	for i = 1; os.IsExist(err) == true && 0 < i; i++ {
		t.fp, err = os.OpenFile(fmt.Sprintf("%s-%d.ts", filePathPre, i), os.O_WRONLY|os.O_EXCL|os.O_CREATE, 0666)
	}

	if err != nil {
		return
	}

	if i <= 0 {
		err = fmt.Errorf("try open file %s-0 ~ %s-int_max  fail", filePathPre, filePathPre)
	} else {
		t.opened = true
	}

	return err
}

func (t *TsRecorder) Dispose() {
	if t.fp == nil {
		return
	}

	if err := t.fp.Close(); err != nil {
		nazalog.Error("open file fail ")
	}
}
