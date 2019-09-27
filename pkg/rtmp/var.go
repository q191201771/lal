package rtmp

// 一些更专业的配置项，暂时只在该源码文件中配置，不提供外部配置接口
var (
	readBufSize                   = 4096 // session 读缓冲的大小
	writeBufSize                  = 4096 // session 写缓冲的大小
	wChanSize                     = 1024 // session 发送数据时，channel 的大小
	serverSessionReadAVTimeoutMS  = 10000
	serverSessionWriteAVTimeoutMS = 10000
	LocalChunkSize                = 4096 // 本端设置的 chunk size
	windowAcknowledgementSize     = 5000000
	peerBandwidth                 = 5000000
)
