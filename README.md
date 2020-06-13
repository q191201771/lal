<p align="center">
<a title="logo" target="_blank" href="https://github.com/q191201771/lal">
<img alt="Wide" src="https://pengrl.com/images/other/lallogo.png">
</a>
<br>
Go live stream lib/client/server and much more.
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

Go直播流媒体网络传输服务器，已支持RTMP，HTTP-FLV，HLS(m3u8+ts)，H264/AVC，H265/HEVC，AAC，GOP缓存，中继转推。更多功能持续迭代中。

### README 目录

1. 编译、运行、体验功能
2. 配置文件说明
3. 仓库目录框架
4. Roadmap
5. 文档
6. 联系我
7. 性能测试，测试过的第三方客户端

### 一. 编译、运行、体验功能

#### 编译

方式1，直接下载编译好的二进制可执行文件

上[最新发布版本页面](https://github.com/q191201771/lal/releases/latest)，下载对应平台编译好的二进制可执行文件的zip压缩包。

方式2，自己编译

```shell
# 不使用 Go module
$go get -u github.com/q191201771/lal
$cd $GOPATH/src/github.com/q191201771/lal
$./build.sh

# 使用 Go module
$export GOPROXY=https://goproxy.io
$git clone https://github.com/q191201771/lal.git
$cd lal
$./build.sh
```

#### 运行

```shell
$./bin/lalserver -c conf/lalserver.conf.json
```

#### 体验功能

快速体验lalserver服务器见：[常见推拉流客户端软件的使用方式](https://pengrl.com/p/20051/)

### 二. 配置文件说明

```
{
  "rtmp": {
    "enable": true,   // 是否开启rtmp服务的监听
    "addr": ":19350", // RTMP服务监听的端口，客户端向lalserver推拉流都是这个地址
    "gop_num": 2      // RTMP拉流的GOP缓存数量，加速秒开
  },
  "httpflv": {
    "enable": true,             // 是否开启HTTP-FLV服务的监听
    "sub_listen_addr": ":8080", // HTTP-FLV拉流地址
    "gop_num": 2
  },
  "hls": {
    "enable": true,               // 是否开启HLS服务的监听
    "sub_listen_addr": ":8081",   // HLS监听地址
    "out_path": "/tmp/lal/hls/",  // HLS文件保存根目录
    "fragment_duration_ms": 3000, // 单个TS文件切片时长，单位毫秒
    "fragment_num": 6             // M3U8文件列表中TS文件的数量
  },
  "relay_push": {
    "enable": false, // 是否开启中继转推功能，开启后，自身接收到的所有流都会转推出去
    "addr_list":[    // 中继转推的对端地址，支持填写多个地址，做1对n的转推。格式举例 "127.0.0.1:19351"
    ]
  },
  "pprof": {
    "enable": true,  // 是否开启Go pprof web服务的监听
    "addr": ":10001" // Go pprof web地址
  },
  "log": {
    "level": 1,                         // 日志级别，1 debug, 2 info, 3 warn, 4 error, 5 fatal
    "filename": "./logs/lalserver.log", // 日志输出文件
    "is_to_stdout": true,               // 是否打印至标志控制台输出
    "is_rotate_daily": true,            // 日志按天翻滚
    "short_file_flag": true,            // 日志末尾是否携带源码文件名以及行号的信息
    "assert_behavior": 1                // 日志断言的行为，1 只打印错误日志 2 打印并退出程序 3 打印并panic
  }
}
```

### 三. 仓库目录框架

简单来说，源码在`pkg/`，`app/lalserver/`，`app/demo/`三个目录下。

- `pkg/`：存放各package包，供本repo的程序以及其他业务方使用
- `app/lalserver`：基于lal编写的一个通用流媒体服务器程序入口
- `app/demo/`：存放各种基于`lal/pkg`开发的小程序（小工具），一个子目录是一个程序，详情见各源码文件中头部的说明

```
pkg/                     ......
|-- rtmp/                ......RTMP协议
|-- httpflv/             ......HTTP-FLV协议
|-- hls/                 ......HLS协议
|-- logic/               ......lalserver服务器程序的上层业务逻辑
|-- aac/                 ......音频AAC编码格式相关
|-- avc/                 ......视频H264/AVC编码格式相关
|-- hevc/                ......视频H265/HEVC编码格式相关
|-- innertest/           ......测试代码

app/                     ......
|-- lalserver/           ......流媒体服务器lalserver的main函数入口

|-- demo/                ......
    |-- analyseflv       ......
    |-- analysehls       ......
    |-- flvfile2rtmppush ......
    |-- rtmppull         ......
    |-- httpflvpull      ......
    |-- modflvfile       ......
    |-- flvfile2es       ......
    |-- learnts          ......
    |-- tscmp            ......

conf/                    ......配置文件目录
bin/                     ......可执行文件编译输出目录
```

后续我再画些源码架构图。

目前唯一的第三方依赖（我自己写的Go基础库）： [github.com/q191201771/naza](https://github.com/q191201771/naza)


### 四. Roadmap

#### lalserver服务器功能

- [x] **pub接收推流：** RTMP
- [x] **sub接收拉流：** RTMP，HTTP-FLV，HLS(m3u8+ts)
- [x] **音频编码格式：** AAC
- [x] **视频编码格式：** H264/AVC，H265/HEVC
- [x] **GOP缓存：** 用于秒开
- [x] **relay push中继转推：** RTMP
- [ ] RTMP回源
- [ ] HTTP-FLV回源
- [ ] 静态转推、回源
- [ ] 动态转推、回源
- [ ] rtsp
- [ ] rtp/rtcp
- [ ] webrtc
- [ ] udp quic
- [ ] udp srt
- [ ] udp kcp
- [ ] mp4
- [ ] 分布式。提供与外部调度系统交互的接口。应对多级分发场景，或平级源站类型场景
- [ ] 调整框架代码
- [ ] 各种推流、拉流客户端兼容性测试
- [ ] 和其它主流服务器的性能对比测试
- [ ] 整理日志
- [ ] 稳定性测试

### 五. 文档

* [流媒体音视频相关的点我](https://pengrl.com/categories/%E6%B5%81%E5%AA%92%E4%BD%93%E9%9F%B3%E8%A7%86%E9%A2%91/)
* [Go语言相关的点我](https://pengrl.com/categories/Go/)
* [我写的其他文章](https://pengrl.com/all/)

### 六. 联系我

扫码加我微信（微信号： q191201771），进行技术交流或扯淡。微信群已开放，加我好友后可拉进群。

<img src="https://pengrl.com/images/yoko_vx.jpeg" width="180" height="180" />

### 七. 性能测试，测试过的第三方客户端

见[TEST.md](https://github.com/q191201771/lal/blob/master/TEST.md)

### 八. 项目star趋势图

觉得这个repo还不错，就点个star支持一下吧 :)

[![Stargazers over time](https://starchart.cc/q191201771/lal.svg)](https://starchart.cc/q191201771/lal)

