package httpflv

import "os"

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

func (ffr *FlvFileReader) ReadTag() (*Tag, error) {
	var err error
	h, rawHeader, err := readTagHeader(ffr.fp)
	if err != nil {
		return nil, err
	}
	needed := int(h.DataSize) + prevTagFieldSize
	tag := &Tag{}
	tag.Header = h
	tag.Raw = make([]byte, TagHeaderSize+needed)
	copy(tag.Raw, rawHeader)

	_, err = ffr.fp.Read(tag.Raw[TagHeaderSize:])
	if err != nil {
		return nil, err
	}
	return tag, nil
}

func (ffr *FlvFileReader) Dispose() {
	if ffr.fp != nil {
		_ = ffr.fp.Close()
	}
}
