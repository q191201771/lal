// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package hls

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/q191201771/lal/pkg/base"
)

type IHlsServerHandlerObserver interface {
	OnNewHlsSubSession(session *SubSession) error
	OnDelHlsSubSession(session *SubSession)
}

type ServerHandler struct {
	outPath        string
	observer       IHlsServerHandlerObserver
	urlPattern     string
	sessionMap     map[string]*SubSession
	mutex          sync.Mutex
	sessionTimeout time.Duration
	sessionHashKey string
}

func NewServerHandler(outPath, urlPattern, sessionHashKey string, sessionTimeoutMs int, observer IHlsServerHandlerObserver) *ServerHandler {
	if strings.HasPrefix(urlPattern, "/") {
		urlPattern = urlPattern[1:]
	}
	if sessionTimeoutMs == 0 {
		sessionTimeoutMs = 30000
	}
	sh := &ServerHandler{
		outPath:        outPath,
		observer:       observer,
		urlPattern:     urlPattern,
		sessionMap:     make(map[string]*SubSession),
		sessionTimeout: time.Duration(sessionTimeoutMs) * time.Millisecond,
		sessionHashKey: sessionHashKey,
	}
	go sh.runLoop()
	return sh
}

func (s *ServerHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	urlCtx, err := base.ParseUrl(base.ParseHttpRequest(req), 80)
	if err != nil {
		Log.Errorf("parse url. err=%+v", err)
		return
	}
	if urlCtx.GetFileType() == "m3u8" {
		redirectUrl, err := s.handleSubSession(req, urlCtx)
		if err != nil {
			Log.Warnf("handle hlsSubSession. err=%+v", err)
			return
		}
		if redirectUrl != "" {
			if redirectUrl != urlCtx.Url {
				resp.Header().Add("Cache-Control", "no-cache")
				resp.Header().Add("Access-Control-Allow-Origin", "*")
				http.Redirect(resp, req, redirectUrl, http.StatusFound)
				return
			} else {
				resp.WriteHeader(http.StatusBadRequest)
				return
			}
		}
	}

	s.ServeHTTPWithUrlCtx(resp, urlCtx)
}

func (s *ServerHandler) ServeHTTPWithUrlCtx(resp http.ResponseWriter, urlCtx base.UrlContext) {
	//Log.Debugf("%+v", req)

	// TODO chef:
	// - check appname in URI path

	filename := urlCtx.LastItemOfPath
	filetype := urlCtx.GetFileType()

	ri := PathStrategy.GetRequestInfo(urlCtx, s.outPath)
	//Log.Debugf("%+v", ri)

	if filename == "" || (filetype != "m3u8" && filetype != "ts") || ri.StreamName == "" || ri.FileNameWithPath == "" {
		Log.Warnf("invalid hls request. url=%+v, request=%+v", urlCtx, ri)
		resp.WriteHeader(404)
		return
	}

	content, err := ReadFile(ri.FileNameWithPath)
	if err != nil {
		Log.Warnf("read hls file failed. request=%+v, err=%+v", ri, err)
		resp.WriteHeader(404)
		return
	}

	switch filetype {
	case "m3u8":
		resp.Header().Add("Content-Type", "application/x-mpegurl")
		resp.Header().Add("Server", base.LalHlsM3u8Server)
	case "ts":
		resp.Header().Add("Content-Type", "video/mp2t")
		resp.Header().Add("Server", base.LalHlsTsServer)
	}
	resp.Header().Add("Cache-Control", "no-cache")
	resp.Header().Add("Access-Control-Allow-Origin", "*")

	_, _ = resp.Write(content)
	return
}

func (s *ServerHandler) keepSessionAlive(sessionIdHash string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	session := s.sessionMap[sessionIdHash]
	if session == nil {
		return base.ErrHlsSessionNotFound
	}
	session.KeepAlive()
	return nil
}

func (s *ServerHandler) createSubSession(req *http.Request, urlCtx base.UrlContext) (*SubSession, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	session := NewSubSession(req, urlCtx, s.urlPattern, s.sessionHashKey, s.sessionTimeout)
	s.sessionMap[session.sessionIdHash] = session
	err := s.observer.OnNewHlsSubSession(session)
	return session, err
}

func (s *ServerHandler) onSubSessionExpired(sessionIdHash string, session *SubSession) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	delete(s.sessionMap, sessionIdHash)
	s.observer.OnDelHlsSubSession(session)
}

func (s *ServerHandler) handleSubSession(req *http.Request, urlCtx base.UrlContext) (redirectUrl string, err error) {
	urlParsed, err := url.Parse(urlCtx.Url)
	if err != nil {
		return "", errors.New("parse url err")
	}
	sessionIdHash := urlParsed.Query().Get("session_id")
	if sessionIdHash != "" {
		err = s.keepSessionAlive(sessionIdHash)
		if err != nil {
			return "", err
		}
	} else {
		session, err := s.createSubSession(req, urlCtx)
		if err != nil {
			return "", err
		}
		query := urlParsed.Query()
		query.Set("session_id", session.sessionIdHash)
		urlParsed.RawQuery = query.Encode()
		return urlParsed.String(), nil
	}
	return "", nil
}

func (s *ServerHandler) clearExpireSession() {
	for sessionIdHash, session := range s.sessionMap {
		if session.IsExpired() {
			s.onSubSessionExpired(sessionIdHash, session)
		}
	}
}

func (s *ServerHandler) runLoop() {
	ticker := time.NewTicker(1 * time.Second)
	for range ticker.C {
		s.clearExpireSession()
	}
}

// m3u8文件用这个也行
//resp.Header().Add("Content-Type", "application/vnd.apple.mpegurl")

//resp.Header().Add("Access-Control-Allow-Origin", "*")
//resp.Header().Add("Access-Control-Allow-Credentials", "true")
//resp.Header().Add("Access-Control-Allow-Methods", "*")
//resp.Header().Add("Access-Control-Allow-Headers", "Content-Type,Access-Token")
//resp.Header().Add("Access-Control-Allow-Expose-Headers", "*")
