package main

import (
	"flag"
	"github.com/q191201771/lal/pkg/httpflv"
	log "github.com/q191201771/naza/pkg/nazalog"
	"os"
)

func main() {
	url := parseFlag()
	session := httpflv.NewPullSession(httpflv.PullSessionConfig{
		ConnectTimeoutMS: 0,
		ReadTimeoutMS:    0,
	})
	err := session.Pull(url, func(tag *httpflv.Tag) {
		log.Infof("ReadFlvTagCB. %+v %t %t", tag.Header, tag.IsAVCKeySeqHeader(), tag.IsAVCKeyNalu())
	})
	if err != nil {
		log.Error(err)
		return
	}
}

func parseFlag() string {
	url := flag.String("i", "", "specify http-flv url")
	flag.Parse()
	if *url == "" {
		flag.Usage()
		os.Exit(1)
	}
	return *url
}
