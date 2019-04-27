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
  "httpflv": {
    "sub_listen_addr": ":8080", // http-flv拉流地址
    "pull_addr": "pull.xxx.com", // 如果设置，则当客户端连接lal拉流且流不存在时，lal会使用http-flv去该域名回
                                 // 源拉流至本地再转发
    "pull_connect_timeout": 2, // 回源连接超时时间
    "pull_read_timeout": 20, // 回源读取数据超时时间
    "sub_idle_timeout": 10, // 往客户端发送数据时的超时时间
    "stop_pull_while_no_sub_timeout": 5, // 回源的流超过多长时间没有客户端播放，则关闭回源的流
    "gop_cache_num": 2 // gop缓存个数，如果设置为0，则只缓存seq header，不缓存gop数据
  }
}
```

TODO 日志配置文件说明

#### 依赖

* cihub/seelog
