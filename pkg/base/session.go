// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package base

//
// | .          | rtmp pub          | rtsp pub              | rtmp sub          | flv sub               | ts sub                | rtsp sub              | rtmp push              | rtmp pull              | flv pull               | rtsp pull |
// | -          | -                 | -                     | -                 | -                     | -                     | -                     | -                      | -                      | -                      | - |
// | file       | server_session.go | server_pub_session.go | server_session.go | server_sub_session.go | server_sub_session.go | server_sub_session.go | client_push_session.go | client_pull_session.go | client_pull_session.go | client_pull_session.go |
// | struct     | ServerSession     | PubSession            | ServerSession     | SubSession            | SubSession            | SubSession            | PushSession            | PullSession            | PullSession            | PullSession |
//
//
//
// | .            | all | rtmppub | rtsppub | rtmpsub | flvsub | tssub | rtspsub | rtmppush | rtmppull | flvpull | rtsppull |
// | -            | -   | -       | -       | -       | -      | -     | -       | -        | -        | -       | - |
// | UniqueKey    | √   | √       | √       | √       | √      | √     | √       | √        | √        | √       | x |
// | StreamName   | x   | √       | √       | √       | √      | √     | √       | √        | √        | x       | x |
// | RunLoop()    | x   | √       | x       | √       | √      | √     | x       | x        | x        | x       | x |
// | Dispose()    | x   | √       | √       | √       | √      | √     | x       | √        | √        | √       | x |
// | GetStat()    | x   | √       | √       | √       | √      | √     | x       | x        | √        | x       | x |
// | UpdateStat() | x   | √       | √       | √       | √      | √     | x       | x        | √        | x       | x |
// | IsAlive()    | x   | √       | √       | √       | √      | √     | x       | x        | √        | x       | x |
// | SingleConn   | x   | √       | x       | √       | √      | √     | √       | √        | √        | √       | x |
// | RemoteAddr() | x   | √       | x       | √       | √      | x     | x       | x        | x        | x       | x |
//
// Dispose由外部调用，表示主动关闭正常的session
// 外部调用Dispose后，不应继续使用该session
// Dispose后，RunLoop结束阻塞
//
// 对端关闭，或session内部关闭也会导致RunLoop结束阻塞
//
// RunLoop结束阻塞后，可通知上层，告知session生命周期结束
