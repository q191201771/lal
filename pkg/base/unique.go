// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package base

import "github.com/q191201771/naza/pkg/unique"

const (
	UkPreRtmpServerSession        = "RTMPPUBSUB"
	UkPreRtmpPushSession          = "RTMPPUSH"
	UkPreRtmpPullSession          = "RTMPPULL"
	UkPreRtspServerCommandSession = "RTSPSRVCMD"
	UkPreRtspPubSession           = "RTSPPUB"
	UkPreRtspSubSession           = "RTSPSUB"
	UkPreRtspPushSession          = "RTSPPUSH"
	UkPreRtspPullSession          = "RTSPPULL"
	UkPreFlvSubSession            = "FLVSUB"
	UkPreTsSubSession             = "TSSUB"
	UkPreFlvPullSession           = "FLVPULL"

	UkPreGroup    = "GROUP"
	UkPreHlsMuxer = "HLSMUXER"
	UkPreStreamer = "STREAMER"
)

//func GenUk(prefix string) string {
//	return unique.GenUniqueKey(prefix)
//}

func GenUkRtmpServerSession() string {
	return siUkRtmpServerSession.GenUniqueKey()
}

func GenUkRtmpPushSession() string {
	return siUkRtmpPushSession.GenUniqueKey()
}

func GenUkRtmpPullSession() string {
	return siUkRtmpPullSession.GenUniqueKey()
}

func GenUkRtspServerCommandSession() string {
	return siUkRtspServerCommandSession.GenUniqueKey()
}

func GenUkRtspPubSession() string {
	return siUkRtspPubSession.GenUniqueKey()
}

func GenUkRtspSubSession() string {
	return siUkRtspSubSession.GenUniqueKey()
}

func GenUkRtspPushSession() string {
	return siUkRtspPushSession.GenUniqueKey()
}

func GenUkRtspPullSession() string {
	return siUkRtspPullSession.GenUniqueKey()
}

func GenUkFlvSubSession() string {
	return siUkFlvSubSession.GenUniqueKey()
}

func GenUkTsSubSession() string {
	return siUkTsSubSession.GenUniqueKey()
}

func GenUkFlvPullSession() string {
	return siUkFlvPullSession.GenUniqueKey()
}

func GenUkGroup() string {
	return siUkGroup.GenUniqueKey()
}

func GenUkHlsMuxer() string {
	return siUkHlsMuxer.GenUniqueKey()
}

func GenUkStreamer() string {
	return siUkStreamer.GenUniqueKey()
}

var (
	siUkRtmpServerSession        *unique.SingleGenerator
	siUkRtmpPushSession          *unique.SingleGenerator
	siUkRtmpPullSession          *unique.SingleGenerator
	siUkRtspServerCommandSession *unique.SingleGenerator
	siUkRtspPubSession           *unique.SingleGenerator
	siUkRtspSubSession           *unique.SingleGenerator
	siUkRtspPushSession          *unique.SingleGenerator
	siUkRtspPullSession          *unique.SingleGenerator
	siUkFlvSubSession            *unique.SingleGenerator
	siUkTsSubSession             *unique.SingleGenerator
	siUkFlvPullSession           *unique.SingleGenerator

	siUkGroup    *unique.SingleGenerator
	siUkHlsMuxer *unique.SingleGenerator
	siUkStreamer *unique.SingleGenerator
)

func init() {
	siUkRtmpServerSession = unique.NewSingleGenerator(UkPreRtmpServerSession)
	siUkRtmpPushSession = unique.NewSingleGenerator(UkPreRtmpPushSession)
	siUkRtmpPullSession = unique.NewSingleGenerator(UkPreRtmpPullSession)
	siUkRtspServerCommandSession = unique.NewSingleGenerator(UkPreRtspServerCommandSession)
	siUkRtspPubSession = unique.NewSingleGenerator(UkPreRtspPubSession)
	siUkRtspSubSession = unique.NewSingleGenerator(UkPreRtspSubSession)
	siUkRtspPushSession = unique.NewSingleGenerator(UkPreRtspPushSession)
	siUkRtspPullSession = unique.NewSingleGenerator(UkPreRtspPullSession)
	siUkFlvSubSession = unique.NewSingleGenerator(UkPreFlvSubSession)
	siUkTsSubSession = unique.NewSingleGenerator(UkPreTsSubSession)
	siUkFlvPullSession = unique.NewSingleGenerator(UkPreFlvPullSession)

	siUkGroup = unique.NewSingleGenerator(UkPreGroup)
	siUkHlsMuxer = unique.NewSingleGenerator(UkPreHlsMuxer)
	siUkStreamer = unique.NewSingleGenerator(UkPreStreamer)
}
