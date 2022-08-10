// Copyright 2022, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package base

import (
	"encoding/hex"
	"fmt"
	"github.com/q191201771/naza/pkg/bele"
	"github.com/q191201771/naza/pkg/nazabytes"
	"os"
	"path/filepath"
	"time"
)

// TODO(chef): [refactor] move to naza 202208

type DumpFile struct {
	file *os.File
}

type DumpFileMessage struct {
	Ver       uint32
	Typ       uint32
	Len       uint32
	Timestamp uint32
	Body      []byte
}

func NewDumpFile() *DumpFile {
	return &DumpFile{}
}

func (d *DumpFile) OpenToWrite(filename string) (err error) {
	dir := filepath.Dir(filename)
	if err = os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	d.file, err = os.Create(filename)
	return
}

func (d *DumpFile) OpenToRead(filename string) (err error) {
	d.file, err = os.Open(filename)
	return
}

func (d *DumpFile) Write(b []byte) error {
	_, err := d.file.Write(d.pack(b))
	return err
}

func (d *DumpFile) ReadOneMessage() (m DumpFileMessage, err error) {
	m.Ver, err = bele.ReadBeUint32(d.file)
	if err != nil {
		return
	}
	m.Typ, err = bele.ReadBeUint32(d.file)
	if err != nil {
		return
	}
	m.Len, err = bele.ReadBeUint32(d.file)
	if err != nil {
		return
	}
	m.Timestamp, err = bele.ReadBeUint32(d.file)
	if err != nil {
		return
	}
	m.Body = make([]byte, m.Len)
	_, err = d.file.Read(m.Body)
	// TODO(chef): [opt] 检查Ver等值 202208
	// TODO(chef): [opt] check Read return value 202208
	return
}

func (d *DumpFile) Close() error {
	if d.file == nil {
		return nil
	}
	return d.file.Close()
}

// ---------------------------------------------------------------------------------------------------------------------

func (m *DumpFileMessage) DebugString() string {
	return fmt.Sprintf("ver: %d, typ: %d, len: %d, timestamp: %d, len: %d, hex: %s",
		m.Ver, m.Typ, m.Len, m.Timestamp, len(m.Body), hex.Dump(nazabytes.Prefix(m.Body, 16)))
}

// ---------------------------------------------------------------------------------------------------------------------

func (d *DumpFile) pack(b []byte) []byte {
	ret := make([]byte, len(b)+16)
	bele.BePutUint32(ret, 1)                  // Ver
	bele.BePutUint32(ret[4:], 1)              // Typ
	bele.BePutUint32(ret[8:], uint32(len(b))) // Len
	//bele.BePutUint32(ret[12:], 0) // Timestamp
	bele.BePutUint32(ret[12:], uint32(time.Now().Unix())) // Timestamp
	copy(ret[16:], b)
	return ret
}
