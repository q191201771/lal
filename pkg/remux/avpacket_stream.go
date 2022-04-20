// Copyright 2022, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package remux

import "github.com/q191201771/lal/pkg/base"

type AvPacketStreamVideoFormat int

const (
	AvPacketStreamVideoFormatUnknown AvPacketStreamVideoFormat = 0
	AvPacketStreamVideoFormatAvcc    AvPacketStreamVideoFormat = 1
	AvPacketStreamVideoFormatAnnexb  AvPacketStreamVideoFormat = 2
)

type AvPacketStreamOption struct {
	VideoFormat AvPacketStreamVideoFormat
}

var DefaultApsOption = AvPacketStreamOption{
	VideoFormat: AvPacketStreamVideoFormatAvcc,
}

type IAvPacketStream interface {
	WithOption(modOption func(option *AvPacketStreamOption))
	FeedAvPacket(packet base.AvPacket)
}
