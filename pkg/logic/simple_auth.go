// Copyright 2022, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package logic

import (
	"net/url"
	"strings"

	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/naza/pkg/nazamd5"
)

func SimpleAuthCalcSecret(key string, streamName string) string {
	return nazamd5.Md5([]byte(key + streamName))
}

// ---------------------------------------------------------------------------------------------------------------------

// TODO(chef): [refactor] 结合 NotifyHandler 整理

const secretName = "lal_secret"

type SimpleAuthCtx struct {
	key    string
	config SimpleAuthConfig
}

// NewSimpleAuthCtx
//
// @param key: 如果为空，则所有鉴权接口都返回成功
//
func NewSimpleAuthCtx(config SimpleAuthConfig) *SimpleAuthCtx {
	return &SimpleAuthCtx{
		config: config,
	}
}

func (s *SimpleAuthCtx) OnPubStart(info base.PubStartInfo) error {
	if s.config.PubRtmpEnable && info.Protocol == base.ProtocolRtmp ||
		s.config.PubRtspEnable && info.Protocol == base.ProtocolRtsp {
		return s.check(info.StreamName, info.UrlParam)
	}

	return nil
}

func (s *SimpleAuthCtx) OnSubStart(info base.SubStartInfo) error {
	if (s.config.SubRtmpEnable && info.Protocol == base.ProtocolRtmp) ||
		(s.config.SubHttpflvEnable && info.Protocol == base.ProtocolHttpflv) ||
		(s.config.SubHttptsEnable && info.Protocol == base.ProtocolHttpts) ||
		(s.config.SubRtspEnable && info.Protocol == base.ProtocolRtsp) {
		return s.check(info.StreamName, info.UrlParam)
	}

	return nil
}

func (s *SimpleAuthCtx) OnHls(streamName string, urlParam string) error {
	return s.check(streamName, urlParam)
}

func (s *SimpleAuthCtx) check(streamName string, urlParam string) error {
	q, err := url.ParseQuery(urlParam)
	if err != nil {
		return err
	}
	v := q.Get(secretName)
	if v == "" {
		return base.ErrSimpleAuthParamNotFound
	}
	v = strings.ToLower(v)
	se := SimpleAuthCalcSecret(s.config.Key, streamName)
	if v != s.config.DangerousLalSecret && v != se {
		return base.ErrSimpleAuthFailed
	}
	return nil
}
