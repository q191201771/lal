package main

import (
	"flag"
	"github.com/q191201771/lal/pkg/rtmp"
	"github.com/q191201771/nezha/pkg/log"
	"os"
)

type Obs struct {
}

func (obs Obs) ReadRTMPAVMsgCB(header rtmp.Header, timestampAbs uint32, message []byte) {
	log.Infof("%+v, abs ts=%d", header, timestampAbs)
}

func main() {
	url := parseFlag()
	var obs Obs
	session := rtmp.NewPullSession(obs, rtmp.PullSessionTimeout{
		ConnectTimeoutMS: 3000,
		PullTimeoutMS:    5000,
		ReadAVTimeoutMS:  10000,
	})
	err := session.Pull(url)
	log.FatalIfErrorNotNil(err)
	err = session.WaitLoop()
	log.FatalIfErrorNotNil(err)
}

func parseFlag() string {
	url := flag.String("i", "", "specify rtmp url")
	flag.Parse()
	if *url == "" {
		flag.Usage()
		os.Exit(1)
	}
	return *url
}
