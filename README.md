<p align="center">
<a title="logo" target="_blank" href="https://github.com/q191201771/lal">
<img alt="Wide" src="https://pengrl.com/images/other/lallogo.png">
</a>
<br>
Go语言编写的直播流媒体网络传输服务器
<br><br>
<a title="TravisCI" target="_blank" href="https://www.travis-ci.org/q191201771/lal"><img src="https://www.travis-ci.org/q191201771/lal.svg?branch=master"></a>
<a title="codecov" target="_blank" href="https://codecov.io/gh/q191201771/lal"><img src="https://codecov.io/gh/q191201771/lal/branch/master/graph/badge.svg?style=flat-square"></a>
<a title="goreportcard" target="_blank" href="https://goreportcard.com/report/github.com/q191201771/lal"><img src="https://goreportcard.com/badge/github.com/q191201771/lal?style=flat-square"></a>
<br>
<a title="codeline" target="_blank" href="https://github.com/q191201771/lal"><img src="https://sloc.xyz/github/q191201771/lal/?category=code"></a>
<a title="license" target="_blank" href="https://github.com/q191201771/lal/blob/master/LICENSE"><img src="https://img.shields.io/badge/license-MIT-brightgreen.svg?style=flat-square"></a>
<a title="lastcommit" target="_blank" href="https://github.com/q191201771/lal/commits/master"><img src="https://img.shields.io/github/commit-activity/m/q191201771/lal.svg?style=flat-square"></a>
<a title="commitactivity" target="_blank" href="https://github.com/q191201771/lal/graphs/commit-activity"><img src="https://img.shields.io/github/last-commit/q191201771/lal.svg?style=flat-square"></a>
<br>
<a title="pr" target="_blank" href="https://github.com/q191201771/lal/pulls"><img src="https://img.shields.io/github/issues-pr-closed/q191201771/lal.svg?style=flat-square&color=FF9966"></a>
<a title="hits" target="_blank" href="https://github.com/q191201771/lal"><img src="https://hits.b3log.org/q191201771/lal.svg?style=flat-square"></a>
<a title="language" target="_blank" href="https://github.com/q191201771/lal"><img src="https://img.shields.io/github/languages/count/q191201771/lal.svg?style=flat-square"></a>
<a title="toplanguage" target="_blank" href="https://github.com/q191201771/lal"><img src="https://img.shields.io/github/languages/top/q191201771/lal.svg?style=flat-square"></a>
<a title="godoc" target="_blank" href="https://godoc.org/github.com/q191201771/lal"><img src="http://img.shields.io/badge/godoc-reference-5272B4.svg?style=flat-square"></a>
<br><br>
<a title="watcher" target="_blank" href="https://github.com/q191201771/lal/watchers"><img src="https://img.shields.io/github/watchers/q191201771/lal.svg?label=Watchers&style=social"></a>&nbsp;&nbsp;
<a title="star" target="_blank" href="https://github.com/q191201771/lal/stargazers"><img src="https://img.shields.io/github/stars/q191201771/lal.svg?label=Stars&style=social"></a>&nbsp;&nbsp;
<a title="fork" target="_blank" href="https://github.com/q191201771/lal/network/members"><img src="https://img.shields.io/github/forks/q191201771/lal.svg?label=Forks&style=social"></a>&nbsp;&nbsp;
</p>

---

Go语言编写的直播流媒体网络传输服务器。本项目遵循的原则或者说最终目标是：

* ~~没有蛀。。~~
* 可读可维护。框架清晰，模块化，按业务逻辑层，协议层，传输层分层。
* 可快速集成各种协议（rtmp / http-flv / hls, rtp / rtcp / webrtc, quic, srt, over tcp, over udp...）
* 高性能

目前 rtmp / http-flv 部分基本完成了。第一个目标大版本会实现直播源站以及直播 CDN 分发相关的功能。

### README 目录

* 源码框架
* 编译和运行
* 配置文件说明
* 性能测试
* 测试过的第三方客户端
* Roadmap
* 联系我

### 源码框架

简单来说，源码在`app/`和`pkg/`两个目录下，后续我再画些源码架构图。

```
pkg/                  ......源码包
|-- aac/              ......音频 aac 编解码格式相关
|-- avc/              ......视频 avc h264 编解码格式相关
|-- rtmp/             ......rtmp 协议
|-- httpflv/          ......http-flv 协议
|-- logic/            ......lals 服务器的上层业务

app/                  ......各种 main 包的源码文件，一个子目录对应一个 main 包，即对应可生成一个可执行文件
|-- lals/             ......[最重要的] 流媒体服务器
|-- flvfile2rtmppush  ......// rtmp 推流客户端，读取本地 flv 文件，使用 rtmp 协议推送出去
                            //
                            // 支持循环推送：文件推送完毕后，可循环推送（rtmp push 流并不断开）
                            // 支持推送多路流：相当于一个 rtmp 推流压测工具
                            
|-- rtmppull          ......// rtmp 拉流客户端，从远端服务器拉取 rtmp 流，存储为本地 flv 文件
                            //
                            // 另外，作为一个 rtmp 拉流压测工具，已经支持：
                            // 1. 对一路流拉取 n 份
                            // 2. 拉取 n 路流
                            
|-- httpflvpull       ......http-flv 拉流客户端
|-- modflvfile        ......修改本地 flv 文件
|-- flvfile2es        ......将本地 flv 文件分离成 h264/avc es 流文件以及 aac es 流文件
bin/                  ......可执行文件编译输出目录
conf/                 ......配置文件目录
```

目前唯一的第三方依赖（我自己写的 Go 基础库）： [github.com/q191201771/naza](https://github.com/q191201771/naza)

### 编译和运行

```
# 不使用 Go module
$go get -u github.com/q191201771/lal
# cd into $GOPATH/src/github.com/q191201771/lal
$./build.sh

# 使用 Go module
$export GOPROXY=https://goproxy.cn
$git clone https://github.com/q191201771/lal.git && cd lal && ./build.sh

# 运行
$./bin/lals -c conf/lals.conf.json
```

### 配置文件说明

```
{
  "rtmp": {
    "addr": ":19350" // rtmp服务监听的端口
  },
  "httpflv": {
    "sub_listen_addr": ":8080"
  },
  "log": {
    "level": 1,                    // 日志级别，1 debug, 2 info, 3 warn, 4 error, 5 fatal
    "filename": "./logs/lals.log", // 日志输出文件
    "is_to_stdout": true,          // 是否打印至标志控制台输出
    "is_rotate_daily": true,       // 日志按天翻滚
    "short_file_flag": true        // 日志末尾是否携带源码文件名以及行号的信息
  },
  "pprof": {
    "addr": ":10001" // Go pprof web 地址
  }
}
```

其它放在代码中的配置：

- [rtmp/var.go](https://github.com/q191201771/lal/blob/master/pkg/rtmp/var.go)
- [httpflv/var.go](https://github.com/q191201771/lal/blob/master/pkg/httpflv/var.go)

### 性能测试

测试场景一：持续推送 n 路 rtmp 流至 lals（没有拉流）

| 推流数量 | CPU 占用 | 内存占用（RES） |
| - | - | - |
| 1000 | （占单个核的）16% | 104MB |

测试场景二：持续推送1路 rtmp 流至 lals，使用 rtmp 协议从 lals 拉取 n 路流

| 拉流数量 | CPU 占用 | 内存占用（RES） |
| - | - | - |
| 1000 | （占单个核的）30% | 120MB |

测试场景三： 持续推送 n 路 rtmp 流至 lals，使用 rtmp 协议从 lals 拉取 n 路流（推拉流为1对1的关系）

| 推流数量 | 拉流数量 | CPU 占用 | 内存占用（RES） |
| - | - | - | - |
| 1000 | 1000 | 125% | 464MB |

* 测试机：32核16G（lals 服务器和压测工具同时跑在这一个机器上）
* 压测工具：lal 中的 `/app/flvfile2rtmppush` 以及 `/app/rtmppull`
* 推流码率：使用 `srs-bench` 中的 flv 文件，大概200kbps
* lals 版本：基于 git commit: xxx

*由于测试机是台共用的机器，上面还跑了许多其他服务，这里列的只是个粗略的数据，还待做更多的性能分析以及优化。如果你对性能感兴趣，欢迎进行测试并将结果反馈给我。*

### 测试过的第三方客户端

```
推流端：
- OBS 21.0.3(mac)
- ffmpeg 3.4.2(mac)
- srs-bench (srs项目配套的一个压测工具)
- flvfile2rtmppush (lal app中的rtmp推流客户端)

拉流端：
- VLC 2.2.6(mac)
- MPV 0.29.1(mac)
- ffmpeg 3.4.2(mac)
- srs-bench (srs项目配套的一个压测工具)
```

### Roadmap

**有建议、意见、bug、功能等等欢迎提 issue 啊，100% 会回复的。**

lals 服务器目标版本功能如下：

**v1.0.0**

- 接收 rtmp 推流 [DONE]
- 转发给 rtmp 拉流 [DONE]
- 转发给 http-flv 拉流 [DONE]
- AAC [DONE]
- H264 [DONE]
- 各种 rtmp 推流、拉流客户端兼容性测试
- 和其它主流 rtmp 服务器的性能对比测试
- 整理日志
- 调整框架代码
- 稳定性测试

**v2.0.0**

- Gop 缓存功能

**v3.0.0**

- rtmp 转推
- rtmp 回源
- http-flv 回源

**v4.0.0**

- udp quic srt
- rtp/rtcp
- webrtc

**v5.0.0**

- 分布式。提供与外部调度系统交互的接口。应对多级分发场景，或平级源站类型场景

**没有排到预期版本中的功能**

- hls
- h265

### 文档

* [rtmp handshake | rtmp握手简单模式和复杂模式](https://pengrl.com/p/20027/)
* [rtmp协议中的chunk stream id, message stream id, transaction id, message type id](https://pengrl.com/p/25610/)

### 联系我

欢迎扫码加我微信，进行技术交流或扯淡。

<img src="https://pengrl.com/images/yoko_vx.jpeg" width="180" height="180" />
