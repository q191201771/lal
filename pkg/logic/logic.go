// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package logic

import (
	"errors"

	"github.com/q191201771/lal/pkg/hls"
	"github.com/q191201771/lal/pkg/httpts"

	"github.com/q191201771/lal/pkg/rtsp"

	"github.com/q191201771/lal/pkg/httpflv"
	"github.com/q191201771/lal/pkg/rtmp"
)

var ErrLogic = errors.New("lal.logic: fxxk")

var _ rtmp.ServerObserver = &ServerManager{}
var _ httpflv.ServerObserver = &ServerManager{}
var _ httpts.ServerObserver = &ServerManager{}
var _ rtsp.ServerObserver = &ServerManager{}

var _ rtmp.PubSessionObserver = &Group{}
var _ rtsp.PubSessionObserver = &Group{}
var _ hls.MuxerObserver = &Group{}
