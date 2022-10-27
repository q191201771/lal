// Copyright 2022, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package logic

import (
	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/lal/pkg/remux"
	"github.com/q191201771/naza/pkg/nazaatomic"
)

type CustomizePubSessionContext struct {
	uniqueKey string

	streamName string
	remuxer    *remux.AvPacket2RtmpRemuxer
	onRtmpMsg  func(msg base.RtmpMsg)

	disposeFlag nazaatomic.Bool
}

func NewCustomizePubSessionContext(streamName string) *CustomizePubSessionContext {
	return &CustomizePubSessionContext{
		uniqueKey:  base.GenUkCustomizePubSession(),
		streamName: streamName,
		remuxer:    remux.NewAvPacket2RtmpRemuxer(),
	}
}

func (ctx *CustomizePubSessionContext) WithOnRtmpMsg(onRtmpMsg func(msg base.RtmpMsg)) *CustomizePubSessionContext {
	ctx.remuxer.WithOnRtmpMsg(onRtmpMsg)
	return ctx
}

func (ctx *CustomizePubSessionContext) UniqueKey() string {
	return ctx.uniqueKey
}

func (ctx *CustomizePubSessionContext) StreamName() string {
	return ctx.streamName
}

func (ctx *CustomizePubSessionContext) Dispose() {
	ctx.disposeFlag.Store(true)
}

// ---------------------------------------------------------------------------------------------------------------------

func (ctx *CustomizePubSessionContext) WithOption(modOption func(option *base.AvPacketStreamOption)) {
	ctx.remuxer.WithOption(modOption)
}

func (ctx *CustomizePubSessionContext) FeedAudioSpecificConfig(asc []byte) error {
	if ctx.disposeFlag.Load() {
		return base.ErrDisposedInStream
	}
	ctx.remuxer.InitWithAvConfig(asc, nil, nil, nil)
	return nil
}

func (ctx *CustomizePubSessionContext) FeedAvPacket(packet base.AvPacket) error {
	if ctx.disposeFlag.Load() {
		return base.ErrDisposedInStream
	}
	ctx.remuxer.FeedAvPacket(packet)
	return nil
}
