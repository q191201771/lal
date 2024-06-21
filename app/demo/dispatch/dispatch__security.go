// Copyright 2024, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package main

import (
	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/naza/pkg/nazalog"
	"net"
	"time"
)

func securityMaxSubSessionPerIp(info base.UpdateInfo) {
	if config.MaxSubSessionPerIp <= 0 {
		return
	}

	ip2SubSessions := make(map[string][]base.StatSub)
	sessionId2StreamName := make(map[string]string)
	for _, g := range info.Groups {
		for _, sub := range g.StatSubs {
			host, _, err := net.SplitHostPort(sub.RemoteAddr)
			if err != nil {
				nazalog.Warnf("securityMaxSubSessionPerIp. split host port failed. remote addr=%s", sub.RemoteAddr)
				continue
			}
			ip2SubSessions[host] = append(ip2SubSessions[host], sub)
			sessionId2StreamName[sub.SessionId] = g.StreamName
		}
	}

	lastUnix := int64(-1)
	lastId := ""
	for ip, subs := range ip2SubSessions {
		if len(subs) <= config.MaxSubSessionPerIp {
			continue
		}
		nazalog.Debugf("securityMaxSubSessionPerIp. close session. ip=%s, session count=%d", ip, len(subs))

		for _, sub := range subs {
			st, err := base.ParseReadableTime(sub.StartTime)
			if err != nil {
				nazalog.Warnf("securityMaxSubSessionPerIp. parse readable time failed. start time=%s, err=%+v", sub.StartTime, err)
				continue
			}
			if st.UnixMilli() > lastUnix {
				lastUnix = st.UnixMilli()
				lastId = sub.SessionId
			}
		}

		for _, sub := range subs {
			if sub.SessionId == lastId {
				nazalog.Debugf("securityMaxSubSessionPerIp. skip kick. last id=%s", lastId)
				continue
			}
			kickSession(info.ServerId, sessionId2StreamName[sub.SessionId], sub.SessionId)
		}
	}
}

func securityMaxSubDurationSec(info base.UpdateInfo) {
	if config.MaxSubDurationSec <= 0 {
		return
	}

	now := time.Now()
	for _, g := range info.Groups {
		for _, sub := range g.StatSubs {
			st, err := base.ParseReadableTime(sub.StartTime)
			if err != nil {
				nazalog.Warnf("parse readable time failed. start time=%s, err=%+v", sub.StartTime, err)
				continue
			}
			diff := int(now.Sub(st).Seconds())
			if diff <= config.MaxSubDurationSec {
				continue
			}
			nazalog.Infof("close session. sub session start time=%s, diff=%d", sub.StartTime, diff)
			kickSession(info.ServerId, g.StreamName, sub.SessionId)
		}
	}
}
