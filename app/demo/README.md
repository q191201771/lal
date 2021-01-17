`/app/demo`示例程序功能简介：

| demo              | push rtmp | push rtsp | pull rtmp | pull httpflv | pull rtsp | 说明 |  
| -                 | -         | -         | -         | -            | -         | -   |  
| pushrtmp          | ✔         | .         | .         | .            | .         | RTMP推流客户端；压力测试工具 |  
| pullrtmp          | .         | .         | ✔         | .            | .         | RTMP拉流客户端；压力测试工具 |  
| pullrtmp2hls      | .         | .         | ✔         | .            | .         | 从远端服务器拉取RTMP流，存储为本地m3u8+ts文件 |  
| pullhttpflv       | .         | .         | .         | ✔            | .         | HTTP-FLV拉流客户端 |  
| pullrtsp          | .         | .         | .         | .            | ✔         | RTSP拉流客户端 |  
| pullrtsp2pushrtsp | .         | ✔         | .         | .            | ✔         | RTSP拉流，并使用RTSP转推出去 |  
| pullrtsp2pushrtmp | ✔         | .         | .         | .            | ✔         | RTSP拉流，并使用RTMP转推出去 |  
| analyseflv        | .         | .         | .         | ✔            | .         | 拉取HTTP-FLV流，并进行分析 |  

.

| demo       | 说明 |
| dispatch   | 简单演示如何实现一个简单的调度服务，使得多个lalserver节点可以组成一个集群 |
| flvfile2es | 将本地FLV文件分离成H264/AVC和AAC的ES流文件 |
| modflvfile | 修改flv文件的一些信息（比如某些tag的时间戳）后另存文件 |

（更具体的功能参加各源码文件的头部说明）
