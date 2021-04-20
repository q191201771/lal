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
	UKPreRTMPServerSession        = "RTMPPUBSUB"
	UKPreRTMPPushSession          = "RTMPPUSH"
	UKPreRTMPPullSession          = "RTMPPULL"
	UKPreRTSPServerCommandSession = "RTSPSRVCMD"
	UKPreRTSPPubSession           = "RTSPPUB"
	UKPreRTSPSubSession           = "RTSPSUB"
	UKPreRTSPPushSession          = "RTSPPUSH"
	UKPreRTSPPullSession          = "RTSPPULL"
	UKPreFLVSubSession            = "FLVSUB"
	UKPreTSSubSession             = "TSSUB"
	UKPreFLVPullSession           = "FLVPULL"

	UKPreGroup    = "GROUP"
	UKPreHLSMuxer = "HLSMUXER"
	UKPreStreamer = "STREAMER"
)

//func GenUK(prefix string) string {
//	return unique.GenUniqueKey(prefix)
//}

func GenUKRTMPServerSession() string {
	return siUKRTMPServerSession.GenUniqueKey()
}

func GenUKRTMPPushSession() string {
	return siUKRTMPPushSession.GenUniqueKey()
}

func GenUKRTMPPullSession() string {
	return siUKRTMPPullSession.GenUniqueKey()
}

func GenUKRTSPServerCommandSession() string {
	return siUKRTSPServerCommandSession.GenUniqueKey()
}

func GenUKRTSPPubSession() string {
	return siUKRTSPPubSession.GenUniqueKey()
}

func GenUKRTSPSubSession() string {
	return siUKRTSPSubSession.GenUniqueKey()
}

func GenUKRTSPPushSession() string {
	return siUKRTSPPushSession.GenUniqueKey()
}

func GenUKRTSPPullSession() string {
	return siUKRTSPPullSession.GenUniqueKey()
}

func GenUKFLVSubSession() string {
	return siUKFLVSubSession.GenUniqueKey()
}

func GenUKTSSubSession() string {
	return siUKTSSubSession.GenUniqueKey()
}

func GenUKFLVPullSession() string {
	return siUKFLVPullSession.GenUniqueKey()
}

func GenUKGroup() string {
	return siUKGroup.GenUniqueKey()
}

func GenUKHLSMuxer() string {
	return siUKHLSMuxer.GenUniqueKey()
}

func GenUKStreamer() string {
	return siUKStreamer.GenUniqueKey()
}

var (
	siUKRTMPServerSession        *unique.SingleGenerator
	siUKRTMPPushSession          *unique.SingleGenerator
	siUKRTMPPullSession          *unique.SingleGenerator
	siUKRTSPServerCommandSession *unique.SingleGenerator
	siUKRTSPPubSession           *unique.SingleGenerator
	siUKRTSPSubSession           *unique.SingleGenerator
	siUKRTSPPushSession          *unique.SingleGenerator
	siUKRTSPPullSession          *unique.SingleGenerator
	siUKFLVSubSession            *unique.SingleGenerator
	siUKTSSubSession             *unique.SingleGenerator
	siUKFLVPullSession           *unique.SingleGenerator

	siUKGroup    *unique.SingleGenerator
	siUKHLSMuxer *unique.SingleGenerator
	siUKStreamer *unique.SingleGenerator
)

func init() {
	siUKRTMPServerSession = unique.NewSingleGenerator(UKPreRTMPServerSession)
	siUKRTMPPushSession = unique.NewSingleGenerator(UKPreRTMPPushSession)
	siUKRTMPPullSession = unique.NewSingleGenerator(UKPreRTMPPullSession)
	siUKRTSPServerCommandSession = unique.NewSingleGenerator(UKPreRTSPServerCommandSession)
	siUKRTSPPubSession = unique.NewSingleGenerator(UKPreRTSPPubSession)
	siUKRTSPSubSession = unique.NewSingleGenerator(UKPreRTSPSubSession)
	siUKRTSPPushSession = unique.NewSingleGenerator(UKPreRTSPPushSession)
	siUKRTSPPullSession = unique.NewSingleGenerator(UKPreRTSPPullSession)
	siUKFLVSubSession = unique.NewSingleGenerator(UKPreFLVSubSession)
	siUKTSSubSession = unique.NewSingleGenerator(UKPreTSSubSession)
	siUKFLVPullSession = unique.NewSingleGenerator(UKPreFLVPullSession)

	siUKGroup = unique.NewSingleGenerator(UKPreGroup)
	siUKHLSMuxer = unique.NewSingleGenerator(UKPreHLSMuxer)
	siUKStreamer = unique.NewSingleGenerator(UKPreStreamer)
}
