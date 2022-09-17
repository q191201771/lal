// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package hls

import (
	"bytes"
	"errors"
	"fmt"
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

	sessionIdHash, err := s.ServeHTTPWithUrlCtx(resp, req, urlCtx)
	if err != nil && sessionIdHash != "" {
		s.delSubSession(sessionIdHash, nil)
		return
	}
}

func (s *ServerHandler) ServeHTTPWithUrlCtx(resp http.ResponseWriter, req *http.Request, urlCtx base.UrlContext) (sessionIdHash string, err error) {
	//Log.Debugf("%+v", req)

	urlObj, _ := url.Parse(urlCtx.Url)

	// TODO chef:
	// - check appname in URI path

	filename := urlCtx.LastItemOfPath
	filetype := urlCtx.GetFileType()

	// handle session
	sessionIdHash = urlObj.Query().Get("session_id")
	if filetype == "ts" && sessionIdHash != "" {
		err = s.keepSessionAlive(sessionIdHash)
		if err != nil {
			Log.Warnf("keepSessionAlive failed. session[%s] err=%+v", sessionIdHash, err)
			return
		}
	} else if filetype == "m3u8" {
		var redirectUrl string
		redirectUrl, err = s.handleSubSession(sessionIdHash, urlObj, req, urlCtx)
		if err != nil {
			Log.Warnf("handle hlsSubSession[%s]. err=%+v", sessionIdHash, err)
			return
		}
		if redirectUrl != "" {
			if redirectUrl != urlCtx.Url {
				resp.Header().Add("Cache-Control", "no-cache")
				resp.Header().Add("Access-Control-Allow-Origin", "*")
				http.Redirect(resp, req, redirectUrl, http.StatusFound)
				return
			} else {
				err = errors.New(fmt.Sprintf("duplicate redirect url[%s]", redirectUrl))
				Log.Error(err.Error())
				resp.WriteHeader(http.StatusBadRequest)
				return
			}
		}
	}

	ri := PathStrategy.GetRequestInfo(urlCtx, s.outPath)
	//Log.Debugf("%+v", ri)

	if filename == "" || (filetype != "m3u8" && filetype != "ts") || ri.StreamName == "" || ri.FileNameWithPath == "" {
		err = errors.New(fmt.Sprintf("invalid hls request. url=%+v, request=%+v", urlCtx, ri))
		Log.Warnf(err.Error())
		resp.WriteHeader(404)
		return
	}

	content, _err := ReadFile(ri.FileNameWithPath)
	if _err != nil {
		err = errors.New(fmt.Sprintf("read hls file failed. request=%+v, err=%+v", ri, _err))
		Log.Warnf(err.Error())
		resp.WriteHeader(404)
		return
	}

	switch filetype {
	case "m3u8":
		resp.Header().Add("Content-Type", "application/x-mpegurl")
		resp.Header().Add("Server", base.LalHlsM3u8Server)
		if sessionIdHash != "" {
			content = bytes.ReplaceAll(content, []byte(".ts"), []byte(".ts?session_id="+sessionIdHash))
		}
	case "ts":
		resp.Header().Add("Content-Type", "video/mp2t")
		resp.Header().Add("Server", base.LalHlsTsServer)
	}
	resp.Header().Add("Cache-Control", "no-cache")
	resp.Header().Add("Access-Control-Allow-Origin", "*")

	if sessionIdHash != "" {
		session := s.getSubSession(sessionIdHash)
		if session != nil {
			session.AddWroteBytesSum(uint64(len(content)))
		}
	}

	_, _ = resp.Write(content)
	return
}

func (s *ServerHandler) getSubSession(sessionIdHash string) *SubSession {
	return s.sessionMap[sessionIdHash]
}

func (s *ServerHandler) keepSessionAlive(sessionIdHash string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	session := s.getSubSession(sessionIdHash)
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

func (s *ServerHandler) delSubSession(sessionIdHash string, session *SubSession) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if session == nil {
		session = s.sessionMap[sessionIdHash]
	}
	if session != nil {
		delete(s.sessionMap, sessionIdHash)
		s.observer.OnDelHlsSubSession(session)
	}
}

func (s *ServerHandler) onSubSessionExpired(sessionIdHash string, session *SubSession) {
	s.delSubSession(sessionIdHash, session)
}

func (s *ServerHandler) handleSubSession(sessionIdHash string, urlObj *url.URL, req *http.Request, urlCtx base.UrlContext) (redirectUrl string, err error) {
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
		query := urlObj.Query()
		query.Set("session_id", session.sessionIdHash)
		urlObj.RawQuery = query.Encode()
		return urlObj.String(), nil
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
