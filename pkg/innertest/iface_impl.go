// Copyright 2021, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package innertest

import (
	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/lal/pkg/hls"
	"github.com/q191201771/lal/pkg/httpflv"
	"github.com/q191201771/lal/pkg/httpts"
	"github.com/q191201771/lal/pkg/logic"
	"github.com/q191201771/lal/pkg/remux"
	"github.com/q191201771/lal/pkg/rtmp"
	"github.com/q191201771/lal/pkg/rtsp"
	"github.com/q191201771/naza/pkg/connection"
)

// TODO(chef): 检查所有 interface是否以I开头 202207
// TODO(chef): 增加 gb28181.PubSession 202207

var (
	_ base.ISession = &rtmp.ServerSession{}
	_ base.ISession = &rtsp.PubSession{}
	_ base.ISession = &rtsp.SubSession{}
	_ base.ISession = &httpflv.SubSession{}
	_ base.ISession = &httpts.SubSession{}

	_ base.ISession = &rtmp.PushSession{}
	_ base.ISession = &rtmp.PullSession{}
	_ base.ISession = &rtsp.PushSession{}
	_ base.ISession = &rtsp.PullSession{}
	_ base.ISession = &httpflv.PullSession{}
)

// IClientSession: 所有Client Session都满足
var (
	_ base.IClientSession = &rtmp.PushSession{}
	_ base.IClientSession = &rtmp.PullSession{}
	_ base.IClientSession = &rtsp.PushSession{}
	_ base.IClientSession = &rtsp.PullSession{}
	_ base.IClientSession = &httpflv.PullSession{}
)

// IServerSession
var (
	_ base.IServerSession = &rtmp.ServerSession{}
	_ base.IServerSession = &rtsp.PubSession{}
	_ base.IServerSession = &rtsp.SubSession{}
	_ base.IServerSession = &httpflv.SubSession{}
	_ base.IServerSession = &httpts.SubSession{}
)

// IClientSessionLifecycle: 所有Client Session都满足
var (
	// client
	_ base.IClientSessionLifecycle = &rtmp.PushSession{}
	_ base.IClientSessionLifecycle = &rtmp.PullSession{}
	_ base.IClientSessionLifecycle = &rtsp.PushSession{}
	_ base.IClientSessionLifecycle = &rtsp.PullSession{}
	_ base.IClientSessionLifecycle = &httpflv.PullSession{}

	// other
	_ base.IClientSessionLifecycle = &rtmp.ClientSession{}
	_ base.IClientSessionLifecycle = &rtsp.ClientCommandSession{}
)

// IServerSessionLifecycle
var (
	// server session
	_ base.IServerSessionLifecycle = &rtmp.ServerSession{}
	_ base.IServerSessionLifecycle = &rtsp.PubSession{}
	_ base.IServerSessionLifecycle = &rtsp.SubSession{}
	_ base.IServerSessionLifecycle = &httpflv.SubSession{}
	_ base.IServerSessionLifecycle = &httpts.SubSession{}

	// other
	_ base.IServerSessionLifecycle = &base.BasicHttpSubSession{}
	_ base.IServerSessionLifecycle = &rtsp.ServerCommandSession{}
)

// ISessionStat: 所有Session(client/server)都满足
var (
	// client
	_ base.ISessionStat = &rtmp.PushSession{}
	_ base.ISessionStat = &rtsp.PushSession{}
	_ base.ISessionStat = &rtmp.PullSession{}
	_ base.ISessionStat = &rtsp.PullSession{}
	_ base.ISessionStat = &httpflv.PullSession{}
	// server session
	_ base.ISessionStat = &rtmp.ServerSession{}
	_ base.ISessionStat = &rtsp.PubSession{}
	_ base.ISessionStat = &rtsp.SubSession{}
	_ base.ISessionStat = &httpflv.SubSession{}
	_ base.ISessionStat = &httpts.SubSession{}
	// other
	_ base.ISessionStat = &base.BasicHttpSubSession{}
	_ base.ISessionStat = &rtmp.ClientSession{}
	_ base.ISessionStat = &rtsp.BaseInSession{}
	_ base.ISessionStat = &rtsp.BaseOutSession{}
	_ base.ISessionStat = &rtsp.ServerCommandSession{}
)

// ISessionUrlContext: 所有Session(client/server)都满足
var (
	// client
	_ base.ISessionUrlContext = &rtmp.PushSession{}
	_ base.ISessionUrlContext = &rtsp.PushSession{}
	_ base.ISessionUrlContext = &rtmp.PullSession{}
	_ base.ISessionUrlContext = &rtsp.PullSession{}
	_ base.ISessionUrlContext = &httpflv.PullSession{}
	// server session
	_ base.ISessionUrlContext = &rtmp.ServerSession{}
	_ base.ISessionUrlContext = &rtsp.PubSession{}
	_ base.ISessionUrlContext = &rtsp.SubSession{}
	_ base.ISessionUrlContext = &httpflv.SubSession{}
	_ base.ISessionUrlContext = &httpts.SubSession{}
	// other
	_ base.ISessionUrlContext = &base.BasicHttpSubSession{}
	_ base.ISessionUrlContext = &rtmp.ClientSession{}
	_ base.ISessionUrlContext = &rtsp.ClientCommandSession{}
)

// IObject: 所有Session(client/server)都满足
var (
	//// client
	_ base.IObject = &rtmp.PushSession{}
	_ base.IObject = &rtsp.PushSession{}
	_ base.IObject = &rtmp.PullSession{}
	_ base.IObject = &rtsp.PullSession{}
	_ base.IObject = &httpflv.PullSession{}
	// server session
	_ base.IObject = &rtmp.ServerSession{}
	_ base.IObject = &rtsp.PubSession{}
	_ base.IObject = &rtsp.SubSession{}
	_ base.IObject = &httpflv.SubSession{}
	_ base.IObject = &httpts.SubSession{}
	//// other
	_ base.IObject = &base.BasicHttpSubSession{}
	_ base.IObject = &rtmp.ClientSession{}
	_ base.IObject = &rtsp.BaseInSession{}
	_ base.IObject = &rtsp.BaseOutSession{}
	_ base.IObject = &rtsp.ClientCommandSession{}
	_ base.IObject = &rtsp.ServerCommandSession{}
)

// ---------------------------------------------------------------------------------------------------------------------

var _ logic.ICustomizePubSessionContext = &logic.CustomizePubSessionContext{}
var _ base.IAvPacketStream = &logic.CustomizePubSessionContext{}

// ---------------------------------------------------------------------------------------------------------------------

var _ logic.ILalServer = &logic.ServerManager{}
var _ rtmp.IServerObserver = &logic.ServerManager{}
var _ logic.IHttpServerHandlerObserver = &logic.ServerManager{}
var _ rtsp.IServerObserver = &logic.ServerManager{}
var _ logic.IGroupCreator = &logic.ServerManager{}
var _ logic.IGroupObserver = &logic.ServerManager{}

var _ logic.INotifyHandler = &logic.HttpNotify{}
var _ logic.IGroupManager = &logic.SimpleGroupManager{}
var _ logic.IGroupManager = &logic.ComplexGroupManager{}

var _ rtmp.IPubSessionObserver = &logic.Group{} //
var _ rtsp.IPullSessionObserver = &logic.Group{}
var _ rtsp.IPullSessionObserver = &remux.AvPacket2RtmpRemuxer{}
var _ rtsp.IPubSessionObserver = &logic.Group{}
var _ rtsp.IPubSessionObserver = &remux.AvPacket2RtmpRemuxer{}
var _ hls.IMuxerObserver = &logic.Group{}
var _ rtsp.IBaseInSessionObserver = &logic.Group{} //
var _ rtsp.IBaseInSessionObserver = &remux.AvPacket2RtmpRemuxer{}
var _ remux.IRtmp2MpegtsRemuxerObserver = &hls.Muxer{}

var _ rtmp.IServerSessionObserver = &rtmp.Server{}
var _ rtmp.IHandshakeClient = &rtmp.HandshakeClientSimple{}
var _ rtmp.IHandshakeClient = &rtmp.HandshakeClientComplex{}

var _ rtsp.IServerCommandSessionObserver = &rtsp.Server{}
var _ rtsp.IClientCommandSessionObserver = &rtsp.PushSession{}
var _ rtsp.IClientCommandSessionObserver = &rtsp.PullSession{}
var _ rtsp.IInterleavedPacketWriter = &rtsp.PushSession{}
var _ rtsp.IInterleavedPacketWriter = &rtsp.PullSession{}
var _ rtsp.IInterleavedPacketWriter = &rtsp.PubSession{}
var _ rtsp.IInterleavedPacketWriter = &rtsp.SubSession{}
var _ rtsp.IInterleavedPacketWriter = &rtsp.ClientCommandSession{}
var _ rtsp.IInterleavedPacketWriter = &rtsp.ServerCommandSession{}

var _ base.IStatable = connection.New(nil)
