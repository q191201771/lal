// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package base

type ISessionStat interface {
	// 周期性调用该函数，用于计算bitrate
	//
	// @param intervalSec 距离上次调用的时间间隔，单位毫秒
	UpdateStat(intervalSec int)

	// 获取session状态
	//
	// @return 注意，结构体中的`Bitrate`的值由最近一次`func UpdateStat`调用计算决定，其他值为当前最新值
	GetStat() StatSession

	// 周期性调用该函数，判断是否有读取、写入数据
	// 注意，判断的依据是，距离上次调用该函数的时间间隔内，是否有读取、写入数据
	// 注意，不活跃，并一定是链路或网络有问题，也可能是业务层没有写入数据
	//
	// @return readAlive  读取是否获取
	// @return writeAlive 写入是否活跃
	IsAlive() (readAlive, writeAlive bool)
}

type ISessionURLContext interface {
	AppName() string
	StreamName() string
	RawQuery() string
}

// TODO chef: rtmp.ClientSession修改为BaseClientSession更好些

//
// | .          | rtmp pub          | rtmp sub          | rtmp push                 | rtmp pull                 |
// | -          | -                 | -                 | -                         | -                         |
// | file       | server_session.go | server_session.go | client_push_session.go    | client_pull_session.go    |
// | struct     | ServerSession     | ServerSession     | PushSession/ClientSession | PullSession/ClientSession |
//
//
// | .          | rtsp pub                                      | rtsp sub                                      | rtsp pull                 | rtsp push |
// | -          | -                                             | -                                             | -                         | -         |
// | file       | server_pub_session.go                         | server_sub_session.go                         | client_pull_session.go    | client_push_session.go |
// | struct     | PubSession/ServerCommandSession/BaseInSession | SubSession/ServerCommandSession | PullSession/BaseInSession | PushSession |
//
//
// | .          | flv sub               | flv pull               |
// | -          | -                     | -                      |
// | file       | server_sub_session.go | client_pull_session.go |
// | struct     | SubSession            | PullSession            |
//
//
// | .          | ts sub                |
// | -          | -                     |
// | file       | server_sub_session.go |
// | struct     | SubSession            |
//
//
// | .                 | rtmppub | rtsppub | rtmpsub | flvsub | tssub | rtspsub | - | rtmppush | rtmppull | flvpull | rtsppull | hls |
// | -                 | -       | -       | -       | -      | -     | -       | - | -        | -        | -       | -        |     |
// | x                 | x       | x       | x       | x      | x     | x       | - | x        | x        | x       | x        |     |
// | UniqueKey<all>    | √       | √       | √       | √      | √     | √       | - | x√       | x√       | √       | √        |     |

// | AppName()<all>    | √       | √       | √       | √      | √     | √       | - | √        | √        | √       | √        |     |
// | StreamName()<all> | √       | √       | √       | √      | √     | √       | - | √        | √        | √       | √        |     |
// | RawQuery()<all>   | √       | √       | √       | √      | √     | √       | - | √        | √        | √       | √        |     |

// | GetStat()<all>    | √       | √       | √       | √      | √     | √       | - | √        | √        | √       | √        |     |
// | UpdateStat()<all> | √       | √       | √       | √      | √     | √       | - | √        | √        | √       | √        |     |
// | IsAlive()<all>    | √       | √       | √       | √      | √     | √       | - | √        | √        | √       | √        |     |

// | RunLoop()         | √       | x√      | √       | √      | √     | x&√     | - | x        | x        | x       | x        |     |
// | Dispose()         | √       | √       | √       | √      | √     | √       | - | √        | √        | √       | √        |     |

// | RemoteAddr()      | √       | x       | √       | √      | x     | x       | - | x        | x        | x       | x        |     |
// | SingleConn        | √       | x       | √       | √      | √     | √       | - | √        | √        | √       | x        |     |
//
// | Opt.PullTimeoutMS | -       | -       | -       | -      | -     | -       | - | -        | x        | √       | √        |     |
// | Wait()            | -       | -       | -       | -      | -     | -       | - | -        | √        | √       | √        |     |
//
// Dispose由外部调用，表示主动关闭正常的session
// 外部调用Dispose后，不应继续使用该session
// Dispose后，RunLoop结束阻塞
//
// 对端关闭，或session内部关闭也会导致RunLoop结束阻塞
//
// RunLoop结束阻塞后，可通知上层，告知session生命周期结束
//
// ---
//
// 对于rtsp.PushSession和rtsp.PullSession
// Push()或Pull成功后，可调用Dispose()主动关闭session
// 当对端关闭导致Wait()触发时，也需要调用Dispose()
//
// 对于rtsp.PubSession和rtsp.SubSession
// ServerCommandSession通知上层，上层调用session的Dispose()
// 当然，session也支持主动调用Dispose()
