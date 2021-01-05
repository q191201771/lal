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

（更具体的功能参加各源码文件的头部说明）
