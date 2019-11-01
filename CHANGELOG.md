#### v0.5.0

- package rtmp:
    - 增加结构体 ClientSessionOption，PushSessionOption，PullSessionOption
    - 增加结构体 AVMsg
    - ClientSession 作为 PushSession 和 PullSession 的私有成员
    - 将绝对时间戳移入到 Header 结构体中
    - PullSession::Pull OnReadAVMsg with AVMsg
    - AVMsgObserver::ReadRTMPAVMsgCB -> OnReadRTMPAVMsg
- package httpflv:
    - PullSessionOption
    - OnReadFLVTag
    - some func use Tag instead of *Tag
    - 整个包的代码做了一次整理
    - FlvFileReader 在 ReadTag 中懒读取 flv header
- package logic:
    - 使用 rtmp.AVMsg
    - 增加两个函数 MakeDefaultRTMPHeader，FLVTagHeader2RTMPHeader

#### v0.4.0

- [功能] 将 rtmp pub session 的音视频转发给httpflv sub session
- [依赖] httpflv ServerSubSession 使用 naza connection
- [其他] 增加测试，加载flv文件后使用rtmp推流至服务器，然后分别使用rtmp和httpflv将流拉取下来，存成文件，判断和输入文件是否相等

#### v0.3.2

- [功能] 默认的rtmp地址
- [依赖] naza 更新为 0.4.3
- [架构调整] lal 中的服务器更名为 lals
- [其他] 从远端下载 flv 测试文件，跑单元测试
- [其他] test.sh 中加入更多 go tool
- [其他] 所有源码文件添加 MIT 许可证

#### v0.3.1

- [功能] 读取配置文件时，部分未配置的字段设置初始值
- [其他] build.sh 中 git信息单引号替换成双引号
- [其他] test.sh 中 加入 gofmt 检查
- [其他] 更新 naza -> 0.4.0

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
