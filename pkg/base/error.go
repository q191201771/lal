// Copyright 2021, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package base

import (
	"errors"
	"fmt"
)

// ----- 通用的 ---------------------------------------------------------------------------------------------------------

var (
	ErrShortBuffer  = errors.New("lal: buffer too short")
	ErrFileNotExist = errors.New("lal: file not exist")
)

// ----- pkg/aac -------------------------------------------------------------------------------------------------------

var ErrSamplingFrequencyIndex = errors.New("lal.aac: invalid sampling frequency index")

// ----- pkg/aac -------------------------------------------------------------------------------------------------------

var ErrAvc = errors.New("lal.avc: fxxk")

// ----- pkg/base ------------------------------------------------------------------------------------------------------

var (
	ErrAddrEmpty               = errors.New("lal.base: http server addr empty")
	ErrMultiRegisterForPattern = errors.New("lal.base: http server multiple registrations for pattern")

	ErrSessionNotStarted = errors.New("lal.base: session has not been started yet")

	ErrInvalidUrl = errors.New("lal.base: invalid url")
)

// ----- pkg/hevc ------------------------------------------------------------------------------------------------------

var ErrHevc = errors.New("lal.hevc: fxxk")

// ----- pkg/hls -------------------------------------------------------------------------------------------------------

var ErrHls = errors.New("lal.hls: fxxk")

// ----- pkg/rtmp ------------------------------------------------------------------------------------------------------

var (
	ErrAmfInvalidType = errors.New("lal.rtmp: invalid amf0 type")
	ErrAmfTooShort    = errors.New("lal.rtmp: too short to unmarshal amf0 data")
	ErrAmfNotExist    = errors.New("lal.rtmp: not exist")

	ErrRtmpShortBuffer   = errors.New("lal.rtmp: buffer too short")
	ErrRtmpUnexpectedMsg = errors.New("lal.rtmp: unexpected msg")
)

// TODO(chef): refactor 整理其他pkg的error

func NewErrAmfInvalidType(b byte) error {
	return fmt.Errorf("%w. b=%d", ErrAmfInvalidType, b)
}

func NewErrRtmpShortBuffer(need, actual int, msg string) error {
	return fmt.Errorf("%w. need=%d, actual=%d, msg=%s", ErrRtmpShortBuffer, need, actual, msg)
}
