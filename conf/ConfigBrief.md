```
{
  "# doc of config": "https://pengrl.com/lal/#/ConfigBrief", //. 配置文件对应的文档说明链接，在程序中没实际用途
  "conf_version": "0.2.2",                                   //. 配置文件版本号，业务方不应该手动修改，程序中会检查该版本
                                                             //  号是否与代码中声明的一致
  "rtmp": {
    "enable": true,           //. 是否开启rtmp服务的监听
    "addr": ":19350",         //. RTMP服务监听的端口，客户端向lalserver推拉流都是这个地址
    "gop_num": 2,             //. RTMP拉流的GOP缓存数量，加速流打开时间，但是可能增加延时
    "merge_write_size": 8192  //. 将小包数据合并进行发送，单位字节，提高服务器性能，但是可能造成卡顿
  },
  "default_http": {                       //. http监听相关的默认配置，如果hls, httpflv, httpts中没有单独配置以下配置项，
                                          //  则使用default_http中的配置
                                          //  注意，hls, httpflv, httpts服务是否开启，不由此处决定
    "http_listen_addr": ":8080",          //. HTTP监听地址
    "https_listen_addr": ":4433",         //. HTTPS监听地址
    "https_cert_file": "./conf/cert.pem", //. HTTPS的本地cert文件地址
    "https_key_file": "./conf/key.pem"    //. HTTPS的本地key文件地址
  },
  "httpflv": {
    "enable": true,          //. 是否开启HTTP-FLV服务的监听
    "enable_https": true,    //. 是否开启HTTPS-FLV监听
    "url_pattern": "/live/", //. 拉流url路由地址。默认值`/live/`，对应`/live/{streamName}.flv`
    "gop_num": 2             //.
  },
  "hls": {
    "enable": true,                  //. 是否开启HLS服务的监听
    "enable_https": true,    //. 是否开启HTTPS-FLV监听
    "url_pattern": "/hls/",          //. 拉流url路由地址，默认值`/hls/`，对应：
                                     //  - `/hls/{streamName}.m3u8` 或
                                     //    `/hls/{streamName}/playlist.m3u8` 或
                                     //    `/hls/{streamName}/record.m3u8`
                                     //  - `/hls/{streamName}/{streamName}-{timestamp}-{index}.ts` 或
                                     //    `/hls/{streamName}-{timestamp}-{index}.ts`
                                     //  注意，hls的url_pattern不能和httpflv、httpts的url_pattern相同
    "out_path": "/tmp/lal/hls/",     //. HLS文件保存根目录
    "fragment_duration_ms": 3000,    //. 单个TS文件切片时长，单位毫秒
    "fragment_num": 6,               //. m3u8文件列表中ts文件的数量
    "cleanup_mode": 1,               //. HLS文件清理模式：
                                     //  0 不删除m3u8+ts文件，可用于录制等场景
                                     //  1 在输入流结束后删除m3u8+ts文件
                                     //    注意，确切的删除时间是推流结束后的<fragment_duration_ms> * <fragment_num> * 2
                                     //    的时间点
                                     //    推迟一小段时间删除，是为了避免输入流刚结束，HLS的拉流端还没有拉取完
                                     //  2 推流过程中，持续删除过期的ts文件，只保留最近的<fragment_num> * 2个左右的ts文件
    "use_memory_as_disk_flag": false //. 是否使用内存取代磁盘，保存m3u8+ts文件
  },
  "httpts": {
    "enable": true,         //. 是否开启HTTP-TS服务的监听。注意，这并不是HLS中的TS，而是在一条HTTP长连接上持续性传输TS流
    "enable_https": true,   //. 是否开启HTTPS-FLV监听
    "url_pattern": "/live/" //. 拉流url路由地址。默认值`/live/`，对应`/live/{streamName}.flv`
  },
  "rtsp": {
    "enable": true, //. 是否开启rtsp服务的监听，目前只支持rtsp推流
    "addr": ":5544" //. rtsp推流地址
  },
  "record": {
    "enable_flv": true,                  //. 是否开启flv录制
    "flv_out_path": "/tmp/lal/flv/",     //. flv录制目录
    "enable_mpegts": true,               //. 是否开启mpegts录制。注意，此处是长ts文件录制，hls录制由上面的hls配置控制
    "mpegts_out_path": "/tmp/lal/mpegts" //. mpegts录制目录
  },
  "relay_push": {
    "enable": false, //. 是否开启中继转推功能，开启后，自身接收到的所有流都会转推出去
    "addr_list":[    //. 中继转推的对端地址，支持填写多个地址，做1对n的转推。格式举例 "127.0.0.1:19351"
    ]
  },
  "relay_pull": {
    "enable": false, //. 是否开启回源拉流功能，开启后，当自身接收到拉流请求，而流不存在时，会从其他服务器拉取这个流到本地
    "addr": ""       //. 回源拉流的地址。格式举例 "127.0.0.1:19351"
  },
  "http_api": {
    "enable": true, //. 是否开启HTTP API接口
    "addr": ":8083" //. 监听地址
  },
  "server_id": "1", //. 当前lalserver唯一ID。多个lalserver HTTP Notify同一个地址时，可通过该ID区分
  "http_notify": {
    "enable": true,                                              //. 是否开启HTTP Notify事件回调
    "update_interval_sec": 5,                                    //. update事件回调间隔，单位毫秒
    "on_server_start": "http://127.0.0.1:10101/on_server_start", //. 各事件HTTP Notify事件回调地址
    "on_update": "http://127.0.0.1:10101/on_update",
    "on_pub_start": "http://127.0.0.1:10101/on_pub_start",
    "on_pub_stop": "http://127.0.0.1:10101/on_pub_stop",
    "on_sub_start": "http://127.0.0.1:10101/on_sub_start",
    "on_sub_stop": "http://127.0.0.1:10101/on_sub_stop",
    "on_rtmp_connect": "http://127.0.0.1:10101/on_rtmp_connect"
  },
  "pprof": {
    "enable": true, //. 是否开启Go pprof web服务的监听
    "addr": ":8084" //. Go pprof web地址
  },
  "log": {
    "level": 1,                         //. 日志级别，0 trace, 1 debug, 2 info, 3 warn, 4 error, 5 fatal
    "filename": "./logs/lalserver.log", //. 日志输出文件
    "is_to_stdout": true,               //. 是否打印至标志控制台输出
    "is_rotate_daily": true,            //. 日志按天翻滚
    "short_file_flag": true,            //. 日志末尾是否携带源码文件名以及行号的信息
    "assert_behavior": 1                //. 日志断言的行为，1 只打印错误日志 2 打印并退出程序 3 打印并panic
  }
}
```
