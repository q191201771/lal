// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtsp

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/q191201771/naza/pkg/nazamd5"
)

// TODO chef: 考虑部分内容移入naza中

const (
	AuthTypeDigest = "Digest"
	AuthTypeBasic  = "Basic"
	AuthAlgorithm  = "MD5"
)

type Auth struct {
	Username string
	Password string

	Typ       string
	Realm     string
	Nonce     string
	Algorithm string
}

func (a *Auth) FeedWwwAuthenticate(auths []string, username, password string) {
	a.Username = username
	a.Password = password
	//目前只处理第一个
	var s string
	if len(auths) > 0 {
		s = auths[0]
	} else {
		return
	}
	s = strings.TrimPrefix(s, HeaderWwwAuthenticate)
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, AuthTypeBasic) {
		a.Typ = AuthTypeBasic
		return
	}
	if !strings.HasPrefix(s, AuthTypeDigest) {
		Log.Warnf("FeedWwwAuthenticate type invalid. v=%s", s)
		return
	}

	a.Typ = AuthTypeDigest
	a.Realm = a.getV(s, `realm="`)
	a.Nonce = a.getV(s, `nonce="`)
	a.Algorithm = a.getV(s, `algorithm="`)

	if a.Realm == "" {
		Log.Warnf("FeedWwwAuthenticate realm invalid. v=%s", s)
	}
	if a.Nonce == "" {
		Log.Warnf("FeedWwwAuthenticate realm invalid. v=%s", s)
	}
	if a.Algorithm == "" {
		a.Algorithm = AuthAlgorithm
		Log.Warnf("FeedWwwAuthenticate algorithm not found fallback to %s. v=%s", AuthAlgorithm, s)
	}
	if a.Algorithm != AuthAlgorithm {
		Log.Warnf("FeedWwwAuthenticate algorithm invalid, only support MD5. v=%s", s)
	}
}

// MakeAuthorization 如果没有调用`FeedWwwAuthenticate`初始化过，则直接返回空字符串
func (a *Auth) MakeAuthorization(method, uri string) string {
	if a.Username == "" {
		return ""
	}
	switch a.Typ {
	case AuthTypeBasic:
		base1 := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf(`%s:%s`, a.Username, a.Password)))
		return fmt.Sprintf(`%s %s`, a.Typ, base1)
	case AuthTypeDigest:
		ha1 := nazamd5.Md5([]byte(fmt.Sprintf("%s:%s:%s", a.Username, a.Realm, a.Password)))
		ha2 := nazamd5.Md5([]byte(fmt.Sprintf("%s:%s", method, uri)))
		response := nazamd5.Md5([]byte(fmt.Sprintf("%s:%s:%s", ha1, a.Nonce, ha2)))
		return fmt.Sprintf(`%s username="%s", realm="%s", nonce="%s", uri="%s", response="%s", algorithm="%s"`, a.Typ, a.Username, a.Realm, a.Nonce, uri, response, a.Algorithm)
	}

	return ""
}

func (a *Auth) getV(s string, pre string) string {
	b := strings.Index(s, pre)
	if b == -1 {
		return ""
	}
	e := strings.Index(s[b+len(pre):], `"`)
	if e == -1 {
		return ""
	}
	return s[b+len(pre) : b+len(pre)+e]
}
