// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package main

import (
	"io/ioutil"
	"net"
	"net/http"

	"github.com/q191201771/naza/pkg/nazalog"
)

var isHTTPServer = false

var listenAddr = ":10101"

func main() {
	l, err := net.Listen("tcp", listenAddr)
	nazalog.Assert(nil, err)

	if isHTTPServer {
		startHTTPServer(l)
	} else {
		startTCPServer(l)
	}
}

func startTCPServer(l net.Listener) {
	for {
		c, err := l.Accept()
		nazalog.Assert(nil, err)
		go func() {
			b := make([]byte, 8192)
			n, err := c.Read(b)
			nazalog.Assert(nil, err)
			//nazalog.Info(hex.Dump(b[:n]))
			nazalog.Info(string(b[:n]))
			_, _ = c.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
			_ = c.Close()
		}()
	}
}

func logHandler(w http.ResponseWriter, r *http.Request) {
	b, _ := ioutil.ReadAll(r.Body)
	nazalog.Infof("r=%+v, body=%s", r, b)
}

func startHTTPServer(l net.Listener) {
	m := http.NewServeMux()
	m.HandleFunc("/on_pub_start", logHandler)
	m.HandleFunc("/on_pub_stop", logHandler)
	m.HandleFunc("/on_sub_start", logHandler)
	m.HandleFunc("/on_sub_stop", logHandler)

	srv := http.Server{
		Handler: m,
	}
	err := srv.Serve(l)
	nazalog.Assert(nil, err)
}
