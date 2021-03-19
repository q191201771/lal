// Copyright 2021, Chef.  All rights reserved.
// https://github.com/cfeeling/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package logic

import (
	"github.com/cfeeling/lal/pkg/base"
	"github.com/cfeeling/lal/pkg/hls"
	"github.com/cfeeling/lal/pkg/httpflv"
	"github.com/cfeeling/lal/pkg/httpts"
	"github.com/cfeeling/lal/pkg/rtmp"
	"github.com/cfeeling/lal/pkg/rtsp"
)

var _ base.ISession = &rtmp.ServerSession{}
var _ base.ISession = &rtsp.PubSession{}
var _ base.ISession = &httpflv.SubSession{}
var _ base.ISession = &httpts.SubSession{}
var _ base.ISession = &rtsp.SubSession{}

var _ base.ISessionURLContext = &rtmp.ServerSession{}
var _ base.ISessionURLContext = &rtsp.PubSession{}
var _ base.ISessionURLContext = &httpflv.SubSession{}
var _ base.ISessionURLContext = &httpts.SubSession{}
var _ base.ISessionURLContext = &rtsp.SubSession{}

//var _ base.ISessionURLContext = &rtmp.PushSession{} //
//var _ base.ISessionURLContext = &rtmp.PullSession{}
//var _ base.ISessionURLContext = &rtsp.PushSession{}
//var _ base.ISessionURLContext = &rtsp.PullSession{}
//var _ base.ISessionURLContext = &httpflv.PullSession{}
//var _ base.ISessionURLContext = &rtmp.ClientSession{} //
//var _ base.ISessionURLContext = &rtsp.ClientCommandSession{}

var _ base.ISessionStat = &rtmp.PushSession{} //
var _ base.ISessionStat = &rtmp.PullSession{}
var _ base.ISessionStat = &rtmp.ServerSession{}
var _ base.ISessionStat = &rtsp.PushSession{}
var _ base.ISessionStat = &rtsp.PullSession{}
var _ base.ISessionStat = &rtsp.PubSession{}
var _ base.ISessionStat = &rtsp.SubSession{}
var _ base.ISessionStat = &httpflv.PullSession{}
var _ base.ISessionStat = &httpflv.SubSession{}
var _ base.ISessionStat = &httpts.SubSession{}
var _ base.ISessionStat = &rtmp.ClientSession{} //
var _ base.ISessionStat = &rtsp.BaseInSession{}
var _ base.ISessionStat = &rtsp.BaseOutSession{}
var _ base.ISessionStat = &rtsp.ServerCommandSession{}

var _ rtmp.ServerObserver = &ServerManager{}
var _ rtsp.ServerObserver = &ServerManager{}
var _ httpflv.ServerObserver = &ServerManager{}
var _ httpts.ServerObserver = &ServerManager{}

var _ HTTPAPIServerObserver = &ServerManager{}

var _ rtmp.PubSessionObserver = &Group{} //
var _ rtsp.PullSessionObserver = &Group{}
var _ rtsp.PubSessionObserver = &Group{}
var _ hls.MuxerObserver = &Group{}
var _ rtsp.BaseInSessionObserver = &Group{} //

var _ rtmp.ServerSessionObserver = &rtmp.Server{}
var _ rtmp.HandshakeClient = &rtmp.HandshakeClientSimple{}
var _ rtmp.HandshakeClient = &rtmp.HandshakeClientComplex{}

var _ rtsp.ServerCommandSessionObserver = &rtsp.Server{}
var _ rtsp.ClientCommandSessionObserver = &rtsp.PushSession{}
var _ rtsp.ClientCommandSessionObserver = &rtsp.PullSession{}
var _ rtsp.IInterleavedPacketWriter = &rtsp.PushSession{}
var _ rtsp.IInterleavedPacketWriter = &rtsp.PullSession{}
var _ rtsp.IInterleavedPacketWriter = &rtsp.PubSession{}
var _ rtsp.IInterleavedPacketWriter = &rtsp.SubSession{}
var _ rtsp.IInterleavedPacketWriter = &rtsp.ClientCommandSession{}
var _ rtsp.IInterleavedPacketWriter = &rtsp.ServerCommandSession{}

var _ hls.StreamerObserver = &hls.Muxer{}
