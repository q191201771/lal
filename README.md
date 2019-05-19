lal - Go语言流媒体服务器

#### 编译和运行

```
$go get -u github.com/q191201771/lal
# cd into lal
$go build

# ./lal -c <配置文件> -l <日志配置文件>，比如：
$./lal -c conf/lal.conf.json -l conf/log.dev.xml
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

#### roadmap

正在实现rtmp服务器部分
