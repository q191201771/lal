package main

import (
	"flag"
	"github.com/q191201771/lal/pkg/rtmp"
	"github.com/q191201771/nezha/pkg/errors"
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
	session := rtmp.NewPullSession(obs, 2000)
	err := session.Pull(url)
	errors.PanicIfErrorOccur(err)
	err = session.WaitLoop()
	errors.PanicIfErrorOccur(err)
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
