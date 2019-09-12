**v0.1.0**

- /app/flvfile2rtmppush 优化推流平稳性
- bugfix rtmp 推拉流信令时可以携带 url 参数，并且在做上下行匹配时去掉 url 参数
- rtmp.ServerSession 处理 typeidAck
- 增加 amf0.WriteNull 和 amf0.WriteBoolean；WriteObject 中增加 bool 类型；bugfix: ReadString 当长度不足时返回 ErrAMFTooShort 而不是 ErrAMFInvalidType
- app lal 接收 USER1 USER2 信号，优雅退出
- 日志相关的配置放入配置文件中
- 整理代码；整理日志；整理 build.sh
- 增加 rtmp.HandshakeClientComplex 复杂握手模式
- 整理一些 struct 的 Dispose 方法
- CaptureAVC 添加错误返回值
- 增加一些单元测试和 benchmark
- 更新 nezha 0.1.0
- errors.PanicIfErrorOccur -> log.FatalIfErrorNotNil

**v0.0.1**

1. 提供 `/app/flvfile2rtmppush` 给业务方使用
