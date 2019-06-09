<p align="center">
<img alt="Wide" src="https://pengrl.com/images/other/lallogo.png">
<br>
Go语言编写的流媒体 库 / 客户端 / 服务器
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
|-- lal/              ......lal main包的源码文件
bin/                  ......可执行文件输出目录
conf/                 ......配置文件目录
demo/                 ......各种demo类型的main包，一个子目录对应一个demo，即对应可生成一个可执行文件
pkg/                  ......源码包
    |-- bele/         ......大小端操作相关
    |-- httpflv/      ......http flv协议
    |-- log/          ......日志相关
    |-- rtmp/         ......rtmp协议
    |-- util/         ......帮助类包
```

#### 编译和运行

```
$go get -u github.com/q191201771/lal
# cd into lal
$./build.sh

# ./bin/lal -c <配置文件> -l <日志配置文件>，比如：
$./bin/lal -c conf/lal.conf.json -l conf/log.dev.xml
```

#### 配置文件说明

```
{
  "sub_idle_timeout": 10, // 往客户端发送数据时的超时时间
  "gop_cache_num": 2,     // gop缓存个数，如果设置为0，则只缓存seq header，不缓存gop数据
  "httpflv": {
    "sub_listen_addr": ":8080" // http-flv拉流地址
  },
  "rtmp": {
    "addr": ":8081" // rtmp服务器监听端口，NOTICE rtmp服务器部分正在开发中
  }
  "pull": { // 如果设置，则当客户端连接lal拉流而lal上该流不存在时，lal会去该配置中的地址回源拉流至本地再转发给客户端
    "type": "httpflv",      // 回源类型，支持"httpflv" 或 "rtmp"
    "addr": "pull.xxx.com", // 回源地址
    "connect_timeout": 2,   // 回源连接超时时间
    "read_timeout": 10,     // 回源读取数据超时时间
    "stop_pull_while_no_sub_timeout": 3000 // 回源的流超过多长时间没有客户端播放，则关闭回源的流
  }
}
```

TODO 日志配置文件说明

#### 简单压力测试

在一台双核腾讯云主机，以后会做更详细的测试以及性能优化。

| ~ | httpflv pull | httpflv sub | 平均%CPU | 入带宽 | 出带宽 | 内存RES |
| - | - | - | - | - | - | - |
| ~ | 1 | 300 | 8.8% | 1.5Mb | 450Mb | 36m |
| ~ | 300 | 300->0 | 18% | 450Mb | ->0Mb | 1.3g |

#### 依赖

* cihub/seelog
* stretchr/testify/assert

#### roadmap

正在实现rtmp服务器部分

#### 文档

* [rtmp handshake | rtmp握手简单模式和复杂模式](https://pengrl.com/p/20027/)
