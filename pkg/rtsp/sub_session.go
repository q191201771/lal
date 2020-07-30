// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtsp

import (
	"github.com/q191201771/naza/pkg/connection"
	"github.com/q191201771/naza/pkg/nazalog"
	"net"
)

type SubSession struct {
	StreamName   	string 					// presentation
	servers     	[]*UDPServer
	remoteRtpAddr	[]*net.UDPAddr			//每个track的客户端RTP端口对应的UDPAddr
	remoteRtcpAddr	[]*net.UDPAddr			//每个track的客户端RTCP端口对应的UDPAddr
	conn			connection.Connection	// rtsp_tcp连接
}

func NewSubSession(streamName string,conn net.Conn) *SubSession {
	return &SubSession{
		StreamName: streamName,
		conn: connection.New(conn, func(option *connection.Option) {
			option.ReadBufSize = readBufSize
			option.WriteChanSize = wChanSize
			option.WriteTimeoutMS = subSessionWriteTimeoutMS
		}),
	}
}

func (p *SubSession) AddConn(conn *net.UDPConn,rtpPort int,rtcpPort int) {
	server := NewUDPServerWithConn(conn, p.onReadUDPPacket)
	go server.RunLoop()
	p.servers = append(p.servers, server)

	var addr *net.UDPAddr;
	// rtp UDPAddr
	addr = &net.UDPAddr{
		IP:   p.conn.RemoteAddr().(*net.TCPAddr).IP,
		Zone: p.conn.RemoteAddr().(*net.TCPAddr).Zone,
		Port: rtpPort,
	}
	p.remoteRtpAddr = append(p.remoteRtpAddr, addr)

	// rtcp UDPAddr
	addr = &net.UDPAddr{
		IP:   p.conn.RemoteAddr().(*net.TCPAddr).IP,
		Zone: p.conn.RemoteAddr().(*net.TCPAddr).Zone,
		Port: rtcpPort,
	}
	p.remoteRtcpAddr = append(p.remoteRtcpAddr, addr)
}
func (p *SubSession) onReadUDPPacket(b []byte, addr string, err error) {
	if len(b) <= 0 {
		return
	}

	// try RTCP
	switch b[1] {
	case RTCPPacketTypeSR:
		parseRTCPPacket(b)
	default:
		nazalog.Errorf("unknown PT. pt=%d", b[1] & 0x7F)
		nazalog.Debug(b)
	}
}
// rtp_over_udp
func (p *SubSession) WriteRtpOverUdp(pkt []byte) {
	// 传输TS数据只有一个Track
	trackID := 0
	p.servers[trackID].conn.WriteTo(pkt,p.remoteRtpAddr[trackID])
}
// rtcp_over_udp
func (p *SubSession) WriteRtcpOverUdp(pkt []byte) {
	// 传输TS数据只有一个Track
	trackID := 0
	p.servers[trackID].conn.WriteTo(pkt,p.remoteRtcpAddr[trackID])
}
// 资源释放
func (p *SubSession) Dispose() {
	// p.conn会在Server.handleTCPConnect中关闭,所以在这里不关闭
	// 关闭关联的udp
	for _, udp := range p.servers {
		//客户端断开的时间第一时间释放本机占用的socket_point,避免在海量连接时资源不足
		udp.conn.Close()
	}
}