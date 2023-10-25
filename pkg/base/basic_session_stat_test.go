package base

import (
	"testing"
)

func TestNewBasicSessionStat(t *testing.T) {
	type args struct {
		sessionType SessionType
		remoteAddr  string
	}
	tests := []struct {
		name string
		args args
		want BasicSessionStat
	}{
		{
			name: "rtmp_server",
			args: args{
				sessionType: SessionTypeRtmpServerSession,
			},
			want: BasicSessionStat{
				stat: StatSession{
					SessionId: GenUkRtmpServerSession(),
					BaseType:  SessionBaseTypePubSubStr,
					Protocol:  SessionProtocolRtmpStr,
				},
			},
		},
		{
			name: "rtmp_push",
			args: args{
				sessionType: SessionTypeRtmpPush,
			},
			want: BasicSessionStat{
				stat: StatSession{
					SessionId: GenUkRtmpPushSession(),
					BaseType:  SessionBaseTypePushStr,
					Protocol:  SessionProtocolRtmpStr,
				},
			},
		},
		{
			name: "rtmp_pull",
			args: args{
				sessionType: SessionTypeRtmpPull,
			},
			want: BasicSessionStat{
				stat: StatSession{
					SessionId: GenUkRtmpPullSession(),
					BaseType:  SessionBaseTypePullStr,
					Protocol:  SessionProtocolRtmpStr,
				},
			},
		},
		{
			name: "rtsp_pub",
			args: args{
				sessionType: SessionTypeRtspPub,
			},
			want: BasicSessionStat{
				stat: StatSession{
					SessionId: GenUkRtspPubSession(),
					BaseType:  SessionBaseTypePubStr,
					Protocol:  SessionProtocolRtspStr,
				},
			},
		},
		{
			name: "rtsp_sub",
			args: args{
				sessionType: SessionTypeRtspSub,
			},
			want: BasicSessionStat{
				stat: StatSession{
					SessionId: GenUkRtspSubSession(),
					BaseType:  SessionBaseTypeSubStr,
					Protocol:  SessionProtocolRtspStr,
				},
			},
		},
		{
			name: "rtsp_push",
			args: args{
				sessionType: SessionTypeRtspPush,
			},
			want: BasicSessionStat{
				stat: StatSession{
					SessionId: GenUkRtspPushSession(),
					BaseType:  SessionBaseTypePushStr,
					Protocol:  SessionProtocolRtspStr,
				},
			},
		},
		{
			name: "rtsp_pull",
			args: args{
				sessionType: SessionTypeRtspPull,
			},
			want: BasicSessionStat{
				stat: StatSession{
					SessionId: GenUkRtspPullSession(),
					BaseType:  SessionBaseTypePullStr,
					Protocol:  SessionProtocolRtspStr,
				},
			},
		},
		{
			name: "flv_sub",
			args: args{
				sessionType: SessionTypeFlvSub,
			},
			want: BasicSessionStat{
				stat: StatSession{
					SessionId: GenUkFlvSubSession(),
					BaseType:  SessionBaseTypeSubStr,
					Protocol:  SessionProtocolFlvStr,
				},
			},
		},
		{
			name: "ps_pub",
			args: args{
				sessionType: SessionTypePsPub,
			},
			want: BasicSessionStat{
				stat: StatSession{
					SessionId: GenUkPsPubSession(),
					BaseType:  SessionBaseTypePubStr,
					Protocol:  SessionProtocolPsStr,
				},
			},
		},
		{
			name: "ts_sub",
			args: args{
				sessionType: SessionTypeTsSub,
			},
			want: BasicSessionStat{
				stat: StatSession{
					SessionId: GenUkTsSubSession(),
					BaseType:  SessionBaseTypeSubStr,
					Protocol:  SessionProtocolTsStr,
				},
			},
		},
		{
			name: "hls_sub",
			args: args{
				sessionType: SessionTypeHlsSub,
			},
			want: BasicSessionStat{
				stat: StatSession{
					SessionId: GenUkHlsSubSession(),
					BaseType:  SessionBaseTypeSubStr,
					Protocol:  SessionProtocolHlsStr,
				},
			},
		},
		{
			name: "customize_pub",
			args: args{
				sessionType: SessionTypeCustomizePub,
			},
			want: BasicSessionStat{
				stat: StatSession{
					SessionId: "",
					BaseType:  "",
					Protocol:  "",
				},
			},
		},
		{
			name: "other",
			args: args{
				sessionType: -1,
			},
			want: BasicSessionStat{
				stat: StatSession{
					SessionId: "",
					BaseType:  "",
					Protocol:  "",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewBasicSessionStat(tt.args.sessionType, tt.args.remoteAddr)
			if got.stat.Protocol != tt.want.stat.Protocol {
				t.Errorf("NewBasicSessionStat() Protocol = %s, want %s", got.stat.Protocol, tt.want.stat.Protocol)
			}
			if got.stat.BaseType != tt.want.stat.BaseType {
				t.Errorf("NewBasicSessionStat() BaseType = %s, want %s", got.stat.BaseType, tt.want.stat.BaseType)
			}
		})
	}
}
