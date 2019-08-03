package httpflv

import "os"

type FlvFileWriter struct {
	fp *os.File
}

func (ffw *FlvFileWriter) Open(filename string) (err error) {
	ffw.fp, err = os.Create(filename)
	return
}

func (ffw *FlvFileWriter) WriteRaw(b []byte) (err error) {
	_, err = ffw.fp.Write(b)
	return
}

func (ffw *FlvFileWriter) WriteTag(tag *Tag) (err error) {
	_, err = ffw.fp.Write(tag.Raw)
	return
}

func (ffw *FlvFileWriter) Dispose() {
	if ffw.fp != nil {
		_ = ffw.fp.Close()
	}
}
