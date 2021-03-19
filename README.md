<p align="center">
<a title="logo" target="_blank" href="https://github.com/cfeeling/lal">
<img alt="Live And Live" src="https://pengrl.com/lal/_media/lallogo.png">
</a>
<br>
<a title="TravisCI" target="_blank" href="https://www.travis-ci.org/q191201771/lal"><img src="https://www.travis-ci.org/q191201771/lal.svg?branch=master"></a>
<a title="codecov" target="_blank" href="https://codecov.io/gh/q191201771/lal"><img src="https://codecov.io/gh/q191201771/lal/branch/master/graph/badge.svg?style=flat-square"></a>
<a title="goreportcard" target="_blank" href="https://goreportcard.com/report/github.com/cfeeling/lal"><img src="https://goreportcard.com/badge/github.com/cfeeling/lal?style=flat-square"></a>
<br>
<a title="codeline" target="_blank" href="https://github.com/cfeeling/lal"><img src="https://sloc.xyz/github/q191201771/lal/?category=code"></a>
<a title="license" target="_blank" href="https://github.com/cfeeling/lal/blob/master/LICENSE"><img src="https://img.shields.io/badge/license-MIT-brightgreen.svg?style=flat-square"></a>
<br>
<a title="hits" target="_blank" href="https://github.com/cfeeling/lal"><img src="https://hits.b3log.org/q191201771/lal.svg?style=flat-square"></a>
<a title="toplanguage" target="_blank" href="https://github.com/cfeeling/lal"><img src="https://img.shields.io/github/languages/top/q191201771/lal.svg?style=flat-square"></a>
<br>
</p>

---

lal是一个开源GoLang直播流媒体网络传输项目，包含三个主要组成部分：

- lalserver：流媒体转发服务器。类似于`nginx-rtmp-module`等应用，但支持更多的协议，提供更丰富的功能。[lalserver简介](https://pengrl.com/lal/#/LALServer)
- demo：一些小应用，比如推、拉流客户端，压测工具，流分析工具，调度示例程序等。类似于ffmpeg、ffprobe等应用。[Demo简介](https://pengrl.com/lal/#/DEMO)
- pkg：流媒体协议库。类似于ffmpeg的libavformat等库。

**lal源码package架构图：**

![lal源码package架构图](https://pengrl.com/lal/_media/lal_src_fullview_frame.jpeg?date=0124)

**lalserver特性图：**

![lalserver特性图](https://pengrl.com/lal/_media/lal_feature.jpeg?date=0124)

了解更多请访问：

* lal github地址: https://github.com/cfeeling/lal
* lal 官方文档: https://pengrl.com/lal
  * **/lalserver/**
    * [简介](https://pengrl.com/lal/#/LALServer.md)
    * [编译、运行、体验功能](https://pengrl.com/lal/#/QuickStart.md)
    * [配置文件说明](https://pengrl.com/lal/#/ConfigBrief.md)
    * [HTTP API接口](https://pengrl.com/lal/#/HTTPAPI.md)
    * [HTTP Notify(Callback/Webhook)事件回调](https://pengrl.com/lal/#/HTTPNotify.md)
  * [Demo简介](https://pengrl.com/lal/#/DEMO.md)
  * [Changelog修改记录](https://pengrl.com/lal/#/CHANGELOG.md)
  * [github star趋势图](https://pengrl.com/lal/#/StarChart.md)
  * [第三方依赖](https://pengrl.com/lal/#/ThirdDeps.md)
  * [联系作者](https://pengrl.com/lal/#/Author.md)
  * **/技术文档/**
    * [常见推拉流客户端使用方式](https://pengrl.com/lal/#/CommonClient.md)
    * [连接类型之session pub/sub/push/pull](https://pengrl.com/lal/#/Session.md)
    * [rtmp url，以及vhost](https://pengrl.com/lal/#/RTMPURLVhost.md)
    * [ffplay播放rtsp花屏](https://pengrl.com/lal/#/RTSPFFPlayBlur.md)
    * [FAQ](https://pengrl.com/lal/#/FAQ.md)
  * **/待整理/**
    * [性能测试](https://pengrl.com/lal/#/Test.md)
  * [图稿](https://pengrl.com/lal/#/Drawing.md)

联系作者：

- email：191201771@qq.com
- 微信：q191201771
- QQ：191201771
- 微信群： 加我微信好友后，告诉我拉你进群
- QQ群： 1090510973
