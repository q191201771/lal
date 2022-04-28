# lalserver 配置文件说明

```
{
  "# doc of config": "https://pengrl.com/lal/#/ConfigBrief", //. 配置文件对应的文档说明链接，在程序中没实际用途
  "conf_version": "0.2.9",                                   //. 配置文件版本号，业务方不应该手动修改，程序中会检查该版本
                                                             //  号是否与代码中声明的一致
  "rtmp": {
    "enable": true,                      //. 是否开启rtmp服务的监听
                                         //  注意，配置文件中控制各协议类型的enable开关都应该按需打开，避免造成不必要的协议转换的开销
    "addr": ":1935",                     //. RTMP服务监听的端口，客户端向lalserver推拉流都是这个地址
    "gop_num": 0,                        //. RTMP拉流的GOP缓存数量，加速流打开时间，但是可能增加延时
                                         //. 如果为0，则不使用缓存发送
    "merge_write_size": 0,               //. 将小包数据合并进行发送，单位字节，提高服务器性能，但是可能造成卡顿
                                         //  如果为0，则不合并发送
    "add_dummy_audio_enable": false,     //. 是否开启动态检测添加静音AAC数据的功能
                                         //  如果开启，rtmp pub推流时，如果超过`add_dummy_audio_wait_audio_ms`时间依然没有
                                         //  收到音频数据，则会自动为这路流叠加AAC的数据
    "add_dummy_audio_wait_audio_ms": 150 //. 单位毫秒，具体见`add_dummy_audio_enable`
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
    "url_pattern": "/",      //. 拉流url路由路径地址。默认值为`/`，表示不受限制，路由地址可以为任意路径地址。
                             //  如果设置为`/live/`，则只能从`/live/`路径下拉流，比如`/live/test110.flv`
    "gop_num": 0             //. 见rtmp.gop_num
  },
  "hls": {
    "enable": true,                  //. 是否开启HLS服务的监听
    "enable_https": true,            //. 是否开启HTTPS-HLS监听
                                     //
    "url_pattern": "/hls/",          //. 拉流url路由地址，默认值`/hls/`，对应的HLS(m3u8)拉流url地址：
                                     //  - `/hls/{streamName}.m3u8`
                                     //  - `/hls/{streamName}/playlist.m3u8`
                                     //  - `/hls/{streamName}/record.m3u8`
                                     //
                                     //  playlist.m3u8文件对应直播hls，列表中只保存<fragment_num>个ts文件名称，会持续增
                                     //  加新生成的ts文件，并去除过期的ts文件
                                     //  record.m3u8文件对应录制hls，列表中会保存从第一个ts文件到最新生成的ts文件，会持
                                     //  续追加新生成的ts文件
                                     //
                                     //  ts文件地址备注如下：
                                     //  - `/hls/{streamName}/{streamName}-{timestamp}-{index}.ts` 或
                                     //    `/hls/{streamName}-{timestamp}-{index}.ts`
                                     //
                                     //  注意，hls的url_pattern不能和httpflv、httpts的url_pattern相同
                                     //
    "out_path": "./lal_record/hls/", //. HLS的m3u8和文件的输出根目录
    "fragment_duration_ms": 3000,    //. 单个TS文件切片时长，单位毫秒
    "fragment_num": 6,               //. playlist.m3u8文件列表中ts文件的数量
                                     //
    "delete_threshold": 6,           //. ts文件的删除时机
                                     //  注意，只在配置项`cleanup_mode`为2时使用
                                     //  含义是只保存最近从playlist.m3u8中移除的ts文件的个数，更早过期的ts文件将被删除
                                     //  如果没有，默认值取配置项`fragment_num`的值
                                     //  注意，该值应该不小于1，避免删除过快导致播放失败
                                     //
    "cleanup_mode": 1,               //. HLS文件清理模式：
                                     //
                                     //  0 不删除m3u8+ts文件，可用于录制等场景
                                     //
                                     //  1 在输入流结束后删除m3u8+ts文件
                                     //    注意，确切的删除时间点是推流结束后的
                                     //    `fragment_duration_ms * (fragment_num + delete_threshold)`
                                     //    推迟一小段时间删除，是为了避免输入流刚结束，HLS的拉流端还没有拉取完
                                     //
                                     //  2 推流过程中，持续删除过期的ts文件，只保留最近的
                                     //    `delete_threshold + fragment_num + 1`
                                     //    个左右的ts文件
                                     //    并且，在输入流结束后，也会执行清理模式1的逻辑
                                     //
                                     //  注意，record.m3u8只在0和1模式下生成
                                     //
    "use_memory_as_disk_flag": false //. 是否使用内存取代磁盘，保存m3u8+ts文件
                                     //  注意，使用该模式要注意内存容量。一般来说不应该搭配`cleanup_mode`为0或1使用
  },
  "httpts": {
    "enable": true,         //. 是否开启HTTP-TS服务的监听。注意，这并不是HLS中的TS，而是在一条HTTP长连接上持续性传输TS流
    "enable_https": true,   //. 是否开启HTTPS-TS监听
    "url_pattern": "/",     //. 拉流url路由路径地址。默认值为`/`，表示不受限制，路由地址可以为任意路径地址。
                            //  如果设置为`/live/`，则只能从`/live/`路径下拉流，比如`/live/test110.ts`
    "gop_num": 0            //. 见rtmp.gop_num
  },
  "rtsp": {
    "enable": true,                 //. 是否开启rtsp服务的监听
    "addr": ":5544",                //. rtsp监听地址
    "out_wait_key_frame_flag": true //. rtsp发送数据时，是否等待视频关键帧数据再发送
                                    //
                                    //  该配置项主要决定首帧、花屏、音视频同步等问题
                                    //
                                    //  如果为true，则音频和视频都等待视频关键帧才开始发送。（也即，视频关键帧到来前，音频或视频全部丢弃不发送）
                                    //
                                    //  如果为false，则音频和视频都直接发送。（也即，音频和视频都不等待视频关键帧，都不等待任何数据）
                                    //
                                    //  注意，纯音频的流，如果该标志为true，理论上音频永远等不到视频关键帧，也即音频没有了发送机会，
                                    //  为了应对这个问题，lalserver会尽最大可能判断是否为纯音频的流，
                                    //  如果判断成功为纯音频的流，音频将直接发送。
                                    //  但是，如果有纯音频流，依然建议将该配置项设置为false
  },
  "record": {
    "enable_flv": true,                      //. 是否开启flv录制
    "flv_out_path": "./lal_record/flv/",     //. flv录制目录
    "enable_mpegts": true,                   //. 是否开启mpegts录制。注意，此处是长ts文件录制，hls录制由上面的hls配置控制
    "mpegts_out_path": "./lal_record/mpegts" //. mpegts录制目录
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
  "simple_auth": {                    // 鉴权文档见： https://pengrl.com/lal/#/auth
    "key": "q191201771",              // 私有key，计算md5鉴权参数时使用
    "dangerous_lal_secret": "pengrl", // 后门鉴权参数，所有的流可通过该参数值鉴权
    "pub_rtmp_enable": false,         // rtmp推流是否开启鉴权，true为开启鉴权，false为不开启鉴权
    "sub_rtmp_enable": false,         // rtmp拉流是否开启鉴权
    "sub_httpflv_enable": false,      // httpflv拉流是否开启鉴权
    "sub_httpts_enable": false,       // httpts拉流是否开启鉴权
    "pub_rtsp_enable": false,         // rtsp推流是否开启鉴权
    "sub_rtsp_enable": false,         // rtsp拉流是否开启鉴权
    "hls_m3u8_enable": true           // m3u8拉流是否开启鉴权
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
  },
  "debug": {
    "log_group_interval_sec": 30,          // 打印group调试日志的间隔时间，单位秒。如果为0，则不打印
    "log_group_max_group_num": 10,         // 最多打印多少个group
    "log_group_max_sub_num_per_group": 10  // 每个group最多打印多少个sub session
  }
}
```
