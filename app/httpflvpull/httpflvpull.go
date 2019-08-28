package main

import (
	"flag"
	"github.com/q191201771/lal/pkg/httpflv"
	"github.com/q191201771/nezha/pkg/log"
	"os"
)

type Obs struct {
}

func (obs *Obs) ReadHTTPRespHeaderCB() {
	log.Info("ReadHTTPRespHeaderCB")
}

func (obs *Obs) ReadFlvHeaderCB(flvHeader []byte) {
	log.Info("ReadFlvHeaderCB")
}

func (obs *Obs) ReadFlvTagCB(tag *httpflv.Tag) {
	log.Infof("ReadFlvTagCB %+v %t %t", tag.Header, tag.IsAVCKeySeqHeader(), tag.IsAVCKeyNalu())
}

func main() {
	url := parseFlag()
	var obs Obs
	session := httpflv.NewPullSession(&obs, 0, 0)
	err := session.Pull(url)
	if err != nil {
		log.Error(err)
		return
	}
	err = session.RunLoop()
	log.Error(err)
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
