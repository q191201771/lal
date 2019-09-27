#### v0.3.0

- [功能] package logic: 增加 func FlvTag2RTMPMsg
- [代码调整] package rtmp: ClientSession 和 ServerSession 使用 nezha 中的 connection 做连接管理
- [代码调整] package rtmp: 增加 struct ChunkDivider
- [代码调整] package rtmp: 调整一些接口
- [代码调整] package httpflv: 删除了 group， gop 相关的代码，后续会放入 package logic 中
- [测试] package rtmp: 增加 `example_test.go` 用于测试整个 rtmp 包的流程
- [其他] 更新 nezha -> 0.3.0

#### v0.2.0

- [结构调整] 将 app/lal 的部分代码抽离到 pkg/logic 中，使得其他 app 可以使用
- [结构调整] 将协议层 rtmp.Group 和 应用层 app/lal 中的 GroupManager 合并为 逻辑层 pkg/logic 的 Group，以后只在逻辑层维护一个 Group，用于处理各种具体协议的输入输出流的挂载
- [功能] pkg/logic 中新增 trans.go: RTMPMsg2FlvTag
- [功能] PubSession 退出时，清空缓存的 meta、avc header、aac header
- [功能] PubSession 已经存在时，后续再连接的 Pub 直接关闭掉
- [功能] app/rtmppull 存储为flv文件
- [优化] chunk divider: calcHeader 在原地计算
- [其他] rtmp 中所有 typeid 相关的类型 int -> uint8，msgLen 相关的类型 int -> uint32
- [其他] 更新 nezha，新版本的日志库
- [其他] 整理日志
- [其他] pprof web 地址放入配置文件中
- [测试] 使用一些开源工具对 app/lal 做推流、拉流测试

#### v0.1.0

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

#### v0.0.1

1. 提供 `/app/flvfile2rtmppush` 给业务方使用
