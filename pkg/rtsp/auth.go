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

	"github.com/q191201771/naza/pkg/nazalog"
	"github.com/q191201771/naza/pkg/nazamd5"
)

// TODO chef: 考虑部分内容移入naza中
// TODO chef: 只支持Digest方式，支持Basic方式

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

func (a *Auth) FeedWWWAuthenticate(s, username, password string) {
	a.Username = username
	a.Password = password

	s = strings.TrimPrefix(s, HeaderWWWAuthenticate)
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, AuthTypeDigest) {
		a.Typ = AuthTypeDigest
	} else {
		a.Typ = AuthTypeBasic
		return
	}
	a.Realm = a.getV(s, `realm="`)
	a.Nonce = a.getV(s, `nonce="`)
	a.Algorithm = a.getV(s, `algorithm="`)

	if a.Typ != AuthTypeDigest {
		nazalog.Warnf("FeedWWWAuthenticate type invalid, only support Digest. v=%s", s)
	}
	if a.Realm == "" {
		nazalog.Warnf("FeedWWWAuthenticate realm invalid. v=%s", s)
	}
	if a.Nonce == "" {
		nazalog.Warnf("FeedWWWAuthenticate realm invalid. v=%s", s)
	}
	if a.Algorithm != AuthAlgorithm {
		nazalog.Warnf("FeedWWWAuthenticate algorithm invalid, only support MD5. v=%s", s)
	}
}

func (a *Auth) MakeAuthorization(method, uri string) string {
	if a.Username == "" {
		return ""
	}
	if a.Typ == AuthTypeDigest {
		if a.Nonce == "" {
			return ""
		}
		ha1 := nazamd5.MD5([]byte(fmt.Sprintf("%s:%s:%s", a.Username, a.Realm, a.Password)))
		ha2 := nazamd5.MD5([]byte(fmt.Sprintf("%s:%s", method, uri)))
		response := nazamd5.MD5([]byte(fmt.Sprintf("%s:%s:%s", ha1, a.Nonce, ha2)))
		return fmt.Sprintf(`%s username="%s", realm="%s", nonce="%s", uri="%s", response="%s", algorithm="%s"`, a.Typ, a.Username, a.Realm, a.Nonce, uri, response, a.Algorithm)
	} else {
		ha1 := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf(`%s:%s`, a.Username, a.Password)))
		return fmt.Sprintf(`%s %s`, a.Typ, ha1)
	}

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
