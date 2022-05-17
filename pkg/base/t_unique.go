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
	UkPreCustomizePubSessionContext = "CUSTOMIZEPUB"
	UkPreRtmpServerSession          = "RTMPPUBSUB" // 两种可能，pub或者sub
	UkPreRtmpPushSession            = "RTMPPUSH"
	UkPreRtmpPullSession            = "RTMPPULL"
	UkPreRtspPubSession             = "RTSPPUB"
	UkPreRtspSubSession             = "RTSPSUB"
	UkPreRtspPushSession            = "RTSPPUSH"
	UkPreRtspPullSession            = "RTSPPULL"
	UkPreFlvSubSession              = "FLVSUB"
	UkPreFlvPullSession             = "FLVPULL"
	UkPreTsSubSession               = "TSSUB"

	UkPreRtspServerCommandSession = "RTSPSRVCMD" // 这个不暴露给上层

	UkPreGroup              = "GROUP"
	UkPreHlsMuxer           = "HLSMUXER"
	UkPreRtmp2MpegtsRemuxer = "RTMP2MPEGTS"
)

//func GenUk(prefix string) string {
//	return unique.GenUniqueKey(prefix)
//}

func GenUkCustomizePubSession() string {
	return siUkCustomizePubSession.GenUniqueKey()
}

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

func GenUkRtmp2MpegtsRemuxer() string {
	return siUkRtmp2MpegtsRemuxer.GenUniqueKey()
}

var (
	siUkCustomizePubSession      *unique.SingleGenerator
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

	siUkGroup              *unique.SingleGenerator
	siUkHlsMuxer           *unique.SingleGenerator
	siUkRtmp2MpegtsRemuxer *unique.SingleGenerator
)

func init() {
	siUkCustomizePubSession = unique.NewSingleGenerator(UkPreCustomizePubSessionContext)
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
	siUkRtmp2MpegtsRemuxer = unique.NewSingleGenerator(UkPreRtmp2MpegtsRemuxer)
}
