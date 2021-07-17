// Copyright 2021, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtsp_test

import (
	"testing"

	"github.com/q191201771/lal/pkg/rtsp"

	"github.com/q191201771/naza/pkg/assert"
)

func TestGetRtspFirstAuth(t *testing.T) {
	var rtspAuth rtsp.Auth
	auths := make([]string, 2)
	auths[0] = `Digest realm="54c41545bbe6", nonce="13991620f27aff5cc046228b7d4434b7", stale="FALSE"`
	auths[1] = `Basic realm="54c41545bbe6"`
	username := "admin"
	password := "admin123"
	rtspAuth.FeedWwwAuthenticate(auths, username, password)

	assert.Equal(t, rtsp.AuthTypeDigest, rtspAuth.Typ)
	assert.Equal(t, "54c41545bbe6", rtspAuth.Realm)
	assert.Equal(t, "13991620f27aff5cc046228b7d4434b7", rtspAuth.Nonce)
	assert.Equal(t, "admin", rtspAuth.Username)
	assert.Equal(t, "admin123", rtspAuth.Password)
}
