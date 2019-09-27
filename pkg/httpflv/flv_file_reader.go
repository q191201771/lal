package httpflv

import (
	"os"
)

type FlvFileReader struct {
	fp *os.File
}

func (ffr *FlvFileReader) Open(filename string) (err error) {
	ffr.fp, err = os.Open(filename)
	return
}

func (ffr *FlvFileReader) ReadFlvHeader() ([]byte, error) {
	flvHeader := make([]byte, flvHeaderSize)
	_, err := ffr.fp.Read(flvHeader)
	return flvHeader, err
}

// TODO chef: 返回 Tag 类型，对比 bench
func (ffr *FlvFileReader) ReadTag() (*Tag, error) {
	return readTag(ffr.fp)
}

func (ffr *FlvFileReader) Dispose() {
	if ffr.fp != nil {
		_ = ffr.fp.Close()
	}
}
