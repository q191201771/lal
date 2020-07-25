### 性能测试

测试场景一：持续推送n路RTMP流至lalserver（没有拉流）

| 推流数量 | CPU占用 | 内存占用（RES） |
| - | - | - |
| 1000 | （占单个核的）16% | 104MB |

测试场景二：持续推送1路RTMP流至lalserver，使用RTMP协议从lalserver拉取n路流

| 拉流数量 | CPU占用 | 内存占用（RES） |
| - | - | - |
| 1000 | （占单个核的）30% | 120MB |

测试场景三： 持续推送n路RTMP流至lalserver，使用RTMP协议从lalserver拉取n路流（推拉流为1对1的关系）

| 推流数量 | 拉流数量 | CPU占用 | 内存占用（RES） |
| - | - | - | - |
| 1000 | 1000 | 125% | 464MB |

* 测试机：32核16G（lalserver服务器和压测工具同时跑在这一个机器上）
* 压测工具：lal中的 `/app/demo/pushrtmp` 以及 `/app/demo/pullrtmp`
* 推流码率：使用`srs-bench`中的FLV文件，大概200kbps
* lalserver版本：基于 git commit: xxx

*由于测试机是台共用的机器，上面还跑了许多其他服务，这里列的只是个粗略的数据，还待做更多的性能分析以及优化。如果你对性能感兴趣，欢迎进行测试并将结果反馈给我。*

性能和可读，有时是矛盾的，存在取舍。我会保持思考，尽量平衡好两者。

### 测试过的第三方客户端

```
RTMP推流端：
- OBS 21.0.3(macos)
- OBS 24.0.3(win10 64 bit)
- ffmpeg 3.4.2(macos)
- srs-bench (macos srs项目配套的一个压测工具)
- pushrtmp (macos lal demo中的RTMP推流客户端)

RTMP / HTTP-FLV 拉流端：
- VLC 3.0.8(macos 10.15.1)
- VLC 3.0.8(win10)
- MPV 0.29.1(macos)
- ffmpeg 3.4.2(macos)
- srs-bench (macos srs项目配套的一个压测工具)
```
