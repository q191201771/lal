package hls

import (
	"github.com/q191201771/lal/pkg/base"
	"strings"
	"time"
)

type SubSession struct {
	LastRequestTime time.Time
	urlCtx          base.UrlContext
	stat            base.StatSession
	hlsUrlPattern   string
	appName         string
	timeout         time.Duration
}

func (s *SubSession) UniqueKey() string {
	return s.stat.SessionId
}

func NewSubSession(urlCtx base.UrlContext, hlsUrlPattern string, timeout time.Duration) *SubSession {
	if strings.HasPrefix(hlsUrlPattern, "/") {
		hlsUrlPattern = hlsUrlPattern[1:]
	}
	session := &SubSession{
		LastRequestTime: time.Now(),
		urlCtx:          urlCtx,
		hlsUrlPattern:   hlsUrlPattern,
		timeout:         timeout,
	}
	session.stat = base.StatSession{
		SessionId: base.GenUkHlsSubSession(),
		Protocol:  base.SessionProtocolHlsStr,
	}
	return session
}

func (s *SubSession) Url() string {
	return s.urlCtx.Url
}

func (s *SubSession) AppName() string {
	if s.appName == "" {
		s.appName = GetAppNameFromUrlCtx(s.urlCtx, s.hlsUrlPattern)
	}
	return s.appName
}

func (s *SubSession) StreamName() string {
	return GetStreamNameFromUrlCtx(s.urlCtx)
}

func (s *SubSession) RawQuery() string {
	return s.urlCtx.RawQuery
}

func (s *SubSession) UpdateStat(intervalSec uint32) {
	// TODO implement hls update stat
	return
}

func (s *SubSession) GetStat() base.StatSession {
	return s.stat
}

func (s *SubSession) IsAlive() (readAlive, writeAlive bool) {
	if !s.IsExpired() {
		return true, true
	}
	return false, false
}

func (s *SubSession) IsExpired() bool {
	return s.LastRequestTime.Add(s.timeout).Before(time.Now())
}

func (s *SubSession) KeepAlive() {
	s.LastRequestTime = time.Now()
}

func GetAppNameFromUrlCtx(urlCtx base.UrlContext, hlsUrlPattern string) string {
	var appName string
	if hlsUrlPattern == "" {
		appName = urlCtx.PathWithoutLastItem
	} else {
		appName = strings.Split(urlCtx.PathWithoutLastItem, hlsUrlPattern)[1]
	}
	return appName
}

func GetStreamNameFromUrlCtx(urlCtx base.UrlContext) string {
	return urlCtx.GetFilenameWithoutType()
}
