package main

import (
	"flag"
	"github.com/q191201771/lal/pkg/rtmp"
	"github.com/q191201771/nezha/pkg/errors"
	"github.com/q191201771/nezha/pkg/log"
	"os"
	"time"
)

type Obs struct {
}

func (obs Obs) ReadRTMPAVMsgCB(header rtmp.Header, timestampAbs int, message []byte) {
	log.Infof("%+v, abs ts=%d", header, timestampAbs)
}

func main() {
	url := parseFlag()
	var obs Obs
	session := rtmp.NewPullSession(obs, 2000)
	err := session.Pull(url)
	errors.PanicIfErrorOccur(err)
	time.Sleep(1 * time.Hour)
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
