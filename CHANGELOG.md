#### v0.30.1 (2022-06-15)

- [feat] HTTP-API：新增start/stop_relay_pull接口，支持rtmp和rtsp，支持设置超时时间，自动关闭，重试次数，rtsp类型等参数
- [feat] HTTP-API：kick_session接口支持踢掉pub/sub/pull类型的session
- [feat] HTTP-Notify：增加on_relay_pull_start和on_relay_pull_stop回调
- [feat] HTTP-Notify：增加hls生成ts文件的事件回调
- [feat] rtmp: client端支持rtmps加密传输
- [feat] rtmp: client端支持adobe auth验证
- [feat] rtsp: server端支持basic/digest auth验证
- [feat] lalserver: 运行参数-p可设置当前工作路径
- [feat] package gb28181: 大体完成ps协议解析
- [feat] 新增remux.Rtmp2AvPacketRemuxer，方便和ffmpeg库协作
- [fix] rtsp: 修复url path路径不存在时，url解析失败的问题
- [fix] rtmp: 解析amf, object中嵌套object导致崩溃
- [fix] rtmp: ChunkComposer的error日志中的对象写错导致崩溃
- [fix] 修复rtmp转ts时，265判断错误
- [fix] lalserver: 修复竞态条件下接收rtsp流崩溃的bug
- [fix] lalserver: relay push判空错误导致崩溃
- [chore] release发版时，增加arm32, arm64, macos arm对应的二进制文件
- [refactor] 新增package h2645
- [refactor] 将所有session的ISessionStat的实现聚合到BasicSessionStat
- [refactor] rename HttpSubSession -> BasicHttpSubSession
- [refactor] HTTP-API: 所有事件都包含的公共字段聚合到EventCommonInfo中
- [opt] aac: 补全AscContext.samplingFrequencyIndex采样率的取值
- [log] 访问非法HTTP-API路径时打印警告日志

#### v0.29.1 (2022-05-03)

- [feat] lalserver: 支持集成第三方协议的输入流 https://pengrl.com/#/customize_pub
- [feat] rtmp: pull session增加ack应答，提高兼容性
- [opt] rtsp: lalserver增加配置项`rtsp->out_wait_key_frame_flag`，用于控制发送rtsp数据时，是否等待关键帧再发送
- [opt] 增强健壮性，检查rtmp消息长度有效性
- [fix] 增强兼容性，rtmp转mpegts时，使用nalu中的sps和pps
- [fix] lalserver鉴权: 修复rtmp拉流鉴权的问题
- [fix] 解析H265类型不够全面，导致推流失败 #140
- [fix] lalserver录制: 是否创建mpegts录制根目录由mpegts录制开关控制
- [fix] demo: dispatch调度程序检测保活时间单位错误
- [perf] mpegts: 加大内存预分配大小

#### v0.28.0 (2022-03-27)

- [feat] httpts: 支持gop缓冲，提高秒开 #129
- [opt] hls: 增加delete_threshold配置，用于配置过期TS文件的保存时间 #120
- [opt] rtsp sub 改为异步发送
- [opt] lalserver: relay push增加超时检查，增加带宽统计
- [opt] lalserver: relay pull的rtmp流也转换为rtsp
- [opt] lalserver: rtsp sub也支持触发relay pull
- [fix] aac: 支持22050采样频率，修复该频率下转rtsp失败的问题
- [fix] avc: 增强兼容性，处理单个seq header中存在多个sps的情况 #135
- [fix] mpegts: 修复单音频场景，有一帧音频重复的问题
- [fix] rtsp: Basic auth的base64编码
- [fix] rtsp: 增强容错性，修复rtmp输入流没有seq header时，rtmp转rtsp内崩溃的问题
- [fix] lalserver: 优雅关闭pprof和http server
- [perf] mpegts: 优化转换mpegts的性能
- [refactor] 将转换mpegts的代码从package hls独立出来，移动到package remux中
- [refactor] lalserver: 大幅重构logic.Group，为支持插件化做准备
- [log] 支持独立设置单个pkg的日志配置 #62
- [log] rtmp和rtsp收包时添加trace级别日志 #63
- [log] rtmp: 优化定位问题的日志 #135
- [test] innertest增加单音频，单视频，httpts sub的测试

#### v0.27.1 (2022-01-23)

- [feat] 新增simple auth鉴权功能，见文档： https://pengrl.com/lal/#/auth
- [feat] httpflv: PullSession支持https，支持302跳转
- [feat] rtmp: client类型的session新增方法用于配置WriteBuf和ReadBuf大小，以及WriteChanSize
- [opt] rtmp: 收到ping request回应ping response
- [fix] rtmp: 增强兼容性，当收到的rtmp message中aac seq header payload长度为0时忽略，避免崩溃 #116
- [fix] rtmp: 增强兼容性，当收到的rtmp message中的payload长度为0时忽略 #112
- [opt] rtsp: 增强兼容性，处理rtsp信令中header存在没有转义的\r\n的情况
- [fix] rtsp: 增强兼容性，修复读取http返回header解析失败的bug #110
- [opt] https: 增强兼容性，服务初始化失败时打印错误日志而不是退出程序
- [opt] avc: 增强兼容性，分隔avcc格式的nal时，如果存在长度为0的nal则忽略
- [fix] sdp: 增强兼容性，fmtp内发生换行时做兼容性处理
- [fix] httpflv: 修复httpflv多级路径下无法播放的问题
- [opt] 整理完所有error返回值，error信息更友好
- [log] 通过配置文件控制group调试日志
- [log] rtsp: client信令增加错误日志
- [fix] 修复logic.Option.NotifyHandler首字母小写外部无法设置的问题
- [refactor] 将logic包中的DummyAudioFilter, GopCache, LazyRtmpChunkDivider, LazyRtmpMsg2FlvTag移入remux中
- [refactor] rtmp: base.Buffer移入naza中
- [chore] CI: 迁移到github action，已支持linux，macos平台，Go1.14和Go1.17，每次push代码和每周定时触发，并自动提交docker hub镜像
- [chore] 修复go vet signal unbound channel的警告
- [test] 提高测试覆盖，目前lal测试覆盖超过60%，文档中增加测试覆盖率徽章
- [test] innertest增加m3u8文件检测，增加http api
- [test] 测试各session的ISessionUrlContext接口
- [test] 修复base/url_test.go中的测试用例

#### v0.26.0 (2021-10-24)

- [perf] rtmp合并发送功能使用writev实现
- [feat] 兼容性: 运行时动态检查所有配置项是否存在
- [refactor] 可定制性: logic: 抽象出ILalServer接口；业务方可在自身代码中创建server，选择是否获取notify通知，以及使用api控制server
- [refactor] 兼容性: 两个不太标准的sdp格式(a=fmtp的前面或后面有多余的分号)
- [refactor] 兼容性: aac解析失败日志; 输入的rtp包格式错误; 输入的rtmp包格式错误; hls中分割nalu增加日志; base.HttpServerManager增加日志
- [refactor] 兼容性: 再增加一个配置文件默认搜索地址
- [refactor] 可读性: logic: ServerManager和Config不再作为全局变量使用；去除entry.go中间层；iface_impl.go移入innertest中；signal_xxx.go移入base中
- [refactor] 易用性: demo/pullrtsp2pushrtsp: 抽象出RtspTunnel结构体，一个对象对应一个转推任务
- [refactor] logic: 新增GroupManager，管理所有Group
- [chore] 配置文件中httpflv和httpts的url_pattern初始值改为没有限制
- [chore] 使用github actions做CI（替换掉之前的travisCI）
- [chore] 修复build.sh在linux下获取git tag信息失败报错的问题；去掉单元测试时不必要的错误日志
- [chore] 增加docker运行脚本run_docker.sh

#### v0.25.0 (2021-08-28)

- [feat] 为rtmp pub推流添加静音AAC音频(可动态检测是否需要添加；配置文件中可开启或关闭这个功能)
- [feat] 优化和统一所有client类型session的使用方式：session由于内部或对端原因导致关闭，外部不再需要显式调用Dispose函数释放资源
- [feat] 增强兼容性：rtsp digest auth时，如果缺少algorithm字段，回复时该字段默认设置为MD5
- [refactor] package avc: 重新实现sps的解析
- [refactor] 新增函数remux.FlvTag2RtmpChunks()
- [refactor] 增强健壮性：package rtmp: 对端协议错误时，主动关闭对端连接而不是主动panic
- [refactor] 整理logic/group的代码
- [refactor] httpflv.Sub和httpts.Sub显式调用base.HttpSubSession的函数
- [fix] rtsp信令打包中部分字段缺少空格
- [chore] 增强易用性：修改配置文件中的默认配置：hls、flv、mpegts的文件输出地址由绝对路径/tmp修改为相对路径./lal_record

#### v0.24.0 (2021-07-31)

- [feat] lalserver支持用rtsp sub协议拉取rtmp的pub推流 (#97)
- [feat] 新增demo pullrtmp2pushrtsp，可以从远端拉取rtmp流并使用rtsp转推出去 (#96)
- [feat] package rtprtcp: 支持h264，h265，aac rtp打包 (#83)
- [feat] package sdp: 支持sdp打包 (#82)
- [fix] 确保rtsp sub拉流从关键帧开始发送数据，避免因此引起的花屏
- [fix] rtsp: 提高兼容性。兼容rtsp auth同时存在Digest和Basic两种字段的情况
- [fix] rtsp: 提高兼容性。兼容rtsp摄像头的sdp中包含aac，但是没有config字段（后续也没有aac rtp包）的情况
- [fix] rtmp: 提高兼容性。兼容rtmp client session处理对端回复两次publish或play信令的情况
- [fix] rtmp: 提高兼容性。修复没有解析amf object中null类型数据导致和其他rtmp开源服务无法建连的问题 (#102)
- [fix] rtmp: 信令打包参考本地chunk size
- [fix] rtsp: 修复rtsp sub session没有正常释放导致协程泄漏的问题
- [fix] 修复lalserver arm32编译失败的问题 (#92)
- [fix] 修复lalserver http服务全部配置为不使用时崩溃的问题 (#58)
- [fix] 修复hls.Muxer没有设置回调会导致崩溃的问题 (#101)
- [fix] 修复demo calcrtmpdelay码率计算大了5倍的问题 (#58)
- [refactor] package httpflv: 新增FlvFilePump，可循环匀速读取flv文件
- [refactor] package aac: 增加adts, asc, seqheader间的转换代码；重构了整个包
- [refactor] package avc: 部分函数提供复用传入参数内存和新申请内存两种实现
- [refactor] demo benchrtmpconnect: 关闭日志，超时时长改为30秒，优化建连时长小于1毫秒的展示 (#58)
- [chore] 增加Dockerfile (#91)

#### v0.23.0 (2021-06-06)

- [feat] HTTP端口复用：HTTP-FLV, HTTP-TS, HLS可使用相同的监听端口。HTTPS也支持端口复用 #64
- [feat] HTTPS：HTTP-FLV，HTTP-TS，HLS都支持HTTPS。WebSocket-FLV，WebSocket-TS都支持WebSockets #76
- [feat] 配置HTTP流的URL路径: HTTP-FLV，HTTP-TS，HLS的URL路由路径可以在配置文件中配置 #77
- [feat] RTMP支持合并发送 #84
- [refactor] 重构整个项目的命名风格 #87
- [fix] RTMP GOP缓存设置为0时，可能花屏 #86
- [feat] 支持海康威视NVR、大华IPC的RTSP流（SDP不包含SPS、PPS等数据，而是通过RTP包发送） #74 #85
- [feat] 配置灵活易用话。增加`default_http`。HTTP-FLV，HTTP-TS，HLS可以独立配置监听地址相关的项，也可以使用公共的`default_http`
- [feat] HLS默认提供两种播放URL地址 #64
- [refactor] package hls: 将HTTP URL路径格式，文件存储路径格式，文件命名格式，映射关系抽象出来，业务方可在外层实现IPathSolver接口做定制 #77
- [feat] 增加几个默认的配置文件加载路径
- [feat] package rtprtcp: 增加用于将H264 Nalu包切割成RTP包的代码 #83
- [refactor] package avc: 增加拆分AnndexB和AVCC Nalu包的代码 #79
- [refactor] 重构httpflv.SubSession和httpts.SubSession的重复代码

#### v0.22.0 (2021-05-03)

- [feat] 录制新增支持：flv和mpegts文件。 录制支持列表见： https://pengrl.com/lal/#/LALServer (#14)
- [feat] h265新增支持： hls拉流，hls录制；http-ts拉流，mpegts录制。h265支持列表见： https://pengrl.com/lal/#/LALServer (#65)
- [feat] 拉流新增支持：websocket-flv，websocket-ts。拉流协议支持列表见： https://pengrl.com/lal/#/LALServer
- [feat] hls: 支持内存切片。 (#50)
- [fix] rtmp ClientSession握手，c2的发送时机，由收到s0s1s2改为收到s0s1就发送，解决握手失败的case。 (#42)
- [fix] rtsp h265: 转rtmp时处理错误导致无法播放
- [fix] rtsp h265: ffmpeg向lalserver推送rtsp h265报错。 (#55)
- [test] travis ci: 自动化单元测试os增加osx, windows, arch增加arm64, ppc64le, s390x。 (#59)
- [feat] rtmp ClientSession支持配置使用简单握手，复杂握手 (#68)

#### v0.21.0 (2021-04-11)

- [feat] package rtmp: 支持Aggregate Message
- [feat] lalserver: 新增配置项hls.cleanup_mode，支持三种清理hls文件的模式，具体说明见 https://pengrl.com/lal/#/ConfigBrief
- [feat] package rtsp: 支持aac fragment格式（一个音频帧被拆分成多个rtp包），之前这种aac格式可能导致崩溃
- [doc] 新增文章《rtmp中的各种ID》，见 https://pengrl.com/lal/#/RTMPID
- [doc] 新增文章《rtmp handshake握手之简单模式和复杂模式》，见 https://pengrl.com/lal/#/RTMPHandshake
- [fix] rtsp推流时，rtp包时间戳翻转导致的错误（比如长时间推流后hls一直强制切片）
- [fix] lalserver的group中，rtsp sub超时时，锁重入导致服务器异常阻塞不响应
- [fix] 修复mipsle架构下rtsp崩溃
- [fix] 修复lalserver中（rtsp.BaseInSession以及logic.Group）的一些竞态读写，https://github.com/q191201771/lal/issues/47
- [fix] demo: 两个拉httpflv流的demo，main函数退出前忘记等待拉流结束
- [refactor] package rtprtcp: 重构一些函数名
- [refactor] package rtprtcp: 重构rtp unpacker，业务方可以使用默认的container，protocol策略，也可以自己实现特定的协议解析组包策略
- [refactor] lalserver: 整理配置文件加载与日志初始化部分的代码
- [doc] 启用英文版本README.md作为github首页文档展示
- [doc] lalserver: 新增配置项conf_version，用于表示配置文件的版本号
- [doc] lalserver: 启动时日志中增加lal logo

#### v0.20.0 (2021-03-21)

- [feat] 新增app/demo/calcrtmpdelay，可用于测量rtmp服务器的转发延时，拉流支持rtmp/httpflv
- [feat] app/demo/pushrtmp 做压测时，修改为完全并行的模式
- [fix] 修复32位arm环境使用rtsp崩溃
- [refactor] 统一各Session接口
- [refactor] 使用新的unique id生成器，提高性能
- [doc] 新增文档 ffplay播放rtsp花屏 https://pengrl.com/lal/#/RTSPFFPlayBlur

#### v0.19.1 (2021-02-01)

- [fix] 获取group中播放者数量时锁没有释放，导致后续无法转发数据

#### v0.19.0 (2021-01-24)

- [feat] demo，新增app/demo/pullrtsp2pushrtsp，可拉取rtsp流，并使用rtsp转推出去
- [feat] demo，新增/app/demo/pullrtmp2pushrtmp，从远端服务器拉取RTMP流，并使用RTMP转推出去，支持1对n转推
- [feat] lalserver，运行参数中没指定配置文件时，尝试从几个常见位置读取
- [feat] windows平台下，执行程序缺少运行参数时，等待用户键入回车再退出程序，避免用户双击打开程序时程序闪退，看不到提示信息
- [feat] rtsp，支持auth basic验证
- [feat] rtsp，实现PushSession
- [feat] rtsp，所有Session类型都支持auth，interleaved
- [fix] rtsp，只有输入流中的音频和视频格式都支持时才使用queue，避免只有音频或视频时造成延迟增加
- [fix] rtsp，输入流只有单路音频或视频时，接收对象设置错误导致崩溃
- [fix] rtsp，client session的所有信令都处理401 auth
- [fix] rtsp，in session使用rtp over tcp时，收到sr回复rr
- [fix] rtsp，setup信令header中的transport字段区分record和play，record时添加mode=record
- [fix] avc，整体解析sps数据失败时，只解析最基础部分
- [refactor] rtsp，重构部分逻辑，聚合至sdp.LogicContext中
- [refactor] rtsp，新增ClientCommandSession，将PushSession和PullSession中共用的信令部分抽离出来
- [refactor] rtsp，新增BaseOutSession，将PushSession和SubSession中共用的发送数据部分抽离出来
- [refactor] rtsp，整理所有session，包含生命周期，ISessionStat、IURLContext、Interleaved收发等函数，整理debug日志
- [doc] 启动lal官方文档页： https://pengrl.com/lal
- [doc] 新增文档《rtmp url，以及vhost》： http://pengrl.com/lal/#/RTMPURLVhost
- [chore] Go最低版本要求从1.9上升到1.13

#### v0.18.0 (2020-12-27)

- [feat] 实现rtsp pull session
- [feat] demo，增加`/app/demo/pullrtsp2pushrtmp`，可拉取rtsp流，并使用rtmp转推出去
- [feat] demo，增加`/app/demo/pullrtsp`，可拉取rtsp流，存储为flv文件
- [feat] rtsp interleaved(rtp over tcp)模式。pub, sub, pull都已支持
- [feat] rtsp，pull支持auth digest验证
- [feat] rtsp，pull支持定时发送`GET_PARAMETER` rtsp message进行保活（对端支持的情况下）
- [feat] rtsp，输入流音频不是AAC格式时，保证视频流可正常remux成其他封装协议
- [feat] rtsp，pull开始时发送dummy rtp/rtcp数据，保证对端能成功发送数据至本地
- [feat] rtsp，修改rtsp.AVPacketQueue的行为：当音频或者视频数量队列满了后，直接出队而不是丢弃
- [feat] logic，rtsp pub转发给rtsp sub
- [feat] logic，rtsp pub转发给relay rtmp push
- [feat] remux，新增package，用于处理协议转封装
- [refactor] 重构所有client session解析url的地方
- [refactor] 所有session实现ISessionStat接口，用于计算、获取bitrate等流相关的信息
- [refactor] 所有session实现ISessionURLContext接口，用于获取流url相关的信息
- [refactor] rtmp/httpflv/rtsp，统一所有PullSession：超时形式；Pull和Wait函数
- [fix] rtsp，将以下包返回给上层：rtsp pub h265, single rtp packet, VPS, SPS, PPS, SEI
- [fix] sdp，修复解析及使用sdp错误的一些case
- [fix] aac，正确处理大于2字节的AudioSpecificConfig
- [fix] avc，尝试解析scaling matrix

#### v0.17.0 (2020-11-21)

- [feat] 增加HTTP Notify事件回调功能，见 https://pengrl.com/p/20101
- [feat] 增加`/app/demo/dispatch`示例程序，用于演示如何结合HTTP Notify加HTTP API构架一个lalserver集群
- [feat] 配置文件中增加配置项，支持配置是否清除过期流的HLS文件
- [feat] lalserver的session增加存活检查，10秒没有数据会主动断开连接
- [feat] lalserver的group没有sub拉流时，停止对应的pull回源
- [feat] HTTP API，增加`/api/ctrl/start_pull`接口，可向lalserver发送命令，主动触发pull回源拉流
- [feat] HTTP API，增加`/api/ctrl/kick_out_session`接口，可向lalserver发送命令，主动踢掉指定的session
- [feat] HTTP API `/api/stat/lal_info` 中增加`server_id`字段
- [feat] HTTP API，group结构体中增加pull结构体，包含了回源拉流的信息
- [fix] 配置文件静态relay push转推方式中，push rtmp url透传pub rtmp url的参数
- [chore] 增加`gen_tag.sh`，用于打tag

#### v0.16.0 (2020-10-23)

- [feat] rtsp pub h265（lal支持接收rtsp h265视频格式的推流）
- [feat] 增加HTTP API接口，用于获取服务的一些信息，具体见： https://pengrl.com/p/20100/
- [fix] 修复部分使用adobe flash player作为rtmp拉流客户端，拉流失败的问题
- [fix] 修复接收rtsp pub推流时，流只有视频（没有音频）流处理的问题

#### v0.15.1 (2020-09-19)

- [fix] 配置文件没有开启HTTPS-FLV时，错误使用nil对象导致崩溃

#### v0.15.0 (2020-09-19)

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

#### v0.14.0 (2020-08-23)

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

#### v0.13.0 (2020-07-18)

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

#### v0.12.0 (2020-06-20)

- [feat] lalserver增加回源功能
- [fix] rtmp.AMF0.ReadObject函数内部，增加解析子类型EcmaArray。避免向某些rtmp服务器推流时，触发断言错误
- [fix] 解析rtmp metadata时，兼容Object和Array两种外层格式
- [refactor] 重写了lalserver的中继转推的代码

#### v0.11.0 (2020-06-13)

- [feat] lalserver增加中继转推(relay push)功能，可将接收到的推流(pub)转推(push)到其他rtmp类型的服务器，支持1对n的转推
- [feat] package rtmp: 新增函数amf0::ReadArray，用于解析amf array数据
- [refactor] `rtmp/client_push_session`增加当前会话连接状态
- [fix] demo/analyseflv: 修复解析metadata的bug
- [perf] httpflv协议关闭时，不做httpflv的GOP缓存
- [refactor] logic中的配置变更为全局变量
- [refactor] lal/demo移动到lal/app/demo
- [refactor] 日志整理

#### v0.10.0 (2020-06-06)

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

#### v0.9.0 (2020-05-30)

- [feat] 新增HLS直播功能
- [fix] 接收rtmp数据时，同一个message的多个chunk混合使用fmt1，2时，可能出现时间戳多加的情况
- [refactor] 将app目录下除lals的其他应用移入demo目录下
- [feat] 新增两个demo：analyseflv和analysehls，分别用于拉取HTTP-FLV和HLS的流，并进行分析
- [fix] 修改rtmp简单握手，修复macOS ffmpeg 4.2.2向lals推rtmp流时的握手警告

#### v0.8.1 (2020-05-01)

- [feat] 新package hevc
- [fix] windows平台缺少USER1信号导致编译失败
- [fix] gop缓存时，不以I帧开始的流会崩溃
- [chore] 提供各平台二进制可执行文件的压缩包
- [doc] package aac增加一些注释
- [refactor] 使用naza v0.10.0

#### v0.8.0 (2020-04-18)

- [feat] 支持H265/HEVC
- [feat] 支持GOP缓存

#### v0.7.0 (2019-12-13)

- [fix] package logic: 转发 rtmp metadata 时，message header 中的 len 字段可能和 body 实际长度不一致
- [feat] rtmp.AVMsg 增加判断包中音视频数据是否为 seq header 等函数
- [feat] app/httpflvpull 使用 naza/bitrate 来统计音频和视频的带宽
- [refactor] logic config 的部分配置移动至 app/lals 中
- [refactor] logic 增加 LazyChunkDivider 组织代码
- [log] package rtmp: 一些错误情况下，对接收到包 dump hex
- [test] 测试推送 n 路 rtmp 流至 lals，再从 lals 拉取这 n 路 rtmp 流的性能消耗
- [doc] README 中增加测试过的推拉流客户端
- [dep] update naza -> v0.7.1

#### v0.6.0 (2019-11-08)

- package rtmp: 结构体的属性重命名 AVMsg.Message -> AVMsg.Payload
- app/flvfile2rtmppush: 支持推送多路 rtmp 流，相当于一个压测工具
- app/rtmppull: 支持对特定的一路流并发拉取多份，相当于一个压测工具
- README 中补充性能测试结果

#### v0.5.0 (2019-11-01)

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

#### v0.4.0 (2019-10-25)

- [功能] 将 rtmp pub session 的音视频转发给httpflv sub session
- [依赖] httpflv ServerSubSession 使用 naza connection
- [其他] 增加测试，加载flv文件后使用rtmp推流至服务器，然后分别使用rtmp和httpflv将流拉取下来，存成文件，判断和输入文件是否相等

#### v0.3.2 (2019-10-19)

- [功能] 默认的rtmp地址
- [依赖] naza 更新为 0.4.3
- [架构调整] lal 中的服务器更名为 lals
- [其他] 从远端下载 flv 测试文件，跑单元测试
- [其他] test.sh 中加入更多 go tool
- [其他] 所有源码文件添加 MIT 许可证

#### v0.3.1 (2019-09-30)

- [功能] 读取配置文件时，部分未配置的字段设置初始值
- [其他] build.sh 中 git信息单引号替换成双引号
- [其他] test.sh 中 加入 gofmt 检查
- [其他] 更新 naza -> 0.4.0

#### v0.3.0 (2019-09-27)

- [功能] package logic: 增加 func FlvTag2RTMPMsg
- [代码调整] package rtmp: ClientSession 和 ServerSession 使用 nezha 中的 connection 做连接管理
- [代码调整] package rtmp: 增加 struct ChunkDivider
- [代码调整] package rtmp: 调整一些接口
- [代码调整] package httpflv: 删除了 group， gop 相关的代码，后续会放入 package logic 中
- [测试] package rtmp: 增加 `example_test.go` 用于测试整个 rtmp 包的流程
- [其他] 更新 nezha -> 0.3.0

#### v0.2.0 (2019-09-21)

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

#### v0.1.0 (2019-09-12)

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

#### v0.0.1 (2019-09-03)

1. 提供 `/app/flvfile2rtmppush` 给业务方使用
