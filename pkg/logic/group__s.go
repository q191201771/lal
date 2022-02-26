// Copyright 2022, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package logic

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/lal/pkg/hls"
	"github.com/q191201771/lal/pkg/httpflv"
	"github.com/q191201771/lal/pkg/mpegts"
	"github.com/q191201771/lal/pkg/remux"
	"github.com/q191201771/lal/pkg/rtmp"
	"github.com/q191201771/lal/pkg/rtsp"
)

func (group *Group) AddRtmpPubSession(session *rtmp.ServerSession) error {
	group.mutex.Lock()
	defer group.mutex.Unlock()

	Log.Debugf("[%s] [%s] add rtmp pub session into group.", group.UniqueKey, session.UniqueKey())

	if group.hasInSession() {
		Log.Errorf("[%s] in stream already exist at group. add=%s, exist=%s",
			group.UniqueKey, session.UniqueKey(), group.inSessionUniqueKey())
		return base.ErrDupInStream
	}

	group.rtmpPubSession = session
	group.addIn()

	if group.config.RtspConfig.Enable {
		group.rtmp2RtspRemuxer = remux.NewRtmp2RtspRemuxer(
			group.onSdpFromRemux,
			group.onRtpPacketFromRemux,
		)
	}

	// TODO(chef): 为rtmp pull以及rtsp也添加叠加静音音频的功能
	if group.config.RtmpConfig.AddDummyAudioEnable {
		// TODO(chef): 从整体控制和锁关系来说，应该让pub的数据回调到group中进锁后再让数据流入filter
		group.dummyAudioFilter = remux.NewDummyAudioFilter(group.UniqueKey, group.config.RtmpConfig.AddDummyAudioWaitAudioMs, group.OnReadRtmpAvMsg)
		session.SetPubSessionObserver(group.dummyAudioFilter)
	} else {
		session.SetPubSessionObserver(group)
	}

	return nil
}

// AddRtspPubSession TODO chef: rtsp package中，增加回调返回值判断，如果是false，将连接关掉
func (group *Group) AddRtspPubSession(session *rtsp.PubSession) error {
	Log.Debugf("[%s] [%s] add RTSP PubSession into group.", group.UniqueKey, session.UniqueKey())

	group.mutex.Lock()
	defer group.mutex.Unlock()

	if group.hasInSession() {
		Log.Errorf("[%s] in stream already exist at group. wanna add=%s", group.UniqueKey, session.UniqueKey())
		return base.ErrDupInStream
	}

	group.rtspPubSession = session
	group.addIn()

	group.rtsp2RtmpRemuxer = remux.NewAvPacket2RtmpRemuxer(group.onRtmpMsgFromRemux)
	session.SetObserver(group)

	return nil
}

func (group *Group) AddRtmpPullSession(session *rtmp.PullSession) bool {
	Log.Debugf("[%s] [%s] add PullSession into group.", group.UniqueKey, session.UniqueKey())

	group.mutex.Lock()
	defer group.mutex.Unlock()

	if group.hasInSession() {
		Log.Errorf("[%s] in stream already exist. wanna add=%s", group.UniqueKey, session.UniqueKey())
		return false
	}

	group.pullProxy.pullSession = session
	group.addIn()

	// TODO(chef): 这里也应该启动rtmp2RtspRemuxer

	return true
}

// ---------------------------------------------------------------------------------------------------------------------

func (group *Group) DelRtmpPubSession(session *rtmp.ServerSession) {
	group.mutex.Lock()
	defer group.mutex.Unlock()
	group.delRtmpPubSession(session)
}

func (group *Group) DelRtspPubSession(session *rtsp.PubSession) {
	group.mutex.Lock()
	defer group.mutex.Unlock()
	group.delRtspPubSession(session)
}

func (group *Group) DelRtmpPullSession(session *rtmp.PullSession) {
	group.mutex.Lock()
	defer group.mutex.Unlock()
	group.delRtmpPullSession(session)
}

// ---------------------------------------------------------------------------------------------------------------------

func (group *Group) delRtmpPubSession(session *rtmp.ServerSession) {
	Log.Debugf("[%s] [%s] del rtmp PubSession from group.", group.UniqueKey, session.UniqueKey())

	if session != group.rtmpPubSession {
		Log.Warnf("[%s] del rtmp pub session but not match. del session=%s, group session=%p",
			group.UniqueKey, session.UniqueKey(), group.rtmpPubSession)
		return
	}

	group.delIn()
}

func (group *Group) delRtspPubSession(session *rtsp.PubSession) {
	Log.Debugf("[%s] [%s] del rtsp PubSession from group.", group.UniqueKey, session.UniqueKey())

	if session != group.rtspPubSession {
		Log.Warnf("[%s] del rtmp pub session but not match. del session=%s, group session=%p",
			group.UniqueKey, session.UniqueKey(), group.rtspPubSession)
		return
	}

	_ = group.rtspPubSession.Dispose()
	group.rtspPubSession = nil
	group.rtsp2RtmpRemuxer = nil
	group.delIn()
}

func (group *Group) delRtmpPullSession(session *rtmp.PullSession) {
	Log.Debugf("[%s] [%s] del rtmp PullSession from group.", group.UniqueKey, session.UniqueKey())

	group.pullProxy.pullSession = nil
	group.setPullingFlag(false)
	group.delIn()
}

// ---------------------------------------------------------------------------------------------------------------------

// addIn 有pub或pull的输入型session加入时，需要调用该函数
//
func (group *Group) addIn() {
	// 是否push转推
	group.pushIfNeeded()

	// 是否启动hls
	if group.config.HlsConfig.Enable {
		if group.hlsMuxer != nil {
			Log.Errorf("[%s] hls muxer exist while addIn. muxer=%+v", group.UniqueKey, group.hlsMuxer)
		}
		enable := group.config.HlsConfig.Enable || group.config.HlsConfig.EnableHttps
		group.hlsMuxer = hls.NewMuxer(group.streamName, enable, &group.config.HlsConfig.MuxerConfig, group)
		group.hlsMuxer.Start()
	}

	now := time.Now().Unix()

	// 是否录制成flv文件
	group.startRecordFlvIfNeeded(now)

	// 是否录制成ts文件
	group.startRecordTsIfNeeded(now)
}

// delIn 有pub或pull的输入型session离开时，需要调用该函数
//
func (group *Group) delIn() {
	// 停止hls
	if group.config.HlsConfig.Enable && group.hlsMuxer != nil {
		group.disposeHlsMuxer()
	}

	// 停止转推
	if group.pushEnable {
		for _, v := range group.url2PushProxy {
			if v.pushSession != nil {
				v.pushSession.Dispose()
			}
			v.pushSession = nil
		}
	}

	// 停止flv录制
	if group.config.RecordConfig.EnableFlv {
		if group.recordFlv != nil {
			if err := group.recordFlv.Dispose(); err != nil {
				Log.Errorf("[%s] record flv dispose error. err=%+v", group.UniqueKey, err)
			}
			group.recordFlv = nil
		}
	}

	// 停止ts录制
	if group.config.RecordConfig.EnableMpegts {
		if group.recordMpegts != nil {
			if err := group.recordMpegts.Dispose(); err != nil {
				Log.Errorf("[%s] record mpegts dispose error. err=%+v", group.UniqueKey, err)
			}
			group.recordMpegts = nil
		}
	}

	group.rtmpPubSession = nil
	group.rtspPubSession = nil
	group.rtsp2RtmpRemuxer = nil
	group.rtmp2RtspRemuxer = nil
	group.dummyAudioFilter = nil
	group.rtmpGopCache.Clear()
	group.httpflvGopCache.Clear()
	group.patpmt = nil
	group.sdpCtx = nil
}

// ---------------------------------------------------------------------------------------------------------------------

// startRecordFlvIfNeeded 是否开启flv录制
//
func (group *Group) startRecordFlvIfNeeded(nowUnix int64) {
	if !group.config.RecordConfig.EnableFlv {
		return
	}

	// 构造文件名
	filename := fmt.Sprintf("%s-%d.flv", group.streamName, nowUnix)
	filenameWithPath := filepath.Join(group.config.RecordConfig.FlvOutPath, filename)
	// 如果已经在录制，则先关闭
	// TODO(chef): 正常的逻辑是否会走到这？
	if group.recordFlv != nil {
		Log.Errorf("[%s] record flv but already exist. new filename=%s, old filename=%s",
			group.UniqueKey, filenameWithPath, group.recordFlv.Name())
		_ = group.recordFlv.Dispose()
	}
	// 初始化录制
	group.recordFlv = &httpflv.FlvFileWriter{}
	if err := group.recordFlv.Open(filenameWithPath); err != nil {
		Log.Errorf("[%s] record flv open file failed. filename=%s, err=%+v",
			group.UniqueKey, filenameWithPath, err)
		group.recordFlv = nil
	}
	if err := group.recordFlv.WriteFlvHeader(); err != nil {
		Log.Errorf("[%s] record flv write flv header failed. filename=%s, err=%+v",
			group.UniqueKey, filenameWithPath, err)
		group.recordFlv = nil
	}
}

func (group *Group) startRecordTsIfNeeded(nowUnix int64) {
	if !group.config.RecordConfig.EnableMpegts {
		return
	}

	// 构造文件名
	filename := fmt.Sprintf("%s-%d.ts", group.streamName, nowUnix)
	filenameWithPath := filepath.Join(group.config.RecordConfig.MpegtsOutPath, filename)
	// 如果已经在录制，则先关闭
	if group.recordMpegts != nil {
		Log.Errorf("[%s] record mpegts but already exist. new filename=%s, old filename=%s",
			group.UniqueKey, filenameWithPath, group.recordMpegts.Name())
		_ = group.recordMpegts.Dispose()
	}
	group.recordMpegts = &mpegts.FileWriter{}
	if err := group.recordMpegts.Create(filenameWithPath); err != nil {
		Log.Errorf("[%s] record mpegts open file failed. filename=%s, err=%+v",
			group.UniqueKey, filenameWithPath, err)
		group.recordMpegts = nil
	}
}

// ---------------------------------------------------------------------------------------------------------------------

func (group *Group) inSessionUniqueKey() string {
	if group.rtmpPubSession != nil {
		return group.rtmpPubSession.UniqueKey()
	}
	if group.rtspPubSession != nil {
		return group.rtspPubSession.UniqueKey()
	}
	return ""
}
