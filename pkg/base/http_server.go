// Copyright 2021, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package base

import (
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"reflect"

	"github.com/q191201771/naza/pkg/nazaerrors"
)

// TODO(chef)
// - 考虑移入naza中
// - 考虑增加一个pattern全部未命中的mux回调

var (
	ErrAddrEmpty             = errors.New("lal.base: http server addr empty")
	ErrMultiRegistForPattern = errors.New("lal.base: http server multiple registrations for pattern")
)

const (
	NetworkTcp = "tcp"
)

type LocalAddrCtx struct {
	IsHttps  bool
	Addr     string
	CertFile string
	KeyFile  string

	Network string // 默认为NetworkTcp
}

type HttpServerManager struct {
	addr2ServerCtx map[string]*ServerCtx
}

type ServerCtx struct {
	addrCtx         LocalAddrCtx
	listener        net.Listener
	httpServer      http.Server
	mux             *http.ServeMux
	pattern2Handler map[string]Handler
}

func NewHttpServerManager() *HttpServerManager {
	return &HttpServerManager{
		addr2ServerCtx: make(map[string]*ServerCtx),
	}
}

type Handler func(http.ResponseWriter, *http.Request)

// @param pattern 必须以`/`开始，并以`/`结束
func (s *HttpServerManager) AddListen(addrCtx LocalAddrCtx, pattern string, handler Handler) error {
	var (
		ctx *ServerCtx
		mux *http.ServeMux
		ok  bool
	)

	if addrCtx.Addr == "" {
		return ErrAddrEmpty
	}

	ctx, ok = s.addr2ServerCtx[addrCtx.Addr]
	if !ok {
		l, err := listen(addrCtx)
		if err != nil {
			return err
		}
		mux = http.NewServeMux()
		ctx = &ServerCtx{
			addrCtx:  addrCtx,
			listener: l,
			httpServer: http.Server{
				Handler: mux,
			},
			mux:             mux,
			pattern2Handler: make(map[string]Handler),
		}
		s.addr2ServerCtx[addrCtx.Addr] = ctx
	}

	// 路径相同，比较回调函数是否相同
	// 如果回调函数也相同，意味着重复绑定，这种情况是允许的
	// 如果回调函数不同，返回错误
	if prevHandler, ok := ctx.pattern2Handler[pattern]; ok {
		if reflect.ValueOf(prevHandler).Pointer() == reflect.ValueOf(handler).Pointer() {
			return nil
		} else {
			return ErrMultiRegistForPattern
		}
	}
	ctx.pattern2Handler[pattern] = handler

	ctx.mux.HandleFunc(pattern, handler)
	return nil
}

func (s *HttpServerManager) RunLoop() error {
	errChan := make(chan error, len(s.addr2ServerCtx))

	for _, v := range s.addr2ServerCtx {
		go func(ctx *ServerCtx) {
			errChan <- ctx.httpServer.Serve(ctx.listener)

			_ = ctx.httpServer.Close()
		}(v)
	}

	// 阻塞直到接到第一个error
	return <-errChan
}

func (s *HttpServerManager) Dispose() error {
	var es []error
	for _, v := range s.addr2ServerCtx {
		err := v.httpServer.Close()
		es = append(es, err)
	}
	return nazaerrors.CombineErrors(es...)
}

func listen(ctx LocalAddrCtx) (net.Listener, error) {
	if ctx.Network == "" {
		ctx.Network = NetworkTcp
	}

	if !ctx.IsHttps {
		return net.Listen(ctx.Network, ctx.Addr)
	}

	cert, err := tls.LoadX509KeyPair(ctx.CertFile, ctx.KeyFile)
	if err != nil {
		return nil, err
	}
	tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}}
	return tls.Listen(ctx.Network, ctx.Addr, tlsConfig)
}
