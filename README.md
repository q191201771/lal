<p align="center">
<img alt="Wide" src="https://pengrl.com/images/other/lallogo.png">
<br>
Go语言编写的流媒体 库 / 客户端 / 服务端
<br><br>
<a title="TravisCI" target="_blank" href="https://www.travis-ci.org/q191201771/lal"><img src="https://www.travis-ci.org/q191201771/lal.svg?branch=master"></a>
<a title="codecov" target="_blank" href="https://codecov.io/gh/q191201771/lal"><img src="https://codecov.io/gh/q191201771/lal/branch/master/graph/badge.svg?style=flat-square"></a>
<a title="goreportcard" target="_blank" href="https://goreportcard.com/report/github.com/q191201771/lal"><img src="https://goreportcard.com/badge/github.com/q191201771/lal?style=flat-square"></a>
<br>
<a title="codesize" target="_blank" href="https://github.com/q191201771/lal"><img src="https://img.shields.io/github/languages/code-size/q191201771/lal.svg?style=flat-square?style=flat-square"></a>
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

#### 工程目录说明

简单来说，主要源码在`app/`和`pkg/`两个目录下，后续我再画些源码架构图。

```
app/                  ......各种main包的源码文件，一个子目录对应一个main包，即对应可生成一个可执行文件
|-- lal/              ......[最重要的] 流媒体服务器
|-- flvfile2es        ......将本地flv文件分离成h264/avc es流文件以及aac es流文件
|-- flvfile2rtmppush  ......rtmp推流客户端，输入是本地flv文件，文件推送完毕后，可循环推送（rtmp push流并不断开）
|-- httpflvpull       ......http-flv拉流客户端
|-- modflvfile        ......修改本地flv文件
|-- rtmppull          ......rtmp拉流客户端，存储为本地flv文件
pkg/                  ......源码包
|-- aac/              ......音频aac编解码格式相关
|-- avc/              ......视频avc h264编解码格式相关
|-- httpflv/          ......http-flv协议
|-- rtmp/             ......rtmp协议
bin/                  ......可执行文件编译输出目录
conf/                 ......配置文件目录
```

#### 编译和运行

```
$go get -u github.com/q191201771/lal
# cd into $GOPATH/src/github.com/q191201771/lal
$./build.sh

$./bin/lal -c conf/lal.conf.json

#如果使用 go module
$git clone https://github.com/q191201771/lal.git && cd lal && ./build.sh
```

#### 配置文件说明

```
{
  "rtmp": {
    "addr": ":19350" // rtmp服务监听的端口
  },
  "log": {
    "level": 1,                   // 日志级别，1 debug, 2 info, 3 warn, 4 error, 5 fatal
    "filename": "./logs/lal.log", // 日志输出文件
    "is_to_stdout": true,         // 是否打印至标志控制台输出
    "is_rotate_daily": true,      // 日志按天翻滚
    "short_file_flag": true       // 日志末尾是否携带源码文件名以及行号的信息
  },
  "pprof": {
    "addr": ":10001" // Go pprof web 地址
  }
}
```

### 测试过的客户端

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

#### roadmap

有建议、意见、bug、功能等等欢迎提 issue 啊，100% 会回复的。

lal 服务器目标版本roadmap如下：

**v1.0.0**

实现 rtmp 转发服务器。目前已经基本完成了。大概在十一假期发布。

- 各种 rtmp 推流、拉流客户端兼容性测试
- 和其它主流 rtmp 服务器的性能对比测试
- 整理日志
- 调整框架代码
- 稳定性测试

**v1.0.0**

- Gop 缓存功能
- http-flv 拉流
- rtmp 转推
- rtmp 回源
- http-flv 回源

**v2.0.0**

- hls

**v3.0.0**

- udp quic srt
- rtp/rtcp
- webrtc

**v4.0.0**

- 分布式。提供与外部调度系统交互的接口。应对多级分发场景，或平级源站类型场景


最终目标：

* 实现一个支持多种流媒体协议（比如rtmp, http-flv, hls, rtp/rtcp 等），多种底层传输协议（比如tcp, udp, srt, quic 等）的服务器
* 所有协议都以模块化的库形式提供给需要的用户使用
* 提供多种协议的推流客户端、拉流客户端，或者说演示demo

#### 依赖

- [github.com/q191201771/nezha](https://github.com/q191201771/nezha) 我自己写的Go基础库

#### 文档

* [rtmp handshake | rtmp握手简单模式和复杂模式](https://pengrl.com/p/20027/)
* [rtmp协议中的chunk stream id, message stream id, transaction id, message type id](https://pengrl.com/p/25610/)
