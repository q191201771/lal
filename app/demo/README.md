`/app/demo`示例程序功能简介：

| demo | push rtmp | pull rtmp | pull httpflv | 说明 |
| - | - | - | - | - |
| pushrtmp     | ✔ | . | . | RTMP推流客户端，读取本地FLV文件，使用RTMP协议推送出去 |
| pullrtmp     |   | ✔ | . | RTMP拉流客户端，从远端服务器拉取RTMP流，存储为本地FLV文件 |
| pullrtmp2hls | . | ✔ | . | 从远端服务器拉取RTMP流，存储为本地m3u8+ts文件 |
| pullhttpflv  | . | . | ✔ | HTTP-FLV拉流客户端 |
| analyseflv   | . | . | ✔ | 拉取HTTP-FLV流，并进行分析 |

（更具体的功能参加各源码文件的头部说明）
