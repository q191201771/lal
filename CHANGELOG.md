#### v0.15.1

- [fix] 配置文件没有开启HTTPS-FLV时，错误使用nil对象导致崩溃

#### v0.15.0

- [feat] 支持HTTP-TS sub长连接拉流
- [feat] 支持HTTPS-FLV
- [feat] 支持跨域请求：HTTP-FLV sub, HTTP-TS sub, HLS这几个HTTP类型的拉流
- [feat] 支持HLS录制与回放（在原有HLS直播的基础之上）
- [fix] 修复record m3u8文件无法更新的问题
- [fix] 修复rtsp pub无法接收IPv6 RTP数据的问题
- [fix] 修复windows平台编译失败的问题（单元测试package innertest中使用syscall.Kill导致）
- [feat] demo pullrtmp2hls: 新增demo，从远端服务器拉取rtmp流，存储为本地hls文件
- [feat] 新增package alpha/stun，学习stun协议
- [feat] 部分rtsp pub支持h265的代码，未完全完成

#### v0.14.0

- [feat] lalserver实现rtsp pub功能。支持接收rtsp(rtp/rtcp)推流，转换为rtmp,httpflv,hls格式供拉流使用
- [feat] hls.Muxer释放时，向m3u8文件写入`#EXT-X-ENDLIST`
- [refactor] 新增package sdp，rtprtcp
- [refactor] 新增package base，整理lal项目中各package的依赖关系
- [refactor] 新增package mpegts，将部分package hls中代码抽离出来
- [refactor] 重写package aac
- [feat] 在各协议的标准字段中写入lal版本信息
- [fix] group Dispose主动释放所有内部资源，与中继转推回调回来的消息，做同步处理，避免崩溃
- [fix] package avc: 修复解析sps中PicOrderCntType为2无法解析的bug
- [refactor] 重命名app/demo中的一些程序名
- [feat] package rtmp: 增加BuildMetadata函数
- [test] 使用wontcry30s.flv作为单元测试用的音视频文件
- [chore] 使用Makefile管理build, test
- [doc] 增加文档: https://pengrl.com/p/20080/
- [log] 整理所有session的日志

#### v0.13.0

- [feat] package httpflv: pull拉流时，携带url参数
- [feat] package avc: 提供一些AVCC转AnnexB相关的代码。学习解析SPS、PPS内部的字段
- [fix] package rtmp: 打包rtmp chunk时扩展时间戳的格式。避免时间戳过大后，发送给vlc的数据无法播放。
- [fix] package hls: 写ts视频数据时，流中没有spspps导致崩溃
- [fix] package logic: 修复重复创建group.RunLoop协程的bug
- [perf] package logic: 广播数据时，内存块不做拷贝
- [perf] package hls: 切片188字节buffer复用一块内存
- [refactor] package hls: 使用package avc
- [refactor] 所有回调函数的命名格式，从CB后缀改为On前缀
- [refactor] 整理日志
- [style] Nalu更改为NALU
- [doc] 增加PR规范
- [test] innertest中对hls生成的m3u8和ts文件做md5验证
- [chore] 下载单元测试用的test.flv失败，本地文件大小为0时，去备用地址下载

#### v0.12.0

- [feat] lalserver增加回源功能
- [fix] rtmp.AMF0.ReadObject函数内部，增加解析子类型EcmaArray。避免向某些rtmp服务器推流时，触发断言错误
- [fix] 解析rtmp metadata时，兼容Object和Array两种外层格式
- [refactor] 重写了lalserver的中继转推的代码

#### v0.11.0

- [feat] lalserver增加中继转推(relay push)功能，可将接收到的推流(pub)转推(push)到其他rtmp类型的服务器，支持1对n的转推
- [feat] package rtmp: 新增函数amf0::ReadArray，用于解析amf array数据
- [refactor] `rtmp/client_push_session`增加当前会话连接状态
- [fix] demo/analyseflv: 修复解析metadata的bug
- [perf] httpflv协议关闭时，不做httpflv的GOP缓存
- [refactor] logic中的配置变更为全局变量
- [refactor] lal/demo移动到lal/app/demo
- [refactor] 日志整理

#### v0.10.0

- [refactor] app/lals重命名为app/lalserver，避免描述时容易和lal造成混淆
- [refactor] 将app/lalserver的大部分逻辑代码移入pkg/logic中
- [test] 将所有package的Server、Session等内容的实例测试收缩至package innertest中，多个package都可以共用它做单元测试
- [refactor] lalserver配置中增加显式enable字段，用于开启关闭特定协议
- [refactor] 各package的Server对象增加独立的Listen函数，使得绑定监听端口失败时上层可以第一时间感知
- [feat] demo/analyseflv，增加I帧间隔检查，增加metadata分析
- [fix] package avc: 修复函数CalcSliceType解析I、P、B帧类型的bug
- [fix] package hls: 检查输入的rtmp message是否完整，避免非法数据造成崩溃
- [perf] gop缓存使用环形队列替换FIFO动态切片队列
- [refactor] package aac: 函数ADTS::PutAACSequenceHeader检查输入切片长度
- [refactor] package aac: 删除函数CaptureAAC
- [feat] 增加demo/learnrtsp，pkg/rtsp，开始学习rtsp

#### v0.9.0

- [feat] 新增HLS直播功能
- [fix] 接收rtmp数据时，同一个message的多个chunk混合使用fmt1，2时，可能出现时间戳多加的情况
- [refactor] 将app目录下除lals的其他应用移入demo目录下
- [feat] 新增两个demo：analyseflv和analysehls，分别用于拉取HTTP-FLV和HLS的流，并进行分析
- [fix] 修改rtmp简单握手，修复macOS ffmpeg 4.2.2向lals推rtmp流时的握手警告

#### v0.8.1

- [feat] 新package hevc
- [fix] windows平台缺少USER1信号导致编译失败
- [fix] gop缓存时，不以I帧开始的流会崩溃
- [chore] 提供各平台二进制可执行文件的压缩包
- [doc] package aac增加一些注释
- [refactor] 使用naza v0.10.0

#### v0.8.0

- [feat] 支持H265/HEVC
- [feat] 支持GOP缓存

#### v0.7.0

- [fix] package logic: 转发 rtmp metadata 时，message header 中的 len 字段可能和 body 实际长度不一致
- [feat] rtmp.AVMsg 增加判断包中音视频数据是否为 seq header 等函数
- [feat] app/httpflvpull 使用 naza/bitrate 来统计音频和视频的带宽
- [refactor] logic config 的部分配置移动至 app/lals 中
- [refactor] logic 增加 LazyChunkDivider 组织代码
- [log] package rtmp: 一些错误情况下，对接收到包 dump hex
- [test] 测试推送 n 路 rtmp 流至 lals，再从 lals 拉取这 n 路 rtmp 流的性能消耗
- [doc] README 中增加测试过的推拉流客户端
- [dep] update naza -> v0.7.1

#### v0.6.0

- package rtmp: 结构体的属性重命名 AVMsg.Message -> AVMsg.Payload
- app/flvfile2rtmppush: 支持推送多路 rtmp 流，相当于一个压测工具
- app/rtmppull: 支持对特定的一路流并发拉取多份，相当于一个压测工具
- README 中补充性能测试结果

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
