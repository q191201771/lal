package httpflv

type Config struct {
	SubListenAddr             string `json:"sub_listen_addr"`
	PullAddr                  string `json:"pull_addr"`
	PullConnectTimeout        int64  `json:"pull_connect_timeout"`
	PullReadTimeout           int64  `json:"pull_read_timeout"`
	SubIdleTimeout            int64  `json:"sub_idle_timeout"`
	StopPullWhileNoSubTimeout int64  `json:"stop_pull_while_no_sub_timeout"`
	GopCacheNum               int    `json:"gop_cache_num"`
}
